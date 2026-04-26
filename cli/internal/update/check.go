// Package update checks for newer versions of the Thask CLI on GitHub Releases
// and prints a one-line notification on stderr. Non-blocking, async refresh.
package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	githubAPI     = "https://api.github.com/repos/kimgh06/Thask/releases/latest"
	checkInterval = 24 * time.Hour
	httpTimeout   = 3 * time.Second
)

type cache struct {
	LastCheck     time.Time `json:"last_check"`
	LatestVersion string    `json:"latest_version"`
}

type release struct {
	TagName string `json:"tag_name"`
}

// Check prints a notification if a cached newer version exists, and starts an
// async refresh when the cache is stale (>24h). Returns a cleanup function
// that blocks until the refresh completes (capped by httpTimeout) — defer it
// after the command runs so the cache reliably updates. Skips silently in CI,
// non-TTY, or when THASK_NO_UPDATE_CHECK is set.
func Check(currentVersion string) func() {
	noop := func() {}
	if shouldSkip() {
		return noop
	}

	c := loadCache()

	if isNewer(c.LatestVersion, currentVersion) {
		fmt.Fprintf(os.Stderr,
			"\033[33m🆕 thask %s available\033[0m  (current: %s)\n   brew upgrade thask  ·  npm i -g @thask-org/cli\n\n",
			c.LatestVersion, currentVersion)
	}

	if time.Since(c.LastCheck) < checkInterval {
		return noop
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		refresh(c)
	}()
	return wg.Wait
}

func shouldSkip() bool {
	if os.Getenv("CI") != "" || os.Getenv("THASK_NO_UPDATE_CHECK") != "" {
		return true
	}
	fi, err := os.Stderr.Stat()
	if err != nil || (fi.Mode()&os.ModeCharDevice) == 0 {
		return true
	}
	return false
}

func cachePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".thask", "update-check.json")
}

func loadCache() cache {
	var c cache
	p := cachePath()
	if p == "" {
		return c
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return c
	}
	_ = json.Unmarshal(data, &c)
	return c
}

func saveCache(c cache) {
	p := cachePath()
	if p == "" {
		return
	}
	data, err := json.Marshal(c)
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	_ = os.WriteFile(p, data, 0o644)
}

func refresh(prev cache) {
	prev.LastCheck = time.Now()
	defer saveCache(prev)

	client := http.Client{Timeout: httpTimeout}
	resp, err := client.Get(githubAPI)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	var r release
	if err := json.Unmarshal(body, &r); err != nil {
		return
	}
	prev.LatestVersion = strings.TrimPrefix(r.TagName, "v")
}

func isNewer(latest, current string) bool {
	latest = strings.TrimPrefix(latest, "v")
	current = strings.TrimPrefix(current, "v")
	if latest == "" || current == "" || current == "dev" {
		return false
	}
	return parseSemver(latest) > parseSemver(current)
}

// parseSemver encodes "MAJOR.MINOR.PATCH" into a sortable int64. Each
// component reserves 6 decimal digits, so values up to 999_999 are safe.
// Pre-release suffixes (e.g. "1.2.3-beta") are ignored.
func parseSemver(v string) int64 {
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	var major, minor, patch int64
	fmt.Sscanf(v, "%d.%d.%d", &major, &minor, &patch)
	return major*1_000_000_000_000 + minor*1_000_000 + patch
}
