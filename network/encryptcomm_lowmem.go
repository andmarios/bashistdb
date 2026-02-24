//go:build lowmem

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

package network

import (
	"bytes"
	"encoding/gob"
	"net"
	"runtime/debug"

	"github.com/andmarios/crypto/nacl/saltsecret"

	conf "github.com/andmarios/bashistdb/configuration"
)

func init() {
	if conf.Log != nil {
		conf.Log.Debug.Println("Lowmem build.")
	}
}

func encryptDispatch(conn net.Conn, m Message) error {
	// We want to sent encrypted data.
	// In order to encrypt, we need to first serialize the message.
	// In order to sent/receive hassle free, we need to serialize the encrypted message
	// So: msg -> [GOB] -> [ENCRYPT] -> [GOB] -> (dispatch)

	// Create encrypter
	var encMsg bytes.Buffer
	encrypter, err := saltsecret.NewWriter(&encMsg, conf.Key, saltsecret.ENCRYPT, true)
	if err != nil {
		return err
	}

	// Serialize message
	enc := gob.NewEncoder(encrypter)
	if err = enc.Encode(m); err != nil {
		return err
	}

	// Flush encrypter to actuall encrypt the message
	if err = encrypter.Flush(); err != nil {
		return err
	}

	// Serialize encrypted message and dispatch it
	dispatch := gob.NewEncoder(conn)
	if err = dispatch.Encode(encMsg.Bytes()); err != nil {
		return err
	}
	debug.FreeOSMemory()
	return nil
}

func receiveDecrypt(conn net.Conn) (Message, error) {
	// Our work is:
	// (receive) -> [de-GOB] -> [DECRYPT] -> [de-GOB] -> msg

	// Receive data and de-serialize to get the encrypted message
	encMsg := new([]byte)
	receive := gob.NewDecoder(conn)
	if err := receive.Decode(encMsg); err != nil {
		return Message{}, err
	}

	// Create decrypter and pass it the encrypted message
	r := bytes.NewReader(*encMsg)
	decrypter, err := saltsecret.NewReader(r, conf.Key, saltsecret.DECRYPT, false)
	if err != nil {
		return Message{}, err
	}

	// Read unencrypted serialized message and de-serialize it
	msg := new(Message)
	dec := gob.NewDecoder(decrypter)
	if err = dec.Decode(msg); err != nil {
		return Message{}, err
	}
	debug.FreeOSMemory()
	return *msg, nil
}
