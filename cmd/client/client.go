package main

import (
	"bufio"
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

	"bitbucket.org/snapbug/hearthreplay-client/common"
	"bitbucket.org/snapbug/hearthreplay-client/linejoin"
	"bitbucket.org/snapbug/hearthreplay-client/regexp"
)

var (
	levelLine = regexp.New(`unloading name=Medal_Ranked_(?P<level>\d+)`)
	typeLine  = `unloading name=RankChangeTwoScoop`

	// these are used to get important information to show the client -- validated by the server
	gameVersion      = regexp.New(`gameVersion = (?P<version>\d+)`)
	screenTransition = regexp.New(`OnSceneLoaded\(\) - prevMode=(?P<prev>\S+) currMode=(?P<curr>\S+)`)
	gameServer       = regexp.New(`GotoGameServer -- address=(?P<ip>.+):(?P<port>\d+), game=(?P<game>\d+), client=(?P<client>\d+), spectateKey=(?P<key>.+)`)

	// these are used to show the client something -- unvalidated by the server, but extracted from logs
	player      = regexp.New(`TAG_CHANGE Entity=(?P<name>.+) tag=PLAYSTATE value=PLAYING`)
	first       = regexp.New(`TAG_CHANGE Entity=(?P<name>.+) tag=FIRST_PLAYER value=1`)
	hero_entity = regexp.New(`TAG_CHANGE Entity=(?P<name>.+) tag=HERO_ENTITY value=(?P<hid>\d+)`)
	winner      = regexp.New(`TAG_CHANGE Entity=(?P<name>.+) tag=PLAYSTATE value=WON`)
	id          = regexp.New(`TAG_CHANGE Entity=(?P<name>.+) tag=PLAYER_ID value=(?P<id>\d+)`)
	hero        = regexp.New(`tag=HERO_ENTITY value=(?P<id>\d+)`)
	create      = regexp.New(` - FULL_ENTITY - Creating ID=(?P<id>\d+) CardID=(?P<cid>.*)`)
	local       = regexp.New(`SendChoices\(\) - id=(?P<id>\d+) ChoiceType=MULLIGAN`)
)

// Hold some basic information of a player in the game
type Player struct {
	Name   string
	Winner bool
	First  bool
	ID     string `json:"-"`
	Hero   string
}

// For debugging, when we print a player, do some nice formatting
func (p Player) String() string {
	w := "✔ "
	if !p.Winner {
		w = "✘"
	}
	return fmt.Sprintf("%s%s (%s)", w, p.Name, p.Hero)
}

// What it is that is being uploaded to the server to parse
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

	// The status of the upload
	Status string
	Reason string

	// These are used to identify some simple information to present
	p1        Player
	p2        Player
	local     string
	heros     []string
	heros_cid [2]string
	spectate  bool

	data bytes.Buffer
}

// For debugging, print a log will show the basic information in line
func (l Log) String() string {
	return fmt.Sprintf("%s vs %s %s(%s) %s/%s (%s->%s)", l.p1, l.p2, l.DisplayType, l.Type, l.Uploader, l.Key, l.Start, l.Finish)
}

// By default logs have an uploading status -- this gets updated in code
func NewLog() Log {
	return Log{
		Status: "Uploading",
		Playrs: make(map[string]Player),
	}
}

const (
	upload_url = "https://hearthreplay.com"
)

// Send the log to the websocket using JSON encoding, put in a function simply
// because it is called from multiple places -- wherever a status is updated.
func send(ws *websocket.Conn, l Log) {
	if err := websocket.JSON.Send(ws, l); err != nil {
		fmt.Println(err)
	}
}

// Upload the log l, inform the websocket ws, and using the waitgroup wg
func upload(l Log, ws *websocket.Conn, wg *sync.WaitGroup) {
	defer wg.Done()

	path := fmt.Sprintf("%s/games/%s/%s/", upload_url, l.Uploader, l.Key)

	// first check that the game hasn't already been uploaded
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

	// now we're sending the log, we have to compress the log lines, and
	// gob encode it -- for simplicity
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

	// use v1 of the upload spec -- this says what information is included in the log
	path = fmt.Sprintf("%sv1", path)
	resp, err = http.Post(path, "appliation/octet-stream", &x)

	// tell the user what happened to the upload
	if err != nil {
		l.Status = "Failed"
		l.Reason = fmt.Sprintf("Error contacting server: %s", err)
	} else if resp.StatusCode != http.StatusAccepted {
		b, _ := ioutil.ReadAll(resp.Body)
		l.Status = "Failed"
		l.Reason = fmt.Sprintf("Server returned: %d (%s)", resp.StatusCode, b)
	} else {
		l.Status = "Success"
		l.Reason = fmt.Sprintf("View at: %s/games/%s/%s/", upload_url, l.Uploader, l.Key)
	}
	send(ws, l)
}

var (
	debug string
)

// given a log folder, get the Logs objects from those logs
func getLogs(logfolder string) chan Log {
	x := make(chan Log)

	fmt.Printf("%s:\n", logfolder)
	filenames := make([]string, 0)

	for _, f := range common.LogFiles {
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
	if len(filenames) != len(common.LogFiles) {
		fmt.Printf("%#v\n%#v\n", filenames, common.LogFiles)
		panic("Did not get all log files!")
	}

	go func(filenames []string) {
		// Sending a log does some finagling with the data that we've gathered
		// and then puts it on the channel ready to be consumed
		send_log := func(log Log, x chan Log) {
			fmt.Printf("Sending: %s\n", log)
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
			} else {
				log.DisplayType = log.Type
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

		var versionLine linejoin.FileAndLine
		var gameTypeLine linejoin.FileAndLine
		var subtypeLine linejoin.FileAndLine
		var ranklevel linejoin.FileAndLine
		var networkLine linejoin.FileAndLine
		otherLines := make([]linejoin.FileAndLine, 0)

		log := NewLog()

		for line := range linejoin.NewJoiner(filenames) {
			if gameVersion.MatchString(line.Text) {
				versionLine = line
				p := gameVersion.NamedMatches(line.Text)
				log.Version = p["version"]
			} else if screenTransition.MatchString(line.Text) {
				trans := screenTransition.NamedMatches(line.Text)

				// when we screen transition _from_ gameplay, we've finished a game
				if trans["prev"] == "GAMEPLAY" {
					// write to debug log -- this will contain the information as uploaded
					// alowing us to see why the server/parser are broken (potentially)
					if debug != "" {
						dbgout, err = os.Create(fmt.Sprintf("logs/%s.%s.%s.log", log.Uploader, log.Key, log.Type))
						if err != nil {
							panic(err)
						}
						fmt.Fprintf(dbgout, "%s\n", versionLine.Text)
						if log.Type == "RANKED" {
							fmt.Fprintf(dbgout, "%s\n", subtypeLine.Text)
						} else {
							fmt.Fprintf(dbgout, "%s\n", gameTypeLine.Text)
						}
						fmt.Fprintf(dbgout, "%s\n", networkLine.Text)
						fmt.Fprintf(dbgout, "%s\n", ranklevel.Text)
						for _, l := range otherLines {
							fmt.Fprintf(dbgout, "%s\n", l.Text)
						}
					}

					// write to the log data
					log.data.WriteString(fmt.Sprintf("%s\n", versionLine.Text))
					if log.Type == "RANKED" {
						log.data.WriteString(fmt.Sprintf("%s\n", subtypeLine.Text))
					} else {
						log.data.WriteString(fmt.Sprintf("%s\n", gameTypeLine.Text))
					}
					log.data.WriteString(fmt.Sprintf("%s\n", networkLine.Text))
					log.data.WriteString(fmt.Sprintf("%s\n", ranklevel.Text))
					for _, l := range otherLines {
						log.data.WriteString(fmt.Sprintf("%s\n", l.Text))
					}

					// calc the times, and send it
					log.Start = networkLine.Ts
					log.Finish = line.Ts
					log.Duration = log.Finish.Sub(log.Start)
					send_log(log, x)

					// reset tracked lines, log status
					subtypeLine.File = ""
					gameTypeLine.File = ""
					networkLine.File = ""
					ranklevel.File = ""
					otherLines = make([]linejoin.FileAndLine, 0)
					log = NewLog()
				} else {
					gameTypeLine = line
					if subtypeLine.File == "" {
						switch trans["prev"] {
						case "ADVENTURE", "ARENA", "DRAFT", "FRIENDLY", "TAVERN_BRAWL", "TOURNAMENT", "RANKED":
							log.Type = trans["prev"]
						}
					}
				}
			} else if strings.Contains(line.Text, typeLine) {
				log.Type = "RANKED"
				subtypeLine = line
			} else if levelLine.MatchString(line.Text) {
				// might be multiple medals for cross-rank play so get the smallest
				// a game between 4/5 is really a game for level 4
				if ranklevel.File != "" {
					p1 := levelLine.NamedMatches(ranklevel.Text)
					p2 := levelLine.NamedMatches(line.Text)
					l1, _ := strconv.Atoi(p1["level"])
					l2, _ := strconv.Atoi(p2["level"])
					if l2 < l1 {
						ranklevel = line
					}
				} else {
					ranklevel = line
				}
			}

			// In general we only need the lines that are GameState updates
			if strings.Contains(line.Text, "GameState") {
				otherLines = append(otherLines, line)

				// now do a bunch of checking to see what information we can
				// present to the user about the game
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
				} else if hero_entity.MatchString(line.Text) {
					parts := hero_entity.NamedMatches(line.Text)
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

			// generate the id for the game from this line
			if gameServer.MatchString(line.Text) {
				gs := gameServer.NamedMatches(line.Text)
				networkLine = line
				log.Key = fmt.Sprintf("%s-%s", gs["game"], gs["key"])
				log.Uploader = gs["client"]
			}
		}
		close(x)
	}(filenames)
	return x
}

// Handles the websocket requests
func logServer(logFolder string) func(ws *websocket.Conn) {
	return func(ws *websocket.Conn) {
		var wg sync.WaitGroup
		// for each log we get from the folder given
		for log := range getLogs(logFolder) {
			// Don't do anything for spectated games -- the spectator doesn't receive
			// the id of the person they are spectating, nor their own, so there'd be a mix of
			// games under the spectated "user"
			if log.Uploader != "0" {
				// if we don't know who our client is, and the game wasn't spectating
				// then write the config so that we can show a link to their games from
				// the start
				if conf.Player == "" {
					conf.Player = log.Uploader
					common.WriteLocalConfig(conf)
				}
				// do the uploading
				wg.Add(1)
				go upload(log, ws, &wg)
			}
			// Even if it's spectated, we want to update the user on what happened
			send(ws, log)
		}
		// Wait for all logs to finish uploading, close the websocket, and quit
		wg.Wait()
		ws.Close()
		// if it's debug don't quit straight away - otherwise we lose the debug information
		// that's printed out.
		if debug != "" {
			reader := bufio.NewReader(os.Stdin)
			_, _ = reader.ReadString('\n')
		}
		os.Exit(0)
	}
}

type TemplateOptions struct {
	Port     string
	Player   string
	Version  string
	Latest   string
	OS, Arch string
}

var (
	Version string
	Options TemplateOptions
	conf    common.Config

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

// Closure over the template name -- simple function that just renders a template
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
		t.Execute(w, Options)
	}
}

// read the config, read the logs, start a local server to show progress
// to the client, and finally open a webbrowser to that client
func main() {
	local_conf := common.GetLocalConfigFile()
	cf, err := os.Open(local_conf)

	if err != nil {
		cf, err = os.Create(local_conf)
		if err != nil {
			panic(err)
		}
	}

	err = json.NewDecoder(cf).Decode(&conf)
	if err != nil {
		panic(err)
	}

	if Version != "" && Version != conf.Version {
		panic("Version mismatch")
	}
	Options.Version = Version
	Options.Player = conf.Player

	if Version == "" {
		Options.Version = "testing"
	}

	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		panic(err)
	}
	_, Options.Port, _ = net.SplitHostPort(listener.Addr().String())

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
		url := fmt.Sprintf("http://localhost:%s/", Options.Port)
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
