# t3

[![main](https://github.com/yamlinson/t3/actions/workflows/main.yml/badge.svg)](https://github.com/yamlinson/t3/actions/workflows/main.yml)

Multiplayer tic-tac-toe in a TUI over SSH

## Quick Start

Note: This application contains no authentication or authorization mechanisms.
It is not currently intended for use on public networks.

To run the latest image with a SQLite data store and exposing SSH on port 2222:

`docker pull ghcr.io/yamlinson/t3:latest`

`docker run -it -p 2222:2222 t3:latest`

## About

This project uses
[wish](https://github.com/charmbracelet/wish)
to serve a multiplayer tic-tac-toe engine with a
[bubbletea](https://github.com/charmbracelet/wish)
TUI client.

Game results can be stored to an external PostgreSQL database if provided,
or a SQLite database is used by default.

## Usage

### Build binary from source

`git clone https://github.com/yamlinson/t3.git`

`cd t3`

`go build -o t3`

`./t3`

### Build container image from source

`git clone https://github.com/yamlinson/t3.git`

`cd t3`

`docker build -t t3:latest`

### Run with Docker

Pull the latest image if you don't already have it:

`docker pull ghcr.io/yamlinson/t3:latest`

Note: `-it` ensures full color support

`docker run -it -p 2222:2222 t3:latest`

#### Optionally, mount /data to a local dir to persist the SQLite database

`docker run --mount type=bind,src=./t3-data,dst=/data -it -p 2222 t3:latest`

### Connect to the game

`ssh -p 2222 localhost`

## Configuration

By default, the server will attempt to create a SQLite database at `/data/t3.db`.
To override this behavior, such as when running the binary outside of a container environment,
set `SQLITE_PATH` in your environment.

This could also be used to persist database state to a preferred location when mounting storage to a container.

To use a PostgreSQL database, set `DATABASE_URL` in the server environment with the standard format:

`postgresql://user:password@host:port/database`
