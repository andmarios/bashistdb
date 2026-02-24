bashistdb
=========

Introduction
-----------

Bashistdb stands for Bash History Database.

Bashistdb stores bash history into a sqlite database.
It can either be run as standalone, or it can be run in server-client mode,
where many clients can store their history into a single database over the
network. In this mode, communications are compressed and encrypted.

Bashistdb stores for each history line the time it was run, the user that run it,
the hostname, the shell session PID, and the working directory. Currently it isn't
meant to be secure against users. This means
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

To build from source with version information from git tags:

    $ go generate ./...
    $ go build

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

### Session and Working Directory Tracking ###

Bashistdb tracks which shell session (by PID) and working directory each command
was run from. This is done via environment variables set in the PROMPT_COMMAND
(handled automatically by `bashistdb -init`).

You can filter queries by session or working directory:

    $ bashistdb -session 12345 -lastk 20
    $ bashistdb -workdir myproject -lastk 20
    $ bashistdb -session 12345 -workdir src git

These filters combine with any query type (`-lastk`, `-topk`, `-A`/`-B`/`-C`, etc.).

Use `-f all` or `-f json` to see session PID and working directory in the output.

### Knobs ###

Run `bashistdb -h` to get a glimpse of available options. They are easy to understand.
Currently the most useful command not covered until here is `-g`. G stands for global
and makes your query to search for commands from all users at any host.

An important knob is the `lowmem` build tag. Bashistdb uses scrypt to generate a new
key for each new network message. Whilst secure, it can make bashistdb in server mode
to use too much RAM and thus lead to a DoS attack. Scrypt's protection model is to use
much RAM and CPU so that an attacker won't be able to bruteforce your password. A light
use bashistdb thus will use about 80MiB of RAM. If some network messages overlap though,
it may use a few hunderd MiBs.

You can use the lowmem build tag (`go get -tags lowmem github.com/andmarios/bashistdb`)
to get a version of bashistdb that tries to free memory agressively, thus usually staying
below 20MiB of RAM and occasionally rising shortly to 44 or more, depending on simultaneous
network connections. Performance wise this is sub-optimal but if you are on a low-end
server it is necessary.

CI/CD
-----

CI runs automatically on every push to `master` and on pull requests. It checks formatting,
runs `go vet`, `golangci-lint`, `govulncheck`, tests (with race detector), and builds both
standard and lowmem variants.

Tagged releases (`v*`) produce cross-platform binaries for Linux (amd64, arm64),
macOS (amd64, arm64), and Windows (amd64) — each in standard and lowmem variants.
Binaries are published as GitHub Release assets.

### Docker ###

A Docker image is published to GHCR on each tagged release:

    $ docker pull ghcr.io/andmarios/bashistdb:latest

Run the server with a persistent volume:

    $ docker run -d \
        -p 25625:25625 \
        -v bashistdb-data:/data \
        -e BASHISTDB_KEY=your_passphrase \
        ghcr.io/andmarios/bashistdb:latest

The image runs in server mode by default, uses `/data` as the database directory,
and exposes port `25625`.

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
