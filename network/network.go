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

// Package network provides network functions for bashistdb.
package network

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	conf "github.com/andmarios/bashistdb/configuration"
	"github.com/andmarios/bashistdb/database"
	"github.com/andmarios/bashistdb/llog"
	"github.com/andmarios/bashistdb/version"
)

// Message Types
const (
	RESULT  = "result"  // (query) results that should be printed
	HISTORY = "history" // history to import
	QUERY   = "query"   // query to run
	LOGINFO = "info"    // results that should go to log.Info
)

// A Message is the communication unit between server and client.
type Message struct {
	Type     string
	Payload  []byte
	User     string
	Hostname string
	QParams  conf.QueryParams
	Version  string
	ShellPID string
	Workdir  string
}

var log *llog.Logger
var db database.Database

func init() {
	if conf.Log != nil {
		log = conf.Log
	}
}

// ServerMode is the server process of bashistdb.
func ServerMode() error {
	var err error
	db, err = database.New()
	if err != nil {
		return err
	}
	defer db.Close()

	s, err := net.Listen("tcp", conf.Address)
	if err != nil {
		return err
	}
	log.Info.Println("bashistdb", version.Version, "listening on:", conf.Address)
	for {
		conn, err := s.Accept()
		if err != nil {
			log.Info.Println("ERROR:", err.Error())
			continue
		}
		log.Info.Printf("Connection from %s.\n", conn.RemoteAddr())
		err = db.LogConn(conn.RemoteAddr())
		if err != nil {
			log.Info.Println("ERROR:", err.Error())
		}
		go handleConn(conn)
	}
	//	return nil // go vet doesn't like this...
}

// ClientMode is the client process fo bashistdb.
func ClientMode() error {
	log.Debug.Println("Connecting to: ", conf.Address)
	conn, err := net.Dial("tcp", conf.Address)
	if err != nil {
		return err
	}
	defer conn.Close()

	var msg Message

	switch conf.Operation {
	case conf.OP_IMPORT: // If Operation == OP_IMPORT, attempt to read from Stdin
		r := bufio.NewReader(os.Stdin)
		history, err := io.ReadAll(r)
		if err != nil {
			return err
		}

		msg = Message{Type: HISTORY, Payload: history, User: conf.User,
			Hostname: conf.Hostname, ShellPID: os.Getenv("BASHISTDB_PID"),
			Workdir: os.Getenv("BASHISTDB_CWD")}

		log.Info.Println("Sent history.")
	case conf.OP_QUERY:
		msg = Message{Type: QUERY, User: conf.User, Hostname: conf.Hostname, QParams: conf.QParams}
	default:
		return errors.New("unknown function")
	}

	msg.Version = version.Version

	if err := encryptDispatch(conn, msg); err != nil {
		return err
	}
	log.Info.Println("Sent request.")

	conn.SetDeadline(time.Now().Add(60 * time.Second))
	reply, err := receiveDecrypt(conn)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return fmt.Errorf("server did not respond within 60 seconds")
		}
		return err
	}

	if reply.Version != version.Version {
		log.Info.Println("Server runs different bashistdb version from client:", reply.Version)
	}

	switch reply.Type {
	case RESULT:
		fmt.Println(string(reply.Payload))
	case LOGINFO:
		log.Info.Println("Received:", string(reply.Payload))
	}
	return nil
}

// handleConn is the server code that handles clients (reads message type and performs relevant operation)
func handleConn(conn net.Conn) {
	defer conn.Close()

	// Deadline for reading the client message.
	conn.SetDeadline(time.Now().Add(30 * time.Second))

	msg, err := receiveDecrypt(conn)
	if err != nil {
		log.Info.Println(err, "["+conn.RemoteAddr().String()+"]")
		return
	}

	// Clear deadline during processing (imports may take a while).
	conn.SetDeadline(time.Time{})

	if msg.Version != version.Version {
		log.Info.Println("Client runs different bashistdb version from server:", msg.Version)
	}

	var result []byte
	switch msg.Type {
	case HISTORY:
		r := bufio.NewReader(bytes.NewReader(msg.Payload))
		res, err := db.AddFromBuffer(r, msg.User, msg.Hostname, msg.ShellPID, msg.Workdir)
		if err != nil {
			result = []byte(err.Error())
		} else {
			result = []byte(res)
		}
		log.Info.Println("Client sent history: ", res)
	case QUERY:
		result, err = db.RunQuery(msg.QParams)
		if err != nil {
			log.Info.Println("ERROR:", err.Error())
			result = []byte(err.Error())
		}
		log.Info.Printf("Client sent %s query for '%s' as '%s'@'%s', '%s' format.\n",
			msg.Type, msg.QParams.User, msg.QParams.Host, msg.QParams.Command, msg.QParams.Format)
	}

	// Deadline for writing the reply.
	conn.SetDeadline(time.Now().Add(30 * time.Second))

	reply := Message{Type: RESULT, Payload: result, Version: version.Version}
	if msg.Type == HISTORY {
		reply.Type = LOGINFO
	}
	if err := encryptDispatch(conn, reply); err != nil {
		log.Println(err)
	}
}
