package browser

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var (
	ldJSONRegex   = regexp.MustCompile(`(?is)<script[^>]*type=["']application/ld\+json["'][^>]*>(.*?)</script>`)
	nextDataRegex = regexp.MustCompile(`(?is)<script[^>]*id=["']__NEXT_DATA__["'][^>]*>(.*?)</script>`)
	nuxtDataRegex = regexp.MustCompile(`(?is)<script[^>]*id=["']__NUXT_DATA__["'][^>]*>(.*?)</script>`)
	metaOgRegex   = regexp.MustCompile(`(?is)<meta[^>]+property=["'](og:[^"']+)["'][^>]+content=["']([^"']*)["']`)
)

// HydrationPayload holds pre-rendered framework states, Schema.org entities, and SEO metadata.
type HydrationPayload struct {
	JSONLD   []map[string]any  `json:"json_ld,omitempty"`   // Schema.org entities (Product, Article, FAQ, etc.)
	NextData map[string]any    `json:"next_data,omitempty"` // Next.js pageProps data
	NuxtData []any             `json:"nuxt_data,omitempty"` // Nuxt.js dehydrated state
	MetaTags map[string]string `json:"meta_tags,omitempty"` // OpenGraph and key metadata
	Summary  string            `json:"summary,omitempty"`   // High-level AI summary of embedded data
}

// ButtonActionMetadata extracts action intent from a button without running any JavaScript.
type ButtonActionMetadata struct {
	Ref           string            `json:"ref"`
	Label         string            `json:"label"`
	Tag           string            `json:"tag"`
	DataAttrs     map[string]string `json:"data_attrs,omitempty"`
	OnClick       string            `json:"onclick,omitempty"`
	FormMethod    string            `json:"form_method,omitempty"`
	FormAction    string            `json:"form_action,omitempty"`
	HiddenInputs  map[string]string `json:"hidden_inputs,omitempty"`
	ActionSummary string            `json:"action_summary,omitempty"`
}

// ExtractHydration parses static data blobs embedded in modern web pages without JS execution.
func ExtractHydration(html string) *HydrationPayload {
	payload := &HydrationPayload{
		MetaTags: make(map[string]string),
	}

	// 1. Extract Schema.org JSON-LD
	ldMatches := ldJSONRegex.FindAllStringSubmatch(html, -1)
	for _, m := range ldMatches {
		if len(m) >= 2 {
			rawJSON := strings.TrimSpace(m[1])
			if rawJSON == "" {
				continue
			}
			// Can be a single object or an array of objects
			if strings.HasPrefix(rawJSON, "[") {
				var arr []map[string]any
				if err := json.Unmarshal([]byte(rawJSON), &arr); err == nil {
					payload.JSONLD = append(payload.JSONLD, arr...)
				}
			} else {
				var obj map[string]any
				if err := json.Unmarshal([]byte(rawJSON), &obj); err == nil {
					payload.JSONLD = append(payload.JSONLD, obj)
				}
			}
		}
	}

	// 2. Extract Next.js __NEXT_DATA__
	nextMatches := nextDataRegex.FindStringSubmatch(html)
	if len(nextMatches) >= 2 {
		var nextFull map[string]any
		if err := json.Unmarshal([]byte(strings.TrimSpace(nextMatches[1])), &nextFull); err == nil {
			// Extract pageProps if available (the cleanest backend data)
			if props, ok := nextFull["props"].(map[string]any); ok {
				if pageProps, ok := props["pageProps"].(map[string]any); ok {
					payload.NextData = pageProps
				} else {
					payload.NextData = props
				}
			} else {
				payload.NextData = nextFull
			}
		}
	}

	// 3. Extract Nuxt.js __NUXT_DATA__
	nuxtMatches := nuxtDataRegex.FindStringSubmatch(html)
	if len(nuxtMatches) >= 2 {
		var nuxtList []any
		if err := json.Unmarshal([]byte(strings.TrimSpace(nuxtMatches[1])), &nuxtList); err == nil {
			payload.NuxtData = nuxtList
		}
	}

	// 4. Extract OpenGraph Meta Tags
	metaMatches := metaOgRegex.FindAllStringSubmatch(html, -1)
	for _, m := range metaMatches {
		if len(m) >= 3 {
			prop := strings.ToLower(strings.TrimSpace(m[1]))
			val := strings.TrimSpace(m[2])
			if prop != "" && val != "" {
				payload.MetaTags[prop] = val
			}
		}
	}

	// Build human/AI-readable summary
	var summaryParts []string
	if len(payload.JSONLD) > 0 {
		for _, item := range payload.JSONLD {
			if typeVal, ok := item["@type"].(string); ok {
				nameVal := ""
				if n, ok := item["name"].(string); ok {
					nameVal = n
				}
				if nameVal != "" {
					summaryParts = append(summaryParts, fmt.Sprintf("%s: %q", typeVal, nameVal))
				} else {
					summaryParts = append(summaryParts, typeVal)
				}
			}
		}
	}
	if len(payload.NextData) > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("Next.js State (%d keys)", len(payload.NextData)))
	}
	if len(payload.MetaTags) > 0 {
		if title, ok := payload.MetaTags["og:title"]; ok {
			summaryParts = append(summaryParts, fmt.Sprintf("og:title: %q", title))
		}
	}
	if len(summaryParts) > 0 {
		payload.Summary = strings.Join(summaryParts, " | ")
	}

	return payload
}

// AnalyzeElementAction inspects an HTML element's attributes to deduce its action without executing JS.
func AnalyzeElementAction(ref, tag, label string, attrs map[string]string, form *Form) ButtonActionMetadata {
	meta := ButtonActionMetadata{
		Ref:          ref,
		Tag:          tag,
		Label:        label,
		DataAttrs:    make(map[string]string),
		HiddenInputs: make(map[string]string),
	}

	// 1. Collect all data-* attributes and event triggers
	for k, v := range attrs {
		lowerK := strings.ToLower(k)
		if strings.HasPrefix(lowerK, "data-") {
			meta.DataAttrs[lowerK] = v
		} else if lowerK == "onclick" || lowerK == "onsubmit" || lowerK == "href" {
			if strings.HasPrefix(lowerK, "on") {
				meta.OnClick = v
			}
		}
	}

	// 2. Form association
	if form != nil {
		meta.FormMethod = strings.ToUpper(form.Method)
		if meta.FormMethod == "" {
			meta.FormMethod = "GET"
		}
		meta.FormAction = form.Action

		// Capture hidden inputs from the form (IDs, tokens, state)
		for _, f := range form.Fields {
			if strings.EqualFold(f.Type, "hidden") && f.Name != "" {
				meta.HiddenInputs[f.Name] = f.Value
			}
		}
	}

	// 3. Synthesize ActionSummary for AI comprehension
	var summaryParts []string
	if meta.FormAction != "" {
		summaryParts = append(summaryParts, fmt.Sprintf("%s %s", meta.FormMethod, meta.FormAction))
	}

	// Key data attributes to highlight
	for _, key := range []string{"data-action", "data-url", "data-endpoint", "data-target", "data-id", "data-sku", "data-price", "data-modal"} {
		if val, ok := meta.DataAttrs[key]; ok && val != "" {
			summaryParts = append(summaryParts, fmt.Sprintf("%s=%s", strings.TrimPrefix(key, "data-"), val))
		}
	}

	if meta.OnClick != "" && len(meta.OnClick) < 60 {
		summaryParts = append(summaryParts, fmt.Sprintf("js:%s", meta.OnClick))
	}

	if len(meta.HiddenInputs) > 0 {
		var hiddenKeys []string
		for k, v := range meta.HiddenInputs {
			if len(v) < 20 {
				hiddenKeys = append(hiddenKeys, fmt.Sprintf("%s=%s", k, v))
			} else {
				hiddenKeys = append(hiddenKeys, k)
			}
		}
		summaryParts = append(summaryParts, fmt.Sprintf("params:[%s]", strings.Join(hiddenKeys, ", ")))
	}

	if len(summaryParts) > 0 {
		meta.ActionSummary = strings.Join(summaryParts, " | ")
	}

	return meta
}
