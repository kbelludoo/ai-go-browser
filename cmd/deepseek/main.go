// Command deepseek is a small DeepSeek Web client: log in once through a real
// browser (CDP or launched), then chat over plain HTTP + SSE with a WASM PoW.
//
//	deepseek login [--cdp-url http://127.0.0.1:9222]
//	deepseek chat "hello" [-model expert] [-thinking] [-search] [-conversation id]
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"time"

	"ai-go-browser/pkg/deepseek"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	var err error
	switch os.Args[1] {
	case "login":
		err = runLogin(ctx, os.Args[2:])
	case "renew":
		err = runRenew(ctx, os.Args[2:])
	case "chat", "ask":
		err = runChat(ctx, os.Args[2:])
	case "-h", "--help", "help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `deepseek — DeepSeek Web client (CDP auth + WASM PoW + HTTP/SSE)

Usage:
  deepseek login [flags]
  deepseek renew [flags]
  deepseek chat "prompt" [flags]

Login flags:
  --cdp-url URL     attach to Chrome started with --remote-debugging-port
  --profile DIR     persistent profile dir when launching a browser
  --browser PATH    browser executable to launch
  --headless        launch without a window
  --timeout DUR     how long to wait for manual sign-in (default 5m)
  --session FILE    session cache path (default ./session/session.json)

Renew flags (reuse an existing sign-in, no window):
  --profile DIR     persistent profile dir to read the session from
  --browser PATH    browser executable to launch headless
  --cdp-url URL     renew from an already-running browser
  --timeout DUR     how long to wait for the app to read the token (default 20s)
  --session FILE    session cache path

Chat flags:
  --session FILE    session cache path
  --model NAME      default|expert (first turn only)
  --thinking        enable DeepThink reasoning
  --search          enable web search
  --conversation ID resume a previous thread (prints its id after each reply)
`)
}

func runLogin(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("login", flag.ExitOnError)
	cdpURL := fs.String("cdp-url", os.Getenv("DEEPSEEK_CDP_URL"), "Chrome DevTools URL (e.g. http://127.0.0.1:9222)")
	profile := fs.String("profile", "", "persistent browser profile dir")
	browserPath := fs.String("browser", "", "browser executable path")
	headless := fs.Bool("headless", false, "launch without a window")
	timeout := fs.Duration("timeout", 5*time.Minute, "how long to wait for sign-in")
	sessionFile := fs.String("session", "", "session cache path")
	if err := fs.Parse(args); err != nil {
		return err
	}

	sess, err := deepseek.Login(ctx, deepseek.LoginOptions{
		CDPURL:      *cdpURL,
		ProfileDir:  *profile,
		BrowserPath: *browserPath,
		Headless:    *headless,
		Timeout:     *timeout,
		SessionFile: *sessionFile,
	})
	if err != nil {
		return err
	}
	file := *sessionFile
	if file == "" {
		file = deepseek.DefaultSessionFile()
	}
	fmt.Printf("session captured: token %s... (%d cookies) -> %s\n",
		first10(sess.Token), len(sess.Cookies), file)
	return nil
}

func runRenew(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("renew", flag.ExitOnError)
	cdpURL := fs.String("cdp-url", "", "renew from a running browser at this CDP URL")
	profile := fs.String("profile", "", "persistent browser profile dir to reuse")
	browserPath := fs.String("browser", "", "browser executable path")
	timeout := fs.Duration("timeout", 20*time.Second, "how long to wait for the app to read the token")
	sessionFile := fs.String("session", "", "session cache path")
	if err := fs.Parse(args); err != nil {
		return err
	}

	sess, err := deepseek.Renew(ctx, deepseek.RenewOptions{
		CDPURL:      *cdpURL,
		ProfileDir:  *profile,
		BrowserPath: *browserPath,
		Timeout:     *timeout,
		SessionFile: *sessionFile,
	})
	if err != nil {
		return err
	}
	file := *sessionFile
	if file == "" {
		file = deepseek.DefaultSessionFile()
	}
	fmt.Printf("session renewed: token %s... (%d cookies) -> %s\n",
		first10(sess.Token), len(sess.Cookies), file)
	return nil
}

func runChat(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("chat", flag.ExitOnError)
	sessionFile := fs.String("session", "", "session cache path")
	model := fs.String("model", "", "default|expert (first turn only)")
	thinking := fs.Bool("thinking", false, "enable DeepThink reasoning")
	search := fs.Bool("search", false, "enable web search")
	conversation := fs.String("conversation", "", "resume a previous thread")
	if err := fs.Parse(args); err != nil {
		return err
	}

	prompt, err := readPrompt(fs.Args())
	if err != nil {
		return err
	}

	sess, err := deepseek.GetSession(ctx, deepseek.LoginOptions{SessionFile: *sessionFile}, false)
	if err != nil {
		return err
	}
	client, err := deepseek.NewClient(ctx, sess)
	if err != nil {
		return err
	}
	defer client.Close()

	cid, err := client.StreamChat(ctx, prompt, deepseek.StreamOptions{
		ConversationID: *conversation,
		Model:          *model,
		Thinking:       *thinking,
		Search:         *search,
	}, func(chunk string) error {
		fmt.Print(chunk)
		return nil
	})
	fmt.Println()
	if err != nil {
		return err
	}
	if cid != "" {
		fmt.Fprintf(os.Stderr, "[conversation] %s\n", cid)
	}
	return nil
}

func readPrompt(args []string) (string, error) {
	if len(args) > 0 {
		return strings.Join(args, " "), nil
	}
	info, _ := os.Stdin.Stat()
	if info != nil && info.Mode()&os.ModeCharDevice != 0 {
		return "", fmt.Errorf("no prompt given (pass it as an argument or on stdin)")
	}
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", err
	}
	prompt := strings.TrimSpace(string(raw))
	if prompt == "" {
		return "", fmt.Errorf("empty prompt")
	}
	return prompt, nil
}

func first10(s string) string {
	if len(s) <= 10 {
		return s
	}
	return s[:10]
}
