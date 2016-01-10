package linejoin

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type FileAndLine struct {
	Ts   time.Time
	Text string
	File string

	base time.Time
	scn  *bufio.Scanner
	add  bool
}

func (fl *FileAndLine) Update() bool {
	if fl.scn.Scan() {
		fl.Text = fl.scn.Text()

		old_time := fl.Ts

		lineT, err := time.Parse("15:04:05.0000000", strings.Split(fl.Text, " ")[1])

		if err != nil {
			panic(fmt.Sprintf("error: %s\n", err))
		}

		fl.Ts = time.Date(
			fl.base.Year(),
			fl.base.Month(),
			fl.base.Day(),
			lineT.Hour(),
			lineT.Minute(),
			lineT.Second(),
			lineT.Nanosecond(),
			time.Now().Location(),
		)

		// There was an issue with an NTP sync that caused a back shift
		// of a few seconds so check that it's 11pm -> 12am
		if fl.Ts.Before(old_time) && fl.Ts.Hour() == 0 && old_time.Hour() == 23 {
			fl.base = fl.base.Add(time.Duration(24) * time.Hour)
			fl.Ts = fl.Ts.Add(time.Duration(24) * time.Hour)
		}

		return true
	}
	return false
}

type fileandlines []*FileAndLine

func (fl fileandlines) Len() int           { return len(fl) }
func (fl fileandlines) Swap(i, j int)      { fl[i], fl[j] = fl[j], fl[i] }
func (fl fileandlines) Less(i, j int) bool { return fl[i].Ts.Before(fl[j].Ts) }

func NewJoiner(filenames []string) chan FileAndLine {
	x := make(chan FileAndLine)

	go func() {
		logsandlines := make(fileandlines, 0)
		for _, fn := range filenames {
			f, err := os.Open(fn)
			if err != nil {
				panic(err)
			} else {
				fi, err := f.Stat()
				if err != nil {
					panic(err)
				}
				fandl := &FileAndLine{scn: bufio.NewScanner(f), File: fn, base: fi.ModTime()}
				if fandl.Update() {
					logsandlines = append(logsandlines, fandl)
				}
			}
		}

		for logsandlines.Len() > 0 {
			sort.Sort(logsandlines)
			x <- FileAndLine{Ts: logsandlines[0].Ts, Text: logsandlines[0].Text, File: filepath.Base(logsandlines[0].File)}
			if !logsandlines[0].Update() {
				logsandlines = logsandlines[1:]
			}
		}

		close(x)
	}()

	return x
}
