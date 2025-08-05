package build

import (
	"fmt"
	"runtime/debug"
	"strconv"
	"time"
)

// Tag returns the truncated git commit used to create the current binary or, if the git workspace was modified (e.g. in
// development) or debug info has been stripped, returns the Unix timestamp.
func Tag() string {
	var (
		revision string
		modified bool
	)

	info, ok := debug.ReadBuildInfo()
	if !ok {
		goto timestamp
	}

	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			modified, _ = strconv.ParseBool(setting.Value)
		}
	}

	if modified || revision == "" {
		goto timestamp
	}

	return revision[:8]

timestamp:
	return fmt.Sprintf("%x", time.Now().Unix())
}
