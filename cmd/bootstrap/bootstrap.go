package main

import (
	"bufio"
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

	"bitbucket.org/snapbug/hearthreplay-client/common"
	"bitbucket.org/snapbug/hearthreplay-client/location"

	"github.com/cheggaaa/pb"
	"github.com/inconshreveable/go-update"
	"github.com/kardianos/osext"
	"github.com/sasbury/mini"
)

// Helper, to print a header section
func header(h string) {
	fmt.Println("")
	fmt.Println(h)
	fmt.Println(strings.Repeat("-", len(h)))
}

// Load the config, and get everything setup correctly
//  - Determine install locations
//  - Determine log locations
//  - Check that the log configuration for HS is setup
func checkLocalConfig() bool {
	header("Checking HSR client config")
	cf, err := os.Open(common.GetLocalConfigFile())

	suffix := "exe"
	if runtime.GOOS == "darwin" {
		suffix = "app"
	}
	suffix = fmt.Sprintf("Hearthstone.%s", suffix)

	// Here, the config file did not exist, so construct all the data
	// that we need, and write it
	if os.IsNotExist(err) {
		fmt.Printf("Determining install location:\n")
		if conf.Install, err = location.Location(); err != nil {
			fmt.Printf("Could not determine location automatically\n")
			reader := bufio.NewScanner(os.Stdin)
			fmt.Println("Please enter it: ")
			for reader.Scan() {
				path := filepath.Clean(reader.Text())
				if !strings.HasSuffix(path, suffix) {
					path = filepath.Join(path, suffix)
				}
				// Check that Hearthstone exists in the directory they specified
				_, err = os.Stat(path)
				if err != nil {
					fmt.Printf("Invalid location, tried %s\n", filepath.Dir(path))
					fmt.Println("Please enter it again:")
				} else {
					conf.Install.LogFolder = filepath.Dir(path)
					break
				}
			}
		}
		// write the config out, and reload it
		common.WriteLocalConfig(conf)
		if cf, err = os.Open(common.GetLocalConfigFile()); err != nil {
			fmt.Printf("%#v", err)
			return false
		}
	} else if err != nil {
		fmt.Printf("%#v", err)
		return false
	} else {
		fmt.Printf("Already determined install location:\n")
	}
	defer cf.Close()

	// Attempt to load the config
	if err = json.NewDecoder(cf).Decode(&conf); err != nil {
		fmt.Printf("%#v", err)
		return false
	}

	// Debug/useful information -- so if it's wrong they can change it
	fmt.Printf("\tLog folder: %#s\n", conf.Install.LogFolder)
	fmt.Printf("\tHS log config: %#s\n", conf.Install.Config)
	return true
}

// A hearthstone log configuration option looks like this
type HSConfigSection struct {
	LogLevel        int64
	FilePrinting    bool
	ConsolePrinting bool
	ScreenPrinting  bool
}

// For debugging, and writing a configuration section to file
func (h HSConfigSection) String() string {
	return fmt.Sprintf(`LogLevel=%d
	FilePrinting=%v
	ConsolePrinting=%v
	ScreenPrinting=%v`,
		h.LogLevel,
		h.FilePrinting,
		h.ConsolePrinting,
		h.ScreenPrinting,
	)
}

// Check that the logs for hearthstone are setup correctly -- that log.config exists
// and is logging the files that we want to parse
func checkHSConfig() (ok bool) {
	header("Setting up HS logging")
	ok = true

	hslog, err := os.Open(conf.Install.Config)
	if os.IsNotExist(err) {
		fmt.Printf("HS log configuration did not exist, creating.\n")
		for _, section := range common.LogFiles {
			hsConf[section] = needed
		}
		ok = false
	} else if err != nil {
		fmt.Printf("%#v", err)
		return false
	} else {
		// Ok, so some logging has been setup before

		// load the previously configured sections
		cf, _ := mini.LoadConfigurationFromReader(hslog)
		for _, section := range cf.SectionNames() {
			sec := HSConfigSection{}
			cf.DataFromSection(section, &sec)
			hsConf[section] = sec
		}

		// now set the bits we want
		for _, section := range common.LogFiles {
			sec := HSConfigSection{}
			k := cf.DataFromSection(section, &sec)
			// the bit we wanted isn't here
			if !k {
				fmt.Printf("%s section missing\n", section)
				hsConf[section] = needed
				ok = false
			} else {
				// it's here, make sure that FilePrinting and LogLevel are set
				if !sec.FilePrinting && sec.LogLevel != 1 {
					// warn the user we're changing the config, so they can check it
					fmt.Printf("%s section doesn't match expected -- being overwritten\n", section)
					sec.FilePrinting = true
					sec.LogLevel = 1
					hsConf[section] = sec
					ok = false
				} else {
					fmt.Printf("%s section ok\n", section)
				}
			}
		}
		hslog.Close()
	}

	// For niceness, sort the config file sections
	sections := make([]string, 0, len(hsConf))
	for k := range hsConf {
		sections = append(sections, k)
	}
	sort.Strings(sections)

	// Because we read it all, or constructed it, we can nuke the old logging config here
	if hslog, err = os.Create(conf.Install.Config); err != nil {
		fmt.Printf("%#v", err)
		return false
	}
	// print out the sections in sorted order, :)
	for _, k := range sections {
		fmt.Fprintf(hslog, "[%s]\n%s\n", k, hsConf[k])
	}
	hslog.Close()
	return
}

// What the server tells us about the version of the client that is current
type VersionUpdate struct {
	Version   string `json:"version"`
	Checksum  string `json:"checksum"`
	Signature string `json:"signature"`
}

// Perform a verified (signed/checksum) update for the client program
func verifiedUpdate(binary io.Reader, givenUpdate VersionUpdate) (err error) {
	f, err := os.Open(client_prog)
	if err != nil {
		if os.IsNotExist(err) {
			f, err = os.Create(client_prog)
			if err != nil {
				fmt.Printf("%#v", err)
				return err
			}
		} else {
			fmt.Printf("%#v", err)
			return err
		}
	}
	// On macos we have to set the executable bit, on windows it's enough
	// to call the program .exe
	if runtime.GOOS == "darwin" {
		stat, err := f.Stat()
		if err != nil {
			fmt.Printf("%#v", err)
			return err
		}
		current := stat.Mode()
		current |= 0111 // set executable bits
		if err := f.Chmod(current); err != nil {
			fmt.Printf("%#v", err)
			return err
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
	// do the update!
	err = update.Apply(binary, opts)
	if err != nil {
		return
	}
	return
}

// The last check the bootstrapper does is check that the version of the client that's
// installed is the latest one
func checkLatest() bool {
	var m VersionUpdate
	header("Checking version of client")

	resp, err := http.Get(update_url)
	if err != nil {
		fmt.Printf("%#v", err)
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Server returned bad status: %s\n", resp.Status)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("%#v", err)
		return false
	}
	err = json.Unmarshal(body, &m)

	if conf.Version == m.Version {
		fmt.Printf("%s is the latest version!\n", conf.Version)
	} else {
		fmt.Printf("Need to download new version: %s\n", m.Version)
		url := fmt.Sprintf("https://s3-us-west-2.amazonaws.com/update.hearthreplay.com/hearthreplay-client-%s-%s-%s", m.Version, runtime.GOOS, runtime.GOARCH)

		resp, err = http.Get(url)

		if err != nil {
			fmt.Printf("%#v", err)
			return false
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			fmt.Printf("Update server returned bad status: %s\n", resp.Status)
			return false
		}

		i, _ := strconv.Atoi(resp.Header.Get("Content-Length"))

		bar := pb.New(i).SetUnits(pb.U_BYTES).SetRefreshRate(time.Millisecond * 100)
		bar.ShowSpeed = true
		bar.Start()

		var buf bytes.Buffer
		writer := io.MultiWriter(&buf, bar)
		io.Copy(writer, resp.Body)
		err := verifiedUpdate(&buf, m)
		if err != nil {
			fmt.Printf("%#v", err)
			return false
		}
		bar.Finish()

		// if we've updated successfully, update the configuration file to reflect that
		conf.Version = m.Version
		if ok := common.WriteLocalConfig(conf); !ok {
			return false
		}
	}
	return true
}

var (
	conf common.Config

	hsConf     = make(map[string]HSConfigSection)
	needed     = HSConfigSection{LogLevel: 1, FilePrinting: true, ConsolePrinting: false, ScreenPrinting: false}
	update_url = fmt.Sprintf("https://hearthreplay.com/version?os=%s&arch=%s", runtime.GOOS, runtime.GOARCH)
)

// Used to check the signature of the executable
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
)

// Does three things:
// - checks the local config is setup
// - checks that hearthstone configuration is setup
// - checks that the version of the client is the latest
//
// After that, it launches the client!
func main() {
	folder, err := osext.ExecutableFolder()
	if err != nil {
		panic(err)
	}

	// the macos version does some weirdness because it's launched
	// from an apple script -- we get the path with '' around it
	client_prog = strings.Trim(client_prog, "'")
	client_prog = filepath.Join(folder, client_prog)
	if runtime.GOOS == "windows" {
		client_prog = fmt.Sprintf("%s.exe", client_prog)
	}

	fmt.Println("==================================")
	fmt.Println("Hearthstone Replay Client Launcher")
	fmt.Println("==================================")

	ok := checkLocalConfig() && checkHSConfig() && checkLatest()

	// Something went wrong -- so generic error message (arguably
	// it'd be better to say _what_ but that should come from
	// within those methods)
	if !ok {
		fmt.Println()
		fmt.Println("==================================")
		fmt.Println("Logging setup for future sessions.")
		fmt.Println("No games to be uploaded this time.")
		fmt.Println("==================================")
		fmt.Println()
	} else {
		// launch the client!
		if err := exec.Command(client_prog).Start(); err != nil {
			fmt.Println(err)
		}
	}

	// don't quit out immediately -- let the user see whatever errors there may be
	fmt.Println("Press any key to quit.")
	reader := bufio.NewReader(os.Stdin)
	_, _ = reader.ReadString('\n')
}
