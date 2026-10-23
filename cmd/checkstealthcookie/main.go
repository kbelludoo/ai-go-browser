package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ai-go-browser/browser"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	b, err := browser.LaunchStealth(ctx, browser.StealthConfig{
		Headless: true,
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer b.Close()

	b.Navigate(ctx, "https://arena.ai/")
	time.Sleep(3 * time.Second)

	res, err := b.Evaluate(ctx, `(() => {
		return document.cookie;
	})()`)
	if err != nil {
		fmt.Printf("Eval err: %v\n", err)
		return
	}
	var c string
	json.Unmarshal(res, &c)
	fmt.Printf("Cookies from headless: %q\n", c)
}
