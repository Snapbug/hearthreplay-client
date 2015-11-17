package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
)

type FileAndLine struct {
	ts  string
	scn *bufio.Scanner
	fn  string
}

func (fl *FileAndLine) Update() bool {
	if fl.scn.Scan() {
		fl.ts = strings.Split(fl.scn.Text(), " ")[1]
		return true
	}
	return false
}

type FileAndLines []*FileAndLine

func (fl FileAndLines) Len() int           { return len(fl) }
func (fl FileAndLines) Swap(i, j int)      { fl[i], fl[j] = fl[j], fl[i] }
func (fl FileAndLines) Less(i, j int) bool { return strings.Compare(fl[i].ts, fl[j].ts) < 0 }

func main() {
	flag.Parse()

	logsandlines := make(FileAndLines, flag.NArg())
	added := 0

	for _, fn := range flag.Args() {
		f, err := os.Open(fn)
		if err != nil {
			panic(err)
		} else {
			fandl := &FileAndLine{scn: bufio.NewScanner(f), fn: fn}
			if fandl.Update() {
				logsandlines[added] = fandl
				added++
			}
		}
	}

	for logsandlines.Len() > 0 {
		sort.Sort(logsandlines)
		fmt.Printf("%s\n", logsandlines[0].scn.Text())
		if !logsandlines[0].Update() {
			logsandlines = logsandlines[1:]
		}
	}
}
