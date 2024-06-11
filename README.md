[![build](https://github.com/linkdata/webserv/actions/workflows/go.yml/badge.svg)](https://github.com/linkdata/webserv/actions/workflows/go.yml)
[![coverage](https://coveralls.io/repos/github/linkdata/webserv/badge.svg?branch=main)](https://coveralls.io/github/linkdata/webserv?branch=main)
[![goreport](https://goreportcard.com/badge/github.com/linkdata/webserv)](https://goreportcard.com/report/github.com/linkdata/webserv)
[![Docs](https://godoc.org/github.com/linkdata/webserv?status.svg)](https://godoc.org/github.com/linkdata/webserv)

# webserv

Thin web server stub.

Given a listen address, certificate directory, user name and data directory:

* If certificate directory is not blank, reads `fullchain.pem` and `privkey.pem` from it.
* If the listen address does not specify a port, default port depends on initial user privileges and if we have a certificate.
* Starts listening on the address and port.
* If user name is given, switch to that user.
* If data directory is given, create it if needed and then switch current directory to it.
