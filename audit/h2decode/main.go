// Command h2decode decodes the HPACK header block captured by audit/h2_probe.py
// so the pseudo-header order the client actually sends is visible.
//
// Run: go run ./audit/h2decode /tmp/lab/h2.json
package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"golang.org/x/net/http2/hpack"
)

type capture struct {
	ALPN     string `json:"alpn"`
	Preface  bool   `json:"h2_preface"`
	Settings map[string]uint32
	Frames   []struct {
		Type  int    `json:"type"`
		Flags int    `json:"flags"`
		Len   int    `json:"len"`
		Hex   string `json:"hex"`
	} `json:"frames"`
}

func main() {
	path := "/tmp/lab/h2.json"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read:", err)
		os.Exit(1)
	}
	var c capture
	if err := json.Unmarshal(data, &c); err != nil {
		fmt.Fprintln(os.Stderr, "json:", err)
		os.Exit(1)
	}
	fmt.Printf("ALPN=%s h2_preface=%v\n", c.ALPN, c.Preface)
	fmt.Printf("SETTINGS=%v\n", c.Settings)
	for _, f := range c.Frames {
		if f.Type != 1 { // HEADERS
			continue
		}
		raw, _ := hex.DecodeString(f.Hex)
		dec := hpack.NewDecoder(4096, func(hf hpack.HeaderField) {
			fmt.Printf("  %s: %s\n", hf.Name, hf.Value)
		})
		if _, err := dec.Write(raw); err != nil {
			fmt.Println("hpack:", err)
		}
	}
}
