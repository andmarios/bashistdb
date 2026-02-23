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

// Package llog provides some basic level logging functionality to bashistdb.
package llog

import (
	"io"
	"io/ioutil"
	"log"
	"os"
)

// Levels for verbosity.
const (
	SILENT = iota // SILENT discards everything (apart those sent to the unnamed logger)
	INFO          // INFO shows only informational messages
	DEBUG         // DEBUG shows both info and debug messages, adding filename and linenumber
)

// A Logger offers an unnamed logger for logging critical events,
// an info loffer for logging informational messages and a debug
// logger for logging debug information.
type Logger struct {
	*log.Logger
	Info  *log.Logger
	Debug *log.Logger
}

// New creates a new Logger of verbosity level.
func New(verbosity int) *Logger {
	var deb, inf *log.Logger
	var debOut, infOut io.Writer
	debMod := log.Ldate | log.Ltime
	infMod := log.Ldate | log.Ltime

	switch verbosity {
	case SILENT:
		infOut = ioutil.Discard
		debOut = ioutil.Discard
	case INFO:
		infOut = os.Stderr
		debOut = ioutil.Discard
	case DEBUG:
		infOut = os.Stderr
		debOut = os.Stderr
		debMod = log.Ldate | log.Ltime | log.Lshortfile
		infMod = log.Ldate | log.Ltime | log.Lshortfile
	default:
		infOut = os.Stderr
		debOut = ioutil.Discard
	}

	// std is used for logging fatal errors
	std := log.New(os.Stderr, "", log.Ldate|log.Ltime|log.Lshortfile)

	// inf is used for logging info messages
	inf = log.New(infOut, "", infMod)

	// std is used for logging debug messages
	deb = log.New(debOut, "", debMod)
	deb.Println("Debug enabled.")

	return &Logger{std, inf, deb}
}
