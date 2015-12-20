package main

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"bitbucket.org/snapbug/hsr/client/location"

	"github.com/cheggaaa/pb"
	"github.com/inconshreveable/go-update"
	"github.com/kardianos/osext"
	"github.com/sasbury/mini"
)

type Config struct {
	Install location.SetupLocation
	Version string
	Player  string
}

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

func checkHSConfig() (ok bool) {
	header("Setting up HS logging")
	ok = true

	hslog, err := os.Open(conf.Install.Config)
	if os.IsNotExist(err) {
		fmt.Printf("HS log configuration did not exist, creating.\n")
		for _, section := range []string{"Power", "Net", "LoadingScreen", "UpdateManager"} {
			hsConf[section] = needed
		}
		ok = false
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
			k := cf.DataFromSection(section, &sec)
			if !k {
				fmt.Printf("%s section missing\n", section)
				hsConf[section] = needed
				ok = false
			} else {
				if sec != needed {
					fmt.Printf("%s section doesn't match expected -- being overwritten\n", section)
					hsConf[section] = needed
					ok = false
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
	return
}

type VersionUpdate struct {
	Version   string `json:"version"`
	Checksum  string `json:"checksum"`
	Signature string `json:"signature"`
}

func verifiedUpdate(binary io.Reader, givenUpdate VersionUpdate) (err error) {
	f, err := os.Open(client_prog)
	if err != nil {
		if os.IsNotExist(err) {
			f, err = os.Create(client_prog)
			if err != nil {
				panic(err)
			}
		} else {
			panic(err)
		}
	}
	if runtime.GOOS == "darwin" {
		stat, err := f.Stat()
		if err != nil {
			n, _ := err.(*os.PathError)
			fmt.Printf("%#v\n", n)
			panic(err)
		}
		current := stat.Mode()
		current |= 0111 // set executable bits
		if err := f.Chmod(current); err != nil {
			panic(err)
		}
	}
	f.Close()

	checksum, err := hex.DecodeString(givenUpdate.Checksum)
	if err != nil {
		return
	}
	signature, err := base64.StdEncoding.DecodeString(givenUpdate.Signature)
	if err != nil {
		return
	}
	opts := update.Options{
		Checksum:   checksum,
		Signature:  signature,
		Verifier:   update.NewRSAVerifier(),
		Patcher:    nil,
		TargetPath: client_prog,
	}
	err = opts.SetPublicKeyPEM([]byte(publicKey))
	if err != nil {
		return
	}
	err = update.Apply(binary, opts)
	if err != nil {
		return
	}
	return
}

func checkLatest() {
	var m VersionUpdate
	header("Checking version of client")

	resp, err := http.Get(update_url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Server returned bad status: %s\n", resp.Status)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(fmt.Sprintf("%#v", err))
	}
	err = json.Unmarshal(body, &m)

	if conf.Version == m.Version {
		fmt.Printf("%s is the latest version!\n", conf.Version)
	} else {
		fmt.Printf("Need to download new version: %s\n", m.Version)
		url := fmt.Sprintf("https://s3-us-west-2.amazonaws.com/update.hearthreplay.com/hearthreplay-client-%s-%s-%s", runtime.GOOS, runtime.GOARCH, m.Version)

		resp, err = http.Get(url)

		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			fmt.Printf("Update server returned bad status: %s\n", resp.Status)
			return
		}

		i, _ := strconv.Atoi(resp.Header.Get("Content-Length"))

		bar := pb.New(i).SetUnits(pb.U_BYTES).SetRefreshRate(time.Millisecond * 10)
		bar.ShowSpeed = true
		bar.Start()

		var buf bytes.Buffer
		writer := io.MultiWriter(&buf, bar)
		io.Copy(writer, resp.Body)
		err := verifiedUpdate(&buf, m)
		if err != nil {
			panic(err)
		}
		bar.Finish()

		conf.Version = m.Version
		writeLocalConfig()
	}
}

var (
	conf Config

	needed     = HSConfigSection{LogLevel: 1, FilePrinting: true, ConsolePrinting: false, ScreenPrinting: false}
	hsConf     = make(map[string]HSConfigSection)
	update_url = fmt.Sprintf("https://hearthreplay.com/v?os=%s&arch=%s", runtime.GOOS, runtime.GOARCH)
)

const (
	publicKey = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAnFz67+ql0kCILF3Ns/Ua
geKyrD1/SaQlfxSriP4PCErbZMa5HWBpaQxRKU+EGkxIQYzSEJlkCnajhXTLVIzJ
FeFiGZdUnv0HQfAmyC0Gsi4h1wrx3f4dgwc0BO2l9H5ExwWT25kdR7EFbj5rNRMq
4qqD+4yj8BnYJ30TXqTCGW/y4aZnMzs/OJapp1ODItRZJk0YeVplo+JrDRKgvXkt
vErvjEBOKwrXmpdIXRY+OXrMrh8KCOa3T785AtK5IYDJrMdhdvo0xSElGvuj/rZT
S7B0I5EtA88/vmKbLqKz9GH3+XxUhWOWYxy73Rmu5zNKQYvqBcBzAFB0Ud8LPr7M
fQIDAQAB
-----END PUBLIC KEY-----`
)

var (
	client_prog = "hearthreplay-client"
	local_conf  = "config.json"
)

func main() {
	folder, err := osext.ExecutableFolder()
	if err != nil {
		panic(err)
	}
	client_prog = strings.Trim(client_prog, "'")
	client_prog = filepath.Join(folder, client_prog)
	local_conf = filepath.Join(folder, local_conf)

	fmt.Println("==================================")
	fmt.Println("Hearthstone Replay Client Launcher")
	fmt.Println("==================================")

	checkLocalConfig()
	ok := checkHSConfig()
	checkLatest()

	if !ok {
		fmt.Println()
		fmt.Println("==================================")
		fmt.Println("Logging setup for future sessions.")
		fmt.Println("No games to be uploaded this time.")
		fmt.Println("==================================")
		fmt.Println()
	} else {
		if err := exec.Command(client_prog).Start(); err != nil {
			fmt.Println(err)
		}
	}
}
