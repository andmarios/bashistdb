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

// Package result handles the output formatting for bashistdb.
package result

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	conf "github.com/andmarios/bashistdb/configuration"
)

// Format Strings
const (
	FORMAT_BASH_HISTORY_S = "#%d\n%s"
	FORMAT_ALL_S          = "%05d | %s | % 10s | % 10s | %6s | %s | %s"
	FORMAT_COMMAND_LINE_S = "%d %s"
	FORMAT_TIMESTAMP_S    = "%s: %s"
	FORMAT_LOG_S          = "%s %s@%s [%s] %s"
	FORMAT_JSON_S         = "" // We use encoding/json for JSON
	FORMAT_EXPORT_S       = "%s %s %s %s %s %s"
	FORMAT_ROWS_S         = "%d"
)

// A Result is used to store the formatted output of a query.
// Result's methods are responsible for formatting according to
// requested output format.
type Result struct {
	out     *bytes.Buffer
	written *bool // we use this to work around json not accepting a trailing comma
	format  string
	digits  *int // we use this to set the width of the count column to that of the first result (max)
}

// Golang's RFC3339 does not comply with all RFC3339 representations
const RFC3339alt = "2006-01-02T15:04:05-0700"

// New returns a new Result
func New(format string) *Result {
	var out bytes.Buffer
	if format == conf.FORMAT_JSON {
		out.WriteString("[\n")
	}
	w := false
	d := 0
	return &Result{out: &out, written: &w, format: format, digits: &d}
}

// A rowJSON is an internal struct to use with json.Marshal
type rowJSON struct {
	Row                                              int
	Datetime, User, Host, Command, ShellPID, Workdir string
}

// AddRow adds a query row to a Result struct. This function is not thread safe!
func (r Result) AddRow(row int, user, host, command, shellpid, workdir string, datetime time.Time) {
	var f string

	switch *r.written {
	case true:
		switch r.format {
		case conf.FORMAT_JSON:
			_, _ = r.out.WriteString(",\n")
		case conf.FORMAT_ROWS:
			_, _ = r.out.WriteString(",")
		default:
			_ = r.out.WriteByte('\n')
		}
	default:
		*r.written = true
	}

	switch r.format {
	case conf.FORMAT_ALL:
		f = fmt.Sprintf(FORMAT_ALL_S, row, datetime, user, host, shellpid, workdir, command)
	case conf.FORMAT_BASH_HISTORY:
		f = fmt.Sprintf(FORMAT_BASH_HISTORY_S, datetime.Unix(), command)
	case conf.FORMAT_TIMESTAMP:
		f = fmt.Sprintf(FORMAT_TIMESTAMP_S, datetime, command)
	case conf.FORMAT_LOG:
		f = fmt.Sprintf(FORMAT_LOG_S, datetime.Format(RFC3339alt), user, host, workdir, command)
	case conf.FORMAT_JSON:
		b, err := json.Marshal(rowJSON{row, datetime.Format(RFC3339alt), user, host, command, shellpid, workdir})
		if err != nil {
			_, _ = r.out.WriteString(fmt.Sprintf(`{"error": "json marshal failed: %s"}`, err.Error()))
		} else {
			_, _ = r.out.Write(b)
		}
		f = ""
	case conf.FORMAT_EXPORT:
		pid := shellpid
		if pid == "" {
			pid = "0"
		}
		cwd := workdir
		if cwd == "" {
			cwd = "."
		}
		f = fmt.Sprintf(FORMAT_EXPORT_S, user, host, pid, url.PathEscape(cwd), datetime.Format(RFC3339alt), command)
	case conf.FORMAT_ROWS:
		f = fmt.Sprintf(FORMAT_ROWS_S, row)
	case conf.FORMAT_COMMAND_LINE:
		fallthrough
	default:
		f = fmt.Sprintf(FORMAT_COMMAND_LINE_S, row, command)

	}
	r.out.WriteString(f)
}

// Formatted returns the result in the desired format after performing any necessary adjustment.
func (r Result) Formatted() []byte {
	if r.format == conf.FORMAT_JSON {
		r.out.WriteString("\n]")
	}
	return r.out.Bytes()
}

// AddCountRow adds a count query row to a Result struct. This function is not thread safe!
// It is used by TopK database function.
func (r Result) AddCountRow(count int, command string) {
	var f string

	switch *r.written {
	case true:
		_, _ = r.out.WriteString("\n")
	default:
		*r.written = true
		*r.digits = digits(count)
	}

	f = fmt.Sprintf("%[2]*.[1]d | %[3]s", count, *r.digits, command)

	r.out.WriteString(f)
}

func digits(n int) int {
	if n < 10 {
		return 1
	}
	return digits(n/10) + 1
}
