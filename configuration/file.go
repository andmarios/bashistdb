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
	"os"
)

// exportFields is a struct used to export some
// configuration variables to JSON and then to a
// file
type exportFields struct {
	Database string `json:"database"`
	Remote   string `json:"remote"`
	Port     string `json:"port"`
	Key      string `json:"key"`
}

// Read configuration file, overrides environment variables.
func readConfFile() error {
	c, err := os.ReadFile(confFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	e := &exportFields{}
	if err = json.Unmarshal(c, e); err != nil {
		return errors.New("Could not parse configuration file: " +
			err.Error())
	}
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
	return nil
}

// Write configuration file using proper JSON encoding.
func writeConfFile() error {
	conf := exportFields{
		Database: Database,
		Remote:   remote,
		Port:     port,
		Key:      string(Key),
	}
	data, err := json.MarshalIndent(conf, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(confFile, data, 0600)
}
