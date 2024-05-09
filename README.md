# muxr

[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white)](https://pkg.go.dev/github.com/btwiuse/muxr?tab=doc)
[![Go 1.18+](https://img.shields.io/github/go-mod/go-version/btwiuse/muxr)](https://golang.org/dl/)
[![License](https://img.shields.io/github/license/btwiuse/muxr?color=%23000&style=flat-round)](https://github.com/btwiuse/muxr/blob/main/LICENSE)

A simple wrapper around the new Go 1.22 muxer. It decorates the net/http muxer with the following:

1. Middleware functionality.
2. Trailing slash footgun protection.
3. Subrouting.
4. HTTP methods as Go methods such as `muxer.Post("/hello", helloHandler)` instead of `muxer.Handle("POST /hello", helloHandler)`. 
