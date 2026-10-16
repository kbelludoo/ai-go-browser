package browser

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestDecodeHTTPBodyLegacyCharsets(t *testing.T) {
	for _, tc := range []struct {
		name, contentType, encoding, want string
		raw                               []byte
	}{
		{"Windows1252", `text/html; charset="WINDOWS-1252"`, "windows-1252", "“Café €”", []byte{0x93, 'C', 'a', 'f', 0xe9, ' ', 0x80, 0x94}},
		{"ISO88591HTMLAlias", "text/html; charset=iso-8859-1", "windows-1252", "Café €", []byte{'C', 'a', 'f', 0xe9, ' ', 0x80}},
		{"ShiftJIS", "text/html; charset=Shift_JIS", "shift_jis", "日本語", []byte{0x93, 0xfa, 0x96, 0x7b, 0x8c, 0xea}},
		{"GBK", "text/plain; charset=GBK", "gbk", "中文€", []byte{0xd6, 0xd0, 0xce, 0xc4, 0x80}},
		{"UTF16LEHeader", "text/html; charset=utf-16le", "utf-16le", "Café 🌍", []byte{'C', 0, 'a', 0, 'f', 0, 0xe9, 0, ' ', 0, 0x3c, 0xd8, 0x0d, 0xdf}},
		{"UTF16BEHeader", "text/plain; charset=utf-16be", "utf-16be", "Café 🌍", []byte{0, 'C', 0, 'a', 0, 'f', 0, 0xe9, 0, ' ', 0xd8, 0x3c, 0xdf, 0x0d}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			name, got, err := DecodeHTTPBody(tc.contentType, tc.raw)
			if err != nil || name != tc.encoding || string(got) != tc.want || !utf8.Valid(got) {
				t.Fatalf("decoded = %q (%s), error=%v; want %q (%s)", got, name, err, tc.want, tc.encoding)
			}
		})
	}
}

func TestDecodeHTTPBodyBOMAndHeaderPrecedence(t *testing.T) {
	for _, tc := range []struct {
		contentType, encoding, want string
		raw                         []byte
	}{
		{"text/html; charset=windows-1252", "utf-8", "Café", append([]byte{0xef, 0xbb, 0xbf}, []byte("Café")...)},
		{"text/html; charset=utf-8", "utf-16le", "A🌍", []byte{0xff, 0xfe, 'A', 0, 0x3c, 0xd8, 0x0d, 0xdf}},
		{"text/plain; charset=utf-16le", "utf-16be", "A🌍", []byte{0xfe, 0xff, 0, 'A', 0xd8, 0x3c, 0xdf, 0x0d}},
		{"text/html; charset=utf-8", "utf-8", `<meta charset="windows-1252">Café`, []byte(`<meta charset="windows-1252">Café`)},
	} {
		name, got, err := DecodeHTTPBody(tc.contentType, tc.raw)
		if err != nil || name != tc.encoding || string(got) != tc.want {
			t.Fatalf("decoded = %q (%s), error=%v; want %q (%s)", got, name, err, tc.want, tc.encoding)
		}
		if bytes.HasPrefix(got, []byte{0xef, 0xbb, 0xbf}) {
			t.Fatal("decoded body retained leading BOM")
		}
	}
}

func TestDecodeHTTPBodyHTMLMetaAndTextDefault(t *testing.T) {
	meta := []byte(`<meta http-equiv="Content-Type" content="text/html; charset=Shift_JIS"><p>`)
	raw := append(meta, 0x93, 0xfa, 0x96, 0x7b)
	name, got, err := DecodeHTTPBody("text/html", raw)
	if err != nil || name != "shift_jis" || !strings.HasSuffix(string(got), "<p>日本") {
		t.Fatalf("HTML meta = %q (%s), error=%v", got, name, err)
	}
	// Text and JSON neither interpret meta tags nor apply HTML's CP1252 fallback.
	for _, contentType := range []string{"text/plain", "application/json", "application/problem+json"} {
		raw := []byte(`<meta charset="windows-1252">Café 日本`)
		name, got, err := DecodeHTTPBody(contentType, raw)
		if err != nil || name != "utf-8" || !bytes.Equal(got, raw) || &got[0] != &raw[0] {
			t.Fatalf("text default = %q (%s), error=%v", got, name, err)
		}
	}
	for _, raw := range [][]byte{[]byte("Café 日本"), []byte(`{"text":"<meta charset=windows-1252>Café 日本"}`)} {
		name, got, err := DecodeHTTPBody("", raw)
		if err != nil || name != "utf-8" || !bytes.Equal(got, raw) || &got[0] != &raw[0] {
			t.Fatalf("headerless text default = %q (%s), error=%v", got, name, err)
		}
	}
	if _, got, err := DecodeHTTPBody("text/plain; charset=unknown-charset", []byte("example")); err == nil || got != nil {
		t.Fatalf("unsupported explicit charset: got=%q error=%v", got, err)
	}
}

func TestDecodeHTTPBodyUTF8FastPathAndMalformedInput(t *testing.T) {
	raw := []byte("<title>Café 日本 🌍</title>")
	name, got, err := DecodeHTTPBody(`text/html; charset="UTF-8"`, raw)
	if err != nil || name != "utf-8" || !bytes.Equal(got, raw) || &got[0] != &raw[0] {
		t.Fatalf("valid UTF-8 copied or changed: %q (%s), error=%v", got, name, err)
	}
	ascii := []byte("<title>ASCII page</title>")
	if _, got, err := DecodeHTTPBody("text/html", ascii); err != nil || &got[0] != &ascii[0] {
		t.Fatalf("ASCII page copied: error=%v", err)
	}
	for _, tc := range []struct {
		contentType, want string
		raw               []byte
	}{
		{"text/html; charset=utf-8", "Caf�", []byte{'C', 'a', 'f', 0xe9}},
		{"application/json", "�", []byte{0xff}},
		{"text/plain; charset=shift_jis", "�", []byte{0x93}},
		{"text/plain; charset=utf-16le", "�", []byte{0x3c, 0xd8}},
	} {
		_, got, err := DecodeHTTPBody(tc.contentType, tc.raw)
		if err != nil || string(got) != tc.want || !utf8.Valid(got) {
			t.Fatalf("malformed sequence = %q, error=%v; want %q", got, err, tc.want)
		}
	}
}

func TestDecodeHTTPBodyBudgets(t *testing.T) {
	for _, limit := range []int{-1, MaxBody + 1} {
		if _, got, err := DecodeHTTPBodyLimit("text/plain", nil, limit); err == nil || got != nil {
			t.Fatalf("invalid limit %d: got=%q error=%v", limit, got, err)
		}
	}
	if _, got, err := DecodeHTTPBodyLimit("text/plain", nil, 0); err != nil || len(got) != 0 {
		t.Fatalf("empty zero budget: got=%q error=%v", got, err)
	}
	if _, got, err := DecodeHTTPBodyLimit("text/plain", []byte("abcd"), 3); err == nil || got != nil || !strings.Contains(err.Error(), "encoded") {
		t.Fatalf("raw budget: got=%q error=%v", got, err)
	}
	if _, got, err := DecodeHTTPBodyLimit("text/html; charset=windows-1252", []byte{0x80, 0x80}, 6); err != nil || string(got) != "€€" || cap(got) > 6 {
		t.Fatalf("exact decoded budget: got=%q cap=%d error=%v", got, cap(got), err)
	}
	if _, got, err := DecodeHTTPBodyLimit("text/html; charset=windows-1252", []byte{0x80, 0x80}, 5); err == nil || got != nil || !strings.Contains(err.Error(), "decoded UTF-8") {
		t.Fatalf("expanded budget: got=%q error=%v", got, err)
	}
	if _, got, err := DecodeHTTPBody("text/html; charset=windows-1252", bytes.Repeat([]byte{0x80}, MaxBody/3+1)); err == nil || got != nil {
		t.Fatalf("MaxBody expansion accepted: len=%d error=%v", len(got), err)
	}
}
