package main

import (
	"strings"
	"unicode"
)

// matchResult holds the result of a fuzzy match attempt.
type matchResult struct {
	matched   bool
	score     int
	positions []int // indices in the original string that matched query chars
}

// fuzzyMatch performs fzf-style fuzzy matching of query against str.
// Returns a matchResult with a score (higher = better) and the matched positions.
func fuzzyMatch(str, query string) matchResult {
	if query == "" {
		return matchResult{matched: true, score: 0, positions: nil}
	}

	strLower := strings.ToLower(str)
	queryLower := strings.ToLower(query)

	// First pass: check if all query chars appear in order (basic fuzzy check)
	positions := make([]int, 0, len(queryLower))
	si := 0
	for _, qc := range queryLower {
		found := false
		for si < len(strLower) {
			if rune(strLower[si]) == qc {
				positions = append(positions, si)
				si++
				found = true
				break
			}
			si++
		}
		if !found {
			return matchResult{matched: false}
		}
	}

	// Score the match — higher is better.
	// Factors: consecutive runs, word boundaries, prefix matches.
	score := scoreMatch(str, strLower, queryLower, positions)

	return matchResult{
		matched:   true,
		score:     score,
		positions: positions,
	}
}

// scoreMatch computes a quality score for a fuzzy match.
// Inspired by fzf's scoring: consecutive bonuses, word-boundary bonuses, prefix bonus.
func scoreMatch(str, strLower, queryLower string, positions []int) int {
	if len(positions) == 0 {
		return 0
	}

	score := 0
	consecutive := 0

	for i, pos := range positions {
		// Base score per matched character
		score += 10

		// Prefix bonus: matching at the start of the string
		if pos == 0 {
			score += 20
		}

		// Word boundary bonus: char after a separator or uppercase after lowercase
		if pos > 0 {
			prev := rune(str[pos-1])
			if isSeparator(prev) {
				score += 15
			} else if unicode.IsUpper(rune(str[pos])) && unicode.IsLower(prev) {
				// camelCase boundary
				score += 10
			}
		}

		// Consecutive run bonus (exponentially increasing)
		if i > 0 && positions[i-1] == pos-1 {
			consecutive++
			score += consecutive * 8
		} else {
			consecutive = 0
		}

		// Gap penalty (small penalty for skipping characters)
		if i > 0 {
			gap := pos - positions[i-1] - 1
			if gap > 0 {
				score -= gap / 4
			}
		}
	}

	// Bonus for matching the full query as a substring (exact substring match)
	if idx := strings.Index(strLower, queryLower); idx >= 0 {
		score += 30
		if idx == 0 {
			score += 20 // prefix exact match gets extra bonus
		}
	}

	// Penalty for longer strings (prefer shorter, more specific matches)
	score -= len(str) / 4

	return score
}

func isSeparator(r rune) bool {
	switch r {
	case '/', '\\', '-', '_', '.', ' ', '(', ')', '[', ']', '{', '}':
		return true
	}
	return false
}
