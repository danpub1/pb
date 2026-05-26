package main

import (
	"image/color"
	"log"
	"math"
	"regexp"
	"strconv"
	"strings"
)

const (
	PointsPerInch      = 72
	InchesPerInch      = 1
	CentimetersPerInch = 2.54
	MillimetersPerInch = 25.4
)

func VerboseLog(message string) {
	if Opts.Verbose("L") {
		log.Print(message)
	}
}

func Atof(astring string) float64 {
	avalue, err := strconv.ParseFloat(astring, 64)
	if err != nil {
		avalue = 0.0
		log.Print("Error converting \"" + astring + "\" to float64")
		log.Print(err)
	}
	return avalue
}

func Atoi(astring string) int {
	avalue, err := strconv.ParseInt(astring, 10, 0)
	if err != nil {
		avalue = 0
		log.Print("Error converting \"" + astring + "\" to int64")
		log.Print(err)
	}
	return int(avalue)
}

func Atob(astring string) bool {
	if astring == "false" || astring == "no" || astring == "0" || len(strings.TrimSpace(astring)) == 0 {
		return false
	}

	if astring == "true" || astring == "yes" || astring == "1" {
		return true
	}

	log.Print("Error converting \"" + astring + "\" to bool")
	return false
}

// W(%)[:x]H(%)[+-]X(%)[+-]Y(%),
// func ParseSize(sSize string, sPageSize string, sMargin string, sDensity string) (float64, float64, float64, float64) {
//     idx := strings.IndexAny(sSize, ":x")
//     sWidth := sSize[:idx]
//     sOperator := sSize[idx:idx+1]
//     sSize = sSize[idx+1:]

//     idx := strings.IndexAny(sSize, "+-")
//     sHeight := sSize[:idx]
//     sXSign := sSize[idx:idx+1]
//     sSize = sSize[idx+1:]

//     idx := strings.IndexAny(sSize, "+-")
//     sX := sSize[:idx]
//     sYSign := sSize[idx:idx+1]
//     sY = sSize[idx+1:]

//     pageWidth, pageHeight := SizeDots(sPageSize, sDensity)
// }

type TRBL struct {
	top    float64
	right  float64
	bottom float64
	left   float64
}

type FrameInfo struct {
	size  TRBL
	color color.NRGBA
	name  string
	above bool
}

// margins: txrxbxl / tbxrl / tbrl -> t, r, b, l
func FourTwoOne(sFourTwoOne string) (float64, float64, float64, float64) {
	parts := strings.SplitN(sFourTwoOne, "x", 4)
	if len(parts) == 1 {
		val := Atof(parts[0])
		return val, val, val, val
	}

	if len(parts) == 2 {
		val1 := Atof(parts[0])
		val2 := Atof(parts[1])
		return val1, val2, val1, val2
	}

	if len(parts) == 4 {
		val1 := Atof(parts[0])
		val2 := Atof(parts[1])
		val3 := Atof(parts[2])
		val4 := Atof(parts[3])
		return val1, val2, val3, val4
	}

	return 0, 0, 0, 0
}

func FourTwoOneTRBL(sFourTwoOne string) *TRBL {
	val1, val2, val3, val4 := FourTwoOne(sFourTwoOne)
	return &TRBL{val1, val2, val3, val4}
}

// WxH
func FloatSize(sSize string) (float64, float64) {
	w, h := Size(sSize)
	return Atof(w), Atof(h)
}

func Size(sSize string) (string, string) {
	if len(sSize) == 0 {
		return "0", "0"
	}

	parts := strings.SplitN(sSize, "x", 2)
	if len(parts) == 2 {
		sWidth := parts[0]
		sHeight := parts[1]
		return sWidth, sHeight
	} else {
		return sSize, sSize
	}
}

// e.g. text box minus margins, page minus margins
func ContainerWidth(sItemSize string, margin string) float64 {
	w, _ := ContainerSize(sItemSize, margin)
	return w
}

func ContainerHeight(sItemSize string, margin string) float64 {
	_, h := ContainerSize(sItemSize, margin)
	return h
}

func ContainerSize(sItemSize string, margin string) (float64, float64) {
	mt, mr, mb, ml := FourTwoOne(margin)
	pw, ph := FloatSize(sItemSize)

	return pw - mr - ml, ph - mt - mb
}

func WidthForContainer(sSize string, sContainerSize string, margin string) float64 {
	w, _ := SizeForContainer(sSize, sContainerSize, margin)
	return w
}

func HeightForContainer(sSize string, sContainerSize string, margin string) float64 {
	_, h := SizeForContainer(sSize, sContainerSize, margin)
	return h
}

// This could be a "size" setting or a "max-size" setting
func SizeForContainer(sSize string, sContainerSize string, margin string) (float64, float64) {
	sWidth, sHeight := Size(sSize)
	width := 0.0
	height := 0.0

	maxWidth, maxHeight := ContainerSize(sContainerSize, margin)

	if before, ok := strings.CutSuffix(sWidth, "%"); ok {
		sWidth = before
		width = math.Min(Atof(sWidth), 100) / 100 * maxWidth
	} else if before, ok := strings.CutSuffix(sWidth, "!"); ok {
		sWidth = before
		width = math.Min(Atof(sWidth), maxWidth)
	} else {
		width = math.Min(Atof(sWidth), maxWidth)
	}

	if before, ok := strings.CutSuffix(sHeight, "%"); ok {
		sHeight = before
		height = math.Min(Atof(sHeight), 100) / 100 * maxHeight
	} else if before, ok := strings.CutSuffix(sHeight, "!"); ok {
		sHeight = before
		height = math.Min(Atof(sHeight), maxHeight)
	} else {
		height = math.Min(Atof(sHeight), maxHeight)
	}

	return width, height
}

var rxValidColor, _ = regexp.Compile("^#([[:xdigit:]]{1,4}|[[:xdigit:]]{6}|[[:xdigit:]]{8})$")

func colorToNRGBA(text string) color.NRGBA {
	rr, gg, bb, aa := int64(0), int64(0), int64(0), int64(255)

	if rxValidColor.MatchString(text) {
		text, _ = strings.CutPrefix(text, "#")
		switch len(text) {
		case 1:
			rr, _ = strconv.ParseInt(text+text, 16, 0)
			gg = rr
			bb = rr
		case 2:
			rr, _ = strconv.ParseInt(text, 16, 0)
			gg = rr
			bb = rr
		case 3:
			rr, _ = strconv.ParseInt(text[0:1]+text[0:1], 16, 0)
			gg, _ = strconv.ParseInt(text[1:2]+text[1:2], 16, 0)
			bb, _ = strconv.ParseInt(text[2:3]+text[2:3], 16, 0)
		case 4:
			rr, _ = strconv.ParseInt(text[0:1]+text[0:1], 16, 0)
			gg, _ = strconv.ParseInt(text[1:2]+text[1:2], 16, 0)
			bb, _ = strconv.ParseInt(text[2:3]+text[2:3], 16, 0)
			aa, _ = strconv.ParseInt(text[3:4]+text[3:4], 16, 0)
		case 6:
			rr, _ = strconv.ParseInt(text[0:2], 16, 0)
			gg, _ = strconv.ParseInt(text[2:4], 16, 0)
			bb, _ = strconv.ParseInt(text[4:6], 16, 0)
		case 8:
			rr, _ = strconv.ParseInt(text[0:2], 16, 0)
			gg, _ = strconv.ParseInt(text[2:4], 16, 0)
			bb, _ = strconv.ParseInt(text[4:6], 16, 0)
			aa, _ = strconv.ParseInt(text[6:8], 16, 0)
		}
	}

	return color.NRGBA{uint8(rr), uint8(gg), uint8(bb), uint8(aa)}
}

func lengthToPoints(length float64, units int) float64 {
	switch units {
	case UnitsPt:
		return length
	case UnitsIn:
		return length * PointsPerInch
	case UnitsCm:
		return length / CentimetersPerInch * PointsPerInch
	case UnitsMm:
		return length / MillimetersPerInch * PointsPerInch
	default:
		log.Printf("Unexpected units %v\n", units)
		return length
	}
}

func dpi(units int, density float64) float64 {
	switch units {
	case UnitsPt:
		return PointsPerInch * density
	case UnitsIn:
		return InchesPerInch * density
	case UnitsCm:
		return CentimetersPerInch * density
	case UnitsMm:
		return MillimetersPerInch * density
	default:
		log.Printf("Unexpected units %v\n", units)
		return PointsPerInch * density
	}
}
