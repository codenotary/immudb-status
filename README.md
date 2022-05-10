# Immudb-status

A simple tool to fetch the current status from all databases in a immudb installation.

## Building from source

```sh
go build
```

## Usage

```
./immudb-status -h
Usage of ./immudb-status:
  -addr string
        IP address of immudb server [IMMUDB_ADDRESS] (default "127.0.0.1")
  -pass string
        Admin password for immudb [IMMUDB_PASSWORD] (default "immudb")
  -port int
        Port number of immudb (default 3322)
  -user string
        Admin username for immudb (default "immudb")
```
