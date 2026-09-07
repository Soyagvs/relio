// Package statchart renders a small, dependency-free line chart as an SVG
// string. It is used to draw Relio's download- and star-history graphs from the
// real snapshots stored in .github/stats/downloads.json — no third-party
// services, no client-side script.
//
// The output is deliberately plain: a transparent background (so it sits well on
// GitHub in both light and dark themes), one coloured line with a faint fill, a
// couple of gridlines, and monospace axis labels.
package statchart

import (
	"fmt"
	"strings"
	"time"
)

// Point is one dated measurement.
type Point struct {
	Date  string // "2006-01-02"
	Value int
}

// Options controls a single chart.
type Options struct {
	Label  string  // small caption drawn top-left; "" to omit
	Points []Point // in chronological order
	Line   string  // stroke colour, e.g. "#F5872B"
}

const (
	width      = 880
	height     = 280
	padL       = 56
	padR       = 24
	padT       = 28
	padB       = 34
	axisColor  = "#8b8b8b"
	gridColor  = "#8b8b8b"
	fontFamily = "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace"
)

// SVG returns a complete <svg> document for the given options.
func SVG(o Options) []byte {
	if o.Line == "" {
		o.Line = "#F5872B"
	}
	plotW := width - padL - padR
	plotH := height - padT - padB
	x0, y1 := padL, height-padB

	var b strings.Builder
	fmt.Fprintf(&b,
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="100%%" `+
			`preserveAspectRatio="xMidYMid meet" role="img" aria-label="%s">`+"\n",
		width, height, esc(ariaLabel(o)))
	b.WriteString(`<style>text{font-family:` + fontFamily + `;font-size:11px;fill:` + axisColor + `}</style>` + "\n")

	if o.Label != "" {
		fmt.Fprintf(&b, `<text x="%d" y="18" font-size="12">%s</text>`+"\n", padL, esc(o.Label))
	}

	if len(o.Points) == 0 {
		fmt.Fprintf(&b, `<text x="%d" y="%d" text-anchor="middle" fill="%s">collecting data…</text>`+"\n",
			width/2, height/2, axisColor)
		b.WriteString("</svg>\n")
		return []byte(b.String())
	}

	maxV := 1
	for _, p := range o.Points {
		if p.Value > maxV {
			maxV = p.Value
		}
	}
	top := niceCeil(maxV)

	// Horizontal gridlines + y labels at 0, mid, top.
	for _, frac := range []float64{0, 0.5, 1} {
		v := int(float64(top) * frac)
		y := y1 - int(float64(plotH)*frac)
		fmt.Fprintf(&b, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-opacity="0.18"/>`+"\n",
			x0, y, x0+plotW, y, gridColor)
		fmt.Fprintf(&b, `<text x="%d" y="%d" text-anchor="end">%s</text>`+"\n",
			x0-8, y+4, humanInt(v))
	}

	// Line + area.
	xAt := func(i int) int {
		if len(o.Points) == 1 {
			return x0 + plotW/2
		}
		return x0 + int(float64(plotW)*float64(i)/float64(len(o.Points)-1))
	}
	yAt := func(v int) int { return y1 - int(float64(plotH)*float64(v)/float64(top)) }

	pts := make([]string, len(o.Points))
	for i, p := range o.Points {
		pts[i] = fmt.Sprintf("%d,%d", xAt(i), yAt(p.Value))
	}
	if len(o.Points) == 1 {
		fmt.Fprintf(&b, `<circle cx="%d" cy="%d" r="3" fill="%s"/>`+"\n",
			xAt(0), yAt(o.Points[0].Value), o.Line)
	} else {
		fmt.Fprintf(&b, `<path d="M %d,%d L %s L %d,%d Z" fill="%s" fill-opacity="0.08"/>`+"\n",
			xAt(0), y1, strings.Join(pts, " "), xAt(len(pts)-1), y1, o.Line)
		fmt.Fprintf(&b, `<polyline points="%s" fill="none" stroke="%s" stroke-width="2" `+
			`stroke-linejoin="round" stroke-linecap="round"/>`+"\n",
			strings.Join(pts, " "), o.Line)
	}

	// Baseline.
	fmt.Fprintf(&b, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-opacity="0.4"/>`+"\n",
		x0, y1, x0+plotW, y1, axisColor)

	// X labels: first and last (plus middle when there is room).
	label := func(i, x int, anchor string) {
		fmt.Fprintf(&b, `<text x="%d" y="%d" text-anchor="%s">%s</text>`+"\n",
			x, y1+18, anchor, esc(shortDate(o.Points[i].Date)))
	}
	label(0, x0, "start")
	if len(o.Points) >= 5 {
		mid := len(o.Points) / 2
		label(mid, xAt(mid), "middle")
	}
	if len(o.Points) > 1 {
		label(len(o.Points)-1, x0+plotW, "end")
	}

	// Discreet wordmark.
	fmt.Fprintf(&b, `<text x="%d" y="%d" text-anchor="end" fill="%s" fill-opacity="0.55" font-size="10">relio</text>`+"\n",
		x0+plotW, padT-14, o.Line)

	b.WriteString("</svg>\n")
	return []byte(b.String())
}

func ariaLabel(o Options) string {
	name := o.Label
	if name == "" {
		name = "history"
	}
	if len(o.Points) == 0 {
		return name + " — no data yet"
	}
	last := o.Points[len(o.Points)-1]
	return fmt.Sprintf("%s — %s on %s", name, humanInt(last.Value), last.Date)
}

// niceCeil rounds v up to 1/2/5 × 10ⁿ so the top gridline is a round number.
func niceCeil(v int) int {
	if v <= 5 {
		return 5
	}
	mag := 1
	for mag*10 < v {
		mag *= 10
	}
	for _, step := range []int{1, 2, 5, 10} {
		if v <= step*mag {
			return step * mag
		}
	}
	return 10 * mag
}

func humanInt(n int) string {
	s := fmt.Sprintf("%d", n)
	if n < 1000 {
		return s
	}
	var out []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, c)
	}
	return string(out)
}

// shortDate turns "2026-09-06" into "Sep 6"; on parse failure it returns input.
func shortDate(iso string) string {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return iso
	}
	return t.Format("Jan 2")
}

func esc(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}
