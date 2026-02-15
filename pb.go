package main

import (
	"bufio"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	"log"
	"math"
	"os"

	"github.com/disintegration/imaging"
)

var (
	inFileFlag  = flag.String("i", "", `pb input filename"`)
	verboseFlag = flag.Bool("v", false, "verbose")
)

func getImageDimensions(items []PbItem) {
	if len(items) == 0 {
		return
	}

	for ii, theItem := range items {
		if theItem.itemType == ItemTypeImage {
			imageFile, err := os.Open(theItem.Setting("image"))
			if err != nil {
				log.Fatal(err)
			}
			imageReader := bufio.NewReader(imageFile)
			config, _, err := image.DecodeConfig(imageReader)
			if err != nil {
				imageFile.Close()
				log.Fatal(err)
			}
			if err := imageFile.Close(); err != nil {
				log.Fatal(err)
			}

			items[ii].imageWidthPx = config.Width
			items[ii].imageHeightPx = config.Height
		}
	}
}

func getTextDimensions(items []PbItem) {
	if len(items) == 0 {
		return
	}

	for ii := range items {
		if items[ii].itemType == ItemTypeText && len(items[ii].Setting("text")) > 0 {
			width := WidthForContainer(items[ii].Setting("text-width"), items[ii].Setting("page-size"), items[ii].Setting("margin"))
			items[ii].textBlockLayouts = MeasureText(items[ii].Setting("text"), width, 0.0, items[ii].TextInfo())
			// TextToImage(&items[ii].textBlockLayouts[0], items[ii].TextInfo())
		} else if items[ii].itemType == ItemTypeImage && len(items[ii].Setting("text")) > 0 {
			maxWidth := ContainerWidth(items[ii].Setting("page-size"), items[ii].Setting("margin"))
			maxHeight := ContainerHeight(items[ii].Setting("page-size"), items[ii].Setting("margin"))
			items[ii].textBlockLayouts = MeasureText(items[ii].Setting("text"), maxWidth, maxHeight, items[ii].TextInfo())
		}
	}
}

func rowHeight(items []PbItem, page int, row int, maxColumn int) float64 {
	rowHeight := 0.0
	columnHeight := 0.0
	curColumn := 0
	for ii := range items {
		if items[ii].page == page && items[ii].row == row && items[ii].column <= maxColumn && (items[ii].itemType == ItemTypeImage || items[ii].itemType == ItemTypeText) {
			if items[ii].column > curColumn {
				curColumn = items[ii].column
				columnHeight = 0.0
			}

			captionGutter := 0.0
			if items[ii].imageHeight > 0 && items[ii].textHeight > 0 {
				captionGutter = Atof(items[ii].Setting("caption-gutter"))
			}

			columnHeight += items[ii].yOffset + items[ii].imageHeight + captionGutter + items[ii].textHeight
			rowHeight = math.Max(rowHeight, columnHeight)
		}
	}
	return rowHeight
}

type breakIntoPageState struct {
	pagesInBook   int
	rowsOnPage    int
	columnsInRow  int
	itemsInColumn int

	pageWidth  float64
	pageHeight float64

	curRowYOffset float64
	curRowHeight  float64

	curColumnXOffset float64
	curColumnWidth   float64
	curColumnHeight  float64
}

func (source *breakIntoPageState) DeepCopy() breakIntoPageState {
	return breakIntoPageState{
		pagesInBook:   source.pagesInBook,
		rowsOnPage:    source.rowsOnPage,
		columnsInRow:  source.columnsInRow,
		itemsInColumn: source.itemsInColumn,

		pageHeight: source.pageHeight,
		pageWidth:  source.pageWidth,

		curRowYOffset: source.curRowYOffset,
		curRowHeight:  source.curRowHeight,

		curColumnXOffset: source.curColumnXOffset,
		curColumnWidth:   source.curColumnWidth,
		curColumnHeight:  source.curColumnHeight,
	}
}

func breakIntoPages(items []PbItem) []PbItem {
	s := breakIntoPageState{}
	stateStack := make([]breakIntoPageState, len(items))

	s.pagesInBook = 0
	s.rowsOnPage = 0
	s.columnsInRow = 0
	s.itemsInColumn = 0

	s.pageHeight = 0.0
	s.pageWidth = 0.0

	s.curRowYOffset = 0.0
	s.curRowHeight = 0.0

	s.curColumnXOffset = 0.0
	s.curColumnWidth = 0.0
	s.curColumnHeight = 0.0

	for ii := 0; ii < len(items); ii++ {

		if ii > 0 {
			stateStack[ii-1] = s.DeepCopy()
		}

		items[ii].textWidth, items[ii].textHeight, items[ii].imageWidth, items[ii].imageHeight, items[ii].bestTextBlockLayout = items[ii].baseDimensions()
		captionGutter := 0.0
		if items[ii].imageHeight > 0 && items[ii].textHeight > 0 {
			captionGutter = Atof(items[ii].Setting("caption-gutter"))
		}

		itemWidth := math.Max(items[ii].textWidth, items[ii].imageWidth)
		itemHeight := items[ii].imageHeight + captionGutter + items[ii].textHeight

		s.pageWidth, s.pageHeight = items[ii].pageDimensions()

		if ii > 0 {
			items[ii].page = items[ii-1].page
			items[ii].row = items[ii-1].row
			items[ii].column = items[ii-1].column
		} else {
			items[ii].page = 0
			items[ii].row = 0
			items[ii].column = 0
		}

		if items[ii].itemType == ItemTypeBook && s.pagesInBook == 0 || (items[ii].itemType == ItemTypePage || items[ii].BoolSetting("page-break")) && s.rowsOnPage > 0 {
			if ii > 0 {
				items[ii].page = items[ii-1].page + 1
			} else {
				items[ii].page = 0
			}
			items[ii].row = 0
			items[ii].column = 0
			s.rowsOnPage = 0
			s.columnsInRow = 0
			s.itemsInColumn = 0
			s.curRowYOffset = 0
			s.curRowHeight = 0
			s.curColumnXOffset = 0
			s.curColumnWidth = 0
			s.curColumnHeight = 0
			items[ii].xOffset = 0
			items[ii].yOffset = 0
		}

		if (items[ii].itemType == ItemTypeRow || items[ii].BoolSetting("row-break")) && s.columnsInRow > 0 {
			if ii > 0 {
				items[ii].row = items[ii-1].row + 1
			} else {
				items[ii].row = 0
			}
			items[ii].column = 0
			s.columnsInRow = 0
			s.itemsInColumn = 0
			s.curRowYOffset = s.curRowYOffset + s.curRowHeight + items[ii].FloatSetting("page-row-gutter")
			s.curRowHeight = 0
			s.curColumnXOffset = 0
			s.curColumnWidth = 0
			s.curColumnHeight = 0
			items[ii].xOffset = s.curColumnXOffset
			items[ii].yOffset = s.curRowYOffset
		}

		if (items[ii].itemType == ItemTypeColumn || items[ii].BoolSetting("column-break")) && s.itemsInColumn > 0 {
			if ii > 0 {
				items[ii].column = items[ii-1].column + 1
			} else {
				items[ii].column = 0
			}
			s.itemsInColumn = 0
			s.curColumnXOffset = s.curColumnXOffset + s.curColumnWidth + items[ii].FloatSetting("row-column-gutter")
			s.curColumnWidth = 0
			s.curColumnHeight = 0
			items[ii].xOffset = s.curColumnXOffset
			items[ii].yOffset = s.curRowYOffset
		}

		if (items[ii].itemType == ItemTypeImage || items[ii].itemType == ItemTypeText) && len(items[ii].Setting("name")) == 0 {
			columnItemGutter := 0.0
			if s.itemsInColumn > 0 {
				columnItemGutter = items[ii].FloatSetting("column-item-gutter")
			}
			rowColumnGutter := 0.0
			if s.columnsInRow > 0 {
				rowColumnGutter = items[ii].FloatSetting("row-column-gutter")
			}
			pageRowGutter := 0.0
			if s.rowsOnPage > 0 {
				pageRowGutter = items[ii].FloatSetting("page-row-gutter")
			}
			startOfColumn := ii
			for startOfColumn > 0 && items[startOfColumn-1].column == items[ii].column && items[startOfColumn-1].row == items[ii].row && items[startOfColumn-1].page == items[ii].page {
				startOfColumn--
			}
			startOfRow := startOfColumn
			for startOfRow > 0 && items[startOfRow-1].row == items[startOfColumn].row && items[startOfRow-1].page == items[startOfColumn].page {
				startOfRow--
			}
			startOfPage := startOfRow
			for startOfPage > 0 && items[startOfPage-1].page == items[startOfRow].page {
				startOfPage--
			}

			curItemYOffset := s.curRowYOffset + s.curColumnHeight + columnItemGutter

			prevColumnsRowHeight := rowHeight(items, items[ii].page, items[ii].row, items[ii].column-1)

			if curItemYOffset+itemHeight <= s.pageHeight && s.curColumnXOffset+itemWidth <= s.pageWidth {
				items[ii].xOffset = s.curColumnXOffset
				items[ii].yOffset = curItemYOffset
				s.curColumnWidth = math.Max(s.curColumnWidth, itemWidth)
				s.curColumnHeight += columnItemGutter + itemHeight
				s.curRowHeight = math.Max(s.curRowHeight, s.curColumnHeight)
				s.itemsInColumn++
				if s.itemsInColumn == 1 {
					s.columnsInRow++
					if s.columnsInRow == 1 {
						s.rowsOnPage++
						if s.rowsOnPage == 1 {
							s.pagesInBook++
						}
					}
				}

			} else if s.curColumnXOffset+itemWidth > s.pageWidth { // Column is too wide for page
				// Column is too wide but there is room for it in the next row
				if s.curRowYOffset+prevColumnsRowHeight+pageRowGutter+s.curColumnHeight+itemHeight < s.pageHeight {
					VerboseLog(fmt.Sprintf("/// VERBOSE: Column too wide, moving column to next row at %v\n", ii))
					items[startOfColumn].Set("row-break", "true")
					ii = startOfColumn - 1
					s = stateStack[ii].DeepCopy()
				} else { // Column is too wide and there is not room for it in the next
					if items[ii].BoolSetting("keep-columns-together") {
						VerboseLog(fmt.Sprintf("/// VERBOSE: Column too wide, Moving column to next page at %v\n", ii))
						items[startOfColumn].Set("page-break", "true")
						ii = startOfColumn - 1
						s = stateStack[ii].DeepCopy()
					} else { // just breaking at the item
						if s.curRowYOffset+s.curRowHeight+pageRowGutter+itemHeight < s.pageHeight {
							VerboseLog(fmt.Sprintf("/// VERBOSE: Column too wide, breaking row at %v\n", ii))
							items[ii].Set("row-break", "true")
							ii = ii - 1
							s = stateStack[ii].DeepCopy()
						} else {
							VerboseLog(fmt.Sprintf("/// VERBOSE: Column too wide, breaking page at %v\n", ii))
							items[ii].Set("page-break", "true")
							ii = ii - 1
							s = stateStack[ii].DeepCopy()
						}
					}
				}

			} else {
				// Is there room for another column?
				if s.itemsInColumn > 0 && s.curColumnXOffset+s.curColumnWidth+rowColumnGutter+itemWidth < s.pageWidth {
					VerboseLog(fmt.Sprintf("/// VERBOSE: Column too tall, breaking column at %v\n", ii))
					items[ii].Set("column-break", "true")
					ii = ii - 1
					s = stateStack[ii].DeepCopy()
				} else {
					VerboseLog(fmt.Sprintf("/// VERBOSE: Column too tall, breaking page at %v\n", ii))
					items[ii].Set("page-break", "true")
					ii = ii - 1
					s = stateStack[ii].DeepCopy()
				}
			}
		}
	}
	return items
}

func writePage(img image.Image, curPage int) {
	out, err := os.Create(fmt.Sprintf("out-%v.png", curPage))
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()
	if err := png.Encode(out, img); err != nil {
		log.Fatal(err)
	}
}

func renderPages(items []PbItem) {

	curPage := -1
	var top float64
	var left float64
	var dst *image.RGBA = nil
	density := 1.0

	for _, item := range items {
		if item.page != curPage {
			if dst != nil {
				writePage(dst, curPage)
			}
			curPage = item.page
			pageWidth, pageHeight := FloatSize(item.Setting("page-size"))
			density = item.FloatSetting("density")
			top, _, _, left = FourTwoOne(item.Setting("margin"))
			widthDots := int(math.Round(dotsFromUnitsFloat(pageWidth, density)))
			heightDots := int(math.Round(dotsFromUnitsFloat(pageHeight, density)))
			dst = image.NewRGBA(image.Rect(0, 0, widthDots, heightDots))

			// fill the destination with the background color
			backColor := colorToNRGBA(item.Setting("background"))
			draw.Draw(dst, dst.Bounds(), image.NewUniform(color.RGBA{backColor.R, backColor.G, backColor.B, backColor.A}), image.Point{}, draw.Src)
		}

		if item.itemType == ItemTypeText && len(item.Setting("name")) == 0 {
			textImage := TextToImage(&item.textBlockLayouts[0], item.TextInfo())
			xDots := int(math.Round(dotsFromUnitsFloat(left+item.xOffset, density)))
			yDots := int(math.Round(dotsFromUnitsFloat(top+item.yOffset, density)))
			draw.Draw(dst, image.Rect(xDots, yDots, xDots+textImage.Bounds().Size().X, yDots+textImage.Bounds().Size().Y), textImage, image.Point{}, draw.Over)
		}

		if item.itemType == ItemTypeImage && len(item.Setting("name")) == 0 {
			picture, err := imaging.Open(item.Setting("image"))
			if err != nil {
				log.Fatalf("failed to open image: %v", err)
			}
			// Resize the cropped image to width = 200px preserving the aspect ratio.
			imageWidthDots := int(math.Round(dotsFromUnitsFloat(item.imageWidth, density)))
			imageHeightDots := int(math.Round(dotsFromUnitsFloat(item.imageHeight, density)))
			picture = imaging.Resize(picture, imageWidthDots, imageHeightDots, imaging.Lanczos)
			textWidthDots := int(math.Round(dotsFromUnitsFloat(item.textWidth, density)))
			textHeightDots := int(math.Round(dotsFromUnitsFloat(item.textHeight, density)))
			xDots := int(math.Round(dotsFromUnitsFloat(left+item.xOffset, density)))
			yDots := int(math.Round(dotsFromUnitsFloat(top+item.yOffset, density)))

			if len(item.textBlockLayouts) > 0 {
				captionGutterDots := int(math.Round(dotsFromUnitsFloat(item.FloatSetting("caption-gutter"), density)))
				textBlockLayout := TextBlockLayout{}
				if item.bestTextBlockLayout >= 0 {
					textBlockLayout = item.textBlockLayouts[item.bestTextBlockLayout]
				}
				textImage := TextToImage(&textBlockLayout, item.TextInfo())
				captionWidthDots := int(math.Round(dotsFromUnitsFloat(textBlockLayout.width, density)))

				xDotsImage := xDots
				if captionWidthDots > imageWidthDots {
					xDotsImage = xDots + (captionWidthDots-imageWidthDots)/2
				}
				draw.Draw(dst, image.Rect(xDotsImage, yDots, xDotsImage+imageWidthDots, yDots+imageHeightDots), picture, image.Point{}, draw.Over)

				xDotsText := xDots
				if captionWidthDots < imageWidthDots {
					xDotsText = xDots + (imageWidthDots-captionWidthDots)/2
				}

				yDots += imageHeightDots + captionGutterDots
				draw.Draw(dst, image.Rect(xDotsText, yDots, xDotsText+textWidthDots, yDots+textHeightDots), textImage, image.Point{}, draw.Over)
			} else {
				draw.Draw(dst, image.Rect(xDots, yDots, xDots+imageWidthDots, yDots+imageHeightDots), picture, image.Point{}, draw.Over)
			}
		}
	}

	if dst != nil {
		writePage(dst, curPage)
	}
}

func main() {
	flag.Parse()

	if *inFileFlag == "" {
		flag.Usage()
		log.Fatal("no input file specified")
	}

	items := ReadPbFile(*inFileFlag)
	getImageDimensions(items)
	getTextDimensions(items)

	// break into columns, rows
	items = breakIntoPages(items)
	//  calculate sizes that fills available space; determine positions on page
	//items = layout(items)

	renderPages(items)

	fmt.Println(printItems(items))
}
