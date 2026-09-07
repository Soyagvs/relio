package statchart

import (
	"strings"
	"testing"
)

func TestNiceCeil(t *testing.T) {
	cases := map[int]int{0: 5, 3: 5, 5: 5, 6: 10, 10: 10, 11: 20, 128: 200, 1500: 2000, 4001: 5000}
	for in, want := range cases {
		if got := niceCeil(in); got != want {
			t.Errorf("niceCeil(%d) = %d, want %d", in, got, want)
		}
	}
}

func TestHumanInt(t *testing.T) {
	cases := map[int]string{0: "0", 42: "42", 999: "999", 1000: "1,000", 12345: "12,345", 1234567: "1,234,567"}
	for in, want := range cases {
		if got := humanInt(in); got != want {
			t.Errorf("humanInt(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestShortDate(t *testing.T) {
	if got := shortDate("2026-09-06"); got != "Sep 6" {
		t.Errorf("shortDate = %q, want \"Sep 6\"", got)
	}
	if got := shortDate("garbage"); got != "garbage" {
		t.Errorf("shortDate passthrough = %q", got)
	}
}

func TestSVGEmpty(t *testing.T) {
	out := string(SVG(Options{Label: "Relio downloads"}))
	if !strings.HasPrefix(out, "<svg") || !strings.Contains(out, "</svg>") {
		t.Fatalf("not a complete svg:\n%s", out)
	}
	if !strings.Contains(out, "collecting data") {
		t.Error("empty chart should say it is collecting data")
	}
	if !strings.Contains(out, `width="100%"`) {
		t.Error("svg must be fluid width for mobile")
	}
}

func TestSVGSinglePoint(t *testing.T) {
	out := string(SVG(Options{Points: []Point{{"2026-09-06", 128}}}))
	if !strings.Contains(out, "<circle") {
		t.Error("a lone point should render as a dot, not a line")
	}
	if strings.Contains(out, "<polyline") {
		t.Error("no polyline for a single point")
	}
}

func TestSVGSeries(t *testing.T) {
	pts := []Point{
		{"2026-09-06", 10}, {"2026-09-07", 40}, {"2026-09-08", 55},
		{"2026-09-09", 90}, {"2026-09-10", 128},
	}
	out := string(SVG(Options{Label: "Relio downloads", Points: pts, Line: "#F5872B"}))

	for _, want := range []string{"<polyline", "#F5872B", "Sep 6", "Sep 10", "relio", "viewBox=\"0 0 880 280\""} {
		if !strings.Contains(out, want) {
			t.Errorf("series svg missing %q", want)
		}
	}
	// top gridline should be the nice ceiling of 128 -> 200
	if !strings.Contains(out, ">200<") {
		t.Error("y axis should top out at the nice ceiling 200")
	}
	// 5+ points -> a middle x label appears
	if strings.Count(out, `text-anchor="middle"`) < 1 {
		t.Error("expected a middle x-axis label for a 5-point series")
	}
}

func TestSVGEscapes(t *testing.T) {
	out := string(SVG(Options{Label: `a<b>&"c`, Points: []Point{{"2026-09-06", 1}}}))
	if strings.Contains(out, "<b>") || !strings.Contains(out, "&lt;b&gt;") {
		t.Errorf("label not escaped:\n%s", out)
	}
}
