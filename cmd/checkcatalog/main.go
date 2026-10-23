package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func main() {
	resp, err := http.Get("https://arena.ai/nextjs-api/model-catalog")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var arenas []struct {
		Arena string `json:"arena"`
		Count int    `json:"-"`
		Models []any `json:"models"`
	}
	json.Unmarshal(body, &arenas)

	fmt.Println("ARENAS FOUND:")
	for _, a := range arenas {
		fmt.Printf("- Arena: %q (Models: %d)\n", a.Arena, len(a.Models))
	}
}
