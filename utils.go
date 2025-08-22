package main

import (
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

func Atof(astring string) float64 {
	avalue, err := strconv.ParseFloat(astring, 64)
	if err != nil {
		log.Fatal(err)
	}
	return avalue
}

func Atoi(astring string) int {
	avalue, err := strconv.ParseInt(astring, 10, 0)
	if err != nil {
		log.Fatal(err)
	}
	return int(avalue)
}

func dotsFromUnitsFloat(length float64, density float64) float64 {
	return length * density
}

func unitsFromDots(dots float64, density float64) float64 {
	return dots / density
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

func MaxWidth(sPageSize string, margin string) float64 {
	_, mr, _, ml := FourTwoOne(margin)
	pw, _ := Size(sPageSize)

	return pw - mr - ml
}

func MaxHeight(sPageSize string, margin string) float64 {
	mt, _, mb, _ := FourTwoOne(margin)
	_, ph := Size(sPageSize)

	return ph - mt - mb
}

var rxRelativeSize, _ = regexp.Compile(`^(much-smaller$|smaller$|normal$|larger$|much-larger$|scale:)`)

func ParseImageWidth(widthName string, item PbItem) float64 {
	maxWidth := MaxWidth(item.Setting("page-size"), item.Setting("margin"))
	sWidth := item.Setting(widthName)

	width := 0.0

	if rxRelativeSize.MatchString(sWidth) {
		sBaseWidth := item.RowSetting(widthName)
		if rxRelativeSize.MatchString(sBaseWidth) {
			sBaseWidth = item.PageSetting(widthName)
			if rxRelativeSize.MatchString(sBaseWidth) {
				sBaseWidth = item.BookSetting(widthName)
				if rxRelativeSize.MatchString(sBaseWidth) {
					sBaseWidth = item.DefaultSetting((widthName))
				}
			}
		}

		baseWidth := 0.0
		if !strings.HasSuffix(sBaseWidth, "%") {
			baseWidth = Atof(sBaseWidth)
		} else {
			sBaseWidth, _ = strings.CutSuffix(sBaseWidth, "%")
			baseWidth = Atof(sBaseWidth) / 100 * maxWidth
		}
		switch sWidth {
		case "much-smaller":
			width = baseWidth / 1.25 / 1.25
		case "smaller":
			width = baseWidth / 1.25
		case "normal":
			width = baseWidth
		case "larger":
			width = baseWidth * 1.25
		case "much-larger":
			width = baseWidth * 1.25 * 1.25
		default: // scale:
			sWidth, _ = strings.CutPrefix(sWidth, "scale:")
			width = baseWidth * Atof(sWidth)
		}
	} else if !strings.HasSuffix(sWidth, "%") {
		width = Atof(sWidth)
	} else {
		sWidth = strings.TrimSuffix(sWidth, "%")
		width = Atof(sWidth) / 100 * maxWidth
	}

	return math.Min(width, maxWidth)
}

func ParseWidth(sWidth string, sPageSize string, margin string) float64 {
	maxWidth := MaxWidth(sPageSize, margin)

	if !strings.HasSuffix(sWidth, "%") {
		return math.Min(Atof(sWidth), maxWidth)
	}

	sWidth = strings.TrimSuffix(sWidth, "%")

	return math.Min(Atof(sWidth), 100) / 100 * maxWidth
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
		log.Fatalf("Unexpected units %v", units)
	}
	return 0
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
		log.Fatalf("Unexpected units %v", units)
	}
	return 0
}
