package main

import (
	"bufio"
	"image"
	"log"
	"maps"
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

// order Left < Center < Right
const (
	TextAlignUnknown = iota
	TextAlignJustified
	TextAlignLeft
	TextAlignCenter
	TextAlignRight
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
	AlignRight        = AlignBottom
	AlignSpreadLeft   = AlignSpreadTop
	AlignSpreadCenter = AlignSpreadMiddle
	AlignSpreadRight  = AlignSpreadBottom
)

const (
	BindingUnknown = iota
	BindingSide
	BindingTop
)

const (
	RowAlignUnknown = iota
	RowAlignTop
	RowAlignMiddle
	RowAlignBottom
)

type PbColumnItem struct {
	item   *PbItem
	weight float64
}

type PbColumn struct {
	xOffset float64
	weight  float64
	items   []PbColumnItem
}

func (theColumn *PbColumn) width() float64 {
	width := 0.0
	for ii := range theColumn.items {
		itemWidth, _ := theColumn.items[ii].item.Size()
		width = math.Max(width, itemWidth)
	}
	return width
}

func (theColumn *PbColumn) height() float64 {
	height := 0.0
	for ii := range theColumn.items {
		_, itemHeight := theColumn.items[ii].item.Size()
		height = math.Max(height, theColumn.items[ii].item.yOffset-theColumn.items[0].item.yOffset+itemHeight)
	}
	return height
}

type PbRow struct {
	yOffset float64
	weight  float64
	columns []PbColumn
}

func (theRow *PbRow) width() float64 {
	width := 0.0
	for ii := range theRow.columns {
		width = math.Max(width, theRow.columns[ii].xOffset-theRow.columns[0].xOffset+theRow.columns[ii].width())
	}
	return width
}

func (theRow *PbRow) height() float64 {
	height := 0.0
	for ii := range theRow.columns {
		height = math.Max(height, theRow.columns[ii].height())
	}
	return height
}

type PbPage struct {
	availableWidth  float64
	availableHeight float64
	rows            []PbRow
}

func (thePage *PbPage) updateOffsets() {
	for row := range thePage.rows {
		setRowOffset := false
		for column := range thePage.rows[row].columns {
			setColumnOffset := false
			for item := range thePage.rows[row].columns[column].items {
				if !setColumnOffset {
					thePage.rows[row].columns[column].xOffset = thePage.rows[row].columns[column].items[item].item.xOffset
					setColumnOffset = true
				}
				if !setRowOffset {
					thePage.rows[row].yOffset = thePage.rows[row].columns[column].items[item].item.yOffset
					setRowOffset = true
				}
			}
		}
	}
}

func (thePage *PbPage) width() float64 {
	width := 0.0
	for ii := range thePage.rows {
		width = math.Max(width, thePage.rows[ii].width())
	}
	return width
}

func (thePage *PbPage) height() float64 {
	height := 0.0
	for ii := range thePage.rows {
		height = math.Max(height, thePage.rows[ii].yOffset-thePage.rows[0].yOffset+thePage.rows[ii].height())
	}
	return height
}

type PbBook struct {
	pages []PbPage
}

func ToPbBook(items []PbItem) *PbBook {
	book := PbBook{}
	book.pages = make([]PbPage, 0)

	curPage := -1
	curRow := -1
	curColumn := -1
	curItem := -1

	for ii, item := range items {
		if item.page != curPage {
			curPage = item.page
			book.pages = append(book.pages, PbPage{})
			book.pages[curPage].rows = make([]PbRow, 0)
			book.pages[curPage].availableWidth, book.pages[curPage].availableHeight = item.pageDimensions()
			curRow = -1
			curColumn = -1
			curItem = -1
		}

		if item.row != curRow {
			curRow = item.row
			curColumn = -1
			curItem = -1

			book.pages[curPage].rows = append(book.pages[curPage].rows, PbRow{})
			book.pages[curPage].rows[curRow].columns = make([]PbColumn, 0)
			book.pages[curPage].rows[curRow].yOffset = item.yOffset
			book.pages[curPage].rows[curRow].weight = item.FloatSetting("row-weight")
		}

		if item.column != curColumn {
			curColumn = item.column
			curItem = -1

			book.pages[curPage].rows[curRow].columns = append(book.pages[curPage].rows[curRow].columns, PbColumn{})
			book.pages[curPage].rows[len(book.pages[curPage].rows)-1].columns[curColumn].items = make([]PbColumnItem, 0)
			book.pages[curPage].rows[curRow].columns[curColumn].xOffset = item.xOffset
			book.pages[curPage].rows[curRow].columns[curColumn].weight = item.FloatSetting("column-weight")
		}

		if (item.itemType == ItemTypeImage || item.itemType == ItemTypeText) && len(item.Setting("name")) == 0 {
			curItem++

			book.pages[curPage].rows[curRow].columns[curColumn].items = append(book.pages[curPage].rows[curRow].columns[curColumn].items, PbColumnItem{})
			book.pages[curPage].rows[curRow].columns[curColumn].items[curItem].item = &items[ii]

			if curItem == 0 && curColumn == 0 {
				book.pages[curPage].rows[curRow].yOffset = items[ii].yOffset
			}

			if item.itemType == ItemTypeImage {
				book.pages[curPage].rows[curRow].columns[curColumn].items[curItem].weight = item.FloatSetting("image-weight")
			} else {
				book.pages[curPage].rows[curRow].columns[curColumn].items[curItem].weight = 1
			}
		}
	}

	for _, page := range book.pages {
		maxWeight := 0.0
		for _, row := range page.rows {
			for _, column := range row.columns {
				for _, item := range column.items {
					aWeight := row.weight * column.weight * item.weight
					if aWeight > maxWeight {
						maxWeight = aWeight
					}
				}
			}
		}
		for _, row := range page.rows {
			for _, column := range row.columns {
				for _, item := range column.items {
					item.weight = row.weight * column.weight * item.weight / maxWeight
				}
				column.weight = 1
			}
			row.weight = 1
		}
	}

	return &book
}

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

	// settings
	hasSettings   bool
	bookSetting   int
	pageSetting   int
	rowSetting    int
	columnSetting int
}

func (item *PbItem) CaptionGutter() float64 {
	captionGutter := item.FloatSetting("caption-gutter")
	if item.imageHeight == 0 || item.textHeight == 0 {
		captionGutter = 0
	}
	return captionGutter
}

func (item *PbItem) Size() (float64, float64) {

	if (item.itemType == ItemTypeImage || item.itemType == ItemTypeText) && len(item.Setting("name")) == 0 {
		return math.Max(item.imageWidth, item.textWidth), item.imageHeight + item.CaptionGutter() + item.textHeight
	}

	return 0, 0
}

func (item *PbItem) SigmoidalSetting() (float64, float64) {
	// factor (-10-10), midpoint (0.5)
	parts := strings.SplitN(item.Setting("sigmoidal"), ",", 2)

	factor := 0.0
	midpoint := 0.5

	if len(parts) > 0 && len(parts[0]) > 0 {
		factor = Atof(parts[0])
	}

	if len(parts) > 1 && len(parts[1]) > 0 {
		midpoint = Atof(parts[1])
	}

	return factor, midpoint
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

func (item *PbItem) Binding() int {
	switch strings.ToLower(item.BookSetting("binding")) {
	case "side":
		return BindingSide
	case "left":
		return BindingSide
	case "right":
		return BindingSide
	case "top":
		return BindingTop
	case "bottom":
		return BindingTop
	default:
		return BindingUnknown
	}
}

func (item *PbItem) Align(whichAlign string) int {
	settingVal := ""

	switch whichAlign {
	case "column-distribute", "column-align":
		settingVal = item.ColumnSetting(whichAlign)
	case "row-distribute", "row-align":
		settingVal = item.RowSetting(whichAlign)
	case "page-distribute":
		settingVal = item.PageSetting(whichAlign)
	default:
		settingVal = item.Setting(whichAlign)
	}

	switch strings.ToLower(settingVal) {
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
		return AlignRight
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
	_, aspect, _, _ := item.ImageRectSetting()
	return aspect
}

func (item *PbItem) Density() float64 {
	return item.FloatBookSetting("density")
}

func (item *PbItem) TextInfo() *TextInfo {
	frameInfo := item.Frame("text-frame")
	return &TextInfo{
		font: item.Setting("font"), height: item.FloatSetting("font-size"), units: item.Units(),
		density: item.Density(), padding: FourTwoOneTRBL(item.Setting("padding")), lineSpacing: item.FloatSetting("linespacing"),
		letterSpacing: item.FloatSetting("letterspacing"), wordSpacing: item.FloatSetting("wordspacing"), breakChars: item.Setting("breakchars"),
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

func (item *PbItem) textForImageDimensions(w float64, h float64) (float64, float64, int) {
	if len(item.textBlockLayouts) == 0 {
		return 0, 0, -1
	}

	squareness := Atof(item.Setting("caption-squareness")) / 100.0
	largerSize := math.Max(w, h)
	targetTextWidth := w*squareness + largerSize*(1-squareness)

	best := item.BestTextBlockLayout(targetTextWidth)

	textBlockLayout := TextBlockLayout{}
	if best >= 0 {
		textBlockLayout = item.textBlockLayouts[best]
	}

	return textBlockLayout.width, textBlockLayout.height, best
}

func (item *PbItem) enlargeImage(amount float64, dx float64, dy float64) (float64, float64) {
	if item.itemType == ItemTypeText && len(item.textBlockLayouts) == 1 {
		return 0, 0
	}

	if item.itemType != ItemTypeImage {
		return 0, 0
	}

	oldWidth, oldHeight := item.Size()

	maxWidth, maxHeight := item.ImageSizeForPage("max-size")

	aspect := item.Aspect()
	// TODO: use aspect from rect if present
	//aspect := float64(item.imageWidthPx) / float64(item.imageHeightPx)

	if aspect >= 1 {
		amount = math.Min(math.Min(amount, math.Max(maxWidth-item.imageWidth, 0)), dx)
		item.imageWidth += amount
		prevHeight := item.imageHeight
		item.imageHeight = item.imageWidth / aspect
		if item.imageHeight > maxHeight || item.imageHeight > prevHeight+dy {
			item.imageHeight = math.Min(maxHeight, prevHeight+dy)
			item.imageWidth = item.imageHeight * aspect
		}
	} else {
		amount = math.Min(math.Min(amount, math.Max(maxHeight-item.imageHeight, 0)), dy)
		item.imageHeight += amount
		prevWidth := item.imageWidth
		item.imageWidth = item.imageHeight * aspect
		if item.imageWidth > maxWidth || item.imageWidth > prevWidth+dx {
			item.imageWidth = math.Min(maxWidth, prevWidth+dx)
			item.imageHeight = item.imageWidth / aspect
		}
	}

	item.textWidth, item.textHeight, item.bestTextBlockLayout = item.textForImageDimensions(item.imageWidth, item.imageHeight)

	newWidth, newHeight := item.Size()

	return newWidth - oldWidth, newHeight - oldHeight
}

func (item *PbItem) baseDimensions() (float64, float64, float64, float64, int) {
	if item.itemType == ItemTypeText && len(item.textBlockLayouts) == 1 {
		return item.textBlockLayouts[0].width, item.textBlockLayouts[0].height, 0, 0, -1
	}

	if item.itemType != ItemTypeImage {
		return 0, 0, 0, 0, -1
	}

	w, h := item.ImageSizeForPage("size")
	tw, th, best := item.textForImageDimensions(w, h)

	return tw, th, w, h, best
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

	if len(parts) > 4 {
		wr, hr, _, _ := calcStraighten(float64(item.imageWidthPx), float64(item.imageHeightPx), item.FloatSetting("straighten"))
		return 1, wr / hr, 50, 50
	}

	zoom := 1.0
	aspect := 1.0

	if parts[0] == "fit" {
		zoom = 1.0
	} else if parts[0] == "trim" {
		zoom = 0.0
	} else if len(parts[0]) > 0 {
		zoom = Atof(parts[0])
		if zoom < 1.0 {
			zoom = 1.0
		}
	}

	if len(parts) > 1 && len(parts[1]) > 0 {
		aspectParts := strings.SplitN(parts[1], ":", 2)
		if len(aspectParts) == 2 {
			aspect = Atof(aspectParts[0]) / Atof(aspectParts[1])
		} else {
			wr, hr, _, _ := calcStraighten(float64(item.imageWidthPx), float64(item.imageHeightPx), item.FloatSetting("straighten"))
			aspect = wr / hr
		}
	} else {
		wr, hr, _, _ := calcStraighten(float64(item.imageWidthPx), float64(item.imageHeightPx), item.FloatSetting("straighten"))
		aspect = wr / hr
	}

	xOffset := 50
	if len(parts) > 2 && len(parts[2]) > 0 {
		xOffset = Atoi(parts[2])
	}

	yOffset := 50
	if len(parts) > 3 && len(parts[3]) > 0 {
		yOffset = Atoi(parts[3])
	} else {
		yOffset = xOffset
	}

	if xOffset < 0 {
		xOffset = 0
	} else if xOffset > 100 {
		xOffset = 100
	}

	if yOffset < 0 {
		yOffset = 0
	} else if yOffset > 100 {
		yOffset = 100
	}

	return zoom, aspect, xOffset, yOffset
}

var rxRelativeSize, _ = regexp.Compile(`^(much-much-much-smaller$|much-much-smaller$|much-smaller$|smaller$|normal$|larger$|much-larger$|much-much-larger$|much-much-much-larger$|scale:)`)

func (item *PbItem) ImageSizeForPage(sizeName string) (float64, float64) {
	maxWidth, maxHeight := ContainerSize(item.PageSetting("page-size"), item.PageSetting("margin"))
	sSize := item.Setting(sizeName)

	// percentage is percentage of the width of the page
	// neither width nor height should be larger than the size

	maxDimension := 0.0

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

		baseSize := 0.0

		if !strings.HasSuffix(sBaseSize, "%") {
			baseSize = Atof(sBaseSize)
		} else {
			sBaseSize = strings.TrimSuffix(sBaseSize, "%")
			baseSize = Atof(sBaseSize) / 100 * maxWidth
		}

		switch sSize {
		case "much-much-much-smaller":
			maxDimension = baseSize / 1.25 / 1.25 / 1.25 / 1.25
		case "much-much-smaller":
			maxDimension = baseSize / 1.25 / 1.25 / 1.25
		case "much-smaller":
			maxDimension = baseSize / 1.25 / 1.25
		case "smaller":
			maxDimension = baseSize / 1.25
		case "normal":
			maxDimension = baseSize
		case "larger":
			maxDimension = baseSize * 1.25
		case "much-larger":
			maxDimension = baseSize * 1.25 * 1.25
		case "much-much-larger":
			maxDimension = baseSize * 1.25 * 1.25 * 1.25
		case "much-much-much-larger":
			maxDimension = baseSize * 1.25 * 1.25 * 1.25 * 1.25
		default: // scale:
			sSize = strings.TrimPrefix(sSize, "scale:")
			maxDimension = baseSize * Atof(sSize)
		}
	} else if !strings.HasSuffix(sSize, "%") {
		maxDimension = Atof(sSize)
	} else {
		sSize = strings.TrimSuffix(sSize, "%")
		maxDimension = Atof(sSize) / 100 * maxWidth
	}

	width := 0.0
	height := 0.0
	aspect := item.Aspect()

	// Way 1: maxDimension is larger dimension
	// if aspect >= 1 { // width > height
	// 	width = maxDimension
	// 	width = math.Min(width, maxWidth)
	// 	height = width / aspect
	// 	if height > maxHeight {
	// 		height = maxHeight
	// 		width = height * aspect
	// 	}
	// } else { // height > width
	// 	height = maxDimension
	// 	height = math.Min(height, maxHeight)
	// 	width = height * aspect
	// 	if width > maxWidth {
	// 		width = maxWidth
	// 		height = width / aspect
	// 	}
	// }

	// Way 2: maxDimension * maxDimension is target area
	// width * height = maxDimension * maxDimension
	// width / heigth = aspect
	// width * width = maxDimension * maxDimension * aspect
	// height * height = maxDimension * maxDimension / aspect
	if aspect > 1 {
		width = math.Sqrt(maxDimension * maxDimension * aspect)
		width = math.Min(width, maxWidth)
		height = width / aspect
		if height > maxHeight {
			height = maxHeight
			width = height * aspect
		}
	} else {
		height = math.Sqrt(maxDimension * maxDimension / aspect)
		height = math.Min(height, maxHeight)
		width = height * aspect
		if width > maxWidth {
			width = maxWidth
			height = width / aspect
		}
	}

	return width, height
}

func (item *PbItem) pageDimensions() (float64, float64) {
	return ContainerSize(item.PageSetting("page-size"), item.PageSetting("margin"))
}

func (item *PbItem) PageSizePts() (int, int) {
	units := item.Units()
	w, h := ContainerSize(item.PageSetting("page-size"), "0")
	return int(lengthToPoints(w, units)), int(lengthToPoints(h, units))
}

func (source *PbItem) DeepCopy() PbItem {
	var dest PbItem
	dest.settings = map[string]string{}
	maps.Copy(dest.settings, source.settings)
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
	"margin":          "24",
	"background":      "#F",
	"page-distribute": "spreadmiddle", // how rows are distributed vertically on the page
	"page-row-gutter": "6",            // gutter between rows
	"current-page":    "false",

	// row
	"row-distribute":    "spreadcenter", // how columns are distributed horizontally in a row
	"row-column-gutter": "6",            // gutter between columns
	"row-weight":        "1",
	"page-break":        "false",

	// column
	"column-distribute":     "spreadmiddle", // how images or text are distributed vertically in a column
	"column-item-gutter":    "6",
	"column-weight":         "1",
	"row-break":             "false",
	"keep-columns-together": "false",

	// image or text
	"column-break": "true",
	"item-align":   "center", // left center right - how images or text of different width are aligned in a column

	// image
	"max-size":     "100%",
	"size":         "25%",
	"rect":         "1", // fit,3:2,50  trim,3:2,50  #,x:y,50,50  #=zoom level where 1=fit, Missing aspect=image aspect, Missing position=50
	"image-weight": "1",
	"image-frame":  "#0000,0",
	"straighten":   "0.0",
	"brightness":   "0.0",
	"contrast":     "0.0",
	"gamma":        "1.0",
	"saturation":   "0.0",
	"sigmoidal":    "0.0,0.50",
	"s-saturation": "0.0,0.50",
	"sharpen":      "0.0",
	"blur":         "0.0",
	"rotate":       "0", // 0, 90, 180, 270
	"recurse":      "true",

	// text
	"caption-position":   "below",
	"caption-squareness": "100",
	"caption-gutter":     "2",
	"text-align":         "left",
	"text-frame":         "#0000,0",
	"font":               "",
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
	"breakchars":         "",
	"text":               "",
	"image":              "",
	"name":               "",
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

func OptimizeSettings(book []PbItem) {
	for ii := range book {
		item := &book[ii]
		if !item.hasSettings {
			item.bookSetting = -1
			item.pageSetting = -1
			item.rowSetting = -1
			item.columnSetting = -1
			for ii, anItem := range item.pb {
				switch anItem.itemType {
				case ItemTypeBook:
					item.bookSetting = ii
				case ItemTypePage:
					item.pageSetting = ii
				case ItemTypeRow:
					item.rowSetting = ii
				case ItemTypeColumn:
					item.columnSetting = ii
				}
				if &item.pb[ii] == item {
					break
				}
			}

			if item.columnSetting < item.rowSetting {
				item.columnSetting = -1
			}
			if item.rowSetting < item.pageSetting {
				item.rowSetting = -1
			}
			if item.pageSetting < item.bookSetting {
				item.pageSetting = -1
			}

			item.hasSettings = true
		}
	}
}

func (item *PbItem) SettingInt(setting string, itemType int) string {
	var book *PbItem
	var page *PbItem
	var row *PbItem
	var column *PbItem
	bookIdx := -1
	pageIdx := -1
	rowIdx := -1
	columnIdx := -1

	var settingValue string
	var exists bool

	if itemType == ItemTypeAny {
		settingValue, exists = item.settings[setting]
		if exists {
			return settingValue
		}
	}

	if !item.hasSettings {
		for ii, anItem := range item.pb {
			if &item.pb[ii] == item {
				break
			} else if anItem.itemType == ItemTypeBook {
				book = &anItem
				bookIdx = ii
			} else if anItem.itemType == ItemTypePage {
				page = &anItem
				pageIdx = ii
			} else if anItem.itemType == ItemTypeRow {
				row = &anItem
				rowIdx = ii
			} else if anItem.itemType == ItemTypeColumn {
				column = &anItem
				columnIdx = ii
			}
		}
		if columnIdx < rowIdx {
			column = nil
		}
		if rowIdx < pageIdx {
			row = nil
		}
		if pageIdx < bookIdx {
			page = nil
		}
	} else {
		if item.bookSetting >= 0 {
			book = &item.pb[item.bookSetting]
		}
		if item.pageSetting >= 0 {
			page = &item.pb[item.pageSetting]
		}
		if item.rowSetting >= 0 {
			row = &item.pb[item.rowSetting]
		}
		if item.columnSetting >= 0 {
			column = &item.pb[item.columnSetting]
		}
	}

	if column != nil && (itemType == ItemTypeAny || itemType == ItemTypeColumn) {
		if settingValue, exists = column.settings[setting]; exists {
			return settingValue
		}
	}

	if row != nil && (itemType == ItemTypeAny || itemType == ItemTypeColumn || itemType == ItemTypeRow) {
		if settingValue, exists = row.settings[setting]; exists {
			return settingValue
		}
	}

	if page != nil && (itemType == ItemTypeAny || itemType == ItemTypeColumn || itemType == ItemTypeRow || itemType == ItemTypePage) {
		if settingValue, exists = page.settings[setting]; exists {
			return settingValue
		}
	}

	if book != nil && (itemType == ItemTypeAny || itemType == ItemTypeColumn || itemType == ItemTypeRow || itemType == ItemTypePage || itemType == ItemTypeBook) {
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
