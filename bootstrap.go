package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"bitbucket.org/snapbug/hsr/client/location"
	"github.com/sasbury/mini"
)

type Config struct {
	Install location.SetupLocation
	Verion  string
}

var conf Config

func header(h string) {
	fmt.Println("")
	fmt.Println(h)
	fmt.Println(strings.Repeat("-", len(h)))
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

func checkLocalConfig() {
	header("Checking HSR client config")
	cf, err := os.Open(local_conf)

	if os.IsNotExist(err) {
		fmt.Printf("Determining install location:\n")
		if conf.Install, err = location.Location(); err != nil {
			panic(err)
		}
		writeLocalConfig()

		if cf, err = os.Open(local_conf); err != nil {
			panic(err)
		}
	} else if err != nil {
		panic(err)
	} else {
		fmt.Printf("Already determined install location:\n")
	}
	defer cf.Close()

	if err = json.NewDecoder(cf).Decode(&conf); err != nil {
		panic(err)
	}

	fmt.Printf("\tLog folder: %#s\n", conf.Install.LogFolder)
	fmt.Printf("\tHS log config: %#s\n", conf.Install.Config)
}

type HSConfigSection struct {
	LogLevel        int64
	FilePrinting    bool
	ConsolePrinting bool
	ScreenPrinting  bool
}

func (h HSConfigSection) String() string {
	return fmt.Sprintf("LogLevel=%d\nFilePrinting=%v\nConsolePrinting=%v\nScreenPrinting=%v\n", h.LogLevel, h.FilePrinting, h.ConsolePrinting, h.ScreenPrinting)
}

var (
	needed = HSConfigSection{LogLevel: 1, FilePrinting: false, ConsolePrinting: true, ScreenPrinting: false}
	hsConf = make(map[string]HSConfigSection)
)

func checkHSConfig() {
	header("Setting up HS logging")

	hslog, err := os.Open(conf.Install.Config)
	if os.IsNotExist(err) {
		fmt.Printf("HS log configuration did not exist, creating.")
		for _, section := range []string{"Power", "Net", "LoadingScreen", "UpdateManager"} {
			hsConf[section] = needed
		}
	} else if err != nil {
		panic(err)
	} else {
		cf, _ := mini.LoadConfigurationFromReader(hslog)
		for _, section := range cf.SectionNames() {
			sec := HSConfigSection{}
			_ = cf.DataFromSection(section, &sec)
			hsConf[section] = sec
		}

		for _, section := range []string{"Power", "Net", "LoadingScreen", "UpdateManager"} {
			sec := HSConfigSection{}
			ok := cf.DataFromSection(section, &sec)
			if !ok {
				fmt.Printf("%s section missing\n", section)
				hsConf[section] = needed
			} else {
				if sec != needed {
					fmt.Printf("%s section doesn't match expected -- being overwritten\n", section)
					hsConf[section] = needed
				} else {
					fmt.Printf("%s section ok\n", section)
				}
			}
		}
		hslog.Close()
	}

	sections := make([]string, 0, len(hsConf))
	for k := range hsConf {
		sections = append(sections, k)
	}
	sort.Strings(sections)

	if hslog, err = os.Create(conf.Install.Config); err != nil {
		panic(err)
	}
	for _, k := range sections {
		fmt.Fprintf(hslog, "[%s]\n%s\n", k, hsConf[k])
	}
	hslog.Close()
}

const (
	update_url = "https://hearthreplay.com/v"
	local_conf = "config.json"
)

func checkLatest() {
	var m struct {
		Version string `json:"version"`
	}

	header("Checking version of client")

	resp, err := http.Get(update_url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("%#v", err)
		return
	}
	err = json.Unmarshal(body, &m)

	// if updater != nil {
	// 	err := updater.BackgroundRun()
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// }
}

func main() {
	fmt.Println("======================================")
	fmt.Println("Hearthstone Replay Client Bootstrapper")
	fmt.Println("======================================")

	checkLocalConfig()
	checkHSConfig()
	checkLatest()
}
