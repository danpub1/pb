package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"log"
	"math"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

type TextInfo struct {
	font          string
	height        float64
	units         int
	density       float64
	padding       *TRBL
	frameColor    color.NRGBA
	frameSize     *TRBL
	lineSpacing   float64
	letterSpacing float64
	wordSpacing   float64
	textColor     color.NRGBA
	backColor     color.NRGBA
	textAlign     int // TextAlignLeft TextAlignCenter TextAlignRight TextAlignJustified
	textWrap      int // TextWrapNormal TextWrapBalanced
	justifyWeight int // Give spaces this many more dots than non-spaces when justifying
}

type TextLineLayout struct {
	line    string
	advance fixed.Int26_6
}

type TextBlockLayout struct {
	lines  []TextLineLayout
	width  float64
	height float64
}

func dotsFromUnitsFloat(length float64, density float64) float64 {
	return length * density
}

func unitsFromDots(dots float64, density float64) float64 {
	return dots / density
}

func fixedFromFloat64(f float64) fixed.Int26_6 {
	return fixed.Int26_6(int(math.Round(f * 64.0)))
}

func floatFromFixed(i fixed.Int26_6) float64 {
	return (float64)(i / 64.0)
}

var fontCache map[string]font.Face = map[string]font.Face{}

func openFont(textInfo *TextInfo) (font.Face, float64, float64) {
	size := lengthToPoints(textInfo.height, textInfo.units)
	dpi := dpi(textInfo.units, textInfo.density)
	key := fmt.Sprintf("%v:%v:%v", textInfo.font, size, dpi)

	var face font.Face
	var exists bool
	if face, exists = fontCache[key]; !exists {
		fontData, err := os.ReadFile(textInfo.font)
		if err != nil {
			log.Fatal(err)
		}
		fnt, err := sfnt.Parse(fontData)
		if err != nil {
			log.Fatal(err)
		}

		face, err = opentype.NewFace(fnt, &opentype.FaceOptions{Size: size, DPI: dpi, Hinting: font.HintingFull})
		if err != nil {
			log.Fatal(err)
		}

		fontCache[key] = face
	}

	fontMetrics := face.Metrics()

	fontHeight := floatFromFixed(fontMetrics.Height)
	fontAscent := floatFromFixed(fontMetrics.Ascent)

	lineHeight := fontHeight * textInfo.lineSpacing

	return face, fontAscent, lineHeight
}

// drawString draws s at the dot and advances the dot's location.
func drawString(d *font.Drawer, s string, letterSpacing fixed.Int26_6, wordSpacing fixed.Int26_6) []fixed.Int26_6 {
	var prevC rune
	advances := make([]fixed.Int26_6, utf8.RuneCountInString(s))
	ii := 0
	for _, c := range s {
		if ii > 0 {
			advanceAdjust := d.Face.Kern(prevC, c) + letterSpacing
			if prevC == rune(32) {
				advanceAdjust += wordSpacing
			}
			d.Dot.X += advanceAdjust
			advances[ii-1] += advanceAdjust
		}
		dr, mask, maskp, advance, _ := d.Face.Glyph(d.Dot, c)
		d.Dot.X += advance
		if d.Dst != nil && d.Src != nil && !dr.Empty() {
			draw.DrawMask(d.Dst, dr, d.Src, image.Point{}, mask, maskp, draw.Over)
		}
		advances[ii] = advance
		prevC = c
		ii++
	}

	return advances
}

func TextToImage(TextBlockLayout *TextBlockLayout, textInfo *TextInfo) *image.RGBA {
	face, fontAscent, lineHeight := openFont(textInfo)

	widthDots := dotsFromUnitsFloat(TextBlockLayout.width, textInfo.density)
	heightDots := dotsFromUnitsFloat(TextBlockLayout.height, textInfo.density)

	dst := image.NewRGBA(image.Rect(0, 0, int(math.Round(widthDots)), int(math.Round(heightDots))))

	d := &font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(color.RGBA{textInfo.textColor.R, textInfo.textColor.G, textInfo.textColor.B, textInfo.textColor.A}),
		Face: face,
	}

	// fill the destination with the frame color
	if textInfo.frameSize.top != 0 || textInfo.frameSize.right != 0 || textInfo.frameSize.bottom != 0 || textInfo.frameSize.left != 0 {
		draw.Draw(dst, dst.Bounds(), image.NewUniform(color.RGBA{textInfo.frameColor.R, textInfo.frameColor.G, textInfo.frameColor.B, textInfo.frameColor.A}), image.Point{}, draw.Src)
	}

	nonFrameRect := dst.Bounds()
	if textInfo.frameSize.top != 0 {
		nonFrameRect.Min.Y += int(dotsFromUnitsFloat(textInfo.frameSize.top, textInfo.density))
	}
	if textInfo.frameSize.right != 0 {
		nonFrameRect.Max.X -= int(dotsFromUnitsFloat(textInfo.frameSize.right, textInfo.density))
	}
	if textInfo.frameSize.bottom != 0 {
		nonFrameRect.Max.Y -= int(dotsFromUnitsFloat(textInfo.frameSize.bottom, textInfo.density))
	}
	if textInfo.frameSize.left != 0 {
		nonFrameRect.Min.X += int(dotsFromUnitsFloat(textInfo.frameSize.left, textInfo.density))
	}

	// fill area inside the frame with the background color
	draw.Draw(dst, nonFrameRect, image.NewUniform(color.RGBA{textInfo.backColor.R, textInfo.backColor.G, textInfo.backColor.B, textInfo.backColor.A}), image.Point{}, draw.Src)

	letterSpacing := fixedFromFloat64(textInfo.letterSpacing)
	wordSpacing := fixedFromFloat64(textInfo.wordSpacing)
	widthFixed := fixedFromFloat64(dotsFromUnitsFloat(TextBlockLayout.width-textInfo.padding.left-textInfo.padding.right-textInfo.frameSize.left-textInfo.frameSize.right, textInfo.density))

	leftEdge := dotsFromUnitsFloat(textInfo.frameSize.left+textInfo.padding.left, textInfo.density)
	d.Dot.X = fixedFromFloat64(leftEdge)
	d.Dot.Y = fixedFromFloat64(fontAscent + dotsFromUnitsFloat(textInfo.frameSize.top+textInfo.padding.top, textInfo.density))
	for ii := range TextBlockLayout.lines {
		switch textInfo.textAlign {
		case TextAlignCenter:
			d.Dot.X += (widthFixed - TextBlockLayout.lines[ii].advance) / 2
		case TextAlignRight:
			d.Dot.X += widthFixed - TextBlockLayout.lines[ii].advance
		}
		drawString(d, TextBlockLayout.lines[ii].line, letterSpacing, wordSpacing)
		d.Dot.X = fixedFromFloat64(leftEdge)
		d.Dot.Y += fixedFromFloat64(lineHeight)
	}

	// out, err := os.Create("out.png")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer out.Close()
	// if err := png.Encode(out, dst); err != nil {
	// 	log.Fatal(err)
	// }

	return dst
}

func layoutForWidth(text string, advances []fixed.Int26_6, width float64, lineHeight float64, textInfo *TextInfo) (*TextBlockLayout, float64) {
	var layout TextBlockLayout
	layout.lines = []TextLineLayout{}
	curWidth := fixed.Int26_6(0)
	lastSpace := fixed.Int26_6(0)
	beginIdx := 0
	breakIdx := 0
	stringRunes := strings.Split(text, "")
	widthDots := fixedFromFloat64(dotsFromUnitsFloat(width-textInfo.padding.left-textInfo.padding.right-textInfo.frameSize.left-textInfo.frameSize.right, textInfo.density))

	breakWidth := fixed.Int26_6(0)
	for idx := 0; idx < len(advances); idx++ {
		if curWidth+advances[idx] > widthDots {
			if beginIdx == breakIdx && stringRunes[idx] != " " {
				return nil, 0.0
			} else {
				if stringRunes[idx] == " " {
					breakIdx = idx
					breakWidth = curWidth
				}
				var textLineLayout TextLineLayout
				textLineLayout.advance = breakWidth
				textLineLayout.line = strings.Join(stringRunes[beginIdx:breakIdx], "")
				layout.lines = append(layout.lines, textLineLayout)
				beginIdx = breakIdx
				idx = breakIdx
				breakWidth = 0
				curWidth = 0
				if stringRunes[idx] == " " {
					idx++
					breakIdx++
					beginIdx++
				}
			}
		}
		if stringRunes[idx] == " " {
			breakIdx = idx
			breakWidth = curWidth
			if lastSpace < curWidth {
				lastSpace = curWidth
			}
		}
		curWidth += advances[idx]
	}

	if curWidth > 0 {
		var textLineLayout TextLineLayout
		textLineLayout.advance = curWidth
		textLineLayout.line = strings.Join(stringRunes[beginIdx:], "")
		layout.lines = append(layout.lines, textLineLayout)
	}

	layout.width = width
	height := math.Ceil(float64(len(layout.lines)) * lineHeight)
	layout.height = unitsFromDots(height, textInfo.density) + textInfo.padding.top + textInfo.padding.bottom + textInfo.frameSize.top + textInfo.frameSize.bottom

	if len(layout.lines) == 1 && curWidth < widthDots {
		lastSpace = curWidth
	}
	return &layout, unitsFromDots(floatFromFixed(lastSpace), textInfo.density)
}

func layoutForBalanced(text string, advances []fixed.Int26_6, width float64, lineHeight float64, textInfo *TextInfo) *TextBlockLayout {
	var lastLayout *TextBlockLayout = nil
	var layout *TextBlockLayout = nil

	curWidth := width

	for {
		lastSpace := 0.0
		layout, lastSpace = layoutForWidth(text, advances, curWidth, lineHeight, textInfo)
		if layout != nil {
			if lastLayout != nil && (len(layout.lines) > len(lastLayout.lines) || layout.width == lastLayout.width) {
				break
			}
			lastLayout = layout
			curWidth = lastSpace + textInfo.padding.left + textInfo.padding.right + textInfo.frameSize.left + textInfo.frameSize.right
		} else {
			break
		}
	}

	if lastLayout != nil {
		lastLayout.width = width
	}
	return lastLayout
}

var rxSpace, _ = regexp.Compile("([[:space:]]+)")

func MeasureText(text string, maxWidth float64, maxHeight float64, textInfo *TextInfo) []TextBlockLayout {
	face, _, lineHeight := openFont(textInfo)

	letterSpacing := fixedFromFloat64(textInfo.letterSpacing)
	wordSpacing := fixedFromFloat64(textInfo.wordSpacing)

	d := &font.Drawer{Face: face}

	// Collapse whitespace
	text = strings.Trim(rxSpace.ReplaceAllString(text, " "), " ")

	advances := drawString(d, text, letterSpacing, wordSpacing)

	// Calculate the TextBlockLayout for a specific maxWidth
	if maxHeight == 0 && textInfo.textWrap != TextWrapBalanced {
		layout, _ := layoutForWidth(text, advances, maxWidth, lineHeight, textInfo)
		if layout != nil {
			return []TextBlockLayout{*layout}
		} else {
			return []TextBlockLayout{}
		}
	} else if maxHeight == 0 {
		layout := layoutForBalanced(text, advances, maxWidth, lineHeight, textInfo)
		if layout != nil {
			return []TextBlockLayout{*layout}
		} else {
			return []TextBlockLayout{}
		}
	}

	// Calculate TextBlockLayouts for various numbers of lines
	curHeight := 0.0
	lastSpace := 1.0
	layouts := []TextBlockLayout{}
	var lastLayout, layout *TextBlockLayout
	for curWidth := maxWidth; curHeight <= maxHeight && curWidth > 0 && lastSpace > 0; curWidth = lastSpace + textInfo.padding.right + textInfo.padding.left + textInfo.frameSize.left + textInfo.frameSize.right {
		layout, lastSpace = layoutForWidth(text, advances, curWidth, lineHeight, textInfo)
		if layout != nil {
			if lastLayout == nil {
				lastLayout = layout
			} else {
				if len(layout.lines) > len(lastLayout.lines) {
					layouts = append(layouts, *lastLayout)
				} else if layout.width == lastLayout.width {
					break
				}
				lastLayout = layout
			}
		}
	}

	if lastLayout != nil {
		layouts = append(layouts, *lastLayout)
	}

	return layouts
}
