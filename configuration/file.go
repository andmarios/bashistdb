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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
)

// exportFields is a struct used to export some
// configuration variables to JSON and then to a
// file
type exportFields struct {
	Database string
	Remote   string
	Port     string
	Key      string
}

// Read configuration file, overrides environment variables.
func readConfFile() error {
	// Try to load settings from configuration file (if exists)
	if c, err := ioutil.ReadFile(confFile); err == nil {
		e := &exportFields{}
		if err = json.Unmarshal(c, e); err == nil {
			if e.Database != "" {
				database = e.Database
			}
			if e.Remote != "" {
				remote = e.Remote
			}
			if e.Port != "" {
				port = e.Port
			}
			if e.Key != "" {
				passphrase = e.Key
			}
			foundConfFile = true
		} else {
			return errors.New("Could not parse configuration file: " +
				err.Error())
		}
	}
	return nil
}

// Write configuration file, pretty prints JSON instead of just Marshal
func writeConfFile() error {
	conf := fmt.Sprintf(`{
"database": %#v,
"remote"  : %#v,
"port"    : %#v,
"key"     : %#v
}
`, Database, remote, port, string(Key))
	err := ioutil.WriteFile(confFile, []byte(conf), 0600)
	if err != nil {
		return err
	}
	return nil
}
