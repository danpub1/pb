package main

import (
	"bufio"
	"image"
	"log"
	"math"
	"os"
	"regexp"
	"strings"
)

const (
	ItemTypeUnknown = iota
	ItemTypeBook
	ItemTypePage
	ItemTypeRow
	ItemTypeColumn
	ItemTypeImage
	ItemTypeText
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

	pb []PbItem

	// measurements
	textBlockLayouts []TextBlockLayout
	imageWidthPx     int
	imageHeightPx    int

	// layout
	page                int
	row                 int
	column              int
	textWidth           float64
	textHeight          float64
	bestTextBlockLayout int
	imageWidth          float64
	imageHeight         float64
	xOffset             float64
	yOffset             float64
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
	return item.FloatSetting("density")
}

func (item *PbItem) TextInfo() *TextInfo {
	frameInfo := item.Frame("text-frame")
	return &TextInfo{
		font: item.Setting("font"), height: item.FloatSetting("font-size"), units: item.Units(),
		density: item.Density(), padding: FourTwoOneTRBL(item.Setting("padding")), lineSpacing: item.FloatSetting("linespacing"),
		letterSpacing: item.FloatSetting("letterspacing"), wordSpacing: item.FloatSetting("wordspacing"),
		textColor: colorToNRGBA(item.Setting("text-color")), backColor: colorToNRGBA(item.Setting("text-back-color")),
		textAlign: item.TextAlign(), textWrap: item.TextWrap(), justifyWeight: item.IntSetting("justify-weight"),
		frameColor: frameInfo.color, frameSize: &frameInfo.size,
	}
}

func (item *PbItem) BestTextBlockLayout(targetTextWidth float64) int {

	if len(item.textBlockLayouts) == 0 {
		return -1
	}

	best := -1
	for ii := range item.textBlockLayouts {
		if item.textBlockLayouts[ii].width < targetTextWidth && (best < 0 || item.textBlockLayouts[ii].width > item.textBlockLayouts[best].width) {
			best = ii
		}
	}

	// nothing found less than target width?  choose smallest found
	if best < 0 {
		best = 0
		for ii := range item.textBlockLayouts {
			if item.textBlockLayouts[ii].width < item.textBlockLayouts[best].width {
				best = ii
			}
		}
	}

	return best
}

func (item *PbItem) baseDimensions() (float64, float64, float64, float64, int) {
	if item.itemType == ItemTypeText && len(item.textBlockLayouts) == 1 {
		return item.textBlockLayouts[0].width, item.textBlockLayouts[0].height, 0, 0, -1
	}

	if item.itemType == ItemTypeImage {
		w, h := item.ImageSizeForPage("size")

		if len(item.textBlockLayouts) == 0 {
			return 0, 0, w, h, -1
		}

		squareness := Atof(item.Setting("caption-squareness")) / 100.0
		largerSize := math.Max(w, h)
		targetTextWidth := w*squareness + largerSize*(1-squareness)

		best := item.BestTextBlockLayout(targetTextWidth)

		textBlockLayout := TextBlockLayout{}
		if best >= 0 {
			textBlockLayout = item.textBlockLayouts[best]
		}

		return textBlockLayout.width, textBlockLayout.height, w, h, best
	}

	return 0, 0, 0, 0, -1
}

func (item *PbItem) GetImage() *image.Image {
	imageFile, err := os.Open(item.Setting("image"))
	if err != nil {
		log.Fatal(err)
	}
	imageReader := bufio.NewReader(imageFile)
	decodedImage, _, err := image.Decode(imageReader)
	if err != nil {
		imageFile.Close()
		log.Fatal(err)
	}
	if err := imageFile.Close(); err != nil {
		log.Fatal(err)
	}

	return &decodedImage
}

func (item *PbItem) ImageRectSetting() (float64, float64, int, int) {
	sRect := item.Setting("rect")
	parts := strings.SplitN(sRect, ",", 4)

	if len(parts) != 4 {
		return 1, float64(item.imageWidthPx) / float64(item.imageHeightPx), 50, 50
	}

	zoom := 1.0
	aspect := 1.0

	if parts[0] == "fit" {
		zoom = 1.0
	} else if parts[0] == "trim" {
		zoom = 1.0 // TODO
	} else if len(parts[0]) > 0 {
		zoom = Atof(parts[0])
	}

	if len(parts[1]) > 0 {
		aspectParts := strings.SplitN(parts[0], ":", 2)
		if len(aspectParts) == 2 {
			aspect = float64(Atoi(aspectParts[0])) / float64(Atoi(aspectParts[1]))
		} else {
			aspect = float64(item.imageWidthPx) / float64(item.imageHeightPx)
		}
	} else {
		aspect = float64(item.imageWidthPx) / float64(item.imageHeightPx)
	}

	return zoom, aspect, Atoi(parts[2]), Atoi(parts[3])
}

var rxRelativeSize, _ = regexp.Compile(`^(much-smaller$|smaller$|normal$|larger$|much-larger$|scale:)`)

func (item *PbItem) ImageSizeForPage(sizeName string) (float64, float64) {
	maxWidth, maxHeight := ContainerSize(item.Setting("page-size"), item.Setting("margin"))
	sSize := item.Setting(sizeName)

	width := 0.0
	height := 0.0

	if rxRelativeSize.MatchString(sSize) {
		sBaseSize := item.RowSetting(sizeName)
		if rxRelativeSize.MatchString(sBaseSize) {
			sBaseSize = item.PageSetting(sizeName)
			if rxRelativeSize.MatchString(sBaseSize) {
				sBaseSize = item.BookSetting(sizeName)
				if rxRelativeSize.MatchString(sBaseSize) {
					sBaseSize = item.DefaultSetting((sizeName))
				}
			}
		}

		baseWidth := 0.0
		baseHeight := 0.0

		if !strings.HasSuffix(sBaseSize, "%") {
			baseWidth = Atof(sBaseSize)
			baseHeight = Atof(sBaseSize)
		} else {
			sBaseSize = strings.TrimSuffix(sBaseSize, "%")
			if maxWidth > maxHeight {
				baseWidth = Atof(sBaseSize) / 100 * maxHeight
				baseHeight = Atof(sBaseSize) / 100 * maxHeight
			} else {
				baseWidth = Atof(sBaseSize) / 100 * maxWidth
				baseHeight = Atof(sBaseSize) / 100 * maxWidth
			}
		}

		switch sSize {
		case "much-smaller":
			width = baseWidth / 1.25 / 1.25
			height = baseHeight / 1.25 / 1.25
		case "smaller":
			width = baseWidth / 1.25
			height = baseHeight / 1.25
		case "normal":
			width = baseWidth
			height = baseHeight
		case "larger":
			width = baseWidth * 1.25
			height = baseHeight * 1.25
		case "much-larger":
			width = baseWidth * 1.25 * 1.25
			height = baseHeight * 1.25 * 1.25
		default: // scale:
			sSize = strings.TrimPrefix(sSize, "scale:")
			width = baseWidth * Atof(sSize)
			height = baseHeight * Atof(sSize)
		}
	} else if !strings.HasSuffix(sSize, "%") {
		width = Atof(sSize)
		height = Atof(sSize)
	} else {
		sSize = strings.TrimSuffix(sSize, "%")
		width = Atof(sSize) / 100 * maxWidth
		height = Atof(sSize) / 100 * maxHeight
	}

	_, aspect, _, _ := item.ImageRectSetting()

	if aspect >= 1 {
		height = height / aspect
	} else {
		width = width * aspect
	}

	return math.Min(width, maxWidth), math.Min(height, maxHeight)
}

func (item *PbItem) pageDimensions() (float64, float64) {
	return ContainerSize(item.PageSetting("page-size"), item.PageSetting("margin"))
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
	"page-distribute": "spreadmiddle", // how
	"page-row-gutter": "6",            // gutter between rows
	"interleave-rows": "true",

	// row
	"row-distribute":    "spreadcenter", // how things are distributed horizontally in a row
	"column-align":      "middle",       // top middle bottom - how columns of different height are aligned in a row
	"row-column-gutter": "6",            // gutter between columns
	"row-weight":        "1",
	"page-break":        "false",

	// column
	"column-distribute":     "spreadmiddle", // how things are distributed horizontally in a row
	"item-align":            "center",       // left center right - how things of different height are aligned in a row
	"column-item-gutter":    "6",
	"column-weight":         "1",
	"row-break":             "false",
	"keep-columns-together": "false",

	// image or text
	"column-break": "true",

	// image
	"max-size":     "100%x100%",
	"size":         "25%x25%",
	"rect":         "1", // fit,3:2,50  trim,3:2,50  #,x:y,50,50  #=zoom level where 1=fit, Missing aspect=image aspect, Missing position=50
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
	"caption-position":   "below",
	"caption-squareness": "100",
	"caption-gutter":     "2",
	"text-align":         "left",
	"text-frame":         "#0000,0",
	"font":               "times.ttf",
	"font-size":          "14",
	"linespacing":        "1",
	"letterspacing":      "0",
	"wordspacing":        "0",
	"padding":            "3.5",
	"text-wrap":          "balanced",
	"text-width":         "100%",
	"text-color":         "#0",
	"text-back-color":    "#0000",
	"justify-weight":     "10",

	"text":  "",
	"image": "",
	"name":  "",
}

func (item *PbItem) Set(setting string, value string) {
	if _, exists := defaultSettings[setting]; !exists {
		log.Fatalf("unrecognized settting: %v", setting)
	}

	item.settings[setting] = value
}

func (item *PbItem) BoolSetting(setting string) bool {
	return Atob(item.Setting(setting))
}

func (item *PbItem) BoolColumnSetting(setting string) bool {
	return Atob(item.ColumnSetting(setting))
}

func (item *PbItem) BoolRowSetting(setting string) bool {
	return Atob(item.RowSetting(setting))
}

func (item *PbItem) BoolPageSetting(setting string) bool {
	return Atob(item.PageSetting(setting))
}

func (item *PbItem) BoolBookSetting(setting string) bool {
	return Atob(item.BookSetting(setting))
}

func (item *PbItem) BoolDefaultSetting(setting string) bool {
	return Atob(item.DefaultSetting(setting))
}

func (item *PbItem) IntSetting(setting string) int {
	return Atoi(item.Setting(setting))
}

func (item *PbItem) IntColumnSetting(setting string) int {
	return Atoi(item.ColumnSetting(setting))
}

func (item *PbItem) IntRowSetting(setting string) int {
	return Atoi(item.RowSetting(setting))
}

func (item *PbItem) IntPageSetting(setting string) int {
	return Atoi(item.PageSetting(setting))
}

func (item *PbItem) IntBookSetting(setting string) int {
	return Atoi(item.BookSetting(setting))
}

func (item *PbItem) IntDefaultSetting(setting string) int {
	return Atoi(item.DefaultSetting(setting))
}

func (item *PbItem) FloatSetting(setting string) float64 {
	return Atof(item.Setting(setting))
}

func (item *PbItem) FloatColumnSetting(setting string) float64 {
	return Atof(item.ColumnSetting(setting))
}

func (item *PbItem) FloatRowSetting(setting string) float64 {
	return Atof(item.RowSetting(setting))
}

func (item *PbItem) FloatPageSetting(setting string) float64 {
	return Atof(item.PageSetting(setting))
}

func (item *PbItem) FloatBookSetting(setting string) float64 {
	return Atof(item.BookSetting(setting))
}

func (item *PbItem) FloatDefaultSetting(setting string) float64 {
	return Atof(item.DefaultSetting(setting))
}

func (item *PbItem) Setting(setting string) string {
	return item.SettingInt(setting, ItemTypeAny)
}

func (item *PbItem) ColumnSetting(setting string) string {
	return item.SettingInt(setting, ItemTypeColumn)
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
	var column PbItem

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
		} else if anItem.itemType == ItemTypeColumn {
			column = anItem
		}
	}

	if itemType == ItemTypeAny || itemType == ItemTypeColumn {
		if settingValue, exists = column.settings[setting]; exists {
			return settingValue
		}
	}

	if itemType == ItemTypeAny || itemType == ItemTypeColumn || itemType == ItemTypeRow {
		if settingValue, exists = row.settings[setting]; exists {
			return settingValue
		}
	}

	if itemType == ItemTypeAny || itemType == ItemTypeColumn || itemType == ItemTypeRow || itemType == ItemTypePage {
		if settingValue, exists = page.settings[setting]; exists {
			return settingValue
		}
	}

	if itemType == ItemTypeAny || itemType == ItemTypeColumn || itemType == ItemTypeRow || itemType == ItemTypePage || itemType == ItemTypeBook {
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
