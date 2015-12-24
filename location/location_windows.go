package location

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func Location() (loc SetupLocation, err error) {
	var s string
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\Hearthstone`, registry.QUERY_VALUE)
	if err != nil {
		defer k.Close()
		s, _, err = k.GetStringValue("DisplayIcon")
	}
	if err != nil {
		k, err = registry.OpenKey(registry.CURRENT_USER, `SOFTWARE\Microsoft\IntelliPoint\AppSpecific\Hearthstone.exe`, registry.QUERY_VALUE)
		if err != nil {
			defer k.Close()
			s, _, err = k.GetStringValue("Path")
			if err != nil {
				fmt.Printf("Could not determine location")
			}
		}
	}

	root := strings.TrimSuffix(s, "Hearthstone.exe")

	loc.LogFolder = filepath.Join(root, "Logs")
	loc.Config = filepath.Join(os.ExpandEnv("$LOCALAPPDATA"), "Blizzard", "Hearthstone", "log.config")
	return
}
