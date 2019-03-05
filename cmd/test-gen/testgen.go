package main

import (
	"flag"

	testgen "github.com/AlexMarco7/aclow/pkg/test-gen"
)

func main() {
	src := flag.String("src", "", "source log file")
	dest := flag.String("dest", "", "dest test file")

	flag.Parse()

	testgen.Generate(*src, *dest)
}
