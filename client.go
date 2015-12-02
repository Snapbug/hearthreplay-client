package main

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"encoding/gob"
	"flag"
	"fmt"
	"golang.org/x/net/websocket"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"bitbucket.org/snapbug/hsr/client/linejoin"
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
	Reason string

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

func upload(l Log, ws *websocket.Conn, wg *sync.WaitGroup) {
	defer wg.Done()

	path := fmt.Sprintf("%s/g/%s/%s/", url, l.Uploader, l.Key)

	resp, err := client.Head(path)

	if err != nil {
		fmt.Printf("head failed: %#v\n", err)
	} else if resp.StatusCode == http.StatusOK {
		l.Status = "Skipped"
		l.Reason = "Already Uploaded"
		websocket.JSON.Send(ws, l)
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
		l.Reason = fmt.Sprintf("Error contacting server: %s", err)
	} else if resp.StatusCode != http.StatusAccepted {
		l.Status = "Failed"
		l.Reason = fmt.Sprintf("Server returned: %d. Report %s/%s", resp.StatusCode, l.Uploader, l.Key)
	} else {
		l.Status = "Success"
		l.Reason = fmt.Sprintf("View at: %s/g/%s/%s", l.Type, url, l.Uploader, l.Key)
	}
	websocket.JSON.Send(ws, l)
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

		for line := range linejoin.NewJoiner(filenames) {
			if gameVersion.MatchString(line.Text) {
				p := gameVersion.NamedMatches(line.Text)
				version = p["version"]
				versionLine = line.Text
			}
			if screenTransition.MatchString(line.Text) {
				trans := screenTransition.NamedMatches(line.Text)
				gameTypeLine = line.Text
				gameType = trans["curr"]
			}
			if player.MatchString(line.Text) {
				parts := player.NamedMatches(line.Text)
				if log.p1.Name == "" {
					log.p1.Name = parts["name"]
				} else {
					log.p2.Name = parts["name"]
				}
			}
			if first.MatchString(line.Text) {
				parts := first.NamedMatches(line.Text)
				if log.p1.Name == parts["name"] {
					log.p1.First = true
				} else {
					log.p2.First = true
				}
			}
			if hent.MatchString(line.Text) {
				parts := hent.NamedMatches(line.Text)
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
			if hero.MatchString(line.Text) {
				p := hero.NamedMatches(line.Text)
				log.heros = append(log.heros, p["id"])
			}
			if create.MatchString(line.Text) {
				p := create.NamedMatches(line.Text)
				if p["id"] == log.heros[0] {
					log.heros_cid[0] = p["cid"]
				} else if p["id"] == log.heros[1] {
					log.heros_cid[1] = p["cid"]
				}
			}
			if winner.MatchString(line.Text) {
				p := winner.NamedMatches(line.Text)
				if log.p1.Name == p["name"] {
					log.p1.Winner = true
				} else {
					log.p2.Winner = true
				}
				log.Finish = line.Ts
				if line.Ts.Before(log.Start) {
					log.Finish = log.Finish.Add(time.Duration(24) * time.Hour)
				}
				log.Duration = log.Finish.Sub(log.Start)
			}
			if id.MatchString(line.Text) {
				p := id.NamedMatches(line.Text)
				if log.p1.Name == p["name"] {
					log.p1.ID = p["id"]
				} else {
					log.p2.ID = p["id"]
				}
			}
			if local.MatchString(line.Text) {
				p := local.NamedMatches(line.Text)
				if log.p1.ID == p["id"] {
					log.local = "1"
				} else {
					log.local = "2"
				}
			}

			if gameServer.MatchString(line.Text) {
				gs := gameServer.NamedMatches(line.Text)

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
					Start:    line.Ts,
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
					fmt.Fprintf(dbgout, "%s\n", line.Text)
				}

				log.data.WriteString(fmt.Sprintf("%s\n", versionLine))
				log.data.WriteString(fmt.Sprintf("%s\n", gameTypeLine))
				log.data.WriteString(fmt.Sprintf("%s\n", line.Text))
			} else {
				if strings.Contains(line.Text, "GameState") {
					if debug != "" && dbgout != nil {
						fmt.Fprintf(dbgout, "%s\n", line.Text)
					}
					log.data.WriteString(fmt.Sprintf("%s\n", line.Text))
				}
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
			var serversocket = new WebSocket("ws://localhost:12345/logs");
			serversocket.onmessage = function(e) {
				var d = JSON.parse(e.data);
				document.getElementById('comms').innerHTML += "<pre>" + JSON.stringify(d, undefined, 2) + "</pre>";
			};
		</script>
	</head>
	<body>
		<div id='comms'></div>
	</body>
</html>
`
)

func logServer(logs chan Log) func(ws *websocket.Conn) {
	return func(ws *websocket.Conn) {
		var wg sync.WaitGroup
		for log := range logs {
			wg.Add(1)
			go upload(log, ws, &wg)
			websocket.JSON.Send(ws, log)
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

	if debug != "" {
		for log := range getLogs(flag.Args()) {
			fmt.Printf("%s v %s\n", log.Players["local"], log.Players["remote"])
		}
	} else {
		http.Handle("/logs", websocket.Handler(logServer(getLogs(flag.Args()))))
		http.HandleFunc("/", root)
		fmt.Println("listening")

		go func() {
			<-time.After(time.Duration(100) * time.Millisecond)
			err := exec.Command("open", "http://localhost:12345").Run()
			if err != nil {
				fmt.Println(err)
			}
		}()

		if err := http.ListenAndServe(":12345", nil); err != nil {
			panic(err)
		}
	}
}
