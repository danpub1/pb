package main

import (
	"archive/zip"
	"embed"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"io"
	"log"
	"math"
	"os"
	"regexp"
	"slices"
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
	breakChars    string
	textColor     color.NRGBA
	backColor     color.NRGBA
	textAlign     int     // TextAlignLeft TextAlignCenter TextAlignRight TextAlignJustified
	textWrap      int     // TextWrapNormal TextWrapBalanced
	justifyWeight float64 // Give spaces this much more weight than non-spaces when justifying
}

type TextLineLayout struct {
	line             string
	advance          fixed.Int26_6
	endedWithNewLine bool
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
	return ((float64)(i)) / 64.0
}

func ReadFontFile(path string) ([]byte, error) {
	if newPath, hasPrefix := strings.CutPrefix(path, "::"); hasPrefix {
		return baseFonts.ReadFile("fonts/" + newPath)
	} else if pathParts := strings.SplitN(path, "::", 2); len(pathParts) == 2 {
		if zipReader, err := zip.OpenReader(pathParts[0]); err != nil {
			return make([]byte, 0), err
		} else {
			defer zipReader.Close()
			for ii := range zipReader.File {
				if zipReader.File[ii].Name == pathParts[1] {
					if file, err := zipReader.File[ii].Open(); err != nil {
						return make([]byte, 0), err
					} else {
						defer file.Close()
						buffer := make([]byte, 64*1024)
						bytes := make([]byte, 0)

						for {
							if n, err := file.Read(buffer); n > 0 && (err == nil || err == io.EOF) {
								bytes = append(bytes, buffer[:n]...)
							} else if err != io.EOF {
								log.Print(err)
								return make([]byte, 0), err
							} else if n == 0 || err == io.EOF {
								break
							}
						}

						return bytes, nil
					}
				}
			}
			return make([]byte, 0), os.ErrNotExist
		}
	} else {
		return os.ReadFile(path)
	}
}

var fontCache map[string]font.Face = map[string]font.Face{}

//go:embed fonts/*.ttf fonts/*.otf
var baseFonts embed.FS

func openFonts(textInfo *TextInfo) ([]font.Face, float64, float64) {
	size := lengthToPoints(textInfo.height, textInfo.units)
	dpi := dpi(textInfo.units, textInfo.density)

	fontNames := textInfo.font
	if len(fontNames) == 0 {
		fontNames = "::FiraSans-Regular.otf,::FiraSans-Bold.otf,::FiraSans-Italic.otf,::FiraSans-BoldItalic.otf,::Merriweather-Regular.ttf,::Merriweather-Bold.ttf,::Merriweather-Italic.ttf,::Merriweather-BoldItalic.ttf,::FiraMono-Medium.otf"
	}

	fonts := strings.Split(fontNames, ",")
	faces := make([]font.Face, 0)
	maxAscent := 0.0
	maxLineHeight := 0.0

	for _, fontName := range fonts {
		key := fmt.Sprintf("%v:%v:%v", fontName, size, dpi)

		var face font.Face
		var exists bool
		if face, exists = fontCache[key]; !exists {
			fontData, err := ReadFontFile(fontName)
			if err != nil {
				log.Print("Error opening font \"" + fontName + "\"")
				log.Fatal(err)
			}
			fnt, err := sfnt.Parse(fontData)
			if err != nil {
				log.Print("Error parsing font \"" + fontName + "\"")
				log.Fatal(err)
			}

			face, err = opentype.NewFace(fnt, &opentype.FaceOptions{Size: size, DPI: dpi, Hinting: font.HintingFull})
			if err != nil {
				log.Print("Error initializing font \"" + fontName + "\"")
				log.Fatal(err)
			}

			fontCache[key] = face
		}

		fontMetrics := face.Metrics()

		fontHeight := floatFromFixed(fontMetrics.Height)
		fontAscent := floatFromFixed(fontMetrics.Ascent)

		lineHeight := fontHeight * textInfo.lineSpacing

		faces = append(faces, face)
		maxAscent = math.Max(maxAscent, fontAscent)
		maxLineHeight = math.Max(maxLineHeight, lineHeight)
	}

	return faces, maxAscent, maxLineHeight
}

// drawString draws s at the dot and advances the dot's location.
func drawString(d []font.Drawer, s string, letterSpacing fixed.Int26_6, wordSpacing fixed.Int26_6, curFont int, justifyLetterDots fixed.Int26_6, justifySpaceDots fixed.Int26_6) ([]fixed.Int26_6, int) {
	var prevC rune
	advances := make([]fixed.Int26_6, utf8.RuneCountInString(s))
	ii := 0
	for _, c := range s {
		isFont := false
		switch c {
		case '\x00':
			isFont = true
		case '\x01':
			curFont = 1
			isFont = true
		case '\x02':
			curFont = 2
			isFont = true
		case '\x03':
			curFont = 3
			isFont = true
		case '\x04':
			curFont = 4
			isFont = true
		case '\x05':
			curFont = 5
			isFont = true
		case '\x06':
			curFont = 6
			isFont = true
		case '\x07':
			curFont = 7
			isFont = true
		case '\x08':
			curFont = 8
			isFont = true
		case '\x0B':
			curFont = 9
			isFont = true
		}
		if curFont > len(d) {
			curFont = len(d)
		}
		if c == '\n' {
			continue
		}
		if isFont {
			advances[ii] = 0
			ii++
			continue
		}
		if ii > 0 {
			advanceAdjust := d[curFont-1].Face.Kern(prevC, c) + letterSpacing
			if prevC == rune(32) {
				advanceAdjust += wordSpacing
			}
			for dd := range d {
				d[dd].Dot.X += advanceAdjust
			}
			advances[ii-1] += advanceAdjust
		}

		dr, mask, maskp, advance, _ := d[curFont-1].Face.Glyph(d[curFont-1].Dot, c)

		justifyDots := fixed.Int26_6(0)
		if ii > 0 && justifySpaceDots != fixed.Int26_6(0) && c == ' ' {
			justifyDots = justifySpaceDots
		} else if ii > 0 && justifyLetterDots != fixed.Int26_6(0) && c != ' ' {
			justifyDots = justifyLetterDots
		}

		for dd := range d {
			d[dd].Dot.X += advance + justifyDots
		}
		if d[curFont-1].Dst != nil && d[curFont-1].Src != nil && !dr.Empty() {
			draw.DrawMask(d[curFont-1].Dst, dr, d[curFont-1].Src, image.Point{}, mask, maskp, draw.Over)
		}
		advances[ii] = advance
		prevC = c
		ii++
	}

	return advances[:ii], curFont
}

func TextToImage(TextBlockLayout *TextBlockLayout, textInfo *TextInfo) *image.NRGBA {
	faces, fontAscent, lineHeight := openFonts(textInfo)

	widthDots := dotsFromUnitsFloat(TextBlockLayout.width, textInfo.density)
	heightDots := dotsFromUnitsFloat(TextBlockLayout.height, textInfo.density)

	dst := image.NewNRGBA(image.Rect(0, 0, int(math.Round(widthDots)), int(math.Round(heightDots))))

	d := make([]font.Drawer, 0)
	for ff := range faces {
		d = append(d, font.Drawer{
			Dst:  dst,
			Src:  image.NewUniform(color.NRGBA{textInfo.textColor.R, textInfo.textColor.G, textInfo.textColor.B, textInfo.textColor.A}),
			Face: faces[ff],
		})
	}

	// fill the destination with the frame color
	if textInfo.frameSize.top != 0 || textInfo.frameSize.right != 0 || textInfo.frameSize.bottom != 0 || textInfo.frameSize.left != 0 {
		draw.Draw(dst, dst.Bounds(), image.NewUniform(color.NRGBA{textInfo.frameColor.R, textInfo.frameColor.G, textInfo.frameColor.B, textInfo.frameColor.A}), image.Point{}, draw.Src)
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
	draw.Draw(dst, nonFrameRect, image.NewUniform(color.NRGBA{textInfo.backColor.R, textInfo.backColor.G, textInfo.backColor.B, textInfo.backColor.A}), image.Point{}, draw.Src)

	letterSpacing := fixedFromFloat64(textInfo.letterSpacing)
	wordSpacing := fixedFromFloat64(textInfo.wordSpacing)
	widthFixed := fixedFromFloat64(dotsFromUnitsFloat(TextBlockLayout.width-textInfo.padding.left-textInfo.padding.right-textInfo.frameSize.left-textInfo.frameSize.right, textInfo.density))

	leftEdge := dotsFromUnitsFloat(textInfo.frameSize.left+textInfo.padding.left, textInfo.density)
	for dd := range d {
		d[dd].Dot.Y = fixedFromFloat64(fontAscent + dotsFromUnitsFloat(textInfo.frameSize.top+textInfo.padding.top, textInfo.density))
	}

	curFont := 1
	for ii := range TextBlockLayout.lines {
		parts := strings.SplitN(TextBlockLayout.lines[ii].line, "\t", 3)
		for jj := range parts {
			textAlign := textInfo.textAlign
			thisAdvance := TextBlockLayout.lines[ii].advance
			line := TextBlockLayout.lines[ii].line
			if len(parts) > 1 {
				thisAdvance, curFont = advance(parts[jj], faces, letterSpacing, wordSpacing, curFont)
				if curFont > len(d) {
					curFont = len(d)
				}
				line = parts[jj]
				switch jj {
				case 0:
					textAlign = TextAlignLeft
				case 1:
					textAlign = TextAlignCenter
				case 2:
					textAlign = TextAlignRight
				}
			}

			if textAlign == TextAlignJustified && (ii == len(TextBlockLayout.lines)-1 || TextBlockLayout.lines[ii].endedWithNewLine) {
				textAlign = TextAlignLeft
			}

			justifySpaceDots := fixed.Int26_6(0)
			justifyLetterDots := fixed.Int26_6(0)
			switch textAlign {
			case TextAlignLeft:
				for dd := range d {
					d[dd].Dot.X = fixedFromFloat64(leftEdge)
				}
			case TextAlignJustified:
				letterCount := 0
				spaceCount := 0
				escapes := []rune{'\x00', '\x01', '\x02', '\x03', '\x04', '\x05', '\x06', '\x07', '\x08', '\x0B', '\n'}
				for _, rr := range line {
					if rr == 0 {
						continue
					}
					if rr == ' ' {
						spaceCount++
					} else if !slices.Contains(escapes, rr) {
						letterCount++
					}
				}
				// spaceCount * n * extraSpaceWeight + letterCount * n = extraDots
				// (spaceCount * extraSpaceWeight + letterCount) * n = extraDots
				// n = extraDots/(spaceCount * extraSpaceWeight + letterCount)
				extraSpaceWeight := textInfo.justifyWeight
				extraDots := widthFixed - thisAdvance
				jld := floatFromFixed(extraDots) / (float64(spaceCount)*extraSpaceWeight + float64(letterCount))
				spd := jld * float64(extraSpaceWeight)
				justifyLetterDots = fixedFromFloat64(jld)
				justifySpaceDots = fixedFromFloat64(spd)
				for dd := range d {
					d[dd].Dot.X = fixedFromFloat64(leftEdge)
				}
			case TextAlignCenter:
				for dd := range d {
					d[dd].Dot.X = fixedFromFloat64(leftEdge) + (widthFixed-thisAdvance)/2
				}
			case TextAlignRight:
				for dd := range d {
					d[dd].Dot.X = fixedFromFloat64(leftEdge) + widthFixed - thisAdvance
				}
			}

			_, curFont = drawString(d, line, letterSpacing, wordSpacing, curFont, justifyLetterDots, justifySpaceDots)
		}
		for dd := range d {
			d[dd].Dot.Y += fixedFromFloat64(lineHeight)
		}
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

func advance(text string, faces []font.Face, letterSpacing fixed.Int26_6, wordSpacing fixed.Int26_6, curFont int) (fixed.Int26_6, int) {
	d := make([]font.Drawer, 0)
	for ff := range faces {
		d = append(d, font.Drawer{Face: faces[ff]})
	}
	var advances []fixed.Int26_6
	advances, curFont = drawString(d, text, letterSpacing, wordSpacing, curFont, fixed.Int26_6(0), fixed.Int26_6(0))
	advance := fixed.Int26_6(0)
	for _, anAdvance := range advances {
		advance += anAdvance
	}
	return advance, curFont
}

func layoutForWidth(text string, advances []fixed.Int26_6, width float64, lineHeight float64, textInfo *TextInfo) (*TextBlockLayout, float64) {
	stringRunes := strings.Split(text, "")

	// Calculate longestBlock
	stringIdx := 0
	longestBlock := fixed.Int26_6(0)
	blockLength := fixed.Int26_6(0)
	for idx := range advances {
		for {
			if stringRunes[stringIdx] != "\n" {
				break
			}
			stringIdx++
		}
		if stringRunes[stringIdx] == " " {
			if longestBlock < blockLength {
				longestBlock = blockLength
			}
			blockLength = 0
		} else if strings.ContainsAny(stringRunes[stringIdx], textInfo.breakChars) {
			if longestBlock < blockLength {
				longestBlock = blockLength + advances[idx]
			}
			blockLength = 0
		} else {
			blockLength += advances[idx]
		}
		stringIdx++
	}

	if longestBlock < blockLength {
		longestBlock = blockLength
	}

	var layout TextBlockLayout
	layout.lines = []TextLineLayout{}
	curWidth := fixed.Int26_6(0)
	lastSpace := fixed.Int26_6(0)
	breakWidth := fixed.Int26_6(0)
	widthDots := fixedFromFloat64(dotsFromUnitsFloat(width-textInfo.padding.left-textInfo.padding.right-textInfo.frameSize.left-textInfo.frameSize.right, textInfo.density))

	stringIdx = 0
	beginIdx := 0
	breakIdx := 0
	beginStringIdx := 0
	breakStringIdx := 0
	for idx := 0; idx < len(advances); idx++ {
		forceBreak := false
		for {
			if stringRunes[stringIdx] != "\n" {
				break
			}
			stringIdx++
			forceBreak = true
		}
		if curWidth+advances[idx] > widthDots || forceBreak { // Need to break the line
			if beginIdx == breakIdx && stringRunes[stringIdx] != " " && !forceBreak && !strings.ContainsAny(stringRunes[stringIdx], textInfo.breakChars) {
				return nil, 0.0
			}
			// Is the current character one we can break on?
			if stringRunes[stringIdx] == " " {
				breakIdx = idx
				breakStringIdx = stringIdx
				breakWidth = curWidth
			} else if (idx > 0 && strings.ContainsAny(stringRunes[stringIdx-1], textInfo.breakChars)) || forceBreak {
				breakIdx = idx
				breakStringIdx = stringIdx
				breakWidth = curWidth
			}
			var textLineLayout TextLineLayout
			textLineLayout.advance = breakWidth
			textLineLayout.line = strings.Join(stringRunes[beginStringIdx:breakStringIdx], "")
			textLineLayout.endedWithNewLine = forceBreak
			layout.lines = append(layout.lines, textLineLayout)
			beginIdx = breakIdx
			beginStringIdx = breakStringIdx
			idx = breakIdx
			stringIdx = breakStringIdx
			breakWidth = 0
			curWidth = 0
			if stringRunes[stringIdx] == " " {
				idx++
				breakIdx++
				beginIdx++

				stringIdx++
				breakStringIdx++
				beginStringIdx++
			}
		}
		if stringRunes[stringIdx] == " " {
			breakIdx = idx
			breakStringIdx = stringIdx
			breakWidth = curWidth
			if lastSpace < curWidth {
				lastSpace = curWidth
			}
		} else if idx > 0 && strings.ContainsAny(stringRunes[stringIdx-1], textInfo.breakChars) {
			breakIdx = idx
			breakStringIdx = stringIdx
			breakWidth = curWidth
			if lastSpace < curWidth {
				lastSpace = curWidth
			}
		}
		curWidth += advances[idx]
		stringIdx++
	}

	if curWidth > 0 {
		var textLineLayout TextLineLayout
		textLineLayout.advance = curWidth
		textLineLayout.line = strings.Join(stringRunes[beginStringIdx:], "")
		layout.lines = append(layout.lines, textLineLayout)
	}

	layout.width = width
	height := math.Ceil(float64(len(layout.lines)) * lineHeight)
	layout.height = unitsFromDots(height, textInfo.density) + textInfo.padding.top + textInfo.padding.bottom + textInfo.frameSize.top + textInfo.frameSize.bottom

	// First time through, lastSpace is the curWidth
	if len(layout.lines) == 1 && curWidth < widthDots {
		lastSpace = curWidth
	} else {
		if longestBlock > lastSpace {
			lastSpace = longestBlock
		}
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

var rxSpace, _ = regexp.Compile("( +)")

func MeasureText(text string, maxWidth float64, maxHeight float64, textInfo *TextInfo) []TextBlockLayout {
	if strings.ContainsRune(text, '\t') {
		parts := strings.Split(text, "\t")
		var layout1, layout2, layout3, destLayout []TextBlockLayout
		maxLines := 0
		if len(parts) > 0 {
			textInfo.textAlign = TextAlignLeft
			layout1 = MeasureText(parts[0], maxWidth, 0, textInfo)
			maxLines = len(layout1[0].lines)
			destLayout = layout1
		}
		if len(parts) > 1 {
			textInfo.textAlign = TextAlignCenter
			layout2 = MeasureText(parts[1], maxWidth, 0, textInfo)
			if len(layout2[0].lines) > maxLines {
				maxLines = len(layout2[0].lines)
				destLayout = layout2
			}
		}
		if len(parts) > 2 {
			textInfo.textAlign = TextAlignRight
			layout3 = MeasureText(parts[2], maxWidth, 0, textInfo)
			if len(layout3[0].lines) > maxLines {
				maxLines = len(layout3[0].lines)
				destLayout = layout3
			}
		}

		for ii := range maxLines {
			var aline strings.Builder
			if layout1 != nil && len(layout1[0].lines) > ii {
				aline.WriteString(layout1[0].lines[ii].line)
			}
			aline.WriteString("\t")
			if layout2 != nil && len(layout2[0].lines) > ii {
				aline.WriteString(layout2[0].lines[ii].line)
			}
			aline.WriteString("\t")
			if layout3 != nil && len(layout3[0].lines) > ii {
				aline.WriteString(layout3[0].lines[ii].line)
			}
			destLayout[0].lines[ii].line = aline.String()
			destLayout[0].lines[ii].advance = fixed.Int26_6(0)
		}

		return destLayout
	}

	faces, _, lineHeight := openFonts(textInfo)

	letterSpacing := fixedFromFloat64(textInfo.letterSpacing)
	wordSpacing := fixedFromFloat64(textInfo.wordSpacing)

	d := make([]font.Drawer, 0)
	for ff := range faces {
		d = append(d, font.Drawer{Face: faces[ff]})
	}

	// Collapse whitespace
	text = strings.Trim(rxSpace.ReplaceAllString(text, " "), " ")

	advances, _ := drawString(d, text, letterSpacing, wordSpacing, 1, fixed.Int26_6(0), fixed.Int26_6(0))

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
