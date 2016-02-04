package common

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kardianos/osext"

	"bitbucket.org/snapbug/hearthreplay-client/location"
)

// What configuration information we need
type Config struct {
	Install location.SetupLocation
	Version string
	Player  string
}

var (
	// Asset - Determine if the game was ranked/casual
	// Bob -
	// Net - Get the client number, spectate key and game number -- used to identify games
	// Power - The plays of the game
	// LoadingScreen - Game type (Adventure, Tavern Brawl, etc.)
	// UpdateManager - The version of the game
	LogFiles = []string{"Asset", "Bob", "Net", "Power", "LoadingScreen", "UpdateManager"}
)

// Get the local config file -- accounts for .app bundled procs as well
func GetLocalConfigFile() string {
	folder, err := osext.ExecutableFolder()
	if err != nil {
		panic(err)
	}
	return filepath.Join(folder, "config.json")
}

// Write the specified config to the config file
func WriteLocalConfig(conf Config) bool {
	cf, err := os.Create(GetLocalConfigFile())
	if err != nil {
		fmt.Printf("%#v", err)
		return false
	}
	b, err := json.MarshalIndent(conf, "", "\t")
	if err != nil {
		fmt.Printf("%#v", err)
		return false
	}
	fmt.Fprintf(cf, "%s", b)
	if err = cf.Close(); err != nil {
		fmt.Printf("%#v", err)
		return false
	}
	return true
}
