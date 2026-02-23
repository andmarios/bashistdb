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

// Package local manages the local operation of bashistdb.
package local

import (
	"bufio"
	"errors"
	"fmt"
	"os"

	conf "github.com/andmarios/bashistdb/configuration"
	"github.com/andmarios/bashistdb/database"
	"github.com/andmarios/bashistdb/llog"
)

var log *llog.Logger

// Run is the local process of bashistdb.
func Run() error {
	db, err := database.New()
	if err != nil {
		return errors.New("Failed to load database: " + err.Error())
	}
	defer db.Close()

	log = conf.Log

	switch conf.Operation {
	case conf.OP_IMPORT:
		r := bufio.NewReader(os.Stdin)
		shellpid := os.Getenv("BASHISTDB_PID")
		workdir := os.Getenv("BASHISTDB_CWD")
		stats, err := db.AddFromBuffer(r, conf.User, conf.Hostname, shellpid, workdir)
		if err != nil {
			return errors.New("Error while processing stdin: " +
				err.Error())
		}
		// We print to log because we usually want this to be quiet
		// as we may run it every time we hit ENTER in a bash prompt.
		log.Info.Println(stats)
	case conf.OP_QUERY:
		res, err := db.RunQuery(conf.QParams)
		if err != nil {
			return err
		}
		fmt.Println(string(res))
	}
	return nil
}
