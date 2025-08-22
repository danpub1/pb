package main

import (
	"bufio"
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"math"
	"os"
	"strings"
)

var (
	inFileFlag = flag.String("i", "", `pb input filename"`)
)

func getImageDimensions(items []PbItem) {
	if len(items) == 0 {
		return
	}

	for _, theItem := range items {
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

			theItem.Set("image-dimensions", fmt.Sprintf("%vx%v", config.Width, config.Height))
		}
	}
}

func getTextDimensions(items []PbItem) {
	if len(items) == 0 {
		return
	}

	for _, item := range items {
		if item.itemType == ItemTypeText && len(item.Setting("text")) > 0 {
			width := ParseWidth(item.Setting("text-width"), item.Setting("page-size"), item.Setting("margin"))
			item.textBlockLayouts = MeasureText(item.Setting("text"), width, 0.0, item.TextInfo())
			if len(item.textBlockLayouts) == 1 {
				item.Set("text-dimensions", fmt.Sprintf("%vx%v", item.textBlockLayouts[0].width, item.textBlockLayouts[0].height))
			}
			TextToImage(&item.textBlockLayouts[0], item.TextInfo())
		} else if item.itemType == ItemTypeImage && len(item.Setting("text")) > 0 {
			maxWidth := MaxWidth(item.Setting("page-size"), item.Setting("margin"))
			maxHeight := MaxHeight(item.Setting("page-size"), item.Setting("margin"))
			item.textBlockLayouts = MeasureText(item.Setting("text"), maxWidth, maxHeight, item.TextInfo())
			var dimensions strings.Builder
			for _, layout := range item.textBlockLayouts {
				if dimensions.Len() > 0 {
					dimensions.WriteRune(';')
				}
				dimensions.WriteString(fmt.Sprintf("%vx%v", layout.width, layout.height))
			}
			if dimensions.Len() > 0 {
				item.Set("text-dimensions", dimensions.String())
			}
		}
	}
}

func insertItemBefore(items []PbItem, item PbItem, idx int) []PbItem {
	if idx == 0 {
		items = append([]PbItem{item}, items...)
	} else if idx < len(items) {
		items = append(items[:idx], append([]PbItem{item}, items[idx:]...)...)
	} else if idx == len(items) {
		items = append(items, item)
	}

	return items
}

type ScaleableItem struct {
	index        int
	weight       int
	width        float64
	maxWidth     float64
	frame        TRBL
	aspect       float64
	captionWidth float64
}

func scaleRow(items []PbItem, beginIdx int, endIdx int) {
	if beginIdx >= 0 && beginIdx < endIdx && beginIdx < len(items) && endIdx < len(items) {
		pageWidth, _ := Size(items[beginIdx].Setting("page-size"))
		extraWidth := pageWidth - items[beginIdx].width
		if extraWidth > 0 {
			scaleableItems := []ScaleableItem{}
			totalWeights := 0.0
			scaleableCount := 0
			for idx := beginIdx; idx <= endIdx; idx++ {
				if items[idx].itemType == ItemTypeImage {
					var scaleableItem ScaleableItem
					frame := items[idx].Frame("image-frame")
					scaleableItem.index = idx
					scaleableItem.weight = Atoi(items[idx].Setting("image-weight"))
					scaleableItem.maxWidth = ParseImageWidth("max-width", items[idx])
					if len(items[idx].textBlockLayouts) > 0 {
						scaleableItem.captionWidth = Atof(items[idx].Setting("caption-width")) / 100.0
					} else {
						scaleableItem.captionWidth = 1.0
					}
					scaleableItem.width = items[idx].width
					scaleableItem.frame = frame.size
					scaleableItem.aspect = items[idx].Aspect()
					scaleableItems = append(scaleableItems, scaleableItem)
					totalWeights += float64(scaleableItem.weight)
					if scaleableItem.width < scaleableItem.maxWidth {
						scaleableCount++
					}
				}
			}

			for scaleableCount > 0 && extraWidth > 0 {
				tempTotalWeights := 0.0
				extraWidthUsed := 0.0
				for idx, scaleableItem := range scaleableItems {
					if scaleableItem.width < scaleableItem.maxWidth {
						weight := float64(scaleableItem.weight) * scaleableItem.captionWidth
						thisExtraWidthUsed := extraWidth * weight / float64(totalWeights)
						orgWidth := scaleableItems[idx].width
						scaleableItems[idx].width = math.Min(scaleableItems[idx].width+thisExtraWidthUsed/scaleableItem.captionWidth, scaleableItem.maxWidth)
						thisExtraWidthUsed = (scaleableItems[idx].width - orgWidth) * scaleableItem.captionWidth
						extraWidthUsed += thisExtraWidthUsed
						if scaleableItems[idx].width < scaleableItem.maxWidth {
							tempTotalWeights += float64(scaleableItem.weight)
						} else {
							scaleableCount--
						}
					}
				}
				totalWeights = tempTotalWeights
				extraWidth -= extraWidthUsed
				extraWidth = math.Round(extraWidth*100.0) / 100.0
			}

			for _, scaleableItem := range scaleableItems {
				items[scaleableItem.index].width = scaleableItem.width
				items[scaleableItem.index].height = scaleableItem.width / scaleableItem.aspect
				items[scaleableItem.index].Set("layout-width", fmt.Sprintf("%v", scaleableItem.width))
				items[scaleableItem.index].Set("layout-height", fmt.Sprintf("%v", items[scaleableItem.index].height))
			}
		}
	}
}

func scaleRows(items []PbItem) {
	rowIdx := -1
	for idx, item := range items {
		if item.itemType == ItemTypeRow {
			if rowIdx != -1 {
				scaleRow(items, rowIdx, idx-1)
			}
			rowIdx = idx
		}
	}
}

func breakIntoRows(items []PbItem) []PbItem {
	if len(items) == 0 {
		return items
	}

	pageWidth := 0.0
	curGutter := 0.0
	curWidth := 0.0
	curItems := 0
	curRowIdx := -1
	for ii := 0; ii < len(items); ii++ {
		switch items[ii].itemType {
		case ItemTypePage:
			pageWidth, _ = Size(items[ii].Setting("page-size"))
			if curRowIdx >= 0 {
				items[curRowIdx].width = curWidth
				curRowIdx = -1
			}
			curWidth = 0.0
			curItems = 0
		case ItemTypeRow:
			if curRowIdx >= 0 {
				items[curRowIdx].width = curWidth
			}
			curWidth = 0.0
			curItems = 0
			curGutter = Atof(items[ii].Setting("v-gutter"))
			curRowIdx = ii
		case ItemTypeImage:
			fallthrough
		case ItemTypeText:
			// Every image or text should have a row
			if curRowIdx == -1 {
				var newRow PbItem
				newRow.itemType = ItemTypeRow
				newRow.settings = map[string]string{}
				items = insertItemBefore(items, newRow, ii)
				curRowIdx = ii
				ii++
			}
			thisAdd := 0.0
			if curItems > 0 {
				thisAdd += curGutter
			}
			var thisWidth float64
			thisWidthWithText := 0.0
			if items[ii].itemType == ItemTypeText {
				thisWidth = ParseWidth(items[ii].Setting("text-width"), items[ii].Setting("page-size"), items[ii].Setting("margin"))
			} else {
				thisWidth = ParseImageWidth("min-width", items[ii])

				if len(items[ii].textBlockLayouts) > 0 {
					captionWidthPercent := Atof(items[ii].Setting("caption-width"))
					if captionWidthPercent > 100 {
						thisWidthWithText = captionWidthPercent * thisWidth / 100.0
					}
				}
			}

			items[ii].width = thisWidth
			if thisWidthWithText > 0 && thisWidthWithText > thisWidth {
				thisWidth = thisWidthWithText
			}
			thisAdd += thisWidth
			if curWidth+thisAdd > pageWidth {
				items[curRowIdx].width = curWidth
				items = insertItemBefore(items, items[curRowIdx], ii)
				curRowIdx = ii
				ii++
				curWidth = 0.0
				curItems = 0
				thisAdd = thisWidth
			}
			curWidth += thisAdd
			curItems++
		}
	}

	if curRowIdx >= 0 {
		items[curRowIdx].width = curWidth
	}

	return items
}

func layoutRow(items []PbItem, beginIdx int, pageNum int) {
	if len(items) == 0 || beginIdx < 0 || beginIdx >= len(items) {
		return
	}

	rowHeight := 0.0
	for idx := beginIdx; idx < len(items) && items[idx].itemType != ItemTypeRow && items[idx].itemType != ItemTypePage && items[idx].itemType != ItemTypeBook; idx++ {
		rowHeight = math.Max(rowHeight, items[idx].height)
	}

	pageWidth, _ := Size(items[beginIdx].Setting("page-size"))
	extraWidth := pageWidth - items[beginIdx].width
	if extraWidth > 0 { // distribute extraWith according to row-align
		//rowAlign := items[beginIdx].Align("row-align")
		for idx := beginIdx; idx < len(items) && items[idx].itemType != ItemTypeRow && items[idx].itemType != ItemTypePage && items[idx].itemType != ItemTypeBook; idx++ {
		}
	}
}

func layoutPages(items []PbItem) []PbItem {

	for idx := 0; idx < len(items); idx++ {
		item := items[idx]

		curPage := -1
		switch item.itemType {
		case ItemTypeBook:
		case ItemTypePage:
			curPage++
		case ItemTypeRow:
		case ItemTypeImage:
		case ItemTypeText:
		}
	}
	return items
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
	items = breakIntoRows(items)
	scaleRows(items)
	items = layoutPages(items)

	fmt.Println(printItems(items, 0))
}
