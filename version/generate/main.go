package main

import (
	"log"

	"github.com/andmarios/go-versiongen"
)

func main() {
	if err := versiongen.Create(); err != nil {
		log.Fatalln(err)
	}
}
