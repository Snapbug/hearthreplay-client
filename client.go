package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"flag"
	"fmt"
	"net/http"
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

type Log struct {
	Type     string
	Version  string
	Uploader string
	Key      string
	Data     []byte

	data bytes.Buffer
}

const (
	url = "http://192.168.99.100:8080"
)

func upload(l Log) {
	var y bytes.Buffer

	gz := gzip.NewWriter(&y)
	gz.Write(l.data.Bytes())
	gz.Close()
	l.Data = y.Bytes()

	var x bytes.Buffer
	enc := gob.NewEncoder(&x)
	err := enc.Encode(l)

	if err != nil {
		panic(err)
	}

	resp, err := http.Post(fmt.Sprintf("%s/g/%s/%s/", url, l.Uploader, l.Key), "appliation/octet-stream", &x)

	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("Uploaded: %s/%s\n", l.Uploader, l.Key)
		fmt.Printf("%s\n", resp)
	}
	panic("") // only upload 1 ... mmmm
}

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

	var dbgout *os.File
	var err error

	var versionLine string
	var version string
	var gameType string

	var shupload bool
	var log Log

	for logsandlines.Len() > 0 {
		// sort the lines -- should really do something else, when one wraps around it'll screw this up!
		sort.Sort(logsandlines)
		text := logsandlines[0].scn.Text()

		if gameVersion.MatchString(text) {
			p := gameVersion.NamedMatches(text)
			version = p["version"]
			versionLine = text
			fmt.Printf("Found game version in: %s\n", version)
		}

		if screenTransition.MatchString(text) {
			trans := screenTransition.NamedMatches(text)
			gameType = trans["curr"]
		}

		if gameServer.MatchString(text) {
			gs := gameServer.NamedMatches(text)
			fmt.Printf("%s Game: %s/%s/%s @ %s:%s\n", gameType, gs["game"], gs["client"], gs["key"], gs["ip"], gs["key"])

			if shupload {
				upload(log)
			}
			log = Log{
				Type:     gameType,
				Uploader: gs["client"],
				Key:      fmt.Sprintf("%s-%s", gs["game"], gs["key"]),
				Version:  version,
			}
			shupload = true

			if dbgout != nil {
				if err = dbgout.Close(); err != nil {
					panic(err)
				}
			}
			dbgout, err = os.Create(fmt.Sprintf("out/%s.%s.%s.(%s).log", gs["client"], gs["game"], gs["key"], gameType))
			if err != nil {
				panic(err)
			}
			fmt.Fprintf(dbgout, "%s\n", versionLine)
			fmt.Fprintf(dbgout, "%s\n", text)

			log.data.Write([]byte(versionLine))
			log.data.Write([]byte("\n"))
			log.data.Write([]byte(text))
			log.data.Write([]byte("\n"))
		} else {
			if dbgout != nil && strings.Contains(text, "GameState") {
				fmt.Fprintf(dbgout, "%s\n", text)

				log.data.Write([]byte(text))
				log.data.Write([]byte("\n"))
			}
		}

		if !logsandlines[0].Update() {
			logsandlines = logsandlines[1:]
		}
	}

	if shupload {
		upload(log)
	}
}
