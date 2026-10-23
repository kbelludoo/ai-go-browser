package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ai-go-browser/browser"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fmt.Println("Launching stealth session...")
	s, err := browser.LaunchStealth(ctx, browser.StealthConfig{
		Headless: true,
		Width:    1366,
		Height:   768,
	})
	if err != nil {
		panic(err)
	}
	defer s.Close()

	fmt.Println("Navigating to https://arena.ai/ ...")
	if err := s.Navigate(ctx, "https://arena.ai/"); err != nil {
		panic(err)
	}

	fmt.Println("Waiting for page hydration and user session...")
	time.Sleep(10 * time.Second)

	checkJS := `(async () => {
		// Check cookies and session
		const cookies = document.cookie;
		let user = null;
		try {
			const res = await fetch('/api/me');
			user = await res.json();
		} catch(e) {
			user = { error: e.message };
		}
		return JSON.stringify({ cookies, user });
	})()`

	res, err := s.Evaluate(ctx, checkJS)
	if err != nil {
		panic(err)
	}

	var resStr string
	json.Unmarshal(res, &resStr)
	fmt.Println("SESSION STATUS:")
	fmt.Println(resStr)
}
