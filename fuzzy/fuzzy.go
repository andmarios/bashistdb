// Package fuzzy provides fuzzy string matching and ranking for command history search.
// It combines substring/prefix matching on individual words with Damerau-Levenshtein
// edit distance for typo tolerance. Each search token must match a word in the target.
package fuzzy

import (
	"sort"
	"strings"
)

// Scoring constants.
const (
	scoreExactWord  = 100 // token equals word exactly
	scorePrefixBase = 60  // token is a prefix of a word
	scoreContains   = 40  // token is a substring of a word
	scoreEditBase   = 20  // token is within edit distance of a word
)

// Match holds the score and original index of a matched target string.
type Match struct {
	Score int
	Index int
}

// damerauLevenshtein computes the Optimal String Alignment distance between two strings,
// which counts insertions, deletions, substitutions, and transpositions of adjacent
// characters as single edits.
func damerauLevenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	d := make([][]int, la+1)
	for i := range d {
		d[i] = make([]int, lb+1)
		d[i][0] = i
	}
	for j := 0; j <= lb; j++ {
		d[0][j] = j
	}

	for i := 1; i <= la; i++ {
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			d[i][j] = min(
				d[i-1][j]+1,
				d[i][j-1]+1,
				d[i-1][j-1]+cost,
			)
			if i > 1 && j > 1 && a[i-1] == b[j-2] && a[i-2] == b[j-1] {
				d[i][j] = min(d[i][j], d[i-2][j-2]+cost)
			}
		}
	}
	return d[la][lb]
}

// splitWords splits a command string into searchable words, splitting on common
// shell separators and stripping punctuation. For example:
//
//	`git commit -am "[OPS-2453] fix"` -> ["git", "commit", "am", "OPS-2453", "OPS", "2453", "fix"]
func splitWords(s string) []string {
	// Replace common separators with spaces
	replacer := strings.NewReplacer(
		"/", " ",
		"=", " ",
		":", " ",
		";", " ",
		"|", " ",
		"\"", " ",
		"'", " ",
		"[", " ",
		"]", " ",
		"(", " ",
		")", " ",
		"{", " ",
		"}", " ",
	)
	s = replacer.Replace(s)

	raw := strings.Fields(s)
	var words []string
	for _, w := range raw {
		// Strip leading dashes (flags like -am, --force)
		clean := strings.TrimLeft(w, "-")
		if clean == "" {
			continue
		}
		// Strip trailing punctuation
		clean = strings.TrimRight(clean, ".,;:!?")
		if clean == "" {
			continue
		}
		words = append(words, clean)
		// Also split on internal hyphens (e.g. "OPS-2453" -> "OPS", "2453")
		if parts := strings.Split(clean, "-"); len(parts) > 1 {
			for _, p := range parts {
				if p != "" {
					words = append(words, p)
				}
			}
		}
		// Also split on dots (e.g. "deploy-prod.yaml" -> also "yaml")
		if parts := strings.Split(clean, "."); len(parts) > 1 {
			for _, p := range parts {
				if p != "" {
					words = append(words, p)
				}
			}
		}
	}
	return words
}

// maxEditDistance returns the maximum allowed edit distance for a
// token of the given length. Short tokens require exact matches; longer tokens
// allow more edits.
func maxEditDistance(tokenLen int) int {
	if tokenLen <= 2 {
		return 0
	}
	if tokenLen <= 5 {
		return 1
	}
	return 2
}

// scoreTokenAgainstWord scores a single token against a single word.
// Returns (score, matched). Higher scores = better matches.
func scoreTokenAgainstWord(token, word string) (int, bool) {
	tLower := strings.ToLower(token)
	wLower := strings.ToLower(word)

	// Exact match
	if tLower == wLower {
		return scoreExactWord, true
	}

	// Prefix match (token is a prefix of word)
	if strings.HasPrefix(wLower, tLower) {
		// Bonus for covering more of the word
		coverage := len(token) * 100 / len(word)
		return scorePrefixBase + coverage/5, true
	}

	// Substring match (token appears inside word)
	if len(token) >= 2 && strings.Contains(wLower, tLower) {
		return scoreContains, true
	}

	// Edit distance match (typo tolerance)
	maxDist := maxEditDistance(len(token))
	if maxDist > 0 {
		dist := damerauLevenshtein(tLower, wLower)
		if dist <= maxDist {
			return scoreEditBase + (maxDist-dist)*10, true
		}
		// Also try prefix edit distance for longer words.
		// This handles truncated+typo input like "srti" matching "strip":
		// try word prefixes from len(token) to len(token)+maxDist chars.
		if len(wLower) > len(tLower) {
			end := min(len(tLower)+maxDist, len(wLower))
			for pl := len(tLower); pl <= end; pl++ {
				dist = damerauLevenshtein(tLower, wLower[:pl])
				if dist <= maxDist {
					return scoreEditBase + (maxDist-dist)*5, true
				}
			}
		}
	}

	return 0, false
}

// scoreToken scores a search token against a target command by finding the
// best-matching word in the target.
func scoreToken(token, target string) (int, bool) {
	words := splitWords(target)
	bestScore := 0
	matched := false

	for _, word := range words {
		s, ok := scoreTokenAgainstWord(token, word)
		if ok && s > bestScore {
			bestScore = s
			matched = true
		}
	}
	return bestScore, matched
}

// Score evaluates how well a multi-word search pattern matches a target string.
// Every whitespace-separated token in the pattern must match a word in the target
// for the overall match to succeed. Returns the total score and whether all tokens matched.
func Score(pattern, target string) (int, bool) {
	tokens := strings.Fields(pattern)
	if len(tokens) == 0 {
		return 0, true
	}

	totalScore := 0
	for _, token := range tokens {
		s, ok := scoreToken(token, target)
		if !ok {
			return 0, false
		}
		totalScore += s
	}
	return totalScore, true
}

// Rank scores every target string against the pattern and returns matches sorted
// by score in descending order. Non-matching targets are excluded from results.
func Rank(pattern string, targets []string) []Match {
	var matches []Match
	for i, t := range targets {
		if score, ok := Score(pattern, t); ok {
			matches = append(matches, Match{Score: score, Index: i})
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})
	return matches
}
