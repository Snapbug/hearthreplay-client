package regexp

import (
	"regexp"
)

// wrapper for regexp to get dict out
type Rx struct {
	*regexp.Regexp
}

func New(r string) Rx {
	return Rx{regexp.MustCompile(r)}
}

// get named group matches as a dict
func (re Rx) NamedMatches(s string) map[string]string {
	m := re.FindStringSubmatch(s)
	r := make(map[string]string)
	for i, n := range re.SubexpNames() {
		r[n] = m[i]
	}
	return r
}
