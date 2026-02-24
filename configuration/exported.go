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

package configuration

import (
	"fmt"
	"io"
	"os"

	"github.com/andmarios/bashistdb/llog"
)

// Exported fields are global settings.
var (
	Mode      int          // Mode of operation (local, server, client, etc)
	Operation int          // function (read, restore, et)
	Log       *llog.Logger // Log is the mail logger to log to
	Address   string       // Address is the remote server's address for client mode or server's address for server mode
	Database  string       // Database is the filename of the sqlite database
	Key       []byte       // Key it the user passphrase to generate keys for net comms
	User      string       // User is the username detected or explicitly set
	Error     error        // Will contain an error message if configuration setup failed
	Hostname  string       // Hostname is the hostname detected or explicitly set
	QParams   QueryParams  // Parameters to query
)

// Output Formats
const (
	FORMAT_BASH_HISTORY = "restore"
	FORMAT_ALL          = "all"
	FORMAT_COMMAND_LINE = "command_line"
	FORMAT_TIMESTAMP    = "timestamp"
	FORMAT_LOG          = "log"
	FORMAT_JSON         = "json"
	FORMAT_EXPORT       = "export"
	FORMAT_ROWS         = "rows"
	FORMAT_DEFAULT      = FORMAT_COMMAND_LINE
)

var availableFormats = map[string]bool{
	FORMAT_BASH_HISTORY: true,
	FORMAT_ALL:          true,
	FORMAT_COMMAND_LINE: true,
	FORMAT_TIMESTAMP:    true,
	FORMAT_LOG:          true,
	FORMAT_JSON:         true,
	FORMAT_EXPORT:       true,
	FORMAT_ROWS:         true,
}

// Run Modes, you may only add entries at the end.
// If many are set, precedence should be PRINT_VERSION > INIT > SERVER > CLIENT > LOCAL
// It is ok that we use ints because these are not communicated between client and server.
const (
	_ = iota
	MODE_SERVER
	MODE_CLIENT
	MODE_LOCAL
	MODE_PRINT_VERSION // version flag overrides anything else
	MODE_INIT
	MODE_ERROR
	MODE_HELP
)

// Operations, you may only add entries at the end.
const (
	_         = iota
	OP_IMPORT // Import history from stdin
	OP_QUERY  // Run a query
)

// A QueryParams contains parameters that are used to run a query.
// Depending on query type, some fields may not be used.
type QueryParams struct {
	Type          string // Query type
	Kappa         int    // If topk or lastk, we store k here
	User          string // Search User
	Host          string // Search Host
	Format        string // Return format
	Command       string // Search Term for command line field
	Unique        bool   // Return unique command lines
	Rows          []int  // Rowids
	Regex         bool   // Search is a regular expression
	AfterContent  int    // Return also this many lines after match
	BeforeContent int    // Return also this many lines before match
	Session       string // Filter by shell PID
	Workdir       string // Filter by working directory
	Fuzzy         bool   // Fuzzy search mode
}

// Available query types
// Since we implement a protocol and client/server could have different versions,
// hardcoded strings instead of Go's autoincrement is better.
const (
	QUERY         = "query"   // A normal search (grep)
	QUERY_LASTK   = "lastk"   // K most recent commands
	QUERY_TOPK    = "topk"    // K most used commands
	QUERY_USERS   = "users"   // users@host in database
	QUERY_DEMO    = "demo"    // Run some demo queries
	QUERY_ROW     = "row"     // Return a plain single row given its rowid
	QUERY_CONTENT = "content" // Content search (n lines before, after or both)
	QUERY_FUZZY   = "fuzzy"   // Fuzzy search
	DELETE        = "delete"  // Delete rows given their rowid
)

// We do this in order to be able to test the parse code (we can't test init).
// In test mode (BASHISTDB_TEST set), skip init parsing to avoid Go test flag conflicts.
// Tests call resetFlags() + parse() themselves with controlled os.Args.
func init() {
	if os.Getenv("BASHISTDB_TEST") != "" {
		Log = llog.New(0)
		return
	}
	if err := parse(); err != nil {
		Mode = MODE_ERROR
		Error = err
	}
}

// PrintHelp prints the help text.
func PrintHelp(w io.Writer) {
	fmt.Fprintln(w, ""+`Usage of bashistdb.
Query or run in server mode:
  bashistdb [OPTIONS] [QUERY]
Import history:
  history | bashistdb [OPTIONS]

The query is run against the command lines only. Special flags exist for user
and hostname search. SQLite wildcard operators are percent (%) instead of
asterisk (*) and undercore (_) instead of question mark (?). You may use
backslash (\) to escape. The query term always run with both a wildcard prefix
and suffix. Think of it as grep.

Available options:
    -db FILE
        Path to database file. It will be created if it doesn't exist.
        Current: `+database+`

    -V
        Print version info and exit.

    -v , -verbose LEVEL
        Verbosity level: 0 for silent, 1 for info, 2 for debug.
        In server mode it is set to 1 if left 0.

    -U, -user USER
        Optional user name to use instead of reading $USER variable. In query
        operations it doubles as search term for the username. Wildcard
        operators (%, _) work but unlike query we search for the exact term.
        Current: `+user+`
    -H, -host HOST
        Optional hostname to use instead of reading it from the system. In query
        operations, it doubles as search term for the hostname. Wildcard
        operators (%, _) work but unlike query we search for the exact term.
        Current: `+host+`
    -g, --global
        Sets user and host to % for query operation. (equiv: -user % -host %)

    -u, -unique
        If the query type permits, return unique results for the
        command line field (returns the most recent execution of each command).
    -R
        The query is a regular expession. This works only for the default query.
        For other types of query (e.g lastk, topk), it works as an exact match
        flag. Normally when you search for “term”, you really search for
        “%term%” which gives a grep like behaviour. With -R, wildcards are gone.
    -F
        Fuzzy search. Matches commands by fragments and approximate spelling.
        “kube deploy prod” finds “kubectl apply -f deploy-prod.yaml”.
        Typo tolerant: “gti comit” finds “git commit”.
        Results ranked by relevance (best match first), top 25 by default.
        Combine with -lastk N to change result limit or get chronological order.
        Combine with -g for global search, -session/-workdir for filtering.
        Incompatible with: -topk, -R, -A, -B, -C, -row, -del, -users.

    -lastk, -tail K
        Return the K most recent commands for the set user and host. If you add
        a query term it will return the K most recent commands that include it.
    -topk K
        Return the K most frequent commands for the set user and host. If you add
        a query term it will return the K most frequent commands that include it.
    -row K
        Return the K row from the database. You can pipe it to bash.
    -del EXPRESSION (e.g: 9-13,100,5)
        Delete rows with the given row ids. Row ids stay unique unless you delete
        the last row, where its id will be given to the next new entry.
    -users
        Return the users in the database. You may use search criteria, eg to
        find users who run a certain commands. By default this option searches
        across all users and host unless you explicitly set them via flags.
    -A K, -B K, -C K  (or grep-style: -A5, -B5, -C5)
        Also print K lines A(fter), B(efore) or before and after C(ontent) of
        each match. Supports both traditional format (-A 5) and grep-style
        format (-A5).
    -session PID
        Filter results to commands from a specific shell session identified
        by its PID. Can be combined with other query types (e.g. -lastk, -A).
    -workdir DIR
        Filter results to commands run in a matching working directory.
        Uses full text search (like grep). Can be combined with other queries.

    -local
        Force local [db] mode, despite remote mode being set by env or conf.
    -s, -server
        Run in server mode. Bashistdb currently binds to 0.0.0.0.
    -r, -remote SERVER_ADDRESS
        Run in network client mode, connect to server address. You may also set
        this with the BASHISTDB_REMOTE env variable. Current: `+remote+`
    -p, -port PORT
        Server port to listen on/connect to. You may also set this with the
        BASHISTDB_PORT env variable. Current: `+port+`
    -k, -key PASSPHRASE
        Passphrase to use for creating keys to encrypt network communications.
        You may also set it via the BASHISTDB_KEY env variable.

    -f, --format FORMAT
        How to format query output. Available types are:
        `+FORMAT_ALL+", "+FORMAT_BASH_HISTORY+", "+FORMAT_COMMAND_LINE+
		", "+FORMAT_JSON+", "+FORMAT_LOG+", "+FORMAT_TIMESTAMP+", "+
		FORMAT_EXPORT+", "+FORMAT_ROWS+`
        Format '`+FORMAT_BASH_HISTORY+`' can be used to restore your history file.
        Format '`+FORMAT_EXPORT+`' can be used to pipe your history to another
        instance of bashistdb, while retaining user and host of each command.
        Format '`+FORMAT_ROWS+`' can be used for advanced delete operations.
        Default: `+FORMAT_DEFAULT+`

    -save
        Write some settings (database, remote, port, key) to configuration file:
        `+confFile+`. These settings override environment variables.
    -init
        Setup system for bashistdb: (1) Save settings to file. (2) Add to bashrc
        functions to timestamp history and sent each command to bashistdb
        (remote or local, taken from settings), (3) add a unique serial
        timestamp to any untimestamped line in your bash_history.

    -h, --help
        This text.`)
}
