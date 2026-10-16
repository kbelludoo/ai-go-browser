package browser

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding"
	"golang.org/x/text/transform"
)

// DecodeHTTPBody converts a buffered HTTP response to UTF-8. Both the encoded
// response and its decoded representation must fit in MaxBody.
// The returned name describes the source encoding, using WHATWG canonical names.
func DecodeHTTPBody(contentType string, raw []byte) (string, []byte, error) {
	return DecodeHTTPBodyLimit(contentType, raw, MaxBody)
}

// DecodeHTTPBodyLimit applies a smaller budget for resource observations. It
// returns an error instead of a partial document when decoding expands past it.
// Valid UTF-8 shares the input buffer; callers must retain ownership of raw.
func DecodeHTTPBodyLimit(contentType string, raw []byte, maxBytes int) (string, []byte, error) {
	if maxBytes < 0 || maxBytes > MaxBody {
		return "", nil, fmt.Errorf("encoding byte limit must be between 0 and %d", MaxBody)
	}
	if len(raw) > maxBytes {
		return "", nil, fmt.Errorf("encoded response exceeds %d bytes", maxBytes)
	}
	e, name, bomBytes, err := httpBodyEncoding(contentType, raw)
	if err != nil {
		return "", nil, err
	}
	raw = raw[bomBytes:]
	if name == "utf-8" && utf8.Valid(raw) {
		return name, raw, nil
	}
	// Undeclared ASCII HTML defaults to Windows-1252, but all its bytes already
	// represent exactly the same UTF-8 text. Avoid allocating another page copy.
	if name == "windows-1252" && asciiBody(raw) {
		return name, raw, nil
	}
	reader := transform.NewReader(bytes.NewReader(raw), e.NewDecoder())
	data, err := readDecodedBody(reader, len(raw), maxBytes)
	if err != nil {
		return name, nil, err
	}
	return name, data, nil
}

func httpBodyEncoding(contentType string, raw []byte) (encoding.Encoding, string, int, error) {
	for _, bom := range []struct {
		bytes []byte
		name  string
	}{
		{[]byte{0xef, 0xbb, 0xbf}, "utf-8"},
		{[]byte{0xff, 0xfe}, "utf-16le"},
		{[]byte{0xfe, 0xff}, "utf-16be"},
	} {
		if bytes.HasPrefix(raw, bom.bytes) {
			e, name := charset.Lookup(bom.name)
			return e, name, len(bom.bytes), nil
		}
	}

	media := responseMediaType(contentType)
	if media == "" {
		media = responseMediaType(http.DetectContentType(raw))
	}
	html := htmlMediaType(media)
	if _, params, err := mime.ParseMediaType(contentType); err == nil {
		if label := strings.TrimSpace(params["charset"]); label != "" {
			if e, name := charset.Lookup(label); e != nil {
				return e, name, 0, nil
			}
			if !html {
				return nil, "", 0, fmt.Errorf("unsupported response charset %q", label)
			}
		}
	}
	if html {
		// HTML follows BOM/header/meta precedence and the browser's legacy
		// fallback. Plain text and JSON do not inspect HTML-like meta strings.
		e, name, _ := charset.DetermineEncoding(raw, contentType)
		return e, name, 0, nil
	}
	e, name := charset.Lookup("utf-8")
	return e, name, 0, nil
}

func asciiBody(raw []byte) bool {
	for _, b := range raw {
		if b >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

func readDecodedBody(reader io.Reader, initial, limit int) ([]byte, error) {
	// Control capacity as well as length: io.ReadAll may grow a backing array
	// beyond the observation budget even when fed by a LimitedReader.
	data := make([]byte, 0, min(initial, limit))
	var chunk [4096]byte
	for {
		n, err := reader.Read(chunk[:])
		if n > limit-len(data) {
			return nil, fmt.Errorf("decoded UTF-8 response exceeds %d bytes", limit)
		}
		if needed := len(data) + n; needed > cap(data) {
			capacity := min(limit, max(needed, 2*cap(data)))
			grown := make([]byte, len(data), capacity)
			copy(grown, data)
			data = grown
		}
		data = append(data, chunk[:n]...)
		if err == io.EOF {
			return data, nil
		}
		if err != nil {
			return nil, fmt.Errorf("decode HTTP response: %w", err)
		}
	}
}
