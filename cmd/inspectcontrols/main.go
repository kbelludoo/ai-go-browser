package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"ai-go-browser/browser"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	session, err := browser.LaunchStealth(ctx, browser.StealthConfig{
		Headless: false,
		Width:    1366,
		Height:   768,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()

	session.Navigate(ctx, "https://arena.ai/?mode=direct")
	time.Sleep(6 * time.Second)

	// Dump all interactive controls in the header / model selection bar
	inspectJS := `(() => {
		const all = Array.from(document.querySelectorAll('button, input, select, [role="button"], [role="combobox"]')).map(el => ({
			tag: el.tagName,
			text: el.innerText.trim(),
			placeholder: el.placeholder || '',
			ariaLabel: el.getAttribute('aria-label') || '',
			class: el.className
		}));
		return JSON.stringify(all);
	})()`

	res, _ := session.Evaluate(ctx, inspectJS)
	var str string
	json.Unmarshal(res, &str)
	fmt.Printf("CONTROLS:\n%s\n", str)
}
