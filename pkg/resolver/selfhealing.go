package resolver

import (
	"strings"
	"unicode/utf8"

	"ai-go-browser/browser"
)

// MatchTarget defines criteria for locating an element during self-healing.
type MatchTarget struct {
	Ref         string `json:"ref,omitempty"`
	Text        string `json:"text,omitempty"`
	Tag         string `json:"tag,omitempty"`
	Role        string `json:"role,omitempty"`
	AriaLabel   string `json:"aria_label,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
	Name        string `json:"name,omitempty"`
}

// HealedMatch represents the result of a self-healing resolution.
type HealedMatch struct {
	Element    browser.Element `json:"element"`
	Confidence float64         `json:"confidence"` // 0.0 to 1.0
	Reason     string          `json:"reason"`
	Healed     bool            `json:"healed"`
}

// SelfHealingResolver performs resilient element resolution against dynamic DOM shifts.
type SelfHealingResolver struct {
	Threshold float64 // Minimum confidence to accept a match (default: 0.65)
}

// NewSelfHealingResolver creates a resolver with recommended defaults.
func NewSelfHealingResolver() *SelfHealingResolver {
	return &SelfHealingResolver{
		Threshold: 0.60,
	}
}

// Resolve attempts exact matching first, falling back to heuristic and fuzzy matching.
func (r *SelfHealingResolver) Resolve(elements []browser.Element, target MatchTarget) *HealedMatch {
	if len(elements) == 0 {
		return nil
	}

	// 1. Direct Reference ID match (e.g., "#ref:3" or "3")
	cleanedRef := strings.TrimPrefix(target.Ref, "#ref:")
	for _, el := range elements {
		elRef := strings.TrimPrefix(el.Ref, "#ref:")
		if elRef != "" && elRef == cleanedRef {
			return &HealedMatch{
				Element:    el,
				Confidence: 1.0,
				Reason:     "exact_ref_match",
				Healed:     false,
			}
		}
	}

	// 2. Exact text / label / name match
	targetText := strings.ToLower(strings.TrimSpace(target.Text))
	targetLabel := strings.ToLower(strings.TrimSpace(target.AriaLabel))
	if targetLabel == "" {
		targetLabel = strings.ToLower(strings.TrimSpace(target.Placeholder))
	}
	targetName := strings.ToLower(strings.TrimSpace(target.Name))

	for _, el := range elements {
		elText := strings.ToLower(strings.TrimSpace(el.Text))
		elLabel := strings.ToLower(strings.TrimSpace(el.Label))
		elName := strings.ToLower(strings.TrimSpace(el.Name))

		if targetText != "" && elText == targetText {
			return &HealedMatch{
				Element:    el,
				Confidence: 0.95,
				Reason:     "exact_text_match",
				Healed:     true,
			}
		}
		if targetLabel != "" && elLabel == targetLabel {
			return &HealedMatch{
				Element:    el,
				Confidence: 0.95,
				Reason:     "exact_label_match",
				Healed:     true,
			}
		}
		if targetName != "" && elName == targetName {
			return &HealedMatch{
				Element:    el,
				Confidence: 0.90,
				Reason:     "exact_name_match",
				Healed:     true,
			}
		}
	}

	// 3. Substring & Role Heuristics
	for _, el := range elements {
		elText := strings.ToLower(strings.TrimSpace(el.Text))
		elLabel := strings.ToLower(strings.TrimSpace(el.Label))

		if targetText != "" && elText != "" && len(targetText) >= 3 && (strings.Contains(elText, targetText) || strings.Contains(targetText, elText)) {
			score := 0.80
			if target.Tag != "" && strings.EqualFold(el.Tag, target.Tag) {
				score += 0.05
			}
			return &HealedMatch{
				Element:    el,
				Confidence: score,
				Reason:     "substring_text_match",
				Healed:     true,
			}
		}
		if targetLabel != "" && elLabel != "" && len(targetLabel) >= 3 && strings.Contains(elLabel, targetLabel) {
			return &HealedMatch{
				Element:    el,
				Confidence: 0.80,
				Reason:     "substring_label_match",
				Healed:     true,
			}
		}
	}

	// 4. Fuzzy Levenshtein Distance Matching
	var bestEl browser.Element
	bestScore := 0.0
	bestReason := ""

	for _, el := range elements {
		var candidateStrs []string
		if el.Text != "" {
			candidateStrs = append(candidateStrs, strings.ToLower(strings.TrimSpace(el.Text)))
		}
		if el.Label != "" {
			candidateStrs = append(candidateStrs, strings.ToLower(strings.TrimSpace(el.Label)))
		}

		for _, cand := range candidateStrs {
			if targetText != "" {
				sim := similarity(targetText, cand)
				if sim > bestScore {
					bestScore = sim
					bestEl = el
					bestReason = "fuzzy_text_similarity"
				}
			}
			if targetLabel != "" {
				sim := similarity(targetLabel, cand)
				if sim > bestScore {
					bestScore = sim
					bestEl = el
					bestReason = "fuzzy_label_similarity"
				}
			}
		}
	}

	if bestScore >= r.Threshold {
		return &HealedMatch{
			Element:    bestEl,
			Confidence: bestScore,
			Reason:     bestReason,
			Healed:     true,
		}
	}

	return nil
}

// similarity calculates normalized Levenshtein similarity between two strings (0.0 to 1.0).
func similarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}
	len1 := utf8.RuneCountInString(s1)
	len2 := utf8.RuneCountInString(s2)
	if len1 == 0 || len2 == 0 {
		return 0.0
	}

	dist := levenshteinDistance(s1, s2)
	maxLen := max(len1, len2)
	return 1.0 - (float64(dist) / float64(maxLen))
}

func levenshteinDistance(s1, s2 string) int {
	r1 := []rune(s1)
	r2 := []rune(s2)
	m := len(r1)
	n := len(r2)

	d := make([][]int, m+1)
	for i := range d {
		d[i] = make([]int, n+1)
		d[i][0] = i
	}
	for j := 0; j <= n; j++ {
		d[0][j] = j
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			cost := 0
			if r1[i-1] != r2[j-1] {
				cost = 1
			}
			d[i][j] = min3(
				d[i-1][j]+1,      // deletion
				d[i][j-1]+1,      // insertion
				d[i-1][j-1]+cost, // substitution
			)
		}
	}
	return d[m][n]
}

func min3(a, b, c int) int {
	if a < b && a < c {
		return a
	}
	if b < c {
		return b
	}
	return c
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
