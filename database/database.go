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

/*
Package database handles a SQLite3 database and access methods for the
specific needs of bashistdb.
*/
package database

import (
	"bufio"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	conf "github.com/andmarios/bashistdb/configuration"
	"github.com/andmarios/bashistdb/llog"
	"github.com/mattn/go-sqlite3"
)

// Golang's RFC3339 does not comply with all RFC3339 representations
const RFC3339alt = "2006-01-02T15:04:05-0700"

// VERSION is the database's schema supported version.
// If your database is older it will be automatically migrated.
// If it is newer you have to update your bashistdb copy.
const VERSION = "2.2"

// A Database holds a bashistdb database.
type Database struct {
	*sql.DB
	statements
}

type statements struct {
	insert *sql.Stmt
}

var log *llog.Logger

func init() {
	log = conf.Log
}

// New returns a new Database instance. It gets the filename for the
// database from the configuration package. If the file does not exist,
// it creates a new database. If it exists, it migrates it if it has an
// older schema version than current.
func New() (Database, error) {
	// If database file does not exist, set a flag to create file and table.
	init := false
	if _, err := os.Stat(conf.Database); os.IsNotExist(err) {
		log.Info.Println("Database file not found. Creating new.")
		init = true
	} else {
		log.Info.Println("Database file found.")
	}
	// Open database. SQLite3 provides concurrency in the library level, thus
	// we don't need to implement locking.
	db, err := sql.Open("sqlite3", conf.Database)
	if err != nil {
		return Database{}, err
	}
	// If database is new, initialize it with our tables.
	// Else migrate it if needed.
	if init {
		if err = initDB(db); err != nil {
			_ = db.Close()
			return Database{}, err
		}
	} else {
		err := migrate(db)
		if err != nil {
			return Database{}, err
		}
	}
	// Prepare various statements that may be used frequently.
	var insert *sql.Stmt
	var err2 error
	insert, err2 = db.Prepare("INSERT INTO history(user, host, command, datetime, shellpid, workdir) VALUES(?, ?, ?, ?, ?, ?)")
	if err2 != nil {
		_ = db.Close()
		return Database{}, err2
	}
	stmts := statements{insert}
	return Database{db, stmts}, nil
}

func initDB(db *sql.DB) error {
	stmt := `
CREATE TABLE history (
    user     TEXT,
    host     TEXT,
    command  TEXT,
    datetime DATETIME,
    shellpid TEXT DEFAULT '',
    workdir  TEXT DEFAULT '',
    PRIMARY KEY (user, command, datetime)
);
CREATE INDEX HistoryDatetimeIdx ON history(datetime);

CREATE TABLE admin (
    key   TEXT PRIMARY KEY,
    value TEXT
 );

CREATE TABLE connlog (
    datetime TEXT PRIMARY KEY,
    remote   TEXT
 );

CREATE TABLE rlookup (
    ip      TEXT PRIMARY KEY,
    reverse TEXT
     );

CREATE VIEW connections AS
    SELECT datetime, remote, reverse
    FROM connlog AS c
        LEFT JOIN rlookup AS r
        ON c.remote=r.ip;`

	if _, err := db.Exec(stmt); err != nil {
		return err
	}

	stmt = `INSERT INTO admin VALUES ("version", ?)`

	if _, err := db.Exec(stmt, VERSION); err != nil {
		return err
	}
	return nil
}

// AddRecord tries to insert a new record in the database,
// if the record already exists, it updates the count
// Note: function isn't used anywhere, may need testing if used.
func (d Database) AddRecord(user, host, command string, time time.Time, shellpid, workdir string) error {
	// Try to insert row
	_, err := d.insert.Exec(user, host, command, time, shellpid, workdir)
	if err != nil {
		// If failed due to duplicate primary key, then ignore error
		// We expect for ease of use, the user to resubmit the whole
		// history from time to time.
		if driverErr, ok := err.(sqlite3.Error); ok {
			if driverErr.ExtendedCode == sqlite3.ErrConstraintPrimaryKey {
				log.Debug.Println("Duplicate entry. Ignoring.", user, host, command, time)
			} else {
				return err
			}
		} else { // Normally we can never reach this. Should we omit it?
			return err
		}
	}
	return nil
}

// A parseline parses history output lines of the following format:
//     LINENUM RFC3339_DATETIME COMMAND
var parseLine = regexp.MustCompile(`^ *[0-9]+\*? *([0-9T:+-]{24,24}) *(.*)`)

// A parseExportLine parses export formatted output from bashistdb:
//     USER HOSTNAME RFC3339_DATETIME COMMAND
//([a-zA-Z_][a-zA-Z0-9_-]*) ([a-zA-Z0-9][a-zA-Z0-9.-]*) *([0-9T:+-]{24,24}) *(.*)
var parseExportLine = regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_-]*) ([a-zA-Z0-9][a-zA-Z0-9.-]*) *([0-9T:+-]{24,24}) *(.*)`)

// A parseExportLineExt parses extended export format with PID and URL-encoded CWD:
//     USER HOSTNAME PID URL_ENCODED_CWD RFC3339_DATETIME COMMAND
var parseExportLineExt = regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_-]*) ([a-zA-Z0-9][a-zA-Z0-9.-]*) +(\d+) +(\S+) +([0-9T:+-]{24,24}) +(.*)`)

// AddFromBuffer reads from a buffered Reader and scans for lines that match
// history command's structure:
//     LINENUM RFC3339_DATETIME COMMAND
// or extended export format:
//     USER HOSTNAME PID URL_ENCODED_CWD RFC3339_DATETIME COMMAND
// or old export format:
//     USER HOSTNAME RFC3339_DATETIME COMMAND
// Upon successful encounter it tries to store it to the database. It counts
// total lines read and lines failed to insert into the database —usually
// because they already exist. It reports the results in a sentence (stats
// string) because we don't anything fancier currently.
func (d Database) AddFromBuffer(r *bufio.Reader, user, host, shellpid, workdir string) (stats string, e error) {
	tx, _ := d.Begin()
	stmt := tx.Stmt(d.insert)
	total, failed := 0, 0
	var once sync.Once
	for {
		historyLine, err := r.ReadString('\n')
		total++
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return "", errors.New("Error while reading stdin: " + err.Error())
			}
		}

		var lineUser, lineHost, lineCommand, linePid, lineCwd string
		var lineTime time.Time

		// Try default history format first (most common):
		//     LINENUM RFC3339_DATETIME COMMAND
		args := parseLine.FindStringSubmatch(historyLine)
		if len(args) == 3 {
			lineTime, err = time.Parse(RFC3339alt, args[1])
			if err != nil {
				tx.Rollback()
				return "", err
			}
			lineUser = user
			lineHost = host
			lineCommand = strings.TrimSuffix(args[2], "\n")
			linePid = shellpid
			lineCwd = workdir
		} else {
			// Try extended export format (new, 6 fields):
			//     USER HOSTNAME PID URL_ENCODED_CWD RFC3339_DATETIME COMMAND
			args = parseExportLineExt.FindStringSubmatch(historyLine)
			if len(args) == 7 {
				once.Do(func() { log.Info.Println("Bashistdb extended export format detected.") })
				lineTime, err = time.Parse(RFC3339alt, args[5])
				if err != nil {
					tx.Rollback()
					return "", err
				}
				lineUser = args[1]
				lineHost = args[2]
				linePid = args[3]
				lineCwd, _ = url.PathUnescape(args[4])
				lineCommand = strings.TrimSuffix(args[6], "\n")
			} else {
				// Try old export format (4 fields):
				//     USER HOSTNAME RFC3339_DATETIME COMMAND
				args = parseExportLine.FindStringSubmatch(historyLine)
				if len(args) == 5 {
					once.Do(func() { log.Info.Println("Bashistdb export format detected.") })
					lineTime, err = time.Parse(RFC3339alt, args[3])
					if err != nil {
						tx.Rollback()
						return "", err
					}
					lineUser = args[1]
					lineHost = args[2]
					lineCommand = strings.TrimSuffix(args[4], "\n")
					linePid = ""
					lineCwd = ""
				} else {
					log.Info.Println("Couldn't decode line, unknown format. Skipping:", historyLine)
					failed++
					continue
				}
			}
		}

		_, err = stmt.Exec(lineUser, lineHost, lineCommand, lineTime, linePid, lineCwd)
		if err != nil {
			// If failed due to duplicate primary key, then ignore error
			// We expect for ease of use, the user to resubmit the whole
			// history from time to time.
			if driverErr, ok := err.(sqlite3.Error); ok {
				if driverErr.ExtendedCode == sqlite3.ErrConstraintPrimaryKey {
					log.Debug.Println("Duplicate entry. Ignoring.", lineUser, lineHost, lineCommand, lineTime)
					failed++
				} else {
					tx.Rollback()
					return "", err
				}
			} else {
				return "", err
			}
		}
	}
	tx.Commit()
	total--
	stats = fmt.Sprintf("Processed %d entries, successful %d, failed %d.", total, total-failed, failed)
	return stats, nil
}

// LogConn logs the remote's IP address and connection time into connlog table.
// Also if it can't find a reverse lookup for the IP address inside table rlookup,
// it performs it asynchronously. Reverse lookup may fail, but we don't care.
func (d Database) LogConn(remote net.Addr) (err error) {
	t := time.Now()
	// Find IP
	if ip, _, err := net.SplitHostPort(remote.String()); err == nil {
		// Store IP and datetime
		_, err = d.Exec(`INSERT INTO connlog VALUES (?, ?);`, t, ip)
		if err == nil {
			// Perform a reverse lookup if needed.
			go func() {
				var rip string
				err = d.QueryRow("SELECT ip FROM rlookup WHERE ip LIKE ?", ip).Scan(&rip)
				if err == sql.ErrNoRows {
					if addr, err := net.LookupAddr(ip); err == nil {
						_, err = d.Exec(`INSERT INTO rlookup(ip, reverse)
                                                           VALUES(? ,?)`,
							ip, strings.Join(addr, ","))
					}
				}
				if err != nil {
					log.Info.Println(err)
				}
			}()
		}
	}
	return
}

// migrate is a unexported function that handles database migrations.
// It is safe to run on databases that already are on latest version.
func migrate(d *sql.DB) error {
	var version string
	err := d.QueryRow(`SELECT value FROM admin WHERE key LIKE "version"`).Scan(&version)
	if err != nil {
		return err
	}

	switch version {
	case "1":
		tx, err := d.Begin()
		if err != nil {
			return err
		}
		stmt := `CREATE TABLE connlog_new(
                             datetime TEXT PRIMARY KEY,
                             remote   TEXT);
                         INSERT INTO connlog_new
                            SELECT datetime, remote FROM connlog;
                         DROP TABLE connlog;
                         ALTER TABLE connlog_new RENAME TO 'connlog';
                         CREATE TABLE rlookup (
                             ip      TEXT PRIMARY KEY,
                             reverse TEXT
                         );
                         CREATE VIEW connections AS
                             SELECT datetime, remote, reverse
                                FROM connlog AS c
                                LEFT JOIN rlookup AS r
                                ON c.remote = r.ip;`
		if _, err = tx.Exec(stmt); err != nil {
			return err
		}
		if _, err = tx.Exec(`UPDATE admin SET value=? WHERE key LIKE 'version'`, "2"); err != nil {
			return err
		}
		if err = tx.Commit(); err != nil {
			return err
		}
		log.Info.Println("Database upgraded to version 2.")
		fallthrough
	case "2":
		tx, err := d.Begin()
		if err != nil {
			return err
		}
		if _, err = tx.Exec(`CREATE INDEX HistoryDatetimeIdx ON history(datetime)`); err != nil {
			return err
		}
		if _, err = tx.Exec(`UPDATE admin SET value=? WHERE key LIKE 'version'`, "2.1"); err != nil {
			return err
		}
		if err = tx.Commit(); err != nil {
			return err
		}
		log.Info.Println("Database upgraded to version 2.1.")
		fallthrough
	case "2.1":
		tx, err := d.Begin()
		if err != nil {
			return err
		}
		if _, err = tx.Exec(`ALTER TABLE history ADD COLUMN shellpid TEXT DEFAULT ''`); err != nil {
			return err
		}
		if _, err = tx.Exec(`ALTER TABLE history ADD COLUMN workdir TEXT DEFAULT ''`); err != nil {
			return err
		}
		if _, err = tx.Exec(`UPDATE admin SET value=? WHERE key LIKE 'version'`, "2.2"); err != nil {
			return err
		}
		if err = tx.Commit(); err != nil {
			return err
		}
		log.Info.Println("Database upgraded to version 2.2.")
		fallthrough
	case "2.2":
		log.Debug.Println("Database on latest version.")
	}

	// Re-read version after migration since the local variable is stale.
	err = d.QueryRow(`SELECT value FROM admin WHERE key LIKE "version"`).Scan(&version)
	if err != nil {
		return err
	}
	if version != VERSION {
		return errors.New("Database version different than code version but couldn't fix it.")
	}

	return nil
}
