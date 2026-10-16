package browser

// Bounded optional JavaScript integration.
// Browser core stays 100% Go; a separate, optional Duktape worker evaluates
// classic inline scripts with a small, explicitly documented DOM shim.
// This is NOT a standards-compliant browser DOM or full modern JS engine.

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"
)

const maxInlineScripts = 16

type JSStats struct {
	Enabled          bool     `json:"enabled"`
	Attempted        int      `json:"attempted"`
	Executed         int      `json:"executed"`
	WorkerPeakRSSMiB float64  `json:"worker_peak_rss_mib,omitempty"`
	Changed          int      `json:"changed"`
	Errors           []string `json:"errors,omitempty"`
	TimedOut         bool     `json:"timed_out,omitempty"`
}

type jsNode struct {
	TextContent string `json:"textContent"`
	Value       string `json:"value,omitempty"`
	OnClick     string `json:"onclick,omitempty"`
}
type jsJob struct {
	Nodes   map[string]jsNode `json:"nodes"`
	Title   string            `json:"title"`
	URL     string            `json:"url"`
	Scripts []string          `json:"scripts"`
}
type jsReply struct {
	Nodes            map[string]jsNode   `json:"nodes"`
	Title            string              `json:"title"`
	Writes           []string            `json:"writes"`
	Appends          map[string][]string `json:"appends"`
	Errors           []string            `json:"errors,omitempty"`
	Executed         int                 `json:"executed"`
	WorkerPeakRSSMiB float64             `json:"worker_peak_rss_mib"`
}

type nodeRange struct {
	ID, Tag, OnClick, Value string
	Begin, End              int
	Text                    string
}
type domFrame struct {
	tag, id, onclick string
	begin            int
}

// scanScriptsAndNodes extracts inline scripts and id-bearing text nodes while
// respecting the raw text element behavior of <script>.
func scanScriptsAndNodes(raw string) ([]string, map[string]nodeRange) {
	var scripts []string
	nodes := make(map[string]nodeRange)
	stack := []domFrame{}
	pos := 0
	lower := strings.ToLower(raw)
	for pos < len(raw) {
		start := pos
		tok := nextToken(raw, &pos)
		if tok.tag == "" || strings.HasPrefix(tok.tag, "!") {
			continue
		}
		if tok.end {
			for i := len(stack) - 1; i >= 0; i-- {
				if stack[i].tag != tok.tag {
					continue
				}
				fr := stack[i]
				stack = stack[:i]
				if fr.id != "" && fr.begin <= start && len(nodes) < 256 {
					contents := raw[fr.begin:start]
					nodes[fr.id] = nodeRange{ID: fr.id, Tag: fr.tag, Begin: fr.begin, End: start, Text: visibleText(contents), OnClick: fr.onclick}
				}
				break
			}
			continue
		}
		if tok.tag == "script" && !tok.selfClose {
			idx := strings.Index(lower[pos:], "</script")
			if idx < 0 {
				break
			}
			end := pos + idx
			typ := strings.ToLower(strings.TrimSpace(tok.attrs["type"]))
			if tok.attrs["src"] == "" && (typ == "" || typ == "text/javascript" || typ == "application/javascript") && len(scripts) < maxInlineScripts {
				scripts = append(scripts, raw[pos:end])
			}
			pos = end // let nextToken consume </script>
			continue
		}
		if voidTag(tok.tag) && tok.attrs["id"] != "" && len(tok.attrs["id"]) <= 128 && len(nodes) < 256 {
			id := tok.attrs["id"]
			nodes[id] = nodeRange{ID: id, Tag: tok.tag, Begin: pos, End: pos, Value: tok.attrs["value"], OnClick: tok.attrs["onclick"]}
		}
		if !voidTag(tok.tag) && !tok.selfClose {
			id := strings.TrimSpace(tok.attrs["id"])
			if len(id) > 128 {
				id = ""
			}
			stack = append(stack, domFrame{tag: tok.tag, id: id, onclick: tok.attrs["onclick"], begin: pos})
		}
	}
	return scripts, nodes
}
func visibleText(raw string) string {
	pos := 0
	b := strings.Builder{}
	for pos < len(raw) {
		t := nextToken(raw, &pos)
		if t.tag == "" {
			b.WriteString(" ")
			b.WriteString(t.text)
		}
	}
	return norm(html.UnescapeString(b.String()))
}

var titleRegex = regexp.MustCompile(`(?is)<title\b[^>]*>.*?</title\s*>`)
var bodyEndRegex = regexp.MustCompile(`(?i)</body\s*>`)

func patchHTML(raw string, orig jsJob, ranges map[string]nodeRange, got jsReply) (string, int) {
	selected := []nodeRange{}
	for id, node := range got.Nodes {
		r, ok := ranges[id]
		if !ok || (r.Text == node.TextContent && len(got.Appends[id]) == 0) {
			continue
		}
		// Work only with existing id-bearing non-script elements.
		if r.Tag == "script" || r.Tag == "style" || r.Tag == "textarea" || r.Tag == "title" {
			continue
		}
		r.Text = node.TextContent
		// Store appended HTML separately; it must not be escaped as literal text.
		selected = append(selected, r)
	}
	// If both parent and child are changed, the outer change takes precedence.
	sort.Slice(selected, func(i, j int) bool { return selected[i].Begin < selected[j].Begin })
	keep := []nodeRange{}
	for _, r := range selected {
		inside := false
		for _, prior := range keep {
			if r.Begin >= prior.Begin && r.End <= prior.End {
				inside = true
				break
			}
		}
		if !inside {
			keep = append(keep, r)
		}
	}
	count := len(keep)
	for i := len(keep) - 1; i >= 0; i-- {
		r := keep[i]
		content := html.EscapeString(r.Text)
		// Append-only changes preserve existing nested markup.
		if old, ok := orig.Nodes[r.ID]; ok && old.TextContent == r.Text {
			content = raw[r.Begin:r.End]
		}
		if len(got.Appends[r.ID]) > 0 {
			content += strings.Join(got.Appends[r.ID], "")
		}
		raw = raw[:r.Begin] + content + raw[r.End:]
	}
	if got.Title != orig.Title {
		title := "<title>" + html.EscapeString(got.Title) + "</title>"
		if titleRegex.MatchString(raw) {
			raw = titleRegex.ReplaceAllLiteralString(raw, title)
		} else {
			raw = title + raw
		}
		count++
	}
	total := 0
	var writes []string
	for _, w := range got.Writes {
		if total+len(w) > 64*1024 {
			break
		}
		total += len(w)
		writes = append(writes, w)
	}
	if len(writes) > 0 {
		insert := strings.Join(writes, "")
		if loc := bodyEndRegex.FindStringIndex(raw); loc != nil {
			raw = raw[:loc[0]] + insert + raw[loc[0]:]
		} else {
			raw += insert
		}
		count += len(writes)
	}
	return raw, count
}

// jsSession owns a single page-scoped Duktape process. The JS VM remains
// alive between agent actions so event handlers and script variables persist.
// The worker has NO injected file/network APIs; the process is not an OS sandbox.
type jsSession struct {
	command *exec.Cmd
	input   io.WriteCloser
	replies chan []byte
	done    chan struct{}
}

func startJSSession(path string) (*jsSession, error) {
	cmd := exec.Command(path)
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		in.Close()
		return nil, err
	}
	cmd.Stderr = io.Discard
	if err = cmd.Start(); err != nil {
		in.Close()
		return nil, err
	}
	ss := &jsSession{command: cmd, input: in, replies: make(chan []byte, 1), done: make(chan struct{})}
	go func() {
		defer close(ss.done)
		scanner := bufio.NewScanner(out)
		scanner.Buffer(make([]byte, 4096), 2*1024*1024)
		for scanner.Scan() {
			bs := append([]byte(nil), scanner.Bytes()...)
			ss.replies <- bs
		}
		close(ss.replies)
	}()
	return ss, nil
}
func (s *jsSession) close() {
	if s == nil {
		return
	}
	_ = s.input.Close()
	if s.command.Process != nil {
		_ = s.command.Process.Kill()
	}
	<-s.done
	_ = s.command.Wait()
}
func (s *jsSession) exchange(req interface{}, timeout time.Duration) (jsReply, error) {
	var reply jsReply
	data, err := json.Marshal(req)
	if err != nil {
		return reply, err
	}
	if len(data) > 256*1024 {
		return reply, fmt.Errorf("JS request exceeds 256 KiB")
	}
	if _, err = s.input.Write(append(data, '\n')); err != nil {
		return reply, err
	}
	if timeout <= 0 {
		timeout = 750 * time.Millisecond
	}
	select {
	case raw, ok := <-s.replies:
		if !ok {
			return reply, fmt.Errorf("JS worker exited unexpectedly")
		}
		if err = json.Unmarshal(raw, &reply); err != nil {
			return reply, fmt.Errorf("invalid JS reply: %w", err)
		}
		return reply, nil
	case <-time.After(timeout):
		return reply, context.DeadlineExceeded
	}
}
func (b *Browser) closeJSSession() {
	if b.jsSession != nil {
		b.jsSession.close()
		b.jsSession = nil
	}
}

// runInlineJS opens a long-lived page-scoped worker instead of replaying scripts
// for every event. Its output is bounded; timed-out workers are killed.
func (b *Browser) runInlineJS(raw []byte, baseURL string) ([]byte, *JSStats) {
	b.closeJSSession()
	if b.JSWorker == "" {
		return raw, nil
	}
	stats := &JSStats{Enabled: true}
	scripts, ranges := scanScriptsAndNodes(string(raw))
	stats.Attempted = len(scripts)
	if len(scripts) == 0 {
		hasHandlers := false
		for _, n := range ranges {
			if n.OnClick != "" {
				hasHandlers = true
				break
			}
		}
		if !hasHandlers {
			return raw, stats
		}
	}
	page := Parse(raw, baseURL, 200, len(raw), b.MaxText)
	job := jsJob{Nodes: map[string]jsNode{}, Title: page.Title, URL: baseURL, Scripts: scripts}
	for id, r := range ranges {
		job.Nodes[id] = jsNode{TextContent: r.Text, Value: r.Value, OnClick: r.OnClick}
	}
	session, err := startJSSession(b.JSWorker)
	if err != nil {
		stats.Errors = []string{err.Error()}
		return raw, stats
	}
	result, err := session.exchange(job, b.JSTimeout)
	if err != nil {
		if err == context.DeadlineExceeded {
			stats.TimedOut = true
		}
		stats.Errors = []string{err.Error()}
		session.close()
		return raw, stats
	}
	b.jsSession = session
	stats.Errors = result.Errors
	stats.Executed = result.Executed
	stats.WorkerPeakRSSMiB = result.WorkerPeakRSSMiB
	output, n := patchHTML(string(raw), job, ranges, result)
	stats.Changed = n
	return []byte(output), stats
}

func (b *Browser) dispatchButton(ref string) error {
	if b.Document == nil {
		return fmt.Errorf("no page open")
	}
	node, ok := b.Document.buttons[ref]
	if !ok {
		return fmt.Errorf("button %q not found", ref)
	}
	id := node.prop("id")
	if id == "" {
		return fmt.Errorf("button %q needs an HTML id for JS event dispatch", ref)
	}
	if b.jsSession == nil {
		return fmt.Errorf("JavaScript session unavailable for button %q", ref)
	}
	result, err := b.jsSession.exchange(map[string]string{"cmd": "event", "id": id, "event": "click"}, b.JSTimeout)
	if err != nil {
		if err == context.DeadlineExceeded {
			b.lastJS = &JSStats{Enabled: true, TimedOut: true, Errors: []string{"JS click timed out"}}
		}
		b.closeJSSession()
		return err
	}
	stats := &JSStats{Enabled: true, Executed: result.Executed, Errors: result.Errors, WorkerPeakRSSMiB: result.WorkerPeakRSSMiB}
	scripts, ranges := scanScriptsAndNodes(string(b.rawHTML))
	_ = scripts
	job := jsJob{Nodes: map[string]jsNode{}, Title: b.Current.Title}
	for id, r := range ranges {
		job.Nodes[id] = jsNode{TextContent: r.Text, Value: r.Value, OnClick: r.OnClick}
	}
	output, n := patchHTML(string(b.rawHTML), job, ranges, result)
	stats.Changed = n
	d, err := ParseDocument([]byte(output), b.Current.URL, b.Current.Status, b.Current.Bytes)
	if err != nil {
		return err
	}
	// JS may update input.value without changing visible innerText.
	for id, node := range result.Nodes {
		if node.Value == "" {
			continue
		}
		var walk func(*Node)
		walk = func(n *Node) {
			if n.prop("id") == id && (n.Tag == "input" || n.Tag == "textarea" || n.Tag == "select") {
				n.setValue(node.Value)
				return
			}
			for _, child := range n.Children {
				walk(child)
			}
		}
		walk(d.Root)
	}
	p := d.Page(b.MaxText)
	b.rawHTML = []byte(output)
	b.Document = d
	b.Current = &p
	b.lastJS = stats
	if len(result.Errors) > 0 {
		return fmt.Errorf("JS event: %s", strings.Join(result.Errors, "; "))
	}
	return nil
}
