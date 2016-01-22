package main

import (
	"flag"
	"fmt"
	"path/filepath"

	"bitbucket.org/snapbug/hsr/client/linejoin"
)

func main() {
	flag.Parse()

	p, err := filepath.Glob(fmt.Sprintf("%s*.log", flag.Arg(0)))
	if err != nil {
		panic(err)
	}
	for line := range linejoin.NewJoiner(p) {
		fmt.Printf("%s - %s\n", line.File, line.Text)
	}
}
