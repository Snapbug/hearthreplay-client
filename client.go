package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"bitbucket.org/snapbug/hsr/common/regexp"
)

var (
	gameVersion      = regexp.New(`gameVersion = (?P<version>\d+)`)
	screenTransition = regexp.New(`OnSceneLoaded\(\) - prevMode=(?P<prev>\S+) currMode=(?P<curr>\S+)`)
	gameServer       = regexp.New(`GotoGameServer -- address=(?P<ip>.+):(?P<port>\d+), game=(?P<game>\d+), client=(?P<client>\d+), spectateKey=(?P<key>.+)`)
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

	var out *os.File
	var version string
	var gameType string
	var err error

	for logsandlines.Len() > 0 {
		// sort the lines -- should really do something else, when one wraps around it'll screw this up!
		sort.Sort(logsandlines)
		text := logsandlines[0].scn.Text()

		if gameVersion.MatchString(text) {
			version = text
			fmt.Printf("Found game version in: %s\n", version)
		}

		if screenTransition.MatchString(text) {
			trans := screenTransition.NamedMatches(text)
			gameType = trans["curr"]
		}

		if gameServer.MatchString(text) {
			gs := gameServer.NamedMatches(text)
			fmt.Printf("%s Game: %s/%s/%s @ %s:%s\n", gameType, gs["game"], gs["client"], gs["key"], gs["ip"], gs["key"])

			if out != nil {
				if err = out.Close(); err != nil {
					panic(err)
				}
			}
			out, err = os.Create(fmt.Sprintf("out/%s.%s.%s.(%s).log", gs["client"], gs["game"], gs["key"], gameType))
			if err != nil {
				panic(err)
			}
			fmt.Fprintf(out, "%s\n", version)
			fmt.Fprintf(out, "%s\n", text)
		} else {
			if out != nil && strings.Contains(text, "GameState") {
				fmt.Fprintf(out, "%s\n", text)
			}
		}

		if !logsandlines[0].Update() {
			logsandlines = logsandlines[1:]
		}
	}
}
