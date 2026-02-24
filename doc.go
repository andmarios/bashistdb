/*
Command bashistdb stores and retrieves bash history into/from a sqlite3
database. It provides advanced functionality such as saving remotely,
keeping history from many users, restoring, quering etc.

bashistdb
=========

Introduction
-----------

Bashistdb stands for Bash History Database.

Bashistdb stores bash history into a sqlite database.
It can either be run as standalone, or it can be run in server-client mode,
where many clients can store their history into a single database over the
network. In this mode, communications are compressed and encrypted.

Bashistdb stores for each history line the time it was run, the user that run it
and the hostname. Currently it isn't meant to be secure against users. This means
that any user may be able to see commands that other users run, or store commands
under different user and hostnames. This is by design. One person may have many
accounts in one or more machines.

It is work in progress. Some features are missing but it has a strong
foundation upon which new features can be build.

Running
-------

### Pre-requisites ###

Install sqlite3 on your machine and go get bashistdb:

	$ go get github.com/andmarios/bashistdb

If you are on a hardened machine, you may need instead:

	$ go get -u -ldflags '-extldflags=-fno-PIC' github.com/andmarios/bashistdb

Bashistdb needs your history to be timestamped in order to work. It understands
the RFC3339 time format.
If you want to also import your current history, you need to add unique
timestamps to it. Bashistdb can perform these steps for you in one step:

	$ bashistdb -init

That's it. Logout and login (or source your bashrc) for the changes to take
effect.

#### Initializing manually ####

If you don't like the automatic setup above, you can perform the steps
needed manually.

In order to set up your bash to log and report RFC3339 timestamps, run:

	$ export HISTTIMEFORMAT="%FT%T%z "
	$ echo 'export HISTTIMEFORMAT="%FT%T%z "' >> ~/.bashrc

Add to your ~/.bashrc (safe to source multiple times):

	if [[ "$PROMPT_COMMAND" != *"bashistdb"* ]]; then
	    [ -n "${PROMPT_COMMAND}" ] && PROMPT_COMMAND="${PROMPT_COMMAND};"
	    export PROMPT_COMMAND="${PROMPT_COMMAND} (history 1 | BASHISTDB_PID=\$\$ BASHISTDB_CWD=\$PWD bashistdb 2>/dev/null &)"
	fi

Add distinct timestamps to your current bash_history:

	$ go get github.com/andmarios/bashistdb/tools/addTimestamp2Hist
	$ addTimestamp2Hist -since 24 -write

This will create timestamps for your current commands that span equally accross
the 24 last months.

### Local mode ###

In local mode your history is stored on your computer.

Import your current history. You can import it as many times as you want. It is
very fast and only new lines will be added.

	$ history | bashistdb

Check some stats:

	$ bashistdb -v 1

Perform a query:

	$ bashistdb <SEARCH TERM>

Restore your history file, percent sign (%) acts as wildcard for the query:

	$ bashistdb -format restore % > ~/.bash_history

### Server - Client mode ###

Start your server¹:

	$ bashistdb -server -key <PASSPHRASE>

From your client machine run bashistdb in client mode:

	$ history | bashistdb -remote <SERVER> -key <PASSPHRASE>

You may use a configuration file or environment variables to setup bashistdb.

Environment variables:

	$ export BASHISTDB_REMOTE=<SERVER>
	$ export BASHISTDB_KEY=<PASSPHRASE>
	$ bashistdb -verbose 1

Configuration file (~/.bashistdb.conf) is better. You can create it and update
it with bashistdb:

	$ bashistdb -r <SERVER> -k <PASSPHRASE> -p <PORT> -save

Update a variable in the configuration:

	$ bashistdb -k <NEW PASSPHRASE> -save

Messages are encrypted using NaCl secret-key authenticated encryption and
scrypt key derivation. Check <https://github.com/andmarios/crypto/nacl/saltsecret>
if you are interested for a higher lever wrapper for golang's crypto/nacl/secretbox.

1: Currently bashistdb listens to all network interfaces (0.0.0.0). It
may get a listen address configuration option in the future.

### Knobs ###

Run `bashistdb -h` to get a glimpse of available options. They are easy to understand.
Currently the most useful command not covered until here is `-g`. G stands for global
and makes your query to search for commands from all users at any host.

License
-------

Copyright (c) 2015-2026, Marios Andreopoulos.

This file is part of bashistdb.

Bashistdb is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

Bashistdb is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with bashistdb.  If not, see <http://www.gnu.org/licenses/>.
*/
package main
