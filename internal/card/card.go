// Package card renders a shareable PNG image of a release.
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
	"golang.org/x/image/font/gofont/goregular"
)

// Shape is the aspect ratio of the output image.
type Shape int

const (
	Horizontal Shape = iota // 1200x630  (Twitter / OpenGraph)
	Vertical                // 1080x1350 (Instagram portrait)
	Square                  // 1080x1080
)

// ParseShape maps a name to a Shape.
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
		return 1080, 1350
	case Square:
		return 1080, 1080
	default:
		return 1200, 630
	}
}

// Row is one "type description" line on the card.
type Row struct {
	Type string
	Text string
	Hash string // abbreviated commit hash; shown only when Render is asked to
}

// Card is everything the image shows.
type Card struct {
	Project string
	Version string
	Meta    string // "2026-09-06 · 13:47 · 4 commits"
	Rows    []Row
}

var (
	fontRegular *truetype.Font
	fontBold    *truetype.Font
)

func init() {
	fontRegular, _ = truetype.Parse(goregular.TTF)
	fontBold, _ = truetype.Parse(gobold.TTF)
}

func face(f *truetype.Font, size float64) font.Face {
	return truetype.NewFace(f, &truetype.Options{Size: size, DPI: 72, Hinting: font.HintingFull})
}

const (
	colBG     = "#0E0E13"
	colBorder = "#26262F"
	colOrange = "#F5872B"
	colPurple = "#A855F7"
	colText   = "#ECECEF"
	colDim    = "#8A8A99"
	colFaint  = "#63636F"
)

// Render draws the card for the given shape. When showHash is true each row is
// prefixed with its commit hash.
func Render(c Card, s Shape, showHash bool) image.Image {
	w, h := s.size()
	fw, fh := float64(w), float64(h)
	dc := gg.NewContext(w, h)

	dc.SetHexColor(colBG)
	dc.Clear()

	// Soft diagonal orange -> purple wash over the dark base (NRGBA so the alpha
	// is honoured; gg/draw expects non-premultiplied here).
	grad := gg.NewLinearGradient(0, 0, fw, fh)
	grad.AddColorStop(0, color.NRGBA{0xF5, 0x87, 0x2B, 0x22})
	grad.AddColorStop(0.5, color.NRGBA{0x0E, 0x0E, 0x13, 0x00})
	grad.AddColorStop(1, color.NRGBA{0xA8, 0x55, 0xF7, 0x2A})
	dc.SetFillStyle(grad)
	dc.DrawRectangle(0, 0, fw, fh)
	dc.Fill()

	u := fh / 24 // scale unit
	pad := u * 1.9

	// Inner rounded frame.
	dc.SetHexColor(colBorder)
	dc.SetLineWidth(2)
	dc.DrawRoundedRectangle(pad*0.55, pad*0.55, fw-pad*1.1, fh-pad*1.1, u*0.9)
	dc.Stroke()

	x := pad
	y := pad + u*0.9

	// Project + version.
	dc.SetFontFace(face(fontBold, u*2.1))
	dc.SetHexColor(colText)
	dc.DrawString(c.Project, x, y+u*1.5)
	projW, _ := dc.MeasureString(c.Project + "  ")
	dc.SetFontFace(face(fontBold, u*1.5))
	dc.SetHexColor(colPurple)
	dc.DrawString(c.Version, x+projW, y+u*1.5)
	y += u * 2.5

	// Meta.
	dc.SetFontFace(face(fontRegular, u*0.82))
	dc.SetHexColor(colDim)
	dc.DrawString(c.Meta, x, y+u*0.6)
	y += u * 1.5

	// Divider.
	dc.SetHexColor(colOrange)
	dc.DrawRectangle(x, y, fw-2*pad, 2)
	dc.Fill()
	y += u * 1.1

	// Rows. Columns: [hash] · type · text.
	gap := u * 0.7
	hashX := x
	typeX := x
	if showHash {
		dc.SetFontFace(face(fontRegular, u))
		hw := 0.0
		for _, r := range c.Rows {
			if w, _ := dc.MeasureString(r.Hash); w > hw {
				hw = w
			}
		}
		typeX = hashX + hw + gap
	}

	dc.SetFontFace(face(fontBold, u))
	typeW := 0.0
	for _, r := range c.Rows {
		if tw, _ := dc.MeasureString(r.Type); tw > typeW {
			typeW = tw
		}
	}
	textX := typeX + typeW + gap
	maxTextW := fw - pad - textX
	lineH := u * 1.7
	bottom := fh - pad - u*1.4

	rows := c.Rows
	more := 0
	if maxN := int((bottom - y) / lineH); maxN >= 1 && len(rows) > maxN {
		more = len(rows) - (maxN - 1)
		rows = rows[:maxN-1]
	}

	if len(rows) == 0 {
		dc.SetFontFace(face(fontRegular, u*0.9))
		dc.SetHexColor(colDim)
		dc.DrawString("no notable changes", x, y+u*0.75)
	}
	for _, r := range rows {
		if showHash && r.Hash != "" {
			dc.SetFontFace(face(fontRegular, u))
			dc.SetHexColor(colFaint)
			dc.DrawString(r.Hash, hashX, y+u*0.75)
		}

		dc.SetFontFace(face(fontBold, u))
		dc.SetHexColor(colOrange)
		dc.DrawString(r.Type, typeX, y+u*0.75)

		dc.SetFontFace(face(fontRegular, u))
		dc.SetHexColor(colText)
		dc.DrawString(truncate(dc, r.Text, maxTextW), textX, y+u*0.75)
		y += lineH
	}
	if more > 0 {
		dc.SetFontFace(face(fontRegular, u*0.82))
		dc.SetHexColor(colDim)
		dc.DrawString(fmt.Sprintf("+%d more", more), x, y+u*0.6)
	}

	// Footer.
	dc.SetFontFace(face(fontRegular, u*0.78))
	dc.SetHexColor(colFaint)
	dc.DrawString("generated with relio", x, fh-pad)

	return dc.Image()
}

// truncate shortens s with a trailing ellipsis until it fits maxW pixels.
func truncate(dc *gg.Context, s string, maxW float64) string {
	if w, _ := dc.MeasureString(s); w <= maxW {
		return s
	}
	r := []rune(s)
	for len(r) > 1 {
		r = r[:len(r)-1]
		if w, _ := dc.MeasureString(string(r) + "…"); w <= maxW {
			return strings.TrimRight(string(r), " ") + "…"
		}
	}
	return "…"
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
