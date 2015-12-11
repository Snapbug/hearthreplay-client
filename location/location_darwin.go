package location

import (
	"bytes"
	"os/exec"
	"os/user"
	"path/filepath"

	"github.com/DHowett/go-plist"
)

func Location() (loc SetupLocation, err error) {
	u, err := user.Current()
	if err != nil {
		return
	}
	loc.Config = filepath.Join(u.HomeDir, "Library", "Preferences", "Blizzard", "Hearthstone", "log.config")

	cmd := exec.Command("system_profiler", "-xml", "SPApplicationsDataType")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	if err := cmd.Start(); err != nil {
		panic(err)
	}
	var b []byte
	buf := bytes.NewBuffer(b)
	_, err = buf.ReadFrom(stdout)

	p := make([]interface{}, 0)
	err = plist.NewDecoder(bytes.NewReader(buf.Bytes())).Decode(&p)
	if err != nil {
		panic(err)
	}
	result, ok := p[0].(map[string]interface{})
	if !ok {
		panic("type conversion")
	}
	for _, iitem := range result["_items"].([]interface{}) {
		item, ok := iitem.(map[string]interface{})
		if !ok {
			panic("inner type conversion")
		}
		if item["_name"] == "Hearthstone" {
			p, ok := item["path"].(string)
			if !ok {
				panic("path conversion")
			}
			loc.LogFolder = filepath.Join(p, "..", "Logs")
		}
	}
	if loc.LogFolder == "" {
		panic("Unable to determine log folder")
	}

	return
}
