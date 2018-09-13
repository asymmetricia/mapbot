// +build ignore

package main

import (
	"github.com/shurcooL/vfsgen"
	"net/http"
)

func main() {
	err := vfsgen.Generate(http.Dir("static"), vfsgen.Options{
		Filename:    "ui/http/assets.go",
		PackageName: "http",
	})
	if err != nil {
		panic(err)
	}
}
