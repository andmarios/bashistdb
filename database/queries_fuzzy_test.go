package database

import (
	"os"
	"strings"
	"testing"

	conf "github.com/andmarios/bashistdb/configuration"
)

func init() {
	os.Setenv("BASHISTDB_TEST", "test")
}

func TestFuzzyQuery(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	insertTestCommands(t, db, []string{
		"git commit -m fix",
		"git checkout main",
		"kubectl apply -f deploy-prod.yaml",
		"docker build -t myapp .",
		"go test ./...",
		"ls -la",
	})

	qp := conf.QueryParams{
		Type:    conf.QUERY_FUZZY,
		User:    "%",
		Host:    "%",
		Command: "git",
		Fuzzy:   true,
		Kappa:   25,
		Format:  conf.FORMAT_COMMAND_LINE,
	}

	res, err := db.FuzzyQuery(qp)
	if err != nil {
		t.Fatalf("FuzzyQuery failed: %v", err)
	}

	result := string(res)
	if !strings.Contains(result, "git commit") {
		t.Errorf("expected result to contain 'git commit', got: %s", result)
	}
	if strings.Contains(result, "docker") {
		t.Errorf("expected result NOT to contain 'docker', got: %s", result)
	}
}

func TestFuzzyQueryTypo(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	insertTestCommands(t, db, []string{
		"git commit -m fix",
		"ls -la",
	})

	qp := conf.QueryParams{
		Type:    conf.QUERY_FUZZY,
		User:    "%",
		Host:    "%",
		Command: "gti",
		Fuzzy:   true,
		Kappa:   25,
		Format:  conf.FORMAT_COMMAND_LINE,
	}

	res, err := db.FuzzyQuery(qp)
	if err != nil {
		t.Fatalf("FuzzyQuery failed: %v", err)
	}

	result := string(res)
	if !strings.Contains(result, "git commit") {
		t.Errorf("typo 'gti' should match 'git commit', got: %s", result)
	}
}

func setupTestDB(t *testing.T) (Database, func()) {
	t.Helper()
	db, err := NewWithPath(":memory:")
	if err != nil {
		t.Fatalf("failed to create test db: %v", err)
	}
	return db, func() { db.Close() }
}

func insertTestCommands(t *testing.T, db Database, commands []string) {
	t.Helper()
	for i, cmd := range commands {
		_, err := db.Exec(
			"INSERT INTO history (user, host, command, datetime, shellpid, workdir) VALUES (?, ?, ?, datetime('2024-01-01', '+' || ? || ' hours'), ?, ?)",
			"testuser", "testhost", cmd, i, "1234", "/home/test",
		)
		if err != nil {
			t.Fatalf("failed to insert command %q: %v", cmd, err)
		}
	}
}
