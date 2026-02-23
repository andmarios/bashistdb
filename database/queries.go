// Copyright (c) 2015, Marios Andreopoulos.
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

/*
Package database handles a SQLite3 database and access methods for the
specific needs of bashistdb.
*/
package database

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	conf "github.com/andmarios/bashistdb/configuration"
	"github.com/andmarios/bashistdb/result"
)

// sessionWorkdirFilter returns SQL clause fragments and args for session/workdir filtering.
func sessionWorkdirFilter(qp conf.QueryParams) (clause string, args []interface{}) {
	if qp.Session != "" {
		clause += " AND shellpid = ?"
		args = append(args, qp.Session)
	}
	if qp.Workdir != "" {
		clause += ` AND workdir LIKE ? ESCAPE '\'`
		args = append(args, qp.Workdir)
	}
	return
}

// TopK returns the k most frequent command lines in history
func (d Database) TopK(qp conf.QueryParams) ([]byte, error) {
	filterClause, filterArgs := sessionWorkdirFilter(qp)
	args := append([]interface{}{qp.User, qp.Host, qp.Command}, filterArgs...)
	args = append(args, qp.Kappa)
	rows, err := d.Query(`SELECT command, count(*) as count FROM history
                               WHERE user LIKE ? AND host LIKE ? AND command LIKE ? ESCAPE '\'`+filterClause+`
                               GROUP BY command ORDER BY count DESC LIMIT ?`,
		args...)
	if err != nil {
		return []byte{}, err
	}
	defer rows.Close()

	res := result.New("")
	for rows.Next() {
		var command string
		var count int
		rows.Scan(&command, &count)
		res.AddCountRow(count, command)
	}
	return res.Formatted(), err
}

// LastK returns the k most recent command lines in history
func (d Database) LastK(qp conf.QueryParams) ([]byte, error) {
	filterClause, filterArgs := sessionWorkdirFilter(qp)
	baseArgs := append([]interface{}{qp.User, qp.Host, qp.Command}, filterArgs...)
	args := append(baseArgs, qp.Kappa)

	var rows *sql.Rows
	var err error
	switch qp.Unique {
	case true:
		rows, err = d.Query(`SELECT * FROM
                                      (SELECT rowid, * FROM history
                                         WHERE user LIKE ? AND host LIKE ? AND command LIKE ? ESCAPE '\'`+filterClause+`
                                         GROUP BY command
                                         ORDER BY datetime DESC LIMIT ?)
                                      ORDER BY datetime ASC`,
			args...)
	default:
		rows, err = d.Query(`SELECT * FROM
                                      (SELECT rowid, * FROM history
                                         WHERE user LIKE ? AND host LIKE ? AND command LIKE ? ESCAPE '\'`+filterClause+`
                                         ORDER BY datetime DESC LIMIT ?)
                                   ORDER BY datetime ASC`,
			args...)
	}
	if err != nil {
		return []byte{}, err
	}
	defer rows.Close()

	res := result.New(qp.Format)
	for rows.Next() {
		var user, host, command, shellpid, workdir string
		var t time.Time
		var row int
		rows.Scan(&row, &user, &host, &command, &t, &shellpid, &workdir)
		res.AddRow(row, user, host, command, shellpid, workdir, t)
	}
	return res.Formatted(), nil
}

// DefaultQuery returns history within the search criteria in the format requested
func (d Database) DefaultQuery(qp conf.QueryParams) ([]byte, error) {
	// SQLite's regexp extension is problematic; most systems don't have it, loading
	// it is extremely error prone (almost impossible to get right), even we manage
	// to load it, it doesn't seem to work with our queries. Thus I use go's regexp
	// library. This makes bashistdb slower when in regexp mode since it has to process
	// all the history entries itself. Also it makes difficult to add pcre support
	// to anything but the DefaultQuery
	var regex *regexp.Regexp
	var err error
	commandQuery := "AND command LIKE ?" // This is used for normal searches. Fast.
	if qp.Regex {
		regex, err = regexp.Compile(qp.Command)
		if err != nil {
			return []byte{}, err
		}
		commandQuery = "" // For PCRE we do the search, so we want everything. Slow.
	}

	filterClause, filterArgs := sessionWorkdirFilter(qp)

	var rows *sql.Rows
	switch qp.Unique {
	case true:
		args := append([]interface{}{qp.User, qp.Host, qp.Command}, filterArgs...)
		rows, err = d.Query(`SELECT rowid, * FROM history
                                        WHERE user LIKE ? AND host LIKE ? `+commandQuery+` ESCAPE '\'`+filterClause+`
                                        GROUP BY command ORDER BY DATETIME ASC`,
			args...)
	default:
		args := append([]interface{}{qp.User, qp.Host, qp.Command}, filterArgs...)
		rows, err = d.Query(`SELECT rowid, * FROM history
                                         WHERE user LIKE ? AND host LIKE ? `+commandQuery+` ESCAPE '\'`+filterClause,
			args...)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := result.New(qp.Format)
	for rows.Next() {
		var user, host, command, shellpid, workdir string
		var t time.Time
		var row int
		rows.Scan(&row, &user, &host, &command, &t, &shellpid, &workdir)
		switch qp.Regex {
		case true:
			if regex.MatchString(command) {
				res.AddRow(row, user, host, command, shellpid, workdir, t)
			}
		default:
			res.AddRow(row, user, host, command, shellpid, workdir, t)
		}
	}
	// Return the result without the newline at the end.
	return res.Formatted(), nil
}

// RunQuery is a wrapper around various queries.
func (d Database) RunQuery(p conf.QueryParams) ([]byte, error) {
	switch p.Type {
	case conf.QUERY:
		return d.DefaultQuery(p)
	case conf.QUERY_LASTK:
		return d.LastK(p)
	case conf.QUERY_TOPK:
		return d.TopK(p)
	case conf.QUERY_USERS:
		return d.Users(p)
	case conf.QUERY_DEMO:
		return d.Demo(p)
	case conf.QUERY_ROW:
		return d.ReturnRow(p)
	case conf.DELETE:
		return d.DeleteRows(p)
	case conf.QUERY_CONTENT:
		return d.ContentQuery(p)
	}

	return []byte{}, errors.New("Unknown query type.")
}

// Users returns unique user@host pairs from the database.
func (d Database) Users(qp conf.QueryParams) (res []byte, e error) {
	filterClause, filterArgs := sessionWorkdirFilter(qp)
	args := append([]interface{}{qp.User, qp.Host, qp.Command}, filterArgs...)
	var result bytes.Buffer
	result.WriteString(fmt.Sprintf("Unique user-hosts pairs:"))
	rows, e := d.Query(`SELECT distinct(user), host FROM history
                               WHERE user LIKE ? AND host LIKE ? AND command LIKE ? ESCAPE '\'`+filterClause,
		args...)
	if e != nil {
		return result.Bytes(), e
	}
	defer rows.Close()

	for rows.Next() {
		var user string
		var host string
		rows.Scan(&user, &host)
		result.WriteString(fmt.Sprintf("\n%s@%s", user, host))
	}
	return result.Bytes(), e
}

// Demo returns some stats from the database to showcase bashistdb.
func (d Database) Demo(qp conf.QueryParams) (res []byte, e error) {
	var result bytes.Buffer

	var numUsers int
	err := d.QueryRow("SELECT count(*) FROM (SELECT distinct(user), host FROM history)").Scan(&numUsers)
	if err != nil {
		return result.Bytes(), err
	}

	var numHosts int
	err = d.QueryRow("SELECT count(distinct(host)) FROM history").Scan(&numHosts)
	if err != nil {
		return result.Bytes(), err
	}

	var numLines int
	err = d.QueryRow("SELECT count(command) FROM history").Scan(&numLines)
	if err != nil {
		return result.Bytes(), err
	}

	var numUniqueLines int
	err = d.QueryRow("SELECT count(distinct(command)) FROM history").Scan(&numUniqueLines)
	if err != nil {
		return result.Bytes(), err
	}

	qp.Kappa = 15
	restop, err := d.TopK(qp)
	if err != nil {
		return result.Bytes(), err
	}

	qp.Kappa = 10
	reslast, err := d.LastK(qp)
	if err != nil {
		return result.Bytes(), err
	}

	result.WriteString(fmt.Sprintf("There are %d command lines (%d unique) in your database from %d users across %d hosts.\n\n", numLines, numUniqueLines, numUsers, numHosts))

	result.WriteString(fmt.Sprintf("Top-15 commands for user %s@%s:\n", qp.User, qp.Host))
	result.Write(restop)

	result.WriteString(fmt.Sprintf("\n\nLast 10 commands user %s@%s ran:\n", qp.User, qp.Host))
	result.Write(reslast)

	return result.Bytes(), nil
}

// ReturnRow returns a single row with no other data.
// It is useful to pipe to bash.
func (d Database) ReturnRow(qp conf.QueryParams) ([]byte, error) {
	var command string
	err := d.QueryRow("SELECT command FROM history WHERE rowid = ?", qp.Kappa).Scan(&command)
	if err != nil {
		return []byte{}, err
	}
	return []byte(command), nil
}

// DeleteRows deletes a range of rows.
func (d Database) DeleteRows(qp conf.QueryParams) ([]byte, error) {
	tx, err := d.Begin()
	defer tx.Rollback()
	if err != nil {
		return []byte{}, err
	}
	stmt, err := tx.Prepare(`DELETE FROM history WHERE rowid=?`)
	if err != nil {
		return []byte{}, err
	}

	for i := len(qp.Rows) - 1; i >= 0; i-- {
		_, err = stmt.Exec(qp.Rows[i])
		if err != nil {
			return []byte{}, err
		}
	}
	tx.Commit()
	return []byte("No errors during deletion."), nil
}

// ContentQuery returns matches of a row plus rows before or after the match.
// Think of it as grep -A(fter) / -B(efore) / -C(ontent)
// It works on 4 stages:
// 1. find matches and get their datetime
// 2. for each match, get the content asked by rowid
// 3. if the content of two matches overlap, join them
// 4. given the sets of rowids, get them from the database
func (d Database) ContentQuery(qp conf.QueryParams) ([]byte, error) {
	// Using Goland regexp library. Check DefaultQuery() for more info.
	var regex *regexp.Regexp
	var err error
	commandQuery := "AND command LIKE ?" // This is used for normal searches. Fast.
	if qp.Regex {
		regex, err = regexp.Compile(qp.Command)
		if err != nil {
			return []byte{}, err
		}
		commandQuery = "" // For PCRE we do the search, so we want everything. Slow.
	}

	filterClause, filterArgs := sessionWorkdirFilter(qp)

	// Stage 1: find matches and get an array with their datetime
	var rows *sql.Rows
	stage1Args := append([]interface{}{qp.User, qp.Host, qp.Command}, filterArgs...)
	rows, err = d.Query(`SELECT datetime, command FROM history
                                         WHERE user LIKE ? AND host LIKE ? `+commandQuery+` ESCAPE '\'`+filterClause,
		stage1Args...)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hits []time.Time
	for rows.Next() {
		var t time.Time
		var command string
		rows.Scan(&t, &command)
		switch qp.Regex {
		case true:
			if regex.MatchString(command) {
				hits = append(hits, t)
			}
		default:
			hits = append(hits, t)
		}
	}

	// Stage 2: for each match create a slice with its content by rowid
	var hitsContent [][]int
	for _, v := range hits {
		var content []int
		// Before query also includes the current command, thus is always run.
		beforeArgs := append([]interface{}{v, qp.User, qp.Host}, filterArgs...)
		beforeArgs = append(beforeArgs, qp.BeforeContent+1)
		rows, err = d.Query(`SELECT rowid, datetime FROM
                                      (SELECT rowid, datetime FROM history
	                                     WHERE datetime <= ? AND user LIKE ? AND host LIKE ? ESCAPE '\'`+filterClause+`
                                         ORDER BY datetime DESC LIMIT ?)
                                      ORDER BY datetime ASC`,
			beforeArgs...) // Here we include current query to before
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var row int
			var datetime time.Time
			rows.Scan(&row, &datetime)
			content = append(content, row)
		}
		// After runs only if needed.
		if qp.AfterContent > 0 {
			afterArgs := append([]interface{}{v, qp.User, qp.Host}, filterArgs...)
			afterArgs = append(afterArgs, qp.AfterContent)
			rows, err = d.Query(`SELECT rowid, datetime FROM history
	                                         WHERE datetime > ? AND user LIKE ? AND host LIKE ? ESCAPE '\'`+filterClause+`
                                             ORDER BY datetime ASC LIMIT ?`,
				afterArgs...)
			if err != nil {
				return nil, err
			}
			defer rows.Close()
			for rows.Next() {
				var row int
				var datetime time.Time
				rows.Scan(&row, &datetime)
				content = append(content, row)
			}
		}
		hitsContent = append(hitsContent, content)
	}

	// Stage 3: let's merge the sets that overlap
	for i := len(hitsContent) - 2; i >= 0; i-- {
		for k, v := range hitsContent[i] {
			if v == hitsContent[i+1][0] {
				hitsContent[i] = append(hitsContent[i][:k], hitsContent[i+1]...)
				if len(hitsContent) > i+2 {
					hitsContent = append(hitsContent[:i+1], hitsContent[i+2:]...)
				} else {
					hitsContent = hitsContent[:i+1]
				}
				break
			}
		}
	}

	var out bytes.Buffer

	// Stage 4: get the tuples for each set's rowids and add them formatted to the result
	for i := 0; i < len(hitsContent); i++ {
		var rowids []string
		for _, v := range hitsContent[i] {
			rowids = append(rowids, strconv.Itoa(v))
		}
		rows, err = d.Query(`SELECT rowid, * FROM history
                                  WHERE rowid IN (` + strings.Join(rowids, ",") + `)
                                  ORDER BY datetime ASC`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		res := result.New(qp.Format)
		for rows.Next() {
			var user, host, command, shellpid, workdir string
			var t time.Time
			var row int
			rows.Scan(&row, &user, &host, &command, &t, &shellpid, &workdir)
			res.AddRow(row, user, host, command, shellpid, workdir, t)
		}
		out.Write(res.Formatted())
		if i < len(hitsContent)-1 {
			out.WriteString("\n------------------\n")
		}
	}
	return out.Bytes(), nil
}
