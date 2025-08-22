package main

import (
	"image/color"
	"log"
	"regexp"
	"strconv"
	"strings"
)

const (
	ItemTypeUnknown = iota
	ItemTypeBook
	ItemTypePage
	ItemTypeRow
	ItemTypeImage
	ItemTypeText
	ItemTypeColumn
	ItemTypeDefault
	ItemTypeAny
)

const (
	TextAlignUnknown = iota
	TextAlignLeft
	TextAlignCenter
	TextAlignRight
	TextAlignJustified
)

const (
	TextWrapUnknown = iota
	TextWrapUnbalanced
	TextWrapBalanced
)

const (
	UnitsUnknown = iota
	UnitsPt
	UnitsIn
	UnitsCm
	UnitsMm
)

const (
	AlignUnknown = iota
	AlignTop
	AlignMiddle
	AlignBottom
	AlignJustify
	AlignBinding
	AlignEdge
	AlignSpreadTop
	AlignSpreadMiddle
	AlignSpreadBottom
	AlignSpreadBinding
	AlignSpreadEdge
	AlignLeft         = AlignTop
	AlignCenter       = AlignMiddle
	AlighRight        = AlignBottom
	AlignSpreadLeft   = AlignSpreadTop
	AlignSpreadCenter = AlignSpreadMiddle
	AlignSpreadRight  = AlignSpreadBottom
)

const (
	RowAlignUnknown = iota
	RowAlignTop
	RowAlignMiddle
	RowAlignBottom
)

type PbItem struct {
	itemType int
	settings map[string]string

	depth int
	pb    []PbItem

	textBlockLayouts []TextBlockLayout
	width            float64 // rowWidth for row items, width for text or image items
	height           float64
}

type TRBL struct {
	top    float64
	right  float64
	bottom float64
	left   float64
}

type FrameInfo struct {
	size  TRBL
	color color.NRGBA
}

// margins: t,r,b,l / tb,rl / tbrl -> t, r, b, l
func FourTwoOne(sFourTwoOne string) (float64, float64, float64, float64) {
	parts := strings.SplitN(sFourTwoOne, ",", 4)
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
func Size(sSize string) (float64, float64) {
	if len(sSize) == 0 {
		return 0, 0
	}

	idx := strings.IndexAny(sSize, "x")
	sOperator := sSize[idx : idx+1]
	if sOperator != "x" {
		log.Fatalf("Unrecognized Size %v", sSize)
	}
	sWidth := sSize[:idx]
	sHeight := sSize[idx+1:]

	return Atof(sWidth), Atof(sHeight)
}

func (item *PbItem) Units() int {
	switch item.Setting("units") {
	case "pt":
		return UnitsPt
	case "in":
		return UnitsIn
	case "cm":
		return UnitsCm
	case "mm":
		return UnitsMm
	default:
		return UnitsUnknown
	}
}

func (item *PbItem) TextAlign() int {
	switch item.Setting("text-align") {
	case "left":
		return TextAlignLeft
	case "center":
		return TextAlignCenter
	case "right":
		return TextAlignRight
	case "justified":
		return TextAlignJustified
	default:
		return TextAlignUnknown
	}
}

func (item *PbItem) TextWrap() int {
	switch item.Setting("text-wrap") {
	case "unbalanced":
		return TextWrapUnbalanced
	case "balanced":
		return TextWrapBalanced
	default:
		return TextWrapUnknown
	}
}

func (item *PbItem) Align(whichAlign string) int {
	switch item.Setting(whichAlign) {
	case "top":
		return AlignTop
	case "middle":
		return AlignMiddle
	case "bottom":
		return AlignBottom
	case "justify":
		return AlignJustify
	case "binding":
		return AlignBinding
	case "edge":
		return AlignEdge
	case "spreadtop":
		return AlignSpreadTop
	case "spreadmiddle":
		return AlignSpreadMiddle
	case "spreadbottom":
		return AlignSpreadBottom
	case "spreadbinding":
		return AlignSpreadBinding
	case "spreadedge":
		return AlignSpreadEdge
	case "left":
		return AlignLeft
	case "center":
		return AlignCenter
	case "right":
		return AlighRight
	case "spreadleft":
		return AlignSpreadLeft
	case "spreadcenter":
		return AlignSpreadCenter
	case "spreadright":
		return AlignSpreadRight
	default:
		return AlignUnknown
	}
}

func (item *PbItem) RowAlign() int {
	switch item.Setting("row-align") {
	case "top":
		return RowAlignTop
	case "middle":
		return RowAlignMiddle
	case "bottom":
		return RowAlignBottom
	default:
		return RowAlignUnknown
	}
}

func (item *PbItem) Frame(whichFrame string) *FrameInfo {
	var frameInfo FrameInfo
	frameString := item.Setting(whichFrame)
	frameParts := strings.SplitN(frameString, ",", 2)
	if len(frameParts) > 0 {
		frameInfo.color = colorToNRGBA(frameParts[0])
		if len(frameParts) > 1 {
			frameInfo.size = *FourTwoOneTRBL(frameParts[1])
		}
	}

	return &frameInfo
}

var rxRect, _ = regexp.Compile(`^(fit|trim|crop),\d+:\d+,\d+(\.\d+)?(,\d+(\.\d+)?)?$`)

func (item *PbItem) Aspect() float64 {
	sRect := item.Setting("rect")
	if rxRect.MatchString(sRect) {
		parts := strings.Split(sRect, ",")
		wh := strings.Split(parts[1], ":")
		width := Atof(wh[0])
		height := Atof(wh[1])
		return width / height
	} else {
		return 1.0
	}
}

func (item *PbItem) Density() float64 {
	return Atof(item.Setting("density"))
}

func (item *PbItem) TextInfo() *TextInfo {
	frameInfo := item.Frame("text-frame")
	return &TextInfo{
		font: item.Setting("font"), height: Atof(item.Setting("font-size")), units: item.Units(),
		density: item.Density(), padding: FourTwoOneTRBL(item.Setting("padding")), lineSpacing: Atof(item.Setting("linespacing")),
		letterSpacing: Atof(item.Setting("letterspacing")), wordSpacing: Atof(item.Setting("wordspacing")),
		textColor: colorToNRGBA(item.Setting("text-color")), backColor: colorToNRGBA(item.Setting("text-back-color")),
		textAlign: item.TextAlign(), textWrap: item.TextWrap(), justifyWeight: Atoi(item.Setting("justify-weight")),
		frameColor: frameInfo.color, frameSize: &frameInfo.size,
	}
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

func (source *PbItem) DeepCopy() PbItem {
	var dest PbItem
	dest.settings = map[string]string{}
	for kk, vv := range source.settings {
		dest.settings[kk] = vv
	}
	dest.itemType = source.itemType
	dest.pb = source.pb
	return dest
}

var defaultSettings = map[string]string{
	// book
	"units":              "pt",
	"density":            "2",
	"binding":            "side",
	"output-gamma":       "1.0",
	"output-sharpen":     "4,1,0.5,0",
	"output-mozjpeg":     "true",
	"output-compression": "97",

	// page
	"page-size":       "576x576",
	"margin":          "36",
	"background":      "#F",
	"page-align":      "spreadmiddle",
	"v-gutter":        "6",
	"interleave-rows": "true",

	// row
	"row-align":  "spreadcenter", // how things are distributed horizontally in a row
	"item-align": "middle",       // top middle bottom - how things of different height are aligned in a row
	"h-gutter":   "6",
	"row-weight": "1",

	// image
	"min-width":    "25%",
	"max-width":    "100%",
	"size":         "normal",
	"rect":         "fit,3:2,50", // trim,3:2,50  crop,x:y,50,50
	"image-weight": "1",
	"image-frame":  "#0000,0",
	"straighten":   "0.0",
	"brightness":   "0.0",
	"contrast":     "0.0",
	"gamma":        "1.0",
	"saturation":   "0.0",
	"s-contrast":   "0.0, 0.50",
	"s-saturation": "0.0, 0.50",

	// text
	"caption-position": "below",
	"caption-width":    "100",
	"text-align":       "left",
	"text-frame":       "#0000,0",
	"font":             "times.ttf",
	"font-size":        "14",
	"linespacing":      "1",
	"letterspacing":    "0",
	"wordspacing":      "0",
	"padding":          "3.5",
	"text-wrap":        "balanced",
	"text-width":       "100%",
	"text-color":       "#0",
	"text-back-color":  "#0000",
	"justify-weight":   "10",

	"text-dimensions":  "",
	"image-dimensions": "",
	"text":             "",
	"image":            "",
	"layout-width":     "",
	"layout-height":    "",
}

func (item *PbItem) Set(setting string, value string) {
	if _, exists := defaultSettings[setting]; !exists {
		log.Fatalf("unrecognized settting: %v", setting)
	}

	item.settings[setting] = value
}

func (item *PbItem) Setting(setting string) string {
	return item.SettingInt(setting, ItemTypeAny)
}

func (item *PbItem) RowSetting(setting string) string {
	return item.SettingInt(setting, ItemTypeRow)
}

func (item *PbItem) PageSetting(setting string) string {
	return item.SettingInt(setting, ItemTypePage)
}

func (item *PbItem) BookSetting(setting string) string {
	return item.SettingInt(setting, ItemTypeBook)
}

func (item *PbItem) DefaultSetting(setting string) string {
	return item.SettingInt(setting, ItemTypeDefault)
}

func (item *PbItem) SettingInt(setting string, itemType int) string {
	var book PbItem
	var page PbItem
	var row PbItem

	var settingValue string
	var exists bool

	if itemType == ItemTypeAny {
		settingValue, exists = item.settings[setting]
		if exists {
			return settingValue
		}
	}

	for _, anItem := range item.pb {
		if &anItem == item {
			break
		} else if anItem.itemType == ItemTypeBook {
			book = anItem
		} else if anItem.itemType == ItemTypePage {
			page = anItem
		} else if anItem.itemType == ItemTypeRow {
			row = anItem
		}
	}

	if itemType == ItemTypeAny || itemType == ItemTypeRow {
		if settingValue, exists = row.settings[setting]; exists {
			return settingValue
		}
	}

	if itemType == ItemTypeAny || itemType == ItemTypeRow || itemType == ItemTypePage {
		if settingValue, exists = page.settings[setting]; exists {
			return settingValue
		}
	}

	if itemType == ItemTypeAny || itemType == ItemTypeRow || itemType == ItemTypePage || itemType == ItemTypeBook {
		if settingValue, exists = book.settings[setting]; exists {
			return settingValue
		}
	}

	if settingValue, exists = defaultSettings[setting]; exists {
		return settingValue
	}

	log.Fatalf("unrecognized settting: %v", setting)
	return ""
}
