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
Package setup provides functions to setup your system for bashistdb.
*/
package setup

import (
	"fmt"
	"os"

	conf "github.com/andmarios/bashistdb/configuration"
	"github.com/andmarios/bashistdb/llog"
	"github.com/andmarios/bashistdb/tools/addTimestamp2Hist/timestamp"
)

const appendLines = `export HISTTIMEFORMAT="%FT%T%z "
[ ! -z "${PROMPT_COMMAND}" ] && export PROMPT_COMMAND="${PROMPT_COMMAND};"
export PROMPT_COMMAND="${PROMPT_COMMAND} (history 1 | BASHISTDB_PID=\$\$ BASHISTDB_CWD=\$PWD bashistdb 2>/dev/null &)"
`

var log *llog.Logger

func init() {
	log = conf.Log
}

// Apply configures your system to use bashistdb:
// 1. It appends to your ~/.bashrc two lines to make your history timestamped
//    and your prompt send your commands to bashistdb.
// 2. It (optionally) adds timestamps to your current history file, so it can
//    be used with bashistdb. This step is also safe to run many times.
func Apply(write bool) error {
	// Setup bashrc for bashistdb
	bashrc := os.Getenv("HOME") + "/.bashrc"
	f, err := os.OpenFile(bashrc, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("could not open bashrc: %w", err)
	}
	defer f.Close()

	if _, err = f.WriteString(appendLines); err != nil {
		return fmt.Errorf("could not write bashrc: %w", err)
	}
	log.Println("Updated " + bashrc + ", appended: \n" + appendLines)

	// Convert bash_history
	if write {
		bashHistory := os.Getenv("HOME") + "/.bash_history"
		historyIn, err := os.ReadFile(bashHistory)
		if err != nil {
			return fmt.Errorf("could not read bash_history: %w", err)
		}

		historyOut := timestamp.Convert(historyIn, 12)

		err = os.WriteFile(bashHistory, historyOut, 0600)
		if err != nil {
			return fmt.Errorf("could not write bash_history: %w", err)
		}
		log.Println("Updated " + bashHistory)
	}

	return nil
}
