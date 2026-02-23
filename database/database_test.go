// Copyright (c) 2015-2026, Marios Andreopoulos.
//
// This file is part of bashistdb.
//
// 	Bashistdb is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// 	Bashistdb is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// 	You should have received a copy of the GNU General Public License
// along with bashistdb.  If not, see <http://www.gnu.org/licenses/>.

package database

import (
	"bufio"
	"bytes"
	"io/ioutil"
	l "log"
	"net"
	"os"
	"testing"
	"time"

	conf "github.com/andmarios/bashistdb/configuration"
)

func TestNew(t *testing.T) {
	f, err := ioutil.TempFile("", "test-bashistdb")
	if err != nil {
		l.Fatalln(err)
	}
	defer f.Close()
	db := f.Name()
	conf.Database = db
	f.Close()
	os.Remove(db)
	testdb, err := New()
	if err != nil {
		l.Fatalln(err)
	}
	defer os.Remove(db)

	// Test add record
	tt := time.Date(2015, 1, 1, 1, 1, 0, 0, time.UTC)
	err = testdb.AddRecord("user1", "host1", "htop", tt, "", "")
	if err != nil {
		t.Fatal("AddRecord failed: " + err.Error())
	}
	// Test try to add duplicate record
	err = testdb.AddRecord("user1", "host1", "htop", tt, "", "")
	if err != nil {
		t.Fatal("AddRecord failed: " + err.Error())
	}

	// Test add from buffer: default (history pipe) import:
	// also test for duplicate records
	br := bufio.NewReader(bytes.NewReader(entriesDefault))
	stats, err := testdb.AddFromBuffer(br, "user", "test", "", "")
	if err != nil {
		t.Fatal("AddFromBuffer failed: ", err.Error())
	}
	if stats != entriesDefaultExpect {
		t.Fatalf("AddFromBuffer returned wrong stats.\n"+
			"Wanted: %s\nGot   : %s", entriesDefaultExpect, stats)
	}

	// Test add from buffer, restore (bashist export) format:
	// also test for bad records
	br = bufio.NewReader(bytes.NewReader(entriesImport))
	stats, err = testdb.AddFromBuffer(br, "", "", "", "")
	if err != nil {
		t.Fatal("AddFromBuffer failed: ", err.Error())
	}
	if stats != entriesImportExpect {
		t.Fatalf("AddFromBuffer returned wrong stats.\n"+
			"Wanted: %s\nGot   : %s", entriesImportExpect, stats)
	}

	// Test log connection
	na, _ := net.ResolveTCPAddr("tcp", "localhost:25625")
	err = testdb.LogConn(na)
	if err != nil {
		t.Fatal("LogConn failed.")
	}

	// Test some of migration
	testdb, err = New()
	if err != nil {
		l.Fatalln(err)
	}

	const (
		_ = iota
		OK
		ER
	)

	queries := []struct {
		params conf.QueryParams
		expect int
		want   string
		test   string
	}{
		{ // TopK
			params: conf.QueryParams{Type: conf.QUERY_TOPK, Kappa: 2, User: "user1", Host: "host1", Command: "%%"},
			expect: OK,
			want:   "4 | topk 1\n" + "3 | topk 2",
			test:   "topk",
		},
		{ // TopK with global
			params: conf.QueryParams{Type: conf.QUERY_TOPK, Kappa: 2, User: "%", Host: "%", Format: conf.FORMAT_COMMAND_LINE, Command: "%%"},
			expect: OK,
			want:   "7 | topk 1\n" + "4 | topk 2",
			test:   "topk global",
		},
		{ // default query
			params: conf.QueryParams{Type: conf.QUERY, User: "user1", Host: "host1", Format: conf.FORMAT_COMMAND_LINE, Command: "%default%"},
			expect: OK,
			want:   "18 default query\n" + "19 default query",
			test:   "default query",
		},
		{ // default query with unique (should return latest command instance)
			params: conf.QueryParams{Type: conf.QUERY, User: "user1", Host: "host1", Format: conf.FORMAT_COMMAND_LINE, Command: "%default%", Unique: true},
			expect: OK,
			want:   "19 default query",
			test:   "default query unique",
		},
		{ // users
			params: conf.QueryParams{Type: conf.QUERY_USERS, User: "%", Host: "%", Format: conf.FORMAT_COMMAND_LINE, Command: "%%", Unique: true},
			expect: OK,
			want:   "Unique user-hosts pairs:\n" + "user@test\n" + "user1@host1\n" + "user1@host2\n" + "user2@host1\n" + "user3@host2",
			test:   "users",
		},
		{ // users at host
			params: conf.QueryParams{Type: conf.QUERY_USERS, User: "%", Host: "host2", Format: conf.FORMAT_COMMAND_LINE, Command: "%%", Unique: true},
			expect: OK,
			want:   "Unique user-hosts pairs:\n" + "user1@host2\n" + "user3@host2",
			test:   "users at host",
		},
		{ // row
			params: conf.QueryParams{Type: conf.QUERY_ROW, Kappa: 20, User: "user1", Host: "host1", Command: "%%"},
			expect: OK,
			want:   "row 20",
			test:   "row",
		},
		{ // delete rows
			params: conf.QueryParams{Type: conf.DELETE, Rows: []int{21, 22, 10000}},
			expect: OK,
			want:   "No errors during deletion.",
			test:   "delete rows",
		},
		{ // check row deleted
			params: conf.QueryParams{Type: conf.QUERY_ROW, Kappa: 21, User: "user1", Host: "host1", Command: "%%"},
			expect: ER,
			want:   "",
			test:   "check deleted rows",
		},
		{ // LastK
			params: conf.QueryParams{Type: conf.QUERY_LASTK, Kappa: 2, User: "%", Host: "%", Format: conf.FORMAT_COMMAND_LINE, Command: "%%"},
			expect: OK,
			want:   "24 lastk 2\n" + "25 lastk 2",
			test:   "lastk",
		},
		{ // LastK unique
			params: conf.QueryParams{Type: conf.QUERY_LASTK, Kappa: 2, User: "%", Host: "%", Format: conf.FORMAT_COMMAND_LINE, Command: "%%", Unique: true},
			expect: OK,
			want:   "23 lastk 1\n" + "25 lastk 2",
			test:   "lastk unique",
		},
		{ // Unkown Command
			params: conf.QueryParams{Type: "bad command", Kappa: 2, User: "%", Host: "%", Format: conf.FORMAT_COMMAND_LINE, Command: "%%", Unique: true},
			expect: ER,
			want:   "",
			test:   "unknown command",
		},
		{ // demo
			params: conf.QueryParams{Type: conf.QUERY_DEMO, User: "%", Host: "%", Format: conf.FORMAT_COMMAND_LINE, Command: "%%"},
			expect: OK,
			want:   demoResponse,
			test:   "demo",
		},
	}

	for _, v := range queries {
		res, err := testdb.RunQuery(v.params)
		switch v.expect {
		case OK:
			if err != nil {
				t.Fatal(err.Error())
			}
			if string(res) != v.want {
				t.Fatalf("Test '%s'\nWanted: %s\nGot   : %s",
					v.test, v.want, string(res))
			}
		case ER:
			if err == nil {
				t.Fatalf("Test '%s' should have returned error. "+
					"Instead  returned: %s.", v.test, string(res))
			}
		}
	}
}

// Test add from buffer, default format
// Out of 5, 4 are accepted, one is duplicate.
var entriesDefault = []byte(`99  2015-10-12T12:00:00+0000 ls
100  2015-10-12T12:00:05+0000 git status
101  2015-10-12T12:00:10+0000 go run bashistdb.go --local -db test.sqlite3 -format rows -lastk 40
102  2015-10-12T12:00:15+0000 history
103  2015-10-12T12:00:15+0000 history
`)
var entriesDefaultExpect = "Processed 5 entries, successful 4, failed 1."

// Test add from buffer, export format
// Out of 18, 17 are accepted, one is bad.
var entriesImport = []byte(`user1 host1 2015-10-12T12:00:40+0000 topk
user1 host1 2015-10-12T12:00:41+0000 topk 1
user1 host1 2015-10-12T12:00:42+0000 topk 1
user1 host1 2015-10-12T12:00:43+0000 topk 1
user1 host1 2015-10-12T12:00:44+0000 topk 1
user2 host1 2015-10-12T12:00:45+0000 topk 1
user2 host1 2015-10-12T12:00:46+0000 topk 1
user3 host2 2015-10-12T12:00:47+0000 topk 1
user1 host2 2015-10-12T12:00:48+0000 topk 2
user1 host1 2015-10-12T12:00:49+0000 topk 2
user1 host1 2015-10-12T12:00:50+0000 topk 2
user1 host1 2015-10-12T12:00:55+0000 topk 2
user1 host1 2015-10-12T12:01:40+0000 default query
user1 host1 2015-10-12T12:01:50+0000 default query
user1 host1 2015-10-12T12:02:40+0000 row 20
user1 host1 2015-10-12T12:02:45+0000 delete 21
user1 host1 2015-10-12T12:02:50+0000 delete 22
user1 host1 2015-10-12T12:03:40+0000 lastk 1
user1 host1 2015-10-12T12:03:45+0000 lastk 2
user1 host1 2015-10-12T12:03:50+0000 lastk 2
user1 host1 nodate command
`)
var entriesImportExpect = "Processed 21 entries, successful 20, failed 1."

var demoResponse = `There are 23 command lines (12 unique) in your database from 5 users across 3 hosts.

Top-15 commands for user %@%:
7 | topk 1
4 | topk 2
2 | default query
2 | lastk 2
1 | git status
1 | go run bashistdb.go --local -db test.sqlite3 -format rows -lastk 40
1 | history
1 | htop
1 | lastk 1
1 | ls
1 | row 20
1 | topk

Last 10 commands user %@% ran:
14 topk 2
15 topk 2
16 topk 2
17 topk 2
18 default query
19 default query
20 row 20
23 lastk 1
24 lastk 2
25 lastk 2`
