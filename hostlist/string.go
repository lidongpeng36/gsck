package hostlist

import (
	"math"
	"regexp"
)

var splitRegexp *regexp.Regexp

func init() {
	splitRegexp = regexp.MustCompile(SplitString)
	RegisterHostlist(func(str string) Hostlist {
		return &fromString{str}
	})
}

type fromString struct {
	str string
}

// pragma mark - Hostlist Interface

func (hs *fromString) Name() string {
	return "string"
}

func (hs *fromString) Priority() int {
	return math.MaxInt32
}

// Get splits input string with /\s+|;|,/
func (hs *fromString) Get() (list HostInfoList, err error) {
	stringList := splitRegexp.Split(hs.str, -1)
	list = MakeHostInfoListFromStringList(stringList)
	return
}

// ShouldBreak always return false
func (hs *fromString) ShouldBreak() bool { return false }
