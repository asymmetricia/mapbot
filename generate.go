//go:generate go build ./vendor/github.com/gopherjs/gopherjs
//go:generate ./gopherjs build -m -o static/mapbot.js ./js
//go:generate go run static_generate.go
package main
