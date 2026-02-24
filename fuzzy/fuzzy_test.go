package fuzzy

import "testing"

func TestDamerauLevenshtein(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "", 3},
		{"", "abc", 3},
		{"git", "git", 0},
		{"gti", "git", 1},        // transposition
		{"comit", "commit", 1},   // deletion
		{"committ", "commit", 1}, // insertion
		{"kitten", "sitting", 3},
	}
	for _, tt := range tests {
		got := damerauLevenshtein(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("damerauLevenshtein(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestSplitWords(t *testing.T) {
	words := splitWords(`git commit -am "[OPS-2453] fix/deploy"`)
	found := make(map[string]bool)
	for _, w := range words {
		found[w] = true
	}
	for _, want := range []string{"git", "commit", "am", "OPS-2453", "OPS", "2453", "fix", "deploy"} {
		if !found[want] {
			t.Errorf("splitWords missing %q, got %v", want, words)
		}
	}
}

func TestScoreTokenAgainstWord(t *testing.T) {
	tests := []struct {
		token, word string
		wantMatch   bool
		desc        string
	}{
		{"git", "git", true, "exact match"},
		{"Git", "git", true, "case insensitive exact"},
		{"git", "github", true, "prefix match"},
		{"kube", "kubectl", true, "prefix match"},
		{"ops", "topics", false, "not a prefix or edit-distance match"},
		{"gti", "git", true, "edit distance (transposition)"},
		{"comit", "commit", true, "edit distance (missing char)"},
		{"zz", "git", false, "no match"},
		{"OPS", "OPS-2453", true, "prefix after split is exact on sub-word"},
		{"srti", "strip", true, "truncated + transposed prefix"},
		{"srtip", "strip", true, "transposed full word"},
	}
	for _, tt := range tests {
		_, ok := scoreTokenAgainstWord(tt.token, tt.word)
		if ok != tt.wantMatch {
			t.Errorf("%s: scoreTokenAgainstWord(%q, %q) matched=%v, want %v", tt.desc, tt.token, tt.word, ok, tt.wantMatch)
		}
	}
}

func TestScoreTokenAgainstWordRanking(t *testing.T) {
	exact, _ := scoreTokenAgainstWord("git", "git")
	prefix, _ := scoreTokenAgainstWord("git", "github")
	edit, _ := scoreTokenAgainstWord("gti", "git")

	if exact <= prefix {
		t.Errorf("exact (%d) should beat prefix (%d)", exact, prefix)
	}
	if prefix <= edit {
		t.Errorf("prefix (%d) should beat edit (%d)", prefix, edit)
	}
}

func TestScore(t *testing.T) {
	tests := []struct {
		pattern, target string
		wantMatch       bool
	}{
		{"git", "git commit -m fix", true},
		{"git commit", "git commit -m fix", true},
		{"git deploy", "git commit -m fix", false},
		{"gti", "git commit -m fix", true},           // typo tolerance
		{"comit", "git commit -m fix", true},         // typo tolerance
		{"zz", "git commit", false},                  // no match
		{"kubernetes", "git commit", false},          // no match
		{"OPS", `git commit -am "[OPS-2453]"`, true}, // word split match
		{"ops", "ag topics", false},                  // should NOT match (not a word match)
		{"ops", "gcloud projects list", false},       // should NOT match
	}
	for _, tt := range tests {
		score, matched := Score(tt.pattern, tt.target)
		if matched != tt.wantMatch {
			t.Errorf("Score(%q, %q) matched=%v, want %v (score=%d)", tt.pattern, tt.target, matched, tt.wantMatch, score)
		}
	}
}

func TestScoreMultiToken(t *testing.T) {
	// "git OPS" should match commands that contain both git and OPS as words
	_, ok := Score("git OPS", `git commit -am "[OPS-2453] Add publish action"`)
	if !ok {
		t.Error("'git OPS' should match git commit with OPS ticket")
	}

	// "git OPS" should NOT match things with scattered o-p-s characters
	_, ok = Score("git OPS", "git checkout public-master")
	if ok {
		t.Error("'git OPS' should NOT match 'git checkout public-master'")
	}
}

func TestRank(t *testing.T) {
	targets := []string{
		"ls -la",
		"git commit -m fix",
		"kubectl apply -f deploy-prod.yaml",
		"docker build -t myapp .",
		"go test ./...",
	}
	matches := Rank("git", targets)
	if len(matches) == 0 {
		t.Fatal("Rank('git', ...) returned no matches, expected at least 1")
	}
	if targets[matches[0].Index] != "git commit -m fix" {
		t.Errorf("best match = %q, want %q", targets[matches[0].Index], "git commit -m fix")
	}
	for i := 1; i < len(matches); i++ {
		if matches[i].Score > matches[i-1].Score {
			t.Errorf("matches not sorted: score[%d]=%d > score[%d]=%d", i, matches[i].Score, i-1, matches[i-1].Score)
		}
	}
}

func TestRankMultiToken(t *testing.T) {
	targets := []string{
		"kubectl apply -f deploy-prod.yaml",
		"kubectl get pods",
		"docker build -t prod .",
	}
	matches := Rank("kube prod", targets)
	if len(matches) == 0 {
		t.Fatal("expected matches for 'kube prod'")
	}
	if targets[matches[0].Index] != "kubectl apply -f deploy-prod.yaml" {
		t.Errorf("best match = %q, want kubectl apply line", targets[matches[0].Index])
	}
}

func TestRankRelevanceOrder(t *testing.T) {
	targets := []string{
		"ag topics",
		"gcloud projects list",
		`git commit -am "[OPS-2453] Add publish action"`,
		"git branch -d fix/ops-2281",
		"emacs gradle.properties",
	}

	matches := Rank("git OPS", targets)
	// Should only match targets that contain both "git" and "OPS" as words
	for _, m := range matches {
		target := targets[m.Index]
		if target == "ag topics" || target == "gcloud projects list" || target == "emacs gradle.properties" {
			t.Errorf("'git OPS' should not match %q", target)
		}
	}
	if len(matches) == 0 {
		t.Fatal("expected at least one match for 'git OPS'")
	}
}
