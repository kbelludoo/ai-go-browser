package browser

import (
	"bytes"
	"math"
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	reMetaCharset = regexp.MustCompile(`(?i)<meta\s+[^>]*charset=["']?([a-zA-Z0-9_\-]+)`)
	reMetaContent = regexp.MustCompile(`(?i)<meta\s+[^>]*content=["'][^"']*charset=([a-zA-Z0-9_\-]+)`)
	reHeaderCharset = regexp.MustCompile(`(?i)charset=([a-zA-Z0-9_\-]+)`)

	win1252Table = [32]rune{
		0x20AC, 0xFFFD, 0x201A, 0x0192, 0x201E, 0x2026, 0x2020, 0x2021,
		0x02C6, 0x2030, 0x0160, 0x2039, 0x0152, 0xFFFD, 0x017D, 0xFFFD,
		0xFFFD, 0x2018, 0x2019, 0x201C, 0x201D, 0x2022, 0x2013, 0x2014,
		0x02DC, 0x2122, 0x0161, 0x203A, 0x0153, 0xFFFD, 0x017E, 0x0178,
	}
)

// Windows1252ToUTF8 converts ISO-8859-1 / Windows-1252 byte stream to UTF-8.
func Windows1252ToUTF8(data []byte) []byte {
	var buf bytes.Buffer
	buf.Grow(len(data) + len(data)/4)
	for _, b := range data {
		if b <= 0x7F {
			buf.WriteByte(b)
		} else if b >= 0x80 && b <= 0x9F {
			buf.WriteRune(win1252Table[b-0x80])
		} else {
			buf.WriteRune(rune(b))
		}
	}
	return buf.Bytes()
}

// DetectAndNormalizeEncoding detects the page encoding and converts it to UTF-8 if needed.
func DetectAndNormalizeEncoding(contentType string, raw []byte) (string, []byte) {
	enc := ""
	if m := reHeaderCharset.FindStringSubmatch(contentType); len(m) > 1 {
		enc = strings.ToLower(strings.TrimSpace(m[1]))
	}
	if enc == "" && len(raw) > 0 {
		headSnippet := raw
		if len(headSnippet) > 4096 {
			headSnippet = headSnippet[:4096]
		}
		if m := reMetaCharset.FindSubmatch(headSnippet); len(m) > 1 {
			enc = strings.ToLower(strings.TrimSpace(string(m[1])))
		} else if m := reMetaContent.FindSubmatch(headSnippet); len(m) > 1 {
			enc = strings.ToLower(strings.TrimSpace(string(m[1])))
		}
	}

	if enc == "" {
		if utf8.Valid(raw) {
			enc = "utf-8"
		} else {
			enc = "windows-1252"
		}
	}

	switch enc {
	case "iso-8859-1", "latin1", "windows-1252", "cp1252", "ansi":
		return enc, Windows1252ToUTF8(raw)
	case "utf-8", "utf8":
		if !utf8.Valid(raw) {
			return enc + " (fixed-cp1252)", Windows1252ToUTF8(raw)
		}
		return "utf-8", raw
	default:
		if !utf8.Valid(raw) {
			return enc + " (fallback-utf8)", Windows1252ToUTF8(raw)
		}
		return enc, raw
	}
}

// AnalyzeTechStack detects the framework, rendering strategy (SSR vs CSR), and content density.
func AnalyzeTechStack(rawHTML string, textLen int, encoding string) TechStack {
	lower := strings.ToLower(rawHTML)
	totalLen := len(rawHTML)
	if totalLen == 0 {
		return TechStack{Framework: "Empty", RenderMode: "None", Encoding: encoding}
	}

	yield := (float64(textLen) / float64(totalLen)) * 100
	yield = math.Round(yield*100) / 100

	framework := "HTML/Static"
	renderMode := "SSR"
	notice := ""

	// Next.js
	if strings.Contains(lower, "__next_data__") || strings.Contains(lower, "/_next/static/") {
		framework = "Next.js"
		if yield >= 2.0 {
			renderMode = "SSR (Server Rendered)"
		} else {
			renderMode = "CSR (Client Hydration)"
		}
	} else if strings.Contains(lower, "__nuxt__") || strings.Contains(lower, "/_nuxt/") {
		framework = "Nuxt (Vue)"
		if yield >= 2.0 {
			renderMode = "SSR (Server Rendered)"
		} else {
			renderMode = "CSR"
		}
	} else if strings.Contains(lower, "data-reactroot") || strings.Contains(lower, `id="root"`) || strings.Contains(lower, `id="app"`) {
		if strings.Contains(lower, "react") {
			framework = "React (SPA)"
		} else if strings.Contains(lower, "vue") {
			framework = "Vue (SPA)"
		} else {
			framework = "SPA / Virtual DOM"
		}
		if yield < 3.0 || textLen < 50 {
			renderMode = "CSR (Client-Side Rendering)"
			notice = "Página depende de JavaScript no cliente (CSR); conteúdo interativo requer execução de JS."
		} else {
			renderMode = "SSR / Híbrido"
		}
	} else if strings.Contains(lower, "ng-version") || strings.Contains(lower, "<app-root>") || strings.Contains(lower, "ng-app") {
		framework = "Angular"
		if yield < 1.0 {
			renderMode = "CSR"
			notice = "Página em Angular com renderização no cliente."
		} else {
			renderMode = "SSR / Angular Universal"
		}
	} else if strings.Contains(lower, "__sveltekit") || strings.Contains(lower, "svelte-") {
		framework = "Svelte / SvelteKit"
		renderMode = "SSR"
	} else if strings.Contains(lower, "/wp-content/") || strings.Contains(lower, "/wp-includes/") {
		framework = "WordPress"
		renderMode = "SSR (PHP/Server)"
	} else if strings.Contains(lower, "hx-get") || strings.Contains(lower, "hx-post") {
		framework = "HTMX"
		renderMode = "SSR + Hypermedia"
	} else if strings.Contains(lower, "csrfmiddlewaretoken") {
		framework = "Python (Django)"
		renderMode = "SSR"
	} else if strings.Contains(lower, "laravel") || strings.Contains(lower, "phpsessid") {
		framework = "PHP / Laravel"
		renderMode = "SSR"
	}

	return TechStack{
		Framework:    framework,
		RenderMode:   renderMode,
		Encoding:     encoding,
		ContentYield: yield,
		Notice:       notice,
	}
}

var (
	reNextData = regexp.MustCompile(`(?s)<script\s+id="__NEXT_DATA__"[^>]*>(.*?)</script>`)
	reJSONLD   = regexp.MustCompile(`(?s)<script\s+type="application/ld\+json"[^>]*>(.*?)</script>`)
)

// ExtractStructuredData extracts embedded JSON-LD and Next.js hydration payload.
func ExtractStructuredData(rawHTML string) map[string]string {
	out := make(map[string]string)
	if m := reNextData.FindStringSubmatch(rawHTML); len(m) > 1 {
		data := strings.TrimSpace(m[1])
		if len(data) > 0 {
			if len(data) > 8192 {
				out["next_data"] = data[:8192] + "... [truncated]"
			} else {
				out["next_data"] = data
			}
		}
	}
	if m := reJSONLD.FindAllStringSubmatch(rawHTML, 3); len(m) > 0 {
		var lds []string
		for _, item := range m {
			if len(item) > 1 {
				trimmed := strings.TrimSpace(item[1])
				if len(trimmed) > 0 && len(trimmed) < 4096 {
					lds = append(lds, trimmed)
				}
			}
		}
		if len(lds) > 0 {
			out["json_ld"] = strings.Join(lds, "\n---\n")
		}
	}
	return out
}
