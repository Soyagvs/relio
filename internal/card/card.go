// Package card renders a shareable PNG "release card" for `relio image`.
//
// Identity: dark charcoal background with a faint dot grid and a subtle warm
// glow, one selectable accent (orange / green / purple), changelog grouped into
// Added / Changed / Fixed with small line icons, and a discreet "generated with
// relio" signature. Every value comes from the real release.
package card

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/goregular"
)

// ---------- public API ----------

// Shape is the aspect ratio of the output image.
type Shape int

const (
	Horizontal Shape = iota // 1200x630  (Twitter / OpenGraph)
	Vertical                // 1080x1920 (Instagram story, 9:16)
	Square                  // 1080x1080
)

func ParseShape(s string) (Shape, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "horizontal", "h", "wide", "landscape":
		return Horizontal, true
	case "vertical", "v", "portrait", "story":
		return Vertical, true
	case "square", "s", "sq":
		return Square, true
	default:
		return Horizontal, false
	}
}

func (s Shape) String() string {
	switch s {
	case Vertical:
		return "vertical"
	case Square:
		return "square"
	default:
		return "horizontal"
	}
}

func (s Shape) size() (int, int) {
	switch s {
	case Vertical:
		return 1080, 1920 // Instagram story
	case Square:
		return 1080, 1080
	default:
		return 1200, 630
	}
}

// GroupKind selects the colour and icon of a changelog section.
type GroupKind int

const (
	KindAdded GroupKind = iota
	KindChanged
	KindFixed
)

// Item is one changelog line.
type Item struct {
	Text string
	Hash string // shown only when Options.ShowHash
}

// Group is a titled section of items.
type Group struct {
	Kind  GroupKind
	Title string
	Items []Item
}

// Card is everything the image shows. All fields come from the real release.
type Card struct {
	Project string
	Version string
	Meta    string // "06.09.26 · 13:47 · 7 commits"
	Groups  []Group
}

// ThemeNames are the accent colours the user can pick, default first.
var ThemeNames = []string{"orange", "green", "purple"}

// Options tune the render.
type Options struct {
	Theme    string // one of ThemeNames; empty = "orange"
	ShowHash bool
}

// Render draws the card.
func Render(c Card, s Shape, opt Options) image.Image {
	return newRenderer(c, s, opt).draw()
}

// Save writes img to path as PNG.
func Save(img image.Image, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// ---------- palette ----------

func hexc(s string) color.NRGBA {
	var r, g, b uint8
	fmt.Sscanf(strings.TrimPrefix(s, "#"), "%02x%02x%02x", &r, &g, &b)
	return color.NRGBA{r, g, b, 0xff}
}

func alpha(c color.NRGBA, a uint8) color.NRGBA { c.A = a; return c }

var (
	colBGTop  = hexc("#0B0B0F")
	colBGBot  = hexc("#070709")
	colBorder = hexc("#20202A")
	colGrid   = color.NRGBA{0xFF, 0xFF, 0xFF, 6}
	colText   = hexc("#F1F1EE")
	colDim    = hexc("#8A8A93")
	colFaint  = hexc("#5B5B64")

	catColor = map[GroupKind]color.NRGBA{
		KindAdded:   hexc("#4ADE80"), // green
		KindChanged: hexc("#F5A524"), // amber
		KindFixed:   hexc("#FF6B6B"), // coral
	}
)

func accentOf(name string) color.NRGBA {
	switch strings.ToLower(name) {
	case "green":
		return hexc("#2DD4BF") // teal-green, distinct from the Added green
	case "purple":
		return hexc("#A855F7")
	default:
		return hexc("#FF7A1A") // intense orange (default)
	}
}

// ---------- fonts ----------

var (
	fSansBold *truetype.Font
	fSans     *truetype.Font
	fMono     *truetype.Font
	fMonoBold *truetype.Font
)

func init() {
	fSansBold, _ = truetype.Parse(gobold.TTF)
	fSans, _ = truetype.Parse(goregular.TTF)
	fMono, _ = truetype.Parse(gomono.TTF)
	fMonoBold, _ = truetype.Parse(gomonobold.TTF)
}

func face(f *truetype.Font, size float64) font.Face {
	return truetype.NewFace(f, &truetype.Options{Size: size, DPI: 72, Hinting: font.HintingFull})
}

// ---------- renderer ----------

type renderer struct {
	c   Card
	s   Shape
	opt Options

	dc     *gg.Context
	W, H   float64
	pad    float64
	u      float64 // base unit (scales with auto-fit)
	accent color.NRGBA

	contentR float64 // right edge available to text
}

func newRenderer(c Card, s Shape, opt Options) *renderer {
	w, h := s.size()
	r := &renderer{c: c, s: s, opt: opt, W: float64(w), H: float64(h), accent: accentOf(opt.Theme)}
	r.dc = gg.NewContext(w, h)

	switch s {
	case Horizontal:
		r.pad = r.H * 0.085
		r.u = r.H / 24
	case Square:
		r.pad = r.W * 0.075
		r.u = r.W / 26
	case Vertical:
		r.pad = r.W * 0.075
		r.u = r.W / 25
	}
	r.contentR = r.W - r.pad
	return r
}

func (r *renderer) draw() image.Image {
	r.background()
	r.fitUnit()
	r.content()
	return r.dc.Image()
}

func (r *renderer) background() {
	dc := r.dc

	g := gg.NewLinearGradient(0, 0, 0, r.H)
	g.AddColorStop(0, colBGTop)
	g.AddColorStop(1, colBGBot)
	dc.SetFillStyle(g)
	dc.DrawRectangle(0, 0, r.W, r.H)
	dc.Fill()

	// Extremely discreet dot grid.
	dc.SetColor(colGrid)
	gap := r.H / 18
	for y := gap; y < r.H; y += gap {
		for x := gap; x < r.W; x += gap {
			dc.DrawCircle(x, y, 1.1)
			dc.Fill()
		}
	}

	// Very subtle warm glow toward the right edge.
	glow := gg.NewRadialGradient(r.W*1.02, r.H*0.42, 0, r.W*1.02, r.H*0.42, r.H*0.95)
	glow.AddColorStop(0, alpha(r.accent, 26))
	glow.AddColorStop(1, alpha(r.accent, 0))
	dc.SetFillStyle(glow)
	dc.DrawRectangle(0, 0, r.W, r.H)
	dc.Fill()

	// Thin rounded border around the whole card.
	dc.SetColor(colBorder)
	dc.SetLineWidth(2)
	m := r.pad * 0.42
	dc.DrawRoundedRectangle(m, m, r.W-2*m, r.H-2*m, r.u*0.9)
	dc.Stroke()
}

// fitUnit scales r.u so the content uses the card height well: it shrinks when
// content overflows (down to a floor — the rest is then truncated with "+N
// more") and grows a little when there is lots of empty space (tall formats).
func (r *renderer) fitUnit() {
	avail := r.H - 2*r.pad
	scale := avail / r.contentHeight(r.u)
	switch {
	case scale < 0.7:
		scale = 0.7
	case scale > 1.22:
		scale = 1.22
	}
	r.u *= scale
}

func (r *renderer) contentHeight(u float64) float64 {
	h := u*2.4 + u*1.5 + u*1.4 // project, meta, divider gap

	itemX := r.pad + u*1.35 + u*0.6
	hashW := 0.0
	if r.opt.ShowHash {
		hashW = u * 4.8
	}
	textMax := r.contentR - (itemX + hashW)
	r.dc.SetFontFace(face(fMono, u*0.95))

	for _, g := range r.c.Groups {
		if len(g.Items) == 0 {
			continue
		}
		h += u * 1.85 // title row
		for _, it := range g.Items {
			n := len(wrapLines(r.dc, it.Text, textMax))
			if n < 1 {
				n = 1
			}
			h += float64(n)*u*1.24 + u*0.1
		}
		h += u * 0.7
	}
	h += u * 1.8 // footer
	return h
}

func (r *renderer) content() {
	dc := r.dc
	x := r.pad
	y := r.pad + r.u*0.6

	// Drift short content toward the vertical centre so tall formats don't look
	// half-empty; tall content stays top-aligned.
	if slack := (r.H - 2*r.pad) - r.contentHeight(r.u); slack > 0 {
		y += slack * 0.42
	}

	// --- header: Project  Version ---
	projFace := face(fSansBold, r.u*2.15)
	dc.SetFontFace(projFace)
	dc.SetColor(colText)
	dc.DrawString(r.c.Project, x, y+r.u*1.6)
	pw, _ := dc.MeasureString(r.c.Project)

	verFace := face(fMonoBold, r.u*1.35)
	dc.SetFontFace(verFace)
	vw, _ := dc.MeasureString(r.c.Version)
	if x+pw+r.u*0.9+vw <= r.contentR {
		dc.SetColor(r.accent)
		dc.DrawString(r.c.Version, x+pw+r.u*0.9, y+r.u*1.45)
		y += r.u * 2.6
	} else {
		y += r.u * 2.5
		dc.SetColor(r.accent)
		dc.DrawString(r.c.Version, x, y+r.u*1.15)
		y += r.u * 1.7
	}

	// --- meta ---
	dc.SetFontFace(face(fMono, r.u*0.82))
	dc.SetColor(colDim)
	dc.DrawString(r.c.Meta, x, y+r.u*0.6)
	y += r.u * 1.4

	// --- fading accent divider ---
	dx := r.contentR - x
	grad := gg.NewLinearGradient(x, y, x+dx, y)
	grad.AddColorStop(0, alpha(r.accent, 220))
	grad.AddColorStop(1, alpha(r.accent, 0))
	dc.SetStrokeStyle(grad)
	dc.SetLineWidth(2)
	dc.DrawLine(x, y, x+dx, y)
	dc.Stroke()
	y += r.u * 1.3

	// --- groups ---
	footerY := r.H - r.pad
	reserve := r.u * 1.5 // always keep room for a trailing "+N more"
	maxY := footerY - r.u*1.2

	box := r.u * 1.35
	itemX := x + box + r.u*0.6
	itemH := r.u * 1.24

	// If it won't all fit, cap items per section so every section gets airtime.
	perGroupCap := 1 << 30
	if r.contentHeight(r.u) > (r.H - 2*r.pad) {
		nonEmpty := 0
		for _, g := range r.c.Groups {
			if len(g.Items) > 0 {
				nonEmpty++
			}
		}
		if nonEmpty > 0 {
			perGroupCap = int(((maxY-y)/itemH - float64(nonEmpty)*2.4) / float64(nonEmpty))
			if perGroupCap < 2 {
				perGroupCap = 2
			}
		}
	}

	hidden := 0
	for _, g := range r.c.Groups {
		if len(g.Items) == 0 {
			continue
		}
		// Enough room for this section's title + at least one line + the reserve?
		if y+box+r.u*0.5+itemH+reserve > maxY {
			hidden += len(g.Items)
			continue
		}

		col := catColor[g.Kind]
		r.icon(g.Kind, x, y, box, col)
		dc.SetFontFace(face(fMonoBold, r.u*1.0))
		dc.SetColor(col)
		dc.DrawString(g.Title, itemX, y+box*0.78)
		y += box + r.u*0.5

		hashW := 0.0
		if r.opt.ShowHash {
			dc.SetFontFace(face(fMono, r.u*0.9))
			for _, it := range g.Items {
				if w, _ := dc.MeasureString(it.Hash); w > hashW {
					hashW = w
				}
			}
			hashW += r.u * 0.6
		}
		textMax := r.contentR - (itemX + hashW)

		shown := 0
		for _, it := range g.Items {
			if shown >= perGroupCap {
				break // rest of this section folds into "+N more"
			}
			dc.SetFontFace(face(fMono, r.u*0.95))
			lines := wrapLines(dc, it.Text, textMax)
			if y+float64(len(lines))*itemH+reserve > maxY {
				break
			}
			if r.opt.ShowHash && it.Hash != "" {
				dc.SetFontFace(face(fMono, r.u*0.9))
				dc.SetColor(colFaint)
				dc.DrawString(it.Hash, itemX, y+r.u*0.72)
			}
			dc.SetFontFace(face(fMono, r.u*0.95))
			dc.SetColor(colText)
			for _, ln := range lines {
				dc.DrawString(ln, itemX+hashW, y+r.u*0.72)
				y += itemH
			}
			y += r.u * 0.1
			shown++
		}
		hidden += len(g.Items) - shown
		y += r.u * 0.7
	}

	if hidden > 0 {
		dc.SetFontFace(face(fMonoBold, r.u*0.85))
		dc.SetColor(colDim)
		dc.DrawString(fmt.Sprintf("+ %d more", hidden), x, y+r.u*0.6)
	}

	// --- footer ---
	dc.SetFontFace(face(fMono, r.u*0.8))
	dc.SetColor(colDim)
	dc.DrawString("generated with ", x, footerY)
	gw, _ := dc.MeasureString("generated with ")
	dc.SetFontFace(face(fMonoBold, r.u*0.8))
	dc.SetColor(r.accent)
	dc.DrawString("relio", x+gw, footerY)
}

// icon draws a simple line glyph inside a rounded square, all in col.
func (r *renderer) icon(k GroupKind, x, y, box float64, col color.NRGBA) {
	dc := r.dc
	dc.SetColor(col)
	dc.SetLineWidth(box * 0.09)
	dc.DrawRoundedRectangle(x, y, box, box, box*0.26)
	dc.Stroke()

	cx, cy := x+box/2, y+box/2
	switch k {
	case KindAdded: // plus
		r := box * 0.24
		dc.DrawLine(cx-r, cy, cx+r, cy)
		dc.DrawLine(cx, cy-r, cx, cy+r)
		dc.Stroke()
	case KindChanged: // pencil
		dc.DrawLine(x+box*0.28, y+box*0.72, x+box*0.66, y+box*0.34)
		dc.DrawLine(x+box*0.28, y+box*0.72, x+box*0.42, y+box*0.72)
		dc.DrawLine(x+box*0.62, y+box*0.30, x+box*0.72, y+box*0.40)
		dc.Stroke()
	case KindFixed: // bug
		bw, bh := box*0.34, box*0.44
		dc.DrawEllipse(cx, cy+box*0.02, bw/2, bh/2)
		dc.Stroke()
		dc.DrawLine(cx-box*0.07, cy-bh/2+box*0.02, cx-box*0.16, cy-box*0.24)
		dc.DrawLine(cx+box*0.07, cy-bh/2+box*0.02, cx+box*0.16, cy-box*0.24)
		dc.Stroke()
		for _, ry := range []float64{-0.05, 0.06, 0.17} {
			dc.DrawLine(cx-bw/2, cy+box*ry, cx-bw/2-box*0.14, cy+box*ry-box*0.03)
			dc.DrawLine(cx+bw/2, cy+box*ry, cx+bw/2+box*0.14, cy+box*ry-box*0.03)
		}
		dc.Stroke()
	}
}

// wrapLines splits s into lines that each fit maxW pixels, breaking on spaces.
// A single word wider than maxW is ellipsised rather than overflowing.
func wrapLines(dc *gg.Context, s string, maxW float64) []string {
	words := strings.Fields(s)
	if len(words) == 0 || maxW <= 0 {
		return []string{s}
	}
	var lines []string
	cur := ""
	for _, w := range words {
		try := w
		if cur != "" {
			try = cur + " " + w
		}
		if tw, _ := dc.MeasureString(try); tw <= maxW || cur == "" {
			cur = try
		} else {
			lines = append(lines, cur)
			cur = w
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	for i, ln := range lines {
		if w, _ := dc.MeasureString(ln); w > maxW {
			lines[i] = truncate(dc, ln, maxW)
		}
	}
	return lines
}

// truncate shortens s with a trailing ellipsis until it fits maxW pixels.
func truncate(dc *gg.Context, s string, maxW float64) string {
	if w, _ := dc.MeasureString(s); w <= maxW || maxW <= 0 {
		return s
	}
	rs := []rune(s)
	for len(rs) > 1 {
		rs = rs[:len(rs)-1]
		if w, _ := dc.MeasureString(string(rs) + "…"); w <= maxW {
			return strings.TrimRight(string(rs), " ") + "…"
		}
	}
	return "…"
}
