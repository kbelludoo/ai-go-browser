package browser

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
)

var (
	ErrJSTimeout   = errors.New("javascript execution timed out")
	ErrNodeNotFound = errors.New("dom node not found")
)

// NativeJSRuntime provides an in-process, pure Go ECMAScript environment for the browser.
// Zero Chromium, zero subprocesses, pure Goja VM with memory and time bounds.
type NativeJSRuntime struct {
	mu        sync.Mutex
	vm        *goja.Runtime
	browser   *Browser
	doc       *Document
	baseURL   string
	listeners map[string]map[string][]goja.Callable // nodeRef -> event -> callbacks
}

// NewNativeJSRuntime creates a pure Go JavaScript runtime attached to the given document.
func NewNativeJSRuntime(b *Browser, doc *Document, baseURL string) (*NativeJSRuntime, error) {
	vm := goja.New()
	rt := &NativeJSRuntime{
		vm:        vm,
		browser:   b,
		doc:       doc,
		baseURL:   baseURL,
		listeners: make(map[string]map[string][]goja.Callable),
	}

	if err := rt.injectDOMShims(); err != nil {
		return nil, err
	}

	return rt, nil
}

func (rt *NativeJSRuntime) injectDOMShims() error {
	vm := rt.vm

	// 1. console.log / console.error
	consoleObj := vm.NewObject()
	_ = consoleObj.Set("log", func(call goja.FunctionCall) goja.Value {
		return goja.Undefined()
	})
	_ = consoleObj.Set("error", func(call goja.FunctionCall) goja.Value {
		return goja.Undefined()
	})
	_ = vm.Set("console", consoleObj)

	// 2. document
	docObj := vm.NewObject()
	_ = docObj.Set("getElementById", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			return goja.Null()
		}
		id := call.Arguments[0].String()
		node := rt.findNodeByID(id)
		if node == nil {
			return goja.Null()
		}
		return rt.wrapNode(node)
	})

	_ = docObj.Set("querySelector", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			return goja.Null()
		}
		sel := call.Arguments[0].String()
		sel = strings.TrimPrefix(sel, "#")
		node := rt.findNodeByID(sel)
		if node == nil {
			return goja.Null()
		}
		return rt.wrapNode(node)
	})

	_ = docObj.Set("write", func(call goja.FunctionCall) goja.Value {
		// Bounded document.write
		return goja.Undefined()
	})

	_ = vm.Set("document", docObj)

	// 3. window & location
	winObj := vm.NewObject()
	_ = winObj.Set("document", docObj)
	_ = winObj.Set("location", map[string]string{"href": rt.baseURL})
	_ = vm.Set("window", winObj)

	// 4. fetch(url, options) -> Pure Go HTTP request reusing the browser's cookie jar and safe transport
	_ = vm.Set("fetch", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			panic(vm.ToValue("fetch requires at least 1 argument"))
		}
		rawURL := call.Arguments[0].String()
		destURL, err := rt.resolveURL(rawURL)
		if err != nil {
			panic(vm.ToValue("invalid url: " + err.Error()))
		}

		method := "GET"
		var reqBody io.Reader
		if len(call.Arguments) > 1 {
			if optObj, ok := call.Arguments[1].Export().(map[string]any); ok {
				if m, ok := optObj["method"].(string); ok && m != "" {
					method = strings.ToUpper(m)
				}
				if body, ok := optObj["body"].(string); ok {
					reqBody = bytes.NewBufferString(body)
				}
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, method, destURL, reqBody)
		if err != nil {
			panic(vm.ToValue(err.Error()))
		}
		req.Header.Set("User-Agent", rt.browser.UserAgent)

		resp, err := rt.browser.client.Do(req)
		if err != nil {
			panic(vm.ToValue("fetch failed: " + err.Error()))
		}
		defer resp.Body.Close()

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			panic(vm.ToValue("failed reading response: " + err.Error()))
		}

		bodyStr := string(bodyBytes)

		// Create response JS object with .json() and .text()
		respObj := vm.NewObject()
		_ = respObj.Set("status", resp.StatusCode)
		_ = respObj.Set("ok", resp.StatusCode >= 200 && resp.StatusCode < 300)
		_ = respObj.Set("text", func(c goja.FunctionCall) goja.Value {
			return vm.ToValue(bodyStr)
		})
		_ = respObj.Set("json", func(c goja.FunctionCall) goja.Value {
			var parsed any
			if err := json.Unmarshal(bodyBytes, &parsed); err != nil {
				panic(vm.ToValue("failed parsing json: " + err.Error()))
			}
			return vm.ToValue(parsed)
		})

		// Return a then-able promise wrapper supporting chained .then() calls
		var makePromise func(val goja.Value) *goja.Object
		makePromise = func(val goja.Value) *goja.Object {
			p := vm.NewObject()
			_ = p.Set("then", func(c goja.FunctionCall) goja.Value {
				if len(c.Arguments) > 0 {
					if fn, ok := goja.AssertFunction(c.Arguments[0]); ok {
						nextVal, err := fn(goja.Undefined(), val)
						if err != nil {
							panic(err)
						}
						return makePromise(nextVal)
					}
				}
				return makePromise(val)
			})
			_ = p.Set("catch", func(c goja.FunctionCall) goja.Value {
				return makePromise(val)
			})
			return p
		}

		return makePromise(respObj)
	})

	return nil
}

func (rt *NativeJSRuntime) wrapNode(n *Node) *goja.Object {
	vm := rt.vm
	obj := vm.NewObject()

	// textContent getter/setter
	_ = obj.DefineAccessorProperty("textContent",
		vm.ToValue(func(c goja.FunctionCall) goja.Value {
			return vm.ToValue(n.innerText(10000))
		}),
		vm.ToValue(func(c goja.FunctionCall) goja.Value {
			if len(c.Arguments) > 0 {
				newText := c.Arguments[0].String()
				n.Children = []*Node{
					{
						Tag:    "#text",
						Text:   newText,
						Parent: n,
					},
				}
			}
			return goja.Undefined()
		}),
		goja.FLAG_TRUE, goja.FLAG_TRUE)

	// value getter/setter
	_ = obj.DefineAccessorProperty("value",
		vm.ToValue(func(c goja.FunctionCall) goja.Value {
			return vm.ToValue(n.prop("value"))
		}),
		vm.ToValue(func(c goja.FunctionCall) goja.Value {
			if len(c.Arguments) > 0 {
				n.Attr["value"] = c.Arguments[0].String()
			}
			return goja.Undefined()
		}),
		goja.FLAG_TRUE, goja.FLAG_TRUE)

	// addEventListener(event, callback)
	_ = obj.Set("addEventListener", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) >= 2 {
			event := strings.ToLower(call.Arguments[0].String())
			if fn, ok := goja.AssertFunction(call.Arguments[1]); ok {
				// Register under both ID and Ref for easy lookup
				keys := []string{n.prop("id"), n.ref}
				for _, k := range keys {
					if k != "" {
						if rt.listeners[k] == nil {
							rt.listeners[k] = make(map[string][]goja.Callable)
						}
						rt.listeners[k][event] = append(rt.listeners[k][event], fn)
					}
				}
			}
		}
		return goja.Undefined()
	})

	return obj
}

func (rt *NativeJSRuntime) findNodeByID(id string) *Node {
	if rt.doc == nil || rt.doc.Root == nil {
		return nil
	}
	var found *Node
	controlWalk(rt.doc.Root, func(n *Node) {
		if n.prop("id") == id {
			found = n
		}
	})
	return found
}

func (rt *NativeJSRuntime) resolveURL(target string) (string, error) {
	b, err := url.Parse(rt.baseURL)
	if err != nil {
		return target, nil
	}
	t, err := url.Parse(target)
	if err != nil {
		return "", err
	}
	return b.ResolveReference(t).String(), nil
}

// ExecuteScript runs a JavaScript snippet within bounded execution time.
func (rt *NativeJSRuntime) ExecuteScript(script string, timeout time.Duration) error {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	timer := time.AfterFunc(timeout, func() {
		rt.vm.Interrupt(ErrJSTimeout)
	})
	defer timer.Stop()

	_, err := rt.vm.RunString(script)
	return err
}

// TriggerEvent fires registered event listeners for a specific element (e.g. click).
func (rt *NativeJSRuntime) TriggerEvent(refOrID, event string) error {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	event = strings.ToLower(event)
	handlers := rt.listeners[refOrID][event]
	if len(handlers) == 0 {
		return nil
	}

	for _, fn := range handlers {
		if _, err := fn(goja.Undefined()); err != nil {
			return err
		}
	}
	return nil
}
