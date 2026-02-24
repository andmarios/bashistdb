package configuration

import (
	"encoding/json"
	"os"
	"testing"
)

func TestWriteConfFileValidJSON(t *testing.T) {
	// Save originals
	origConfFile := confFile
	origDatabase := Database
	origRemote := remote
	origPort := port
	origKey := Key

	// Use a temp file
	f, err := os.CreateTemp("", "bashistdb-conf-test-*.json")
	if err != nil {
		t.Fatal(err)
	}
	tmpPath := f.Name()
	f.Close()
	defer os.Remove(tmpPath)

	confFile = tmpPath
	Database = "/tmp/test.db"
	remote = "10.0.0.1"
	port = "25625"
	Key = []byte("test-key")

	defer func() {
		confFile = origConfFile
		Database = origDatabase
		remote = origRemote
		port = origPort
		Key = origKey
	}()

	if err := writeConfFile(); err != nil {
		t.Fatal("writeConfFile failed:", err)
	}

	// Read it back and verify it's valid JSON
	data, err := os.ReadFile(tmpPath)
	if err != nil {
		t.Fatal("could not read conf file:", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("writeConfFile produced invalid JSON: %v\nContent: %s", err, string(data))
	}

	// Verify round-trip: values should match
	if v, ok := parsed["database"]; !ok || v != "/tmp/test.db" {
		t.Fatalf("database field wrong, got: %v", parsed["database"])
	}
	if v, ok := parsed["remote"]; !ok || v != "10.0.0.1" {
		t.Fatalf("remote field wrong, got: %v", parsed["remote"])
	}
	if v, ok := parsed["port"]; !ok || v != "25625" {
		t.Fatalf("port field wrong, got: %v", parsed["port"])
	}
	if v, ok := parsed["key"]; !ok || v != "test-key" {
		t.Fatalf("key field wrong, got: %v", parsed["key"])
	}
}

func TestWriteConfFileSpecialChars(t *testing.T) {
	origConfFile := confFile
	origDatabase := Database
	origRemote := remote
	origPort := port
	origKey := Key

	f, err := os.CreateTemp("", "bashistdb-conf-test-*.json")
	if err != nil {
		t.Fatal(err)
	}
	tmpPath := f.Name()
	f.Close()
	defer os.Remove(tmpPath)

	confFile = tmpPath
	Database = `/tmp/path with "quotes" and \backslash`
	remote = ""
	port = "25625"
	Key = []byte(`pass"word`)

	defer func() {
		confFile = origConfFile
		Database = origDatabase
		remote = origRemote
		port = origPort
		Key = origKey
	}()

	if err := writeConfFile(); err != nil {
		t.Fatal("writeConfFile failed:", err)
	}

	data, err := os.ReadFile(tmpPath)
	if err != nil {
		t.Fatal("could not read conf file:", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("writeConfFile produced invalid JSON with special chars: %v\nContent: %s", err, string(data))
	}

	if parsed["database"] != Database {
		t.Fatalf("database didn't round-trip. Want %q, got %q", Database, parsed["database"])
	}
	if parsed["key"] != string(Key) {
		t.Fatalf("key didn't round-trip. Want %q, got %q", string(Key), parsed["key"])
	}
}
