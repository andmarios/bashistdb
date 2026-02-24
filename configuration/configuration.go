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

// Package configuration handles the configuration of bashistdb.
package configuration

import (
	"errors"
	"flag"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/andmarios/bashistdb/llog"
)

const (
	bashistPort = "25625"
)

// flag variables
// Vars ending in Set are bool
// For non-bool vars we may create a bool counterpart using flag.Visit
// We set them all here so we can copy this to test for resetting vars
var (
	// These are used as actual flagvars
	database      = os.Getenv("HOME") + "/.bashistdb.sqlite3"
	versionSet    = false
	verbosity     = 0
	user          = os.Getenv("USER")
	host, _       = os.Hostname()
	serverSet     = false
	remote        = os.Getenv("BASHISTDB_REMOTE")
	port          = os.Getenv("BASHISTDB_PORT")
	passphrase    = os.Getenv("BASHISTDB_KEY")
	format        = FORMAT_DEFAULT
	helpSet       = false
	globalSet     = false
	writeconfSet  = false
	setupSet      = false
	topk          = 20
	lastk         = 20
	localSet      = false
	uniqueSet     = false
	usersSet      = false
	row           = 0
	delRows       = ""
	regexSet      = false
	afterContent  = 5
	beforeContent = 5
	content       = 5
	session       = ""
	workdirQuery  = ""
	fuzzySet      = false
	// Custom Flags that need custom (non-flag package code) to parse and set. //
	// These are not parsed from flags but we set them with flag.Visit
	userSet          = false
	hostSet          = false
	remoteSet        = false
	topkSet          = false
	lastkSet         = false
	rowSet           = false
	delRowsSet       = false
	afterContentSet  = false
	beforeContentSet = false
	contentSet       = false
	sessionSet       = false
	workdirSet       = false
	// These are set with manual searches
	querySet = false
	stdinSet = false
	// Vars below can not be overriden by user
	confFile      = os.Getenv("HOME") + "/.bashistdb.conf"
	foundConfFile = false
)

// Set visited flags so we may have boolean expression criteria
func setVisitedFlags(f *flag.Flag) {
	switch f.Name {
	case "U", "user":
		userSet = true
	case "H", "host":
		hostSet = true
	case "r", "remote":
		remoteSet = true
	case "topk":
		topkSet = true
	case "lastk", "tail":
		lastkSet = true
	case "row":
		rowSet = true
	case "del":
		delRowsSet = true
	case "A":
		afterContentSet = true
	case "B":
		beforeContentSet = true
	case "C":
		contentSet = true
	case "session":
		sessionSet = true
	case "workdir":
		workdirSet = true
	}
}

// Set all boolean flags. Besides flag vars, this is also for stdin
// detection and query detection (non-flag argument).
func parseCustomFlags() {
	// Set boolean counterparts for non-boolean flag vars.
	flag.Visit(setVisitedFlags)

	// Detect if there is a QUERY in the command line (that is non-flag arguments)
	if len(flag.Args()) > 0 {
		querySet = true
	}

	// Detect if there are data coming from stdin:
	stats, err := os.Stdin.Stat()
	if err == nil && (stats.Mode()&os.ModeCharDevice) != os.ModeCharDevice {
		stdinSet = true
	}
}

// checkFlagCombination checks if non-compatible flags were used
// This may not be exhaustive. There won't be real errors if we don't
// detect incompatible arguments, just that the user will see only
// one of the arguments executed. This function serves more as a help
// mode.
func checkFlagCombination() error {
	// Do not mix server, client and local modes:
	// Server mode incompatible with client mode.
	if Mode == MODE_SERVER && remoteSet { // User may just set his BASHISTDB_REMOTE env var
		return errors.New("Incompatible options: server and client.")
	}

	if lastkSet && topkSet {
		return errors.New("Incompatible options: -lastk and -topk.")
	}

	if topkSet && uniqueSet {
		return errors.New("Incompatible options: -topk and -unique.")
	}

	if rowSet && (lastkSet || topkSet) {
		return errors.New("Incompatible options: -rows and one of -lastk, -topk")
	}

	if usersSet && (lastkSet || topkSet || querySet || rowSet) {
		return errors.New("Incompatible options: -users with other type of query")
	}

	if (sessionSet || workdirSet) && delRowsSet {
		return errors.New("Incompatible options: -session or -workdir with -del")
	}

	if rowSet && querySet {
		return errors.New("Incompatible options: -row combined with query")
	}

	if delRowsSet && (lastkSet || topkSet || querySet || rowSet || usersSet) {
		return errors.New("Incompatible options: -del combined with other type of query")
	}

	if regexSet && (lastkSet || topkSet || rowSet || usersSet || delRowsSet) {
		Log.Info.Println("R(egexp) flag works only for simple queries. For other types it works as an exact match flag.")
	}

	if uniqueSet && (afterContentSet || beforeContentSet || contentSet) {
		Log.Info.Println("u(nique) flag doesn't work with content, before, after search")
	}

	if (afterContentSet || beforeContentSet || contentSet) && (lastkSet || topkSet || rowSet || usersSet || delRowsSet) {
		return errors.New("Incompatible options: content search (-A, -B, -C) and a non standard query")
	}

	if fuzzySet && !querySet {
		return errors.New("Fuzzy search (-F) requires a search term.")
	}

	if fuzzySet && (topkSet || regexSet || rowSet || usersSet || delRowsSet || afterContentSet || beforeContentSet || contentSet) {
		return errors.New("Incompatible options: -F (fuzzy) with -topk, -R, -row, -users, -del, -A, -B, or -C")
	}

	// Check mode-operation incompatibility
	if Mode == MODE_SERVER && QParams.Type != QUERY_DEMO {
		return errors.New("Incompatible options: asked for server mode and other functions.\n\n")
	}
	return nil
}

// Sets Operation Query Parameters
func setOpAndQParams() error {
	var err error
	// Determine operation (used in local and client mode)
	switch {
	case fuzzySet && querySet:
		Operation = OP_QUERY
		QParams.Type = QUERY_FUZZY
		QParams.Fuzzy = true
		QParams.Kappa = 25 // default result limit
		if lastkSet {
			QParams.Kappa = lastk
		}
	case topkSet:
		Operation = OP_QUERY
		QParams.Type = QUERY_TOPK
		QParams.Kappa = topk
	case lastkSet:
		Operation = OP_QUERY
		QParams.Type = QUERY_LASTK
		QParams.Kappa = lastk
	case usersSet:
		Operation = OP_QUERY
		QParams.Type = QUERY_USERS
	case afterContentSet, beforeContentSet, contentSet:
		Operation = OP_QUERY
		QParams.Type = QUERY_CONTENT
		if afterContentSet {
			QParams.AfterContent = afterContent
		}
		if beforeContentSet {
			QParams.BeforeContent = beforeContent
		}
		if contentSet {
			QParams.AfterContent, QParams.BeforeContent = content, content
		}
	case querySet: // We have non-flag arguments -> it is a query
		Operation = OP_QUERY
		QParams.Type = QUERY
	case rowSet:
		Operation = OP_QUERY
		QParams.Type = QUERY_ROW
		QParams.Kappa = row
	case delRowsSet:
		Operation = OP_QUERY
		QParams.Type = DELETE
		QParams.Rows, err = parseRange(delRows)
		if err != nil {
			return err
		}
	case stdinSet:
		Operation = OP_IMPORT
	default: // Demo mode
		Operation = OP_QUERY
		QParams.Type = QUERY_DEMO
	}

	if availableFormats[format] { // Query uses output format
		QParams.Format = format
	} else {
		Log.Info.Println("The specified format doesn't exist. Reverting to default:", FORMAT_DEFAULT)
		QParams.Format = FORMAT_DEFAULT
	}

	switch uniqueSet {
	case true:
		QParams.Unique = true
	default:
		QParams.Unique = false
	}

	QParams.User = user
	if usersSet && !userSet { // simple users query should be global
		QParams.User = "%"
	}

	QParams.Host = host
	if usersSet && !hostSet { // simple users query should be global
		QParams.Host = "%"
	}

	// Check for global (search) flag
	if Operation == OP_QUERY && globalSet {
		// User, Hostname = "%", "%" // TODO: remove
		QParams.User, QParams.Host = "%", "%"
	}

	// Query is the non flag os.Args parts.
	// Depending on the pcre flag, we prepare the query differently.
	switch regexSet {
	case true:
		QParams.Regex = true
		QParams.Command = strings.Join(flag.Args(), " ")
	default:
		QParams.Regex = false
		QParams.Command = "%" + strings.Join(flag.Args(), " ") + "%" // Grep like behaviour
	}

	// Fuzzy search uses the raw query, not SQL LIKE patterns
	if fuzzySet {
		QParams.Command = strings.Join(flag.Args(), " ")
	}

	// Session and workdir are filters applied to any query type.
	if sessionSet {
		QParams.Session = session
	}
	if workdirSet {
		QParams.Workdir = "%" + workdirQuery + "%" // Grep like behaviour
	}

	return nil
}

// Sets and parses flags. Helps for testing to separate these.
func setParseFlags() {
	// flagVars, we keep actual documentation separated
	flag.StringVar(&database, "db", database, "Database file")
	flag.BoolVar(&versionSet, "V", versionSet, "Show version.")
	flag.IntVar(&verbosity, "v", verbosity, "verbosity level")
	flag.IntVar(&verbosity, "verbose", verbosity, "verbosity level")
	flag.StringVar(&user, "U", user, "custom username")
	flag.StringVar(&user, "user", user, "custom username")
	flag.StringVar(&host, "H", host, "custom hostname")
	flag.StringVar(&host, "host", host, " custom hostname")
	flag.BoolVar(&serverSet, "s", serverSet, "run as server")
	flag.BoolVar(&serverSet, "server", serverSet, "run as server")
	flag.StringVar(&remote, "r", remote, "run as client, connect to SERVER")
	flag.StringVar(&remote, "remote", remote, "run as client, connect to SERVER")
	flag.StringVar(&port, "p", port, "port")
	flag.StringVar(&port, "port", port, "port")
	flag.StringVar(&passphrase, "k", passphrase, "passphrase")
	flag.StringVar(&passphrase, "key", passphrase, "passphrase")
	flag.StringVar(&format, "f", format, "query output format")
	flag.StringVar(&format, "format", format, "query output format")
	flag.BoolVar(&helpSet, "h", helpSet, "help")
	flag.BoolVar(&helpSet, "help", helpSet, "help")
	flag.BoolVar(&globalSet, "g", globalSet, "global: '-user % -host %'")
	flag.BoolVar(&writeconfSet, "save", writeconfSet, "write ~/.bashistdb.conf")
	flag.BoolVar(&setupSet, "init", setupSet, "set-up system to use bashistdb")
	flag.BoolVar(&uniqueSet, "u", uniqueSet, "show unique (distinct) command lines")
	flag.BoolVar(&uniqueSet, "unique", uniqueSet, "show unique (distinct) command lines")
	flag.IntVar(&topk, "topk", topk, "return K most used command lines")
	flag.IntVar(&lastk, "lastk", lastk, "return K most recent command lines")
	flag.IntVar(&lastk, "tail", lastk, "return K most recent command lines")
	flag.BoolVar(&usersSet, "users", usersSet, "show users in database")
	flag.BoolVar(&localSet, "local", localSet, "force local mode")
	flag.IntVar(&row, "row", row, "return this row")
	flag.StringVar(&delRows, "del", delRows, "delete these rows")
	flag.BoolVar(&regexSet, "R", regexSet, "regular expression search")
	flag.IntVar(&afterContent, "A", afterContent, "return this many rows after match")
	flag.IntVar(&beforeContent, "B", beforeContent, "return this many rows before match")
	flag.IntVar(&content, "C", content, "return this many rows before and after match")
	flag.StringVar(&session, "session", session, "filter by shell session PID")
	flag.StringVar(&workdirQuery, "workdir", workdirQuery, "filter by working directory")
	flag.BoolVar(&fuzzySet, "F", fuzzySet, "fuzzy search")
	flag.Parse()
}

// preprocessGrepStyleFlags handles grep-style context flags (-A5, -B5, -C5)
// It converts them to standard flag format (-A 5, -B 5, -C 5)
func preprocessGrepStyleFlags() {
	// Pattern to match grep-style context flags: -A123, -B123, -C123
	grepFlagPattern := regexp.MustCompile(`^-([ABC])(\d+)$`)

	newArgs := []string{os.Args[0]} // Start with program name

	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]

		if matches := grepFlagPattern.FindStringSubmatch(arg); matches != nil {
			// Split grep-style flag into flag and value
			newArgs = append(newArgs, "-"+matches[1], matches[2])
		} else {
			// Keep the argument as-is
			newArgs = append(newArgs, arg)
		}
	}

	os.Args = newArgs
}

// parse is the "main" of out configuration code.
// Configuration seems a bit messy but that's the way it is.
func parse() error {
	// Create logger early so it's available even if parse fails.
	// It will be recreated with the correct verbosity level later.
	Log = llog.New(0)

	// If this is set, skip reading settings from configuration file.
	if t := os.Getenv("BASHISTDB_TEST"); t == "" {
		if err := readConfFile(); err != nil {
			return err
		}
	}

	// If port isn't set yet, set default port.
	if port == "" {
		port = bashistPort
	}

	// Preprocess grep-style flags before parsing
	preprocessGrepStyleFlags()

	// Set flag vars and parse them
	setParseFlags()

	// Set custom flags (non flag-package variables)
	parseCustomFlags()

	if helpSet {
		Mode = MODE_HELP
		return nil
	}

	// Verbosity reaches up to 2 (DEBUG)
	if verbosity > 2 {
		verbosity = 2
	}

	// Recreate logger with the requested verbosity level.
	Log = llog.New(verbosity)

	// Determine run mode. A run mode is expected to run and then bashistdb toexit.
	switch { // Cases are in precedence order
	case setupSet:
		Mode = MODE_INIT
	case versionSet:
		Mode = MODE_PRINT_VERSION
	case serverSet:
		Mode = MODE_SERVER
		Address = ":" + port
		if verbosity < 1 { // Server mode sets min verbosity of 1 (INFO)
			verbosity = 1
		}
	case remote != "" && !localSet:
		Mode = MODE_CLIENT
		Address = remote + ":" + port
	default:
		Mode = MODE_LOCAL
	}

	if err := setOpAndQParams(); err != nil {
		return err
	}

	if err := checkFlagCombination(); err != nil {
		return err
	}

	// Protest about username issues.
	if user == "" {
		return errors.New("Couldn't read username from $USER system variable and none was provided by -user flag.")
	}
	User = user // TODO: remove

	// Protest about hostname issues.
	if host == "" {
		return errors.New("Couldn't get hostname from system and none was provided by -host flag.")
	}
	Hostname = host // TODO: remove

	welcomeMessages()

	// Set database filename
	Database = database

	// When we setup the system, we should also save settings
	if setupSet {
		writeconfSet = true
	}

	// Passphrase may come from environment or flag
	if Mode == MODE_SERVER || Mode == MODE_CLIENT || writeconfSet {
		if passphrase == "" {
			log.Println("Using empty passphrase.")
		}
		Key = []byte(passphrase)
	}

	if writeconfSet {
		if err := writeConfFile(); err != nil {
			return err
		}
		Log.Info.Println("Wrote settings to ", confFile)
	}

	Log.Debug.Printf("Database: %s, mode: %d, operation: %d, address: %s\n", Database, Mode, Operation, Address)

	return nil
}

func welcomeMessages() {
	// Welcome message
	m := ""
	switch Mode {
	case MODE_SERVER:
		m = "server"
	case MODE_CLIENT:
		m = "client"
	case MODE_LOCAL:
		m = "local"
	}

	Log.Info.Println("Welcome " + User + "@" + Hostname + ". Bashistdb is in " + m + " mode.")
	Log.Debug.Println("Loaded some settings from environment. Configuration file and flags can override them.")
	if foundConfFile {
		Log.Info.Println("Loaded some settings from ~/.bashistdb.conf. Command line flags can override them.")
	}

	if Operation == OP_QUERY {
		Log.Info.Printf("Your query parameters are user: %s, host: %s, command line: %s.\n", QParams.User, QParams.Host, QParams.Command)
	}
}
