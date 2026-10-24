package browser

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

var (
	// Noise/boilerplate patterns commonly found in web headers/footers
	boilerplateRegex = regexp.MustCompile(`(?i)^(pular para|skip to|menu principal|toggle navigation|pol[íi]tica de privacidade|privacy policy|termos de uso|terms of (use|service)|todos os direitos reservados|all rights reserved|ir para o conte[úu]do|cookies?|fale conosco|voltar ao topo|back to top|compartilhar|share on)\b`)
	multiSpaceRegex  = regexp.MustCompile(`[ \t]+`)
	multiNewlineRegex = regexp.MustCompile(`\n{3,}`)
)

// EstimateTokens provides a realistic BPE tokenizer approximation (calibrated against cl100k_base / Llama).
// Words are ~1.3 tokens each, plus penalty for punctuation and special syntax symbols (brackets, parentheses, pipes).
func EstimateTokens(text string) int {
	if len(text) == 0 {
		return 0
	}
	words := 0
	specialPunct := 0
	inWord := false

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if !inWord {
				words++
				inWord = true
			}
		} else {
			inWord = false
			// Punctuation heavily penalized in Markdown: [], (), {}, |, #, *, _, `
			if r == '[' || r == ']' || r == '(' || r == ')' || r == '|' || r == '#' || r == '*' || r == '_' || r == '`' || r == '>' {
				specialPunct++
			}
		}
	}

	// Base word tokens (~1.3x words for BPE subwords) + isolated punctuation tokens
	tokenEstimate := int(float64(words)*1.25) + int(float64(specialPunct)*0.75)
	if tokenEstimate < 1 && len(text) > 0 {
		return 1
	}
	return tokenEstimate
}

// ToCSO transforms a parsed Page into Compact Semantic Outline (CSO) format.
// CSO eliminates decorative markdown punctuation, strips navigation noise, and optimizes link syntax for LLMs.
func (p *Page) ToCSO(maxChars int) string {
	if maxChars <= 0 {
		maxChars = 15000
	}

	var sb strings.Builder
	domain := ""
	if u, err := url.Parse(p.URL); err == nil {
		domain = strings.TrimPrefix(u.Hostname(), "www.")
	}

	// 1. Ultra-compact Header (1 line)
	title := strings.TrimSpace(p.Title)
	if title == "" {
		title = domain
	}
	sb.WriteString(fmt.Sprintf("@site: %s [%s]\n", title, domain))

	if p.Description != "" {
		desc := strings.TrimSpace(p.Description)
		if len(desc) > 200 {
			desc = desc[:200] + "..."
		}
		sb.WriteString(fmt.Sprintf("@desc: %s\n", desc))
	}

	// 2. Headings & Content Outline
	if len(p.Headings) > 0 {
		sb.WriteString("@seções:\n")
		for _, h := range p.Headings {
			txt := strings.TrimSpace(h.Text)
			if txt == "" || isBoilerplate(txt) {
				continue
			}
			if h.Level <= 2 {
				sb.WriteString(fmt.Sprintf("  H%d: %s\n", h.Level, txt))
			} else {
				sb.WriteString(fmt.Sprintf("    - %s\n", txt))
			}
		}
	}

	// 3. Main Text Body (Pruned & Cleaned)
	cleanBody := cleanBodyText(p.Text)
	if cleanBody != "" {
		sb.WriteString("@conteúdo:\n")
		lines := strings.Split(cleanBody, "\n")
		count := 0
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || isBoilerplate(line) {
				continue
			}
			sb.WriteString(line)
			sb.WriteString("\n")
			count++
			if count >= 35 { // Focus on primary content
				break
			}
		}
	}

	// 4. Compact Links (Key navigation paths, stripping tracking UTMs)
	if len(p.Links) > 0 {
		var meaningfulLinks []string
		for _, l := range p.Links {
			lbl := strings.TrimSpace(l.Label)
			if len(lbl) < 3 || isBoilerplate(lbl) {
				continue
			}
			cleanTarget := cleanURLPath(l.URL)
			if cleanTarget == "" || cleanTarget == "/" {
				continue
			}
			meaningfulLinks = append(meaningfulLinks, fmt.Sprintf("%s -> %s", lbl, cleanTarget))
			if len(meaningfulLinks) >= 8 {
				break
			}
		}
		if len(meaningfulLinks) > 0 {
			sb.WriteString("@links:\n")
			for _, ml := range meaningfulLinks {
				sb.WriteString("  * ")
				sb.WriteString(ml)
				sb.WriteString("\n")
			}
		}
	}

	// 5. Interactive Form Controls (if any)
	if len(p.Inputs) > 0 || len(p.Buttons) > 0 {
		var actions []string
		for _, in := range p.Inputs {
			if in.Type == "hidden" {
				continue
			}
			actions = append(actions, fmt.Sprintf("in[%s:%s]", in.Type, in.Name))
			if len(actions) >= 4 {
				break
			}
		}
		for _, b := range p.Buttons {
			lbl := strings.TrimSpace(b.Label)
			if lbl != "" {
				actions = append(actions, fmt.Sprintf("btn[%s]", lbl))
				if len(actions) >= 6 {
					break
				}
			}
		}
		if len(actions) > 0 {
			sb.WriteString("@ações: ")
			sb.WriteString(strings.Join(actions, " | "))
			sb.WriteString("\n")
		}
	}

	result := sb.String()
	if len(result) > maxChars {
		result = result[:maxChars]
	}
	return strings.TrimSpace(result)
}

// MarkdownToCSO transforms an existing Markdown string into CSO by stripping markdown decorations.
func MarkdownToCSO(md string, maxChars int) string {
	if maxChars <= 0 {
		maxChars = 10000
	}
	lines := strings.Split(md, "\n")
	var out []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || isBoilerplate(line) {
			continue
		}

		// Convert markdown headings ### to H3:
		if strings.HasPrefix(line, "#") {
			level := 0
			for level < len(line) && line[level] == '#' {
				level++
			}
			title := strings.TrimSpace(line[level:])
			if !isBoilerplate(title) {
				out = append(out, fmt.Sprintf("H%d: %s", level, title))
			}
			continue
		}

		// Convert markdown links [Label](url) to Label -> /path
		linkMatches := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`).FindAllStringSubmatch(line, -1)
		if len(linkMatches) > 0 {
			for _, m := range linkMatches {
				label := strings.TrimSpace(m[1])
				target := cleanURLPath(m[2])
				if isBoilerplate(label) {
					line = strings.ReplaceAll(line, m[0], "")
				} else {
					line = strings.ReplaceAll(line, m[0], fmt.Sprintf("%s -> %s", label, target))
				}
			}
		}

		// Remove bold/italics
		line = strings.ReplaceAll(line, "**", "")
		line = strings.ReplaceAll(line, "__", "")
		line = strings.ReplaceAll(line, "`", "")

		// Strip table pipe characters
		if strings.HasPrefix(line, "|") && strings.HasSuffix(line, "|") {
			parts := strings.Split(line, "|")
			var cleanParts []string
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" && !strings.Contains(p, "---") {
					cleanParts = append(cleanParts, p)
				}
			}
			if len(cleanParts) > 0 {
				line = strings.Join(cleanParts, " | ")
			} else {
				continue
			}
		}

		line = multiSpaceRegex.ReplaceAllString(line, " ")
		line = strings.TrimSpace(line)
		if line != "" && !isBoilerplate(line) {
			out = append(out, line)
		}
	}

	res := strings.Join(out, "\n")
	res = multiNewlineRegex.ReplaceAllString(res, "\n\n")
	if len(res) > maxChars {
		res = res[:maxChars]
	}
	return strings.TrimSpace(res)
}

func isBoilerplate(s string) bool {
	lower := strings.ToLower(strings.TrimSpace(s))
	if len(lower) == 0 {
		return true
	}
	if boilerplateRegex.MatchString(lower) {
		return true
	}
	if strings.Contains(lower, "cookie") && (strings.Contains(lower, "concordo") || strings.Contains(lower, "aceitar") || strings.Contains(lower, "aceito")) {
		return true
	}
	return false
}

func cleanBodyText(t string) string {
	t = multiSpaceRegex.ReplaceAllString(t, " ")
	lines := strings.Split(t, "\n")
	var kept []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if len(l) > 10 && !isBoilerplate(l) {
			kept = append(kept, l)
		}
	}
	return strings.Join(kept, "\n")
}

func cleanURLPath(raw string) string {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if u.Path != "" && u.Host == "" {
		return u.Path
	}
	// Strip tracking query params
	if u.RawQuery != "" {
		q := u.Query()
		for k := range q {
			if strings.HasPrefix(k, "utm_") || k == "fbclid" || k == "gclid" || k == "ref" {
				q.Del(k)
			}
		}
		u.RawQuery = q.Encode()
	}
	if u.Host != "" {
		return u.Path
	}
	return u.String()
}
