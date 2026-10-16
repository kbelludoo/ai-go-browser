package browser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"unicode/utf8"
)

func responseMediaType(header string) string {
	media, _, err := mime.ParseMediaType(header)
	if err == nil {
		return strings.ToLower(media)
	}
	return strings.ToLower(strings.TrimSpace(strings.SplitN(header, ";", 2)[0]))
}

func htmlMediaType(media string) bool {
	return media == "text/html" || media == "application/xhtml+xml"
}

func sniffResponseMediaType(content []byte) string {
	// Sniff only a small decoded preview. BOM-encoded HTML must still enter
	// the DOM path; the original buffer remains intact for byte accounting.
	preview := content[:min(len(content), 1024)]
	if _, decoded, err := DecodeHTTPBodyLimit("application/octet-stream", preview, 4096); err == nil {
		preview = decoded
	}
	return responseMediaType(http.DetectContentType(preview))
}

// Resource snapshots have a smaller budget than the DOM. JSON is emitted only
// after complete parsing and redaction; a truncated JSON prefix is never logged.
func resourceSnapshot(content []byte, target string, status int, contentType string, truncated bool) *Resource {
	out := &Resource{URL: target, Status: status, ContentType: contentType, Truncated: truncated}
	media := responseMediaType(contentType)
	if media == "" {
		media = sniffResponseMediaType(content)
		// Classification must see the decoded text too: a BOM otherwise makes
		// valid JSON look like text or binary and bypasses field redaction.
		if _, data, err := DecodeHTTPBodyLimit(contentType, content, maxResourceBytes); err == nil {
			trimmed := bytes.TrimSpace(data)
			objectOrArray := len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[')
			if (!truncated && json.Valid(data)) || objectOrArray {
				media = "application/json"
			}
		}
	}
	switch {
	case media == "application/json" || strings.HasSuffix(media, "+json"):
		_, data, err := DecodeHTTPBodyLimit(contentType, content, maxResourceBytes)
		if err != nil {
			out.Body = "[JSON could not be decoded within observation budget]"
			out.Truncated = true
			return out
		}
		var value any
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.UseNumber()
		var extra any
		if !truncated && decoder.Decode(&value) == nil && decoder.Decode(&extra) == io.EOF {
			out.Data = redactJSON(value, 0)
			if value == nil {
				out.Body = "null"
			}
		} else {
			out.Body = "[JSON too large or invalid to parse safely]"
		}
	case htmlMediaType(media):
		_, data, err := DecodeHTTPBodyLimit(contentType, content, maxResourceBytes)
		if err != nil {
			out.Body = "[HTML could not be decoded within observation budget]"
			out.Truncated = true
			return out
		}
		page := Parse(data, target, status, len(content), 5000)
		out.Body = fmt.Sprintf("title: %s\ntext: %s", page.Title, page.Text)
	case strings.HasPrefix(media, "text/") || media == "application/xml" || strings.HasSuffix(media, "+xml"):
		// A byte limit may split the final rune; omit only that incomplete tail.
		_, encoding, _, _ := httpBodyEncoding(contentType, content)
		if truncated && encoding == "utf-8" {
			start := len(content) - 1
			for start > 0 && content[start]&0xC0 == 0x80 && len(content)-start < utf8.UTFMax {
				start--
			}
			if start >= 0 && !utf8.FullRune(content[start:]) {
				content = content[:start]
			}
		}
		_, data, err := DecodeHTTPBodyLimit(contentType, content, maxResourceBytes)
		if err != nil {
			out.Body = "[text could not be decoded within observation budget]"
			out.Truncated = true
			return out
		}
		out.Body = strings.ToValidUTF8(string(data), "\uFFFD")
	default:
		out.Body = "[binary response omitted]"
	}
	return out
}

func (b *Browser) installResource(out *Resource, count int, saveHistory bool) {
	if saveHistory {
		b.rememberCurrent()
	}
	b.closeLive()
	b.closeJSSession()
	b.rawHTML, b.lastJS, b.Document = nil, nil, nil
	b.scannedScripts, b.scannedScriptText = false, ""
	b.resource = out
	text := out.Body
	if out.Data != nil {
		data, _ := json.Marshal(out.Data)
		text = string(data)
	}
	collector := pageText{limit: b.MaxText}
	collector.add(text)
	b.Current = &Page{URL: out.URL, Status: out.Status, Text: collector.text.String(), Bytes: count, Truncated: out.Truncated || collector.truncated, Stack: TechStack{RenderMode: "Resource"}}
}
