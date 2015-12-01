package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"encoding/gob"
	"encoding/json"
	"flag"
	"fmt"
	"golang.org/x/net/websocket"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"bitbucket.org/snapbug/hsr/common/regexp"
)

var (
	gameVersion      = regexp.New(`gameVersion = (?P<version>\d+)`)
	screenTransition = regexp.New(`OnSceneLoaded\(\) - prevMode=(?P<prev>\S+) currMode=(?P<curr>\S+)`)
	gameServer       = regexp.New(`GotoGameServer -- address=(?P<ip>.+):(?P<port>\d+), game=(?P<game>\d+), client=(?P<client>\d+), spectateKey=(?P<key>.+)`)

	player = regexp.New(`GameState[^_]+TAG_CHANGE Entity=(?P<name>.+) tag=PLAYSTATE value=PLAYING`)
	first  = regexp.New(`GameState[^_]+TAG_CHANGE Entity=(?P<name>.+) tag=FIRST_PLAYER value=1`)
	hent   = regexp.New(`GameState[^_]+TAG_CHANGE Entity=(?P<name>.+) tag=HERO_ENTITY value=(?P<hid>\d+)`)
	winner = regexp.New(`GameState[^_]+TAG_CHANGE Entity=(?P<name>.+) tag=PLAYSTATE value=WON`)
	id     = regexp.New(`GameState[^_]+TAG_CHANGE Entity=(?P<name>.+) tag=PLAYER_ID value=(?P<id>\d+)`)
	local  = regexp.New(`SendChoices\(\) - id=(?P<id>\d+) ChoiceType=MULLIGAN`)

	hero   = regexp.New(`GameState[^_]+tag=HERO_ENTITY value=(?P<id>\d+)`)
	create = regexp.New(` - FULL_ENTITY - Creating ID=(?P<id>\d+) CardID=(?P<cid>.*)`)
)

type FileAndLine struct {
	ts  time.Time
	scn *bufio.Scanner
	fn  string
	add bool
}

const (
	timeparse = "15:04:05.0000000"
)

func (fl *FileAndLine) Update() bool {
	var err error
	if fl.scn.Scan() {
		old_time := fl.ts
		fl.ts, err = time.Parse(timeparse, strings.Split(fl.scn.Text(), " ")[1])

		if err != nil {
			panic(fmt.Sprintf("error: %s\n", err))
		}

		if fl.ts.Before(old_time) {
			fl.add = true
		}
		if fl.add {
			fl.ts = fl.ts.Add(time.Duration(24) * time.Hour)
		}
		return true
	}
	return false
}

type FileAndLines []*FileAndLine

func (fl FileAndLines) Len() int           { return len(fl) }
func (fl FileAndLines) Swap(i, j int)      { fl[i], fl[j] = fl[j], fl[i] }
func (fl FileAndLines) Less(i, j int) bool { return fl[i].ts.Before(fl[j].ts) }

type Player struct {
	Name string
	// Class  string
	Winner bool
	First  bool
	ID     string `json:"-"`
	Hero   string
}

func (p Player) String() string {
	w := "✔ "
	if !p.Winner {
		w = "" // w = "✘"
	}
	return fmt.Sprintf("%s%s (%s)", w, p.Name, p.Hero)
}

type Log struct {
	Start    time.Time
	Finish   time.Time
	Duration time.Duration
	Type     string
	Version  string
	Uploader string
	Key      string
	Data     []byte `json:"-"`
	Players  map[string]Player

	Status string

	p1        Player
	p2        Player
	local     string
	heros     []string
	heros_cid [2]string

	data bytes.Buffer
}

const (
	url = "https://hearthreplay.com"
)

var (
	tr = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client = &http.Client{Transport: tr}
)

func sendLogToWS(l Log, ws *websocket.Conn) {
	d, _ := json.MarshalIndent(l, "", "\t")
	fmt.Fprintf(ws, "%s", string(d))
}

func upload(l Log, ws *websocket.Conn, wg *sync.WaitGroup) {
	defer wg.Done()

	path := fmt.Sprintf("%s/g/%s/%s/", url, l.Uploader, l.Key)

	resp, err := client.Head(path)

	if err != nil {
		fmt.Printf("head failed: %#v\n", err)
	} else if resp.StatusCode == http.StatusOK {
		l.Status = "Skipped"
		sendLogToWS(l, ws)
		fmt.Printf("Already uploaded %s/%s -- skipping\n", l.Uploader, l.Key)
		return
	} else {
		fmt.Printf("head failed: %#v\n", resp)
	}

	var y bytes.Buffer

	gz := gzip.NewWriter(&y)
	gz.Write(l.data.Bytes())
	gz.Close()
	l.Data = y.Bytes()

	var x bytes.Buffer
	enc := gob.NewEncoder(&x)
	err = enc.Encode(l)

	if err != nil {
		panic(err)
	}

	resp, err = client.Post(path, "appliation/octet-stream", &x)

	if err != nil {
		l.Status = "Failed"
		fmt.Printf("Error contacting server: %s\n", err)
	} else if resp.StatusCode != http.StatusAccepted {
		l.Status = "Failed"
		fmt.Printf("Server returned: %s -- report %s/%s", l.Uploader, l.Key)
	} else {
		l.Status = "Success"
		fmt.Printf("Uploaded %s game: %s/g/%s/%s\n", l.Type, url, l.Uploader, l.Key)
	}
	sendLogToWS(l, ws)
}

var (
	debug string
)

func getLogs(filenames []string) chan Log {
	x := make(chan Log)
	go func(filenames []string) {
		var dbgout *os.File
		var err error

		var versionLine string
		var version string
		var gameType string
		var gameTypeLine string

		var log Log
		var found_log bool

		logsandlines := make(FileAndLines, 0)

		for _, fn := range filenames {
			f, err := os.Open(fn)
			if err != nil {
				panic(err)
			} else {
				fandl := &FileAndLine{scn: bufio.NewScanner(f), fn: fn}
				if fandl.Update() {
					logsandlines = append(logsandlines, fandl)
				}
			}
		}

		for logsandlines.Len() > 0 {
			sort.Sort(logsandlines)
			text := logsandlines[0].scn.Text()

			if gameVersion.MatchString(text) {
				p := gameVersion.NamedMatches(text)
				version = p["version"]
				versionLine = text
			}
			if screenTransition.MatchString(text) {
				trans := screenTransition.NamedMatches(text)
				gameTypeLine = text
				gameType = trans["curr"]
			}
			if player.MatchString(text) {
				parts := player.NamedMatches(text)
				if log.p1.Name == "" {
					log.p1.Name = parts["name"]
				} else {
					log.p2.Name = parts["name"]
				}
			}
			if first.MatchString(text) {
				parts := first.NamedMatches(text)
				if log.p1.Name == parts["name"] {
					log.p1.First = true
				} else {
					log.p2.First = true
				}
			}
			if hent.MatchString(text) {
				parts := hent.NamedMatches(text)
				if log.p1.Name == parts["name"] {
					if log.p1.Hero == "" {
						log.p1.Hero = parts["hid"]
					}
				} else {
					if log.p2.Hero == "" {
						log.p2.Hero = parts["hid"]
					}
				}
			}
			if hero.MatchString(text) {
				p := hero.NamedMatches(text)
				log.heros = append(log.heros, p["id"])
			}
			if create.MatchString(text) {
				p := create.NamedMatches(text)
				if p["id"] == log.heros[0] {
					log.heros_cid[0] = p["cid"]
				} else if p["id"] == log.heros[1] {
					log.heros_cid[1] = p["cid"]
				}
			}
			if winner.MatchString(text) {
				p := winner.NamedMatches(text)
				if log.p1.Name == p["name"] {
					log.p1.Winner = true
				} else {
					log.p2.Winner = true
				}
				log.Finish = logsandlines[0].ts
				if logsandlines[0].ts.Before(log.Start) {
					log.Finish = log.Finish.Add(time.Duration(24) * time.Hour)
				}
				log.Duration = log.Finish.Sub(log.Start)
			}
			if id.MatchString(text) {
				p := id.NamedMatches(text)
				if log.p1.Name == p["name"] {
					log.p1.ID = p["id"]
				} else {
					log.p2.ID = p["id"]
				}
			}
			if local.MatchString(text) {
				p := local.NamedMatches(text)
				if log.p1.ID == p["id"] {
					log.local = "1"
				} else {
					log.local = "2"
				}
			}

			if gameServer.MatchString(text) {
				gs := gameServer.NamedMatches(text)

				if found_log {
					if log.p1.Hero == log.heros[0] {
						log.p1.Hero = log.heros_cid[0]
						log.p2.Hero = log.heros_cid[1]
					} else if log.p1.Hero == log.heros[1] {
						log.p1.Hero = log.heros_cid[1]
						log.p2.Hero = log.heros_cid[0]
					} else {
						fmt.Println("Unable to determine hero classes")
					}

					if log.local == "1" {
						log.Players["local"] = log.p1
						log.Players["remote"] = log.p2
					} else {
						log.Players["local"] = log.p2
						log.Players["remote"] = log.p1
					}

					x <- log
				}

				log = Log{
					Status:   "Uploading",
					Start:    logsandlines[0].ts,
					Type:     gameType,
					Uploader: gs["client"],
					Key:      fmt.Sprintf("%s-%s", gs["game"], gs["key"]),
					Version:  version,
					Players:  make(map[string]Player),
				}
				found_log = true

				if debug != "" && dbgout != nil {
					if err = dbgout.Close(); err != nil {
						panic(err)
					}
				}
				if debug != "" {
					dbgout, err = os.Create(fmt.Sprintf("out/%s.%s.%s.(%s).log", gs["client"], gs["game"], gs["key"], gameType))
					if err != nil {
						panic(err)
					}
					fmt.Fprintf(dbgout, "%s\n", versionLine)
					fmt.Fprintf(dbgout, "%s\n", gameTypeLine)
					fmt.Fprintf(dbgout, "%s\n", text)
				}

				log.data.WriteString(fmt.Sprintf("%s\n", versionLine))
				log.data.WriteString(fmt.Sprintf("%s\n", gameTypeLine))
				log.data.WriteString(fmt.Sprintf("%s\n", text))
			} else {
				if strings.Contains(text, "GameState") {
					if debug != "" && dbgout != nil {
						fmt.Fprintf(dbgout, "%s\n", text)
					}
					log.data.WriteString(fmt.Sprintf("%s\n", text))
				}
			}

			if !logsandlines[0].Update() {
				logsandlines = logsandlines[1:]
			}
		}

		if found_log {
			if log.p1.Hero == log.heros[0] {
				log.p1.Hero = log.heros_cid[0]
				log.p2.Hero = log.heros_cid[1]
			} else if log.p1.Hero == log.heros[1] {
				log.p1.Hero = log.heros_cid[1]
				log.p2.Hero = log.heros_cid[0]
			} else {
				fmt.Println("Unable to determine hero classes")
			}
			if log.local == "1" {
				log.Players["local"] = log.p1
				log.Players["remote"] = log.p2
			} else {
				log.Players["local"] = log.p2
				log.Players["remote"] = log.p1
			}
			x <- log
		}

		close(x)
	}(filenames)
	return x
}

const (
	index = `
<html>
	<head>
		<meta charset="UTF-8" />
		<script>
			var serversocket = new WebSocket("ws://localhost:12345/echo");
			serversocket.onmessage = function(e) {
				//var d = JSON.parse(e.data);
				document.getElementById('comms').innerHTML += "<pre>" + e.data + "</pre>";
			};
		</script>
	</head>
	<body>
		<div id='comms'></div>
	</body>
</html>
`
)

func echoServer(logs chan Log) func(ws *websocket.Conn) {
	return func(ws *websocket.Conn) {
		var wg sync.WaitGroup
		for log := range logs {
			wg.Add(1)
			go upload(log, ws, &wg)
			sendLogToWS(log, ws)
		}
		wg.Wait()
		os.Exit(1)
	}
}
func root(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, index)
}

func main() {
	flag.Parse()

	http.Handle("/echo", websocket.Handler(echoServer(getLogs(flag.Args()))))
	http.HandleFunc("/", root)
	fmt.Println("listening")

	go func() {
		<-time.After(time.Duration(10) * time.Millisecond)
		err := exec.Command("open", "http://localhost:12345").Run()
		if err != nil {
			fmt.Println(err)
		}
	}()

	if err := http.ListenAndServe(":12345", nil); err != nil {
		panic(err)
	}

	fmt.Println("Fin!")
}
