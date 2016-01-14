package main

import (
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"golang.org/x/net/websocket"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/icub3d/graceful"
	"github.com/kardianos/osext"

	"bitbucket.org/snapbug/hsr/client/linejoin"
	"bitbucket.org/snapbug/hsr/client/location"
	"bitbucket.org/snapbug/hsr/common/regexp"
)

var (
	levelLine = regexp.New(`unloading name=Medal_Ranked_(?P<level>\d+)`)
	typeLine  = `unloading name=RankChangeTwoScoop`

	// these are used to get important information to show the client -- validated by the server
	gameVersion      = regexp.New(`gameVersion = (?P<version>\d+)`)
	screenTransition = regexp.New(`OnSceneLoaded\(\) - prevMode=(?P<prev>\S+) currMode=(?P<curr>\S+)`)
	gameServer       = regexp.New(`GotoGameServer -- address=(?P<ip>.+):(?P<port>\d+), game=(?P<game>\d+), client=(?P<client>\d+), spectateKey=(?P<key>.+)`)

	// these are used to show the client something -- unvalidated by the server, but extracted from logs
	player = regexp.New(`TAG_CHANGE Entity=(?P<name>.+) tag=PLAYSTATE value=PLAYING`)
	first  = regexp.New(`TAG_CHANGE Entity=(?P<name>.+) tag=FIRST_PLAYER value=1`)
	hent   = regexp.New(`TAG_CHANGE Entity=(?P<name>.+) tag=HERO_ENTITY value=(?P<hid>\d+)`)
	winner = regexp.New(`TAG_CHANGE Entity=(?P<name>.+) tag=PLAYSTATE value=WON`)
	id     = regexp.New(`TAG_CHANGE Entity=(?P<name>.+) tag=PLAYER_ID value=(?P<id>\d+)`)
	hero   = regexp.New(`tag=HERO_ENTITY value=(?P<id>\d+)`)
	create = regexp.New(` - FULL_ENTITY - Creating ID=(?P<id>\d+) CardID=(?P<cid>.*)`)
	local  = regexp.New(`SendChoices\(\) - id=(?P<id>\d+) ChoiceType=MULLIGAN`)
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
	Start       time.Time
	Finish      time.Time
	Duration    time.Duration
	Type        string
	DisplayType string
	Version     string
	Uploader    string
	Key         string
	Data        []byte `json:"-"`
	Playrs      map[string]Player

	Status string
	Reason string

	p1        Player
	p2        Player
	local     string
	heros     []string
	heros_cid [2]string
	spectate  bool

	data bytes.Buffer
}

func (l Log) String() string {
	return fmt.Sprintf("%s vs %s %s(%s) %s/%s", l.p1, l.p2, l.DisplayType, l.Type, l.Uploader, l.Key)
}

const (
	upload_url = "https://hearthreplay.com"
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
		fmt.Printf("head failed: %#v\n", err)
	} else if resp.StatusCode == http.StatusOK {
		l.Status = "Skipped"
		l.Reason = "Already Uploaded"
		send(ws, l)
		fmt.Printf("Already uploaded %s/%s -- skipping\n", l.Uploader, l.Key)
		return
	} else {
		fmt.Printf("head failed: %#v\n", resp)
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

	path = fmt.Sprintf("%sv1", path)
	resp, err = http.Post(path, "appliation/octet-stream", &x)

	fmt.Printf("%s\n", l)
	if err != nil {
		l.Status = "Failed"
		l.Reason = fmt.Sprintf("Error contacting server: %s", err)
	} else if resp.StatusCode != http.StatusAccepted {
		l.Status = "Failed"
		l.Reason = fmt.Sprintf("Server returned: %d. Report %s/%s", resp.StatusCode, l.Uploader, l.Key)
		b, _ := ioutil.ReadAll(resp.Body)
		fmt.Printf("%s", b)
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

	fmt.Printf("%s:\n", logfolder)
	filenames := make([]string, 0)
	for _, f := range []string{"Asset", "Bob", "Net", "Power", "LoadingScreen", "UpdateManager"} {
		fmt.Printf("%s::\n", f)
		for _, suf := range []string{"", "_old"} {
			fn := fmt.Sprintf("%s%s.log", f, suf)
			p := filepath.Join(logfolder, fn)
			if _, err := os.Stat(p); os.IsNotExist(err) {
				fmt.Printf("%s: %#v\n", p, err)
				continue
			} else {
				filenames = append(filenames, filepath.Join(logfolder, fn))
			}
			break
		}
	}
	fmt.Printf("%#v\n", filenames)

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

			if ty, ok := gameTypeMap[log.Type]; ok {
				log.DisplayType = ty
			}
			if log.Uploader == "0" {
				log.Status = "Failed"
				log.Reason = "Spectated Game"
				log.DisplayType = "Spectated"
			}
			x <- log
		}

		var dbgout *os.File
		var err error

		var versionLine string
		var version string
		var gameType string
		var gameTypeLine string
		var subtypeLine string
		var ranklevel string

		var log Log
		var found_log bool

		for line := range linejoin.NewJoiner(filenames) {
			if gameVersion.MatchString(line.Text) {
				p := gameVersion.NamedMatches(line.Text)
				version = p["version"]
				versionLine = line.Text
			} else if screenTransition.MatchString(line.Text) {
				trans := screenTransition.NamedMatches(line.Text)
				gameTypeLine = line.Text
				if subtypeLine == "" {
					gameType = trans["curr"]
				}
			} else if strings.Contains(line.Text, typeLine) {
				gameType = "RANKED"
				subtypeLine = line.Text
			} else if levelLine.MatchString(line.Text) {
				// might be multiple medals for cross-rank play so get the smallest
				// a game between 4/5 is really a game for level 4
				if ranklevel != "" {
					p1 := levelLine.NamedMatches(ranklevel)
					p2 := levelLine.NamedMatches(line.Text)
					l1, _ := strconv.Atoi(p1["level"])
					l2, _ := strconv.Atoi(p2["level"])
					if l2 < l1 {
						ranklevel = line.Text
					}
				} else {
					ranklevel = line.Text
				}
			}

			if strings.Contains(line.Text, "GameState") {
				if player.MatchString(line.Text) {
					parts := player.NamedMatches(line.Text)
					if log.p1.Name == "" {
						log.p1.Name = parts["name"]
					} else {
						log.p2.Name = parts["name"]
					}
				} else if first.MatchString(line.Text) {
					parts := first.NamedMatches(line.Text)
					if log.p1.Name == parts["name"] {
						log.p1.First = true
					} else {
						log.p2.First = true
					}
				} else if hent.MatchString(line.Text) {
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
				} else if hero.MatchString(line.Text) {
					p := hero.NamedMatches(line.Text)
					log.heros = append(log.heros, p["id"])
					fmt.Println(line.Text)
				} else if create.MatchString(line.Text) {
					p := create.NamedMatches(line.Text)
					if p["id"] == log.heros[0] {
						log.heros_cid[0] = p["cid"]
					} else if p["id"] == log.heros[1] {
						log.heros_cid[1] = p["cid"]
					}
				} else if winner.MatchString(line.Text) {
					p := winner.NamedMatches(line.Text)
					if log.p1.Name == p["name"] {
						log.p1.Winner = true
					} else {
						log.p2.Winner = true
					}
					log.Finish = line.Ts
					log.Duration = log.Finish.Sub(log.Start)
				} else if id.MatchString(line.Text) {
					p := id.NamedMatches(line.Text)
					if log.p1.Name == p["name"] {
						log.p1.ID = p["id"]
					} else {
						log.p2.ID = p["id"]
					}
				} else if local.MatchString(line.Text) {
					p := local.NamedMatches(line.Text)
					if log.p1.ID == p["id"] {
						log.local = "1"
					} else {
						log.local = "2"
					}
				}
			}

			if gameServer.MatchString(line.Text) {
				gs := gameServer.NamedMatches(line.Text)

				if found_log {
					ranklevel = ""
					subtypeLine = ""
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
					dbgout, err = os.Create(fmt.Sprintf("logs/%s.%s.%s.(%s).log", gs["client"], gs["game"], gs["key"], gameType))
					if err != nil {
						panic(err)
					}
					fmt.Fprintf(dbgout, "%s\n", versionLine)
					if log.Type == "RANKED" {
						fmt.Fprintf(dbgout, "%s\n", subtypeLine)
					} else {
						fmt.Fprintf(dbgout, "%s\n", gameTypeLine)
					}
					fmt.Fprintf(dbgout, "%s\n", line.Text)
					fmt.Fprintf(dbgout, "%s\n", ranklevel)
				}

				log.data.WriteString(fmt.Sprintf("%s\n", versionLine))
				if log.Type == "RANKED" {
					log.data.WriteString(fmt.Sprintf("%s\n", subtypeLine))
				} else {
					log.data.WriteString(fmt.Sprintf("%s\n", gameTypeLine))
				}
				log.data.WriteString(fmt.Sprintf("%s\n", line.Text))
				log.data.WriteString(fmt.Sprintf("%s\n", ranklevel))
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
			if conf.Player == "" && log.Uploader != "0" {
				conf.Player = log.Uploader
				writeLocalConfig()
			}
			if log.Uploader != "0" {
				wg.Add(1)
				go upload(log, ws, &wg)
			}
			send(ws, log)
		}
		wg.Wait()
		ws.Close()
		os.Exit(0)
	}
}

type PP struct {
	Port     string
	Player   string
	Version  string
	Latest   string
	OS, Arch string
}

func make_tmpl_handler(tn string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		d, err := Asset(fmt.Sprintf("tmpl/%s.html", tn))
		if err != nil {
			panic(err)
		}
		t, err := template.New(tn).Parse(string(d))
		if err != nil {
			panic(err)
		}
		t.Execute(w, p)
	}
}

var (
	Version string
	p       PP
)

type Config struct {
	Install location.SetupLocation
	Version string
	Player  string
}

var conf Config

var (
	HeroClass = map[string]string{
		"HERO_01":  "Warrior",
		"HERO_01a": "Warrior",
		"HERO_02":  "Shaman",
		"HERO_03":  "Rogue",
		"HERO_04":  "Paladin",
		"HERO_05":  "Hunter",
		"HERO_05a": "Hunter",
		"HERO_06":  "Druid",
		"HERO_07":  "Warlock",
		"HERO_08":  "Mage",
		"HERO_08a": "Mage",
		"HERO_09":  "Priest",
	}
	HeroName = map[string]string{
		"HERO_01":  "Garrosh Hellscream",
		"HERO_01a": "Magni Bronzebeard",
		"HERO_02":  "Thrall",
		"HERO_03":  "Valeera Sanguinar",
		"HERO_04":  "Uther Lightbringer",
		"HERO_05":  "Rexxar",
		"HERO_05a": "Alleria Windrunner",
		"HERO_06":  "Malfurion Stormrage",
		"HERO_07":  "Gul'dan",
		"HERO_08":  "Jaina Proudmoore",
		"HERO_08a": "Medivh",
		"HERO_09":  "Anduin Wrynn",
	}
)

var (
	local_conf = "config.json"

	gameTypeMap = map[string]string{
		"ADVENTURE":    "Adventure",
		"ARENA":        "Arena",
		"DRAFT":        "Arena",
		"FRIENDLY":     "Friendly",
		"TAVERN_BRAWL": "Tavern Brawl",
		"TOURNAMENT":   "Casual",
		"RANKED":       "Ranked",
	}
)

func main() {
	root, err := osext.ExecutableFolder()
	if err != nil {
		panic(err)
	}

	local_conf = filepath.Join(root, local_conf)
	cf, err := os.Open(local_conf)

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
	p.Version = Version
	p.Player = conf.Player

	if Version == "" {
		p.Version = "testing"
	}

	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		panic(err)
	}
	_, p.Port, _ = net.SplitHostPort(listener.Addr().String())

	s := graceful.NewServer(&http.Server{
		Addr:    listener.Addr().String(),
		Handler: nil,
	})

	http.HandleFunc("/", make_tmpl_handler("index"))
	http.Handle("/logs", websocket.Handler(logServer(conf.Install.LogFolder)))

	fmt.Printf("Listening on: %s\n", listener.Addr())

	go func() {
		var err error
		var cmd *exec.Cmd
		url := fmt.Sprintf("http://localhost:%s/", p.Port)
		if runtime.GOOS == "darwin" {
			cmd = exec.Command("open", url)
		} else {
			cmd = exec.Command("cmd", "/c", "start", url)
		}
		if err = cmd.Run(); err != nil {
			fmt.Println(err)
		}
	}()

	if err = s.Serve(listener); err != nil {
		panic(err)
	}
}

func writeLocalConfig() {
	cf, err := os.Create(local_conf)
	if err != nil {
		panic(err)
	}
	b, err := json.MarshalIndent(conf, "", "\t")
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(cf, "%s", b)
	if err = cf.Close(); err != nil {
		panic(err)
	}
}
