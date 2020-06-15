![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/raphaelreyna/ez-cgi)
[![Go Report Card](https://goreportcard.com/badge/github.com/raphaelreyna/ez-cgi)](https://goreportcard.com/report/github.com/raphaelreyna/ez-cgi)
[![GoDoc](https://godoc.org/github.com/spf13/pflag?status.svg)](https://godoc.org/github.com/raphaelreyna/ez-cgi/pkg/cgi)

# ez-cgi

An HTTP CGI server thats easy to use.
The name ez-cgi actually applies to two things, the CLI application and the Go package.
This README is about the Go package, click here for the README for the CLI application.

## About
The ez-cgi package make it easy to implement a more flexible CGI server in your Go applications.
Ez-cgi supports default headers and user defined output handlers to modify the client executables output before serving it over HTTP.
