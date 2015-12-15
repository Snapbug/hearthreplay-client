package main

import (
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"golang.org/x/net/websocket"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"

	"bitbucket.org/snapbug/hsr/client/linejoin"
	"bitbucket.org/snapbug/hsr/client/location"
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
	Playrs   map[string]Player

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
	upload_url = "https://hearthreplay.com"
	update_url = "https://update.hearthreplay.com"
)

func send(ws *websocket.Conn, l Log) {
	if err := websocket.JSON.Send(ws, l); err != nil {
		fmt.Println(err)
	}
}

func upload(l Log, ws *websocket.Conn, wg *sync.WaitGroup) {
	defer wg.Done()

	path := fmt.Sprintf("%s/g/%s/%s/", upload_url, l.Uploader, l.Key)

	resp, err := http.Head(path)

	if err != nil {
		// fmt.Printf("head failed: %#v\n", err)
	} else if resp.StatusCode == http.StatusOK {
		l.Status = "Skipped"
		l.Reason = "Already Uploaded"
		send(ws, l)
		// fmt.Printf("Already uploaded %s/%s -- skipping\n", l.Uploader, l.Key)
		return
	} else {
		// fmt.Printf("head failed: %#v\n", resp)
	}

	var y bytes.Buffer

	gz := gzip.NewWriter(&y)
	if _, err := gz.Write(l.data.Bytes()); err != nil {
		panic(err)
	}
	if err := gz.Close(); err != nil {
		panic(err)
	}
	l.Data = y.Bytes()

	var x bytes.Buffer
	enc := gob.NewEncoder(&x)
	err = enc.Encode(l)

	if err != nil {
		panic(err)
	}

	resp, err = http.Post(path, "appliation/octet-stream", &x)

	if err != nil {
		l.Status = "Failed"
		l.Reason = fmt.Sprintf("Error contacting server: %s", err)
	} else if resp.StatusCode != http.StatusAccepted {
		l.Status = "Failed"
		l.Reason = fmt.Sprintf("Server returned: %d. Report %s/%s", resp.StatusCode, l.Uploader, l.Key)
	} else {
		l.Status = "Success"
		l.Reason = fmt.Sprintf("View at: %s/g/%s/%s/", upload_url, l.Uploader, l.Key)
	}
	send(ws, l)
}

var (
	debug string
)

func getLogs(logfolder string) chan Log {
	x := make(chan Log)

	filenames := make([]string, 0)
	for _, f := range []string{"Power", "Net", "LoadingScreen", "UpdateManager"} {
		for _, suf := range []string{"", "_old"} {
			fn := fmt.Sprintf("%s%s.log", f, suf)
			p := filepath.Join(logfolder, fn)
			if _, err := os.Stat(p); os.IsNotExist(err) {
				continue
			} else {
				filenames = append(filenames, filepath.Join(logfolder, fn))
			}
			break
		}
	}

	go func(filenames []string) {
		send_log := func(log Log, x chan Log) {
			if log.p1.Hero == log.heros[0] {
				log.p1.Hero = HeroClass[log.heros_cid[0]]
				log.p2.Hero = HeroClass[log.heros_cid[1]]
			} else if log.p1.Hero == log.heros[1] {
				log.p1.Hero = HeroClass[log.heros_cid[1]]
				log.p2.Hero = HeroClass[log.heros_cid[0]]
			} else {
				fmt.Println("Unable to determine hero classes")
			}

			if log.local == "1" {
				log.Playrs["local"] = log.p1
				log.Playrs["remote"] = log.p2
			} else {
				log.Playrs["local"] = log.p2
				log.Playrs["remote"] = log.p1
			}

			p := strings.Split(log.Type, "_")
			for i, _ := range p {
				p[i] = strings.Title(strings.ToLower(p[i]))
			}
			log.Type = strings.Join(p, " ")
			x <- log
		}

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
					send_log(log, x)
				}
				log = Log{
					Status:   "Uploading",
					Start:    line.Ts,
					Type:     gameType,
					Uploader: gs["client"],
					Key:      fmt.Sprintf("%s-%s", gs["game"], gs["key"]),
					Version:  version,
					Playrs:   make(map[string]Player),
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

				if _, err := log.data.WriteString(fmt.Sprintf("%s\n", versionLine)); err != nil {
					panic(err)
				}
				if _, err := log.data.WriteString(fmt.Sprintf("%s\n", gameTypeLine)); err != nil {
					panic(err)
				}
				if _, err := log.data.WriteString(fmt.Sprintf("%s\n", line.Text)); err != nil {
					panic(err)
				}
			} else {
				if strings.Contains(line.Text, "GameState") {
					if debug != "" && dbgout != nil {
						fmt.Fprintf(dbgout, "%s\n", line.Text)
					}
					if _, err := log.data.WriteString(fmt.Sprintf("%s\n", line.Text)); err != nil {
						panic(err)
					}
				}
			}
		}

		if found_log {
			send_log(log, x)
		}

		close(x)
	}(filenames)
	return x
}

func logServer(logFolder string) func(ws *websocket.Conn) {
	return func(ws *websocket.Conn) {
		var wg sync.WaitGroup
		for log := range getLogs(logFolder) {
			wg.Add(1)
			go upload(log, ws, &wg)
			send(ws, log)
		}
		wg.Wait()
		os.Exit(1)
	}
}

type PP struct {
	Port     string
	Version  string
	Latest   string
	OS, Arch string
}

func root(w http.ResponseWriter, r *http.Request) {
	d, err := Asset("tmpl/index.html")
	if err != nil {
		panic(err)
	}
	t, err := template.New("index").Parse(string(d))
	if err != nil {
		panic(err)
	}
	t.Execute(w, p)
}

func changelog(w http.ResponseWriter, r *http.Request) {
	d, err := Asset("tmpl/changelog.html")
	if err != nil {
		panic(err)
	}
	t, err := template.New("changelog").Parse(string(d))
	if err != nil {
		panic(err)
	}
	t.Execute(w, p)
}

var (
	Version string
	p       PP
)

type Config struct {
	Install location.SetupLocation
	Version string
}

var conf Config

var (
	HeroClass = map[string]string{
		"HERO_01":  "Warrior",
		"HERO_02":  "Shaman",
		"HERO_03":  "Rogue",
		"HERO_04":  "Paladin",
		"HERO_05":  "Hunter",
		"HERO_06":  "Druid",
		"HERO_07":  "Warlock",
		"HERO_08":  "Mage",
		"HERO_09":  "Priest",
		"HERO_08a": "Mage",
		"HERO_05a": "Hunter",
		"HERO_01a": "Warrior",
	}
	HeroName = map[string]string{
		"HERO_01":  "Garrosh Hellscream",
		"HERO_02":  "Thrall",
		"HERO_03":  "Valeera Sanguinar",
		"HERO_04":  "Uther Lightbringer",
		"HERO_05":  "Rexxar",
		"HERO_06":  "Malfurion Stormrage",
		"HERO_07":  "Gul'dan",
		"HERO_08":  "Jaina Proudmoore",
		"HERO_09":  "Anduin Wrynn",
		"HERO_08a": "Medivh",
		"HERO_05a": "Alleria Windrunner",
		"HERO_01a": "Magni Bronzebeard",
	}
)

func main() {
	cf, err := os.Open("config.json")

	if err != nil {
		panic(err)
	}

	err = json.NewDecoder(cf).Decode(&conf)
	if err != nil {
		panic(err)
	}

	if Version != "" && Version != conf.Version {
		panic("Version mismatch")
	}

	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		panic(err)
	}

	_, p.Port, _ = net.SplitHostPort(listener.Addr().String())
	p.Version = conf.Version

	http.HandleFunc("/", root)
	http.Handle("/logs", websocket.Handler(logServer(conf.Install.LogFolder)))
	http.Handle("/s/", http.StripPrefix("/s/", http.FileServer(http.Dir("tmpl"))))
	http.HandleFunc("/changelog", changelog)
	http.HandleFunc("/quit", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		go func() {
			<-time.After(1 * time.Millisecond)
			os.Exit(1)
		}()
	})

	go func() {
		var err error
		if runtime.GOOS == "darwin" {
			err = exec.Command("open", fmt.Sprintf("http://localhost:%s/", p.Port)).Run()
		} else {
			err = exec.Command("cmd", []string{"/c", "start", fmt.Sprintf("http://localhost:%s/", p.Port)}).Run()
		}
		if err != nil {
			fmt.Println(err)
		}
	}()

	if err = http.Serve(listener, nil); err != nil {
		panic(err)
	}
}
