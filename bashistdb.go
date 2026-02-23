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

//go:generate go run version/generate/main.go

package main

import (
	"fmt"
	"os"

	conf "github.com/andmarios/bashistdb/configuration"
	"github.com/andmarios/bashistdb/llog"
	"github.com/andmarios/bashistdb/local"
	"github.com/andmarios/bashistdb/network"
	"github.com/andmarios/bashistdb/setup"
	"github.com/andmarios/bashistdb/version"
)

var log *llog.Logger
var v = version.Version // a debug build will append pprof to this

func init() {
	version.Version = vgVersion
	v = vgVersion
}

func main() {
	log = conf.Log

	switch conf.Mode {
	case conf.MODE_PRINT_VERSION:
		fmt.Println("bashistdb " + v)
		fmt.Println("https://github.com/andmarios/bashistdb")
	case conf.MODE_SERVER:
		if err := network.ServerMode(); err != nil {
			log.Fatalln(err)
		}
	case conf.MODE_CLIENT:
		if err := network.ClientMode(); err != nil {
			log.Fatalln(err)
		}
	case conf.MODE_LOCAL:
		if err := local.Run(); err != nil {
			log.Fatalln(err)
		}
	case conf.MODE_INIT:
		if err := setup.Apply(true); err != nil {
			log.Fatalln(err)
		}
	case conf.MODE_ERROR:
		fmt.Printf("%s\n\n", conf.Error)
		conf.PrintHelp(os.Stderr) // On error help goes to stderr
		os.Exit(1)
	case conf.MODE_HELP:
		conf.PrintHelp(os.Stdout) // Help when asked goes to stdout
	}
}
