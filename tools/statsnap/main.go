// Command statsnap records one dated snapshot of Relio's public GitHub
// download and star counts, then regenerates the history SVGs from the whole
// series. It is run by .github/workflows/stats.yml and is not part of the relio
// binary.
//
// What it touches:
//
//	reads   api.github.com/repos/soyagvs/relio  and  /releases  (anonymous GET)
//	writes  .github/stats/downloads.json   one {date,total,stars} row per UTC day
//	writes  assets/download-history.svg
//	writes  assets/star-history.svg
//
// No user data and no third-party services are involved. "downloads" is the sum
// of release-asset download counts for Relio's own distribution archives
// (.tar.gz / .zip), excluding checksums and signatures — it is not a count of
// unique installs.
//
// Usage: statsnap [repo-root]   (defaults to the current directory)
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/soyagvs/relio/internal/ghstats"
	"github.com/soyagvs/relio/internal/statchart"
)

const repo = "soyagvs/relio"

type snapshot struct {
	Date  string `json:"date"`  // YYYY-MM-DD, UTC
	Total int    `json:"total"` // cumulative release-asset downloads
	Stars int    `json:"stars"`
}

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}

	dataPath := filepath.Join(root, ".github", "stats", "downloads.json")
	series, err := load(dataPath)
	if err != nil {
		die(err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	stats, err := ghstats.Fetch(context.Background(), client, repo)
	if err != nil {
		die(err)
	}

	today := time.Now().UTC().Format("2006-01-02")
	series = upsert(series, snapshot{Date: today, Total: stats.TotalDownloads, Stars: stats.Stars})

	changed := false

	wrote, err := saveJSON(dataPath, series)
	if err != nil {
		die(err)
	}
	changed = changed || wrote

	charts := []struct {
		path string
		body []byte
	}{
		{filepath.Join(root, "assets", "download-history.svg"),
			statchart.SVG(statchart.Options{Label: "Relio downloads", Line: "#F5872B", Points: points(series, func(s snapshot) int { return s.Total })})},
		{filepath.Join(root, "assets", "star-history.svg"),
			statchart.SVG(statchart.Options{Label: "Relio stars", Line: "#8b8b8b", Points: points(series, func(s snapshot) int { return s.Stars })})},
	}
	for _, c := range charts {
		wrote, err := writeIfChanged(c.path, c.body)
		if err != nil {
			die(err)
		}
		changed = changed || wrote
	}

	if changed {
		fmt.Printf("stats updated: %s total=%d stars=%d\n", today, stats.TotalDownloads, stats.Stars)
	} else {
		fmt.Println("stats unchanged")
	}
}

func load(path string) ([]snapshot, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var s []snapshot
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return s, nil
}

// upsert replaces the row for p.Date if it exists (last write wins for the same
// day) or appends it, keeping the series sorted by date.
func upsert(series []snapshot, p snapshot) []snapshot {
	for i := range series {
		if series[i].Date == p.Date {
			series[i] = p
			return series
		}
	}
	series = append(series, p)
	sort.Slice(series, func(i, j int) bool { return series[i].Date < series[j].Date })
	return series
}

func points(series []snapshot, pick func(snapshot) int) []statchart.Point {
	out := make([]statchart.Point, len(series))
	for i, s := range series {
		out[i] = statchart.Point{Date: s.Date, Value: pick(s)}
	}
	return out
}

func saveJSON(path string, series []snapshot) (bool, error) {
	b, err := json.MarshalIndent(series, "", "  ")
	if err != nil {
		return false, err
	}
	b = append(b, '\n')
	return writeIfChanged(path, b)
}

// writeIfChanged writes body only when it differs from what is on disk, so the
// workflow commits nothing on a no-op run. It returns whether it wrote.
func writeIfChanged(path string, body []byte) (bool, error) {
	if old, err := os.ReadFile(path); err == nil && string(old) == string(body) {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return false, err
	}
	return true, nil
}

func die(err error) {
	fmt.Fprintln(os.Stderr, "statsnap:", err)
	os.Exit(1)
}
