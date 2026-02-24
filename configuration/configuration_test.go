// Copyright (c) 2015-2026, Marios Andreopoulos.
//
// This file is part of bashistdb.
//
//      Bashistdb is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
//      Bashistdb is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
//      You should have received a copy of the GNU General Public License
// along with bashistdb.  If not, see <http://www.gnu.org/licenses/>.

/*
Testing configuration is difficult because it is meant to run on the init stage,
use flags, use non-flag command line arguments, use environment variables and settings
from a file. All these are almost impossible to test. Even more so since they overlap.

Thus we test parts of our code. In general we accept that the process below is correct:

1. Set default variables. Some are read from the environment.
2. If there is a configuration file, read it and update the variables it includes values for.

And we test the rest of the code that uses the variables and set flags to set the exported
set of variables.

It is important to test and use go coverage tool for this test. It will help you see
if the test passed from the codepaths you would expect.
*/

package configuration

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"testing"
)

func init() {
	os.Setenv("BASHISTDB_TEST", "test")
}

func resetFlags(args ...string) {
	os.Args = args
	// This is where magic happens. Think of it as resetting the flag package.
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// These are used as actual flagvars
	database = "test.sqlite3"
	versionSet = false
	verbosity = 0
	user = "test"
	host = "test"
	serverSet = false
	remote = ""
	port = ""
	passphrase = ""
	format = FORMAT_DEFAULT
	helpSet = false
	globalSet = false
	writeconfSet = false
	setupSet = false
	topk = 20
	lastk = 20
	localSet = false
	uniqueSet = false
	usersSet = false
	row = 0
	regexSet = false
	// Here we will store the non flag arguments //
	// These are not parsed from flags but we set them with flag.Visit
	userSet = false
	hostSet = false
	remoteSet = false
	topkSet = false
	lastkSet = false
	rowSet = false
	// These are set with manual searches
	querySet = false
	stdinSet = false
	// Vars below can not be overriden by user
	confFile = "test.conf"
	foundConfFile = false

	// Exported variables
	Mode = 0
	Operation = 0
	Address = ""
	Database = ""
	Key = []byte{}
	User = ""
	Hostname = ""
	QParams = *new(QueryParams)
}

func TestParse(t *testing.T) {
	const (
		_  = iota
		OK // We expect test to pass
		ER // We expect test to return error
	)

	test := []struct {
		want   exportedVars
		expect int
		input  []string
		test   string
	}{
		{
			want: exportedVars{Mode: MODE_CLIENT, Operation: OP_QUERY, Address: "10.10.0.1:4000", Database: "test.sqlite", User: "test", Hostname: "test",
				QParams: QueryParams{Type: QUERY_DEMO, User: "test", Host: "test", Format: FORMAT_DEFAULT, Command: "%%"}},
			expect: OK,
			input:  []string{"cmd", "-r", "10.10.0.1", "-p", "4000", "-db", "test.sqlite"},
			test:   "Test a simple demo in remote mode: ",
		},
		{
			expect: ER,
			input:  []string{"cmd", "-lastk", "5", "-topk", "5"},
			test:   "Test topk and lastk incompatibility: ",
		},
		{
			want: exportedVars{Mode: MODE_LOCAL, Operation: OP_QUERY, Address: "", Database: "test.sqlite3", User: "test1", Hostname: "test2",
				QParams: QueryParams{Type: QUERY_LASTK, User: "test1", Host: "test2", Format: FORMAT_JSON, Command: "%git%", Unique: true, Kappa: 5}},
			expect: OK,
			input:  []string{"cmd", "-U", "test1", "-host", "test2", "-lastk", "5", "-format", "json", "-unique", "git"},
			test:   "Test lastk and non flag local: ",
		},
		{
			expect: ER,
			input:  []string{"cmd", "-tail", "5", "-topk", "5"},
			test:   "Test tail and topk incompatibility: ",
		},
		{
			expect: ER,
			input:  []string{"cmd", "-U", "test1", "-host", "test2", "-lastk", "5", "-format", "json", "-unique", "-s", "git"},
			test:   "Test server and operation incompatibility: ",
		},
		{
			expect: ER,
			input:  []string{"cmd", "-s", "-r", "localhost"},
			test:   "Test server and remote incompatibility: ",
		},
		{
			expect: ER,
			input:  []string{"cmd", "-r", "localhost", "-row", "500", "-unique", "ls"},
			test:   "Test row - default query incompatibility: ",
		},
		{
			want: exportedVars{Mode: MODE_CLIENT, Operation: OP_QUERY, Address: "localhost:25625", Database: "test.sqlite3", User: "test", Hostname: "test",
				QParams: QueryParams{Type: QUERY_ROW, User: "test", Host: "test", Format: FORMAT_BASH_HISTORY, Command: "%%", Unique: true, Kappa: 500}},
			expect: OK,
			input:  []string{"cmd", "-r", "localhost", "-row", "500", "-format", "restore", "-unique"},
			test:   "Test row query: ",
		},
		{
			want: exportedVars{Mode: MODE_CLIENT, Operation: OP_QUERY, Address: "localhost:25625", Database: "test.sqlite3", User: "test", Hostname: "test",
				QParams: QueryParams{Type: QUERY_ROW, User: "test", Host: "test", Format: FORMAT_DEFAULT, Command: "%%", Unique: true, Kappa: 500}},
			expect: OK,
			input:  []string{"cmd", "-r", "localhost", "-row", "500", "-format", "abadformat", "-unique"},
			test:   "Test non-existant format, use default: ",
		},
		{
			want: exportedVars{Mode: MODE_LOCAL, Operation: OP_QUERY, Address: "", Database: "test.sqlite3", User: "test", Hostname: "test",
				QParams: QueryParams{Type: QUERY, User: "%", Host: "%", Format: FORMAT_DEFAULT, Command: "%s%s%"}},
			expect: OK,
			input:  []string{"cmd", "-g", "s%s"},
			test:   "Test global flag: ",
		},
		{
			want: exportedVars{Mode: MODE_LOCAL, Operation: OP_QUERY, Address: "", Database: "test.sqlite3", User: "test", Hostname: "test",
				QParams: QueryParams{Type: QUERY_USERS, User: "%", Host: "%", Format: FORMAT_DEFAULT, Command: "%%", Unique: false, Kappa: 0}},
			expect: OK,
			input:  []string{"cmd", "--local", "-users"},
			test:   "Test users flag: ",
		},
		{
			want: exportedVars{Mode: MODE_LOCAL, Operation: OP_QUERY, Address: "", Database: "test.sqlite3", User: "test", Hostname: "test",
				QParams: QueryParams{Type: DELETE, User: "test", Host: "test", Format: FORMAT_DEFAULT, Command: "%%", Rows: []int{1, 3, 4, 5, 9}}},
			expect: OK,
			input:  []string{"cmd", "-del", "1,3-5,9,3"},
			test:   "Test del flag: ",
		},
		{
			expect: ER,
			input:  []string{"cmd", "-del", "1,3-5,9,f"},
			test:   "Test del flag with bad row ids: ",
		},
		{
			expect: ER,
			input:  []string{"cmd", "-del", "1,3-5", "-row", "5"},
			test:   "Test del flag with non-compatible row flag: ",
		},
		{
			want:   exportedVars{Mode: MODE_HELP},
			expect: OK,
			input:  []string{"cmd", "-s", "-k", "pass", "-h"},
			test:   "Test help flag: ",
		},
	}

	// Test remote override by server (remote set by env or conf file)
	resetFlags("cmd", "-s")
	remote = "localhost"
	if err := parse(); err != nil {
		t.Fatalf("Test remote override by server failed. %s", err.Error())
	}

	for _, v := range test {
		resetFlags(v.input...)
		err := parse()
		switch v.expect {
		case OK:
			if err != nil {
				t.Fatal(v.test + err.Error())
			}
			if err := compare(v.want); err != nil {
				t.Fatal(v.test + err.Error())
			}
		case ER:
			if err == nil {
				t.Fatal(v.test + "should not get error")
			}
		}
	}

	PrintHelp(io.Discard)

}

type exportedVars struct {
	Mode      int         // Mode of operation (local, server, client, etc)
	Operation int         // function (read, restore, et)
	Address   string      // Address is the remote server's address for client mode or server's address for server mode
	Database  string      // Database is the filename of the sqlite database
	Key       []byte      // Key it the user passphrase to generate keys for net comms
	User      string      // User is the username detected or explicitly set
	Hostname  string      // Hostname is the hostname detected or explicitly set
	QParams   QueryParams // Parameters to query
}

func compare(v exportedVars) error {
	s := ""
	if Mode != v.Mode {
		s += fmt.Sprintf("Mode wrong. Wanted %d, got %d.\n", v.Mode, Mode)
	}
	if Operation != v.Operation {
		s += fmt.Sprintf("Operation wrong. Wanted %d, got %d.\n", v.Operation, Operation)
	}
	if Address != v.Address {
		s += fmt.Sprintf("Address wrong. Wanted %s, got %s.\n", v.Address, Address)
	}
	if Database != v.Database {
		s += fmt.Sprintf("Database wrong. Wanted %s, got %s.\n", v.Database, Database)
	}
	if string(Key) != string(v.Key) {
		s += fmt.Sprintf("Key wrong. Wanted %s, got %s.\n", string(v.Key), string(Key))
	}
	if User != v.User {
		s += fmt.Sprintf("User wrong. Wanted %s, got %s.\n", v.User, User)
	}
	if Hostname != v.Hostname {
		s += fmt.Sprintf("Hostname wrong. Wanted %s, got %s.\n", v.Hostname, Hostname)
	}

	if QParams.Type != v.QParams.Type {
		s += fmt.Sprintf("QParams.Type wrong. Wanted %s, got %s.\n", v.QParams.Type, QParams.Type)
	}
	if QParams.Kappa != v.QParams.Kappa {
		s += fmt.Sprintf("QParams.Kappa wrong. Wanted %d, got %d.\n", v.QParams.Kappa, QParams.Kappa)
	}
	if QParams.User != v.QParams.User {
		s += fmt.Sprintf("QParams.User wrong. Wanted %s, got %s.\n", v.QParams.User, QParams.User)
	}
	if QParams.Host != v.QParams.Host {
		s += fmt.Sprintf("QParams.Host wrong. Wanted %s, got %s.\n", v.QParams.Host, QParams.Host)
	}

	if QParams.Format != v.QParams.Format {
		s += fmt.Sprintf("QParams.Format wrong. Wanted %s, got %s.\n", v.QParams.Format, QParams.Format)
	}
	if QParams.Command != v.QParams.Command {
		s += fmt.Sprintf("QParams.Command wrong. Wanted %s, got %s.\n", v.QParams.Command, QParams.Command)
	}
	if QParams.Unique != v.QParams.Unique {
		s += fmt.Sprintf("QParams.Unique wrong. Wanted %v, got %v.\n", v.QParams.Unique, QParams.Unique)
	}
	if !compareIntSlice(QParams.Rows, v.QParams.Rows) {
		s += fmt.Sprintf("QParams.Rows wrong. Wanted %v, got %v.\n", v.QParams.Rows, QParams.Rows)
	}

	if s != "" {
		return errors.New(s)
	}
	return nil
}
