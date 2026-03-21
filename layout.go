package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
)

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

func breakIntoPages(items []PbItem) *PbBook {
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

	return ToPbBook(items)
}

func loadResizeCache() map[string]string {
	cache := map[string]string{}

	if *nocacheFlag {
		return cache
	}

	bytes, err := os.ReadFile(".pbresizecache")
	if err == nil {
		json.Unmarshal(bytes, &cache)
	}

	return cache
}

func saveResizeCache(cache *map[string]string) {
	bytes, err := json.Marshal(cache)
	if err == nil {
		os.WriteFile(".pbresizecache", bytes, 0666)
	}
}

func checkResizeCacheEntry(cache *map[string]string, jsonValue string) (string, string) {
	hashbytes := sha256.Sum256([]byte(jsonValue))
	jsonhash := hex.EncodeToString(hashbytes[:])
	if entry, exists := (*cache)[jsonhash]; exists {
		return entry, jsonhash
	}
	return "", jsonhash
}

func updateResizeCacheEntry(cache *map[string]string, jsonValue string, jsonhash string) {
	(*cache)[jsonhash] = jsonValue
}

func spaceToDistribute(page *PbPage, row *PbRow, column *PbColumn) (float64, float64, bool) {
	spareRowWidth := page.availableWidth - row.width()
	sparePageHeight := page.availableHeight - page.height()
	spareColumnHeight := row.height() - column.height()
	return spareRowWidth, math.Max(sparePageHeight, spareColumnHeight), spareRowWidth > 0 && (sparePageHeight > 0 || spareColumnHeight > 0)
}

func resizeItem(itemColumnItemNum int, itemColumnNum int, itemRowNum int, pbPage *PbPage, dx float64, dy float64) bool {
	pbRow := &pbPage.rows[itemRowNum]
	pbColumn := &pbRow.columns[itemColumnNum]
	pbColumnItem := &pbColumn.items[itemColumnItemNum]
	pbItem := pbColumnItem.item

	if pbItem.itemType != ItemTypeImage {
		return false
	}

	amount := 1.0 / pbItem.Density() * pbColumnItem.weight

	oldColumnWidth := pbColumn.width()
	oldRowHeight := pbRow.height()

	deltaWidth, deltaHeight := pbItem.enlargeImage(amount, dx, dy)
	if deltaWidth == 0 && deltaHeight == 0 {
		return false
	}

	newColumnWidth := pbColumn.width()
	deltaColumnWidth := newColumnWidth - oldColumnWidth

	for rowNum := range pbPage.rows {
		for columnNum := range pbPage.rows[rowNum].columns {
			for columnItemNum := range pbPage.rows[rowNum].columns[columnNum].items {
				if rowNum == itemRowNum && columnNum == itemColumnNum && columnItemNum > itemColumnItemNum {
					// Below the item in the same column, move the item down
					pbPage.rows[rowNum].columns[columnNum].items[columnItemNum].item.yOffset += deltaHeight
				} else if rowNum == itemRowNum && columnNum > itemColumnNum {
					// later in same row, move the item to the right
					pbPage.rows[rowNum].columns[columnNum].items[columnItemNum].item.xOffset += deltaColumnWidth
				}
			}
		}
	}

	pbPage.updateOffsets()

	newRowHeight := pbRow.height()
	deltaRowHeight := newRowHeight - oldRowHeight

	for rowNum := range pbPage.rows {
		for columnNum := range pbPage.rows[rowNum].columns {
			for columnItemNum := range pbPage.rows[rowNum].columns[columnNum].items {
				if rowNum > itemRowNum {
					// later row, move the item down
					pbPage.rows[rowNum].columns[columnNum].items[columnItemNum].item.yOffset += deltaRowHeight
				}
			}
		}
	}

	pbPage.updateOffsets()

	return true
}

type ResizeCacheEntryItem struct {
	TextWidth           float64
	TextHeight          float64
	BestTextBlockLayout int
	ImageWidth          float64
	ImageHeight         float64
	XOffset             float64
	YOffset             float64
}

type ResizeCacheEntryColumn struct {
	Items []ResizeCacheEntryItem
}

type ResizeCacheEntryRow struct {
	Columns []ResizeCacheEntryColumn
}

type ResizeCacheEntry struct {
	Rows []ResizeCacheEntryRow
}

func serializePage(page *PbPage) string {
	entry := ResizeCacheEntry{}

	for row := range page.rows {
		entry.Rows = append(entry.Rows, ResizeCacheEntryRow{})
		for column := range page.rows[row].columns {
			entry.Rows[row].Columns = append(entry.Rows[row].Columns, ResizeCacheEntryColumn{})
			for ii, columnItem := range page.rows[row].columns[column].items {
				entry.Rows[row].Columns[column].Items = append(entry.Rows[row].Columns[column].Items, ResizeCacheEntryItem{})
				entry.Rows[row].Columns[column].Items[ii].TextWidth = columnItem.item.textWidth
				entry.Rows[row].Columns[column].Items[ii].TextHeight = columnItem.item.textHeight
				entry.Rows[row].Columns[column].Items[ii].BestTextBlockLayout = columnItem.item.bestTextBlockLayout
				entry.Rows[row].Columns[column].Items[ii].ImageWidth = columnItem.item.imageWidth
				entry.Rows[row].Columns[column].Items[ii].ImageHeight = columnItem.item.imageHeight
				entry.Rows[row].Columns[column].Items[ii].XOffset = columnItem.item.xOffset
				entry.Rows[row].Columns[column].Items[ii].YOffset = columnItem.item.yOffset
			}
		}
	}

	bytes, _ := json.Marshal(&entry)
	return string(bytes[:])
}

func deserializePage(jsonValue string, page *PbPage) {
	var entry ResizeCacheEntry
	json.Unmarshal([]byte(jsonValue), &entry)

	for row := range page.rows {
		for column := range page.rows[row].columns {
			for ii := range page.rows[row].columns[column].items {
				page.rows[row].columns[column].items[ii].item.textWidth = entry.Rows[row].Columns[column].Items[ii].TextWidth
				page.rows[row].columns[column].items[ii].item.textHeight = entry.Rows[row].Columns[column].Items[ii].TextHeight
				page.rows[row].columns[column].items[ii].item.bestTextBlockLayout = entry.Rows[row].Columns[column].Items[ii].BestTextBlockLayout
				page.rows[row].columns[column].items[ii].item.imageWidth = entry.Rows[row].Columns[column].Items[ii].ImageWidth
				page.rows[row].columns[column].items[ii].item.imageHeight = entry.Rows[row].Columns[column].Items[ii].ImageHeight
				page.rows[row].columns[column].items[ii].item.xOffset = entry.Rows[row].Columns[column].Items[ii].XOffset
				page.rows[row].columns[column].items[ii].item.yOffset = entry.Rows[row].Columns[column].Items[ii].YOffset
			}
		}
	}
}

func resizePages(pb *PbBook, outPageRange string) {
	resizeCache := loadResizeCache()

	for pp := range pb.pages {
		if isPageInRange(outPageRange, pp) || isCurrentPage(pb, pp) {
			changed := false
			if changed, _ = fileChanged(*inFileFlag, lastModTime); changed {
				break
			}
			page := &pb.pages[pp]
			sPage := serializePage(page)
			if newSPage, jsonhash := checkResizeCacheEntry(&resizeCache, sPage); newSPage != "" {
				deserializePage(newSPage, page)
				page.updateOffsets()
			} else {
				for {
					resized := false
					for row := range page.rows {
						for column := range page.rows[row].columns {
							for item := range page.rows[row].columns[column].items {
								if dx, dy, canResize := spaceToDistribute(page, &page.rows[row], &page.rows[row].columns[column]); canResize {
									resizedOne := resizeItem(item, column, row, page, dx, dy)
									resized = resized || resizedOne
								}
							}
						}
					}
					if !resized {
						break
					}
				}

				sPage = serializePage(page)
				updateResizeCacheEntry(&resizeCache, sPage, jsonhash)

				if globalVerboseFlag&4 != 0 {
					log.Printf("Resized Page %v / %v", pp+1, len(pb.pages))
				}
			}
		}
	}

	saveResizeCache(&resizeCache)
}

func BindingAlign(align int, binding int, pageNum int) int {
	if binding == BindingTop && (align == AlignEdge && pageNum%2 == 1 || align == AlignBinding && pageNum%2 == 0) {
		return AlignBottom
	}
	if binding == BindingTop && (align == AlignEdge && pageNum%2 == 0 || align == AlignBinding && pageNum%2 == 1) {
		return AlignTop
	}
	if binding == BindingSide && (align == AlignEdge && pageNum%2 == 1 || align == AlignBinding && pageNum%2 == 0) {
		return AlignRight
	}
	if binding == BindingSide && (align == AlignEdge && pageNum%2 == 0 || align == AlignBinding && pageNum%2 == 1) {
		return AlignLeft
	}
	if binding == BindingTop && (align == AlignSpreadEdge && pageNum%2 == 1 || align == AlignSpreadBinding && pageNum%2 == 0) {
		return AlignSpreadBottom
	}
	if binding == BindingTop && (align == AlignSpreadEdge && pageNum%2 == 0 || align == AlignSpreadBinding && pageNum%2 == 1) {
		return AlignSpreadTop
	}
	if binding == BindingSide && (align == AlignSpreadEdge && pageNum%2 == 1 || align == AlignSpreadBinding && pageNum%2 == 0) {
		return AlignSpreadRight
	}
	if binding == BindingSide && (align == AlignSpreadEdge && pageNum%2 == 0 || align == AlignSpreadBinding && pageNum%2 == 1) {
		return AlignSpreadLeft
	}
	return align
}

func layoutPages(pbBook *PbBook, outPageRange string) {
	binding := BindingUnknown
	if len(pbBook.pages) > 0 {
		binding = pbBook.pages[0].rows[0].columns[0].items[0].item.Binding()
	}

	for pp := range pbBook.pages {
		if isPageInRange(outPageRange, pp) || isCurrentPage(pbBook, pp) {
			page := &pbBook.pages[pp]
			// pageHeight := page.height()
			for row := range page.rows {
				rowHeight := page.rows[row].height()
				// rowWidth := page.rows[row].width()
				for column := range page.rows[row].columns {
					columnWidth := page.rows[row].columns[column].width() // distribute items across this width
					for item := range page.rows[row].columns[column].items {
						w, _ := page.rows[row].columns[column].items[item].item.Size()
						if w < columnWidth {
							itemAlign := page.rows[row].columns[column].items[item].item.Align("item-align")
							switch BindingAlign(itemAlign, binding, pp) {
							case AlignRight:
								page.rows[row].columns[column].items[item].item.xOffset += columnWidth - w
							case AlignCenter:
								page.rows[row].columns[column].items[item].item.xOffset += (columnWidth - w) / 2
							}
						}
					}
					extraColumnHeight := rowHeight - page.rows[row].columns[column].height() // distribute items across this height
					if extraColumnHeight > 0 {
						columnDistribute := AlignTop
						if len(page.rows[row].columns[column].items) > 0 {
							columnDistribute = page.rows[row].columns[column].items[0].item.Align("column-distribute")
						}
						switch BindingAlign(columnDistribute, binding, pp) {
						case AlignBottom:
							for item := range page.rows[row].columns[column].items {
								page.rows[row].columns[column].items[item].item.yOffset += extraColumnHeight
							}
						case AlignMiddle:
							for item := range page.rows[row].columns[column].items {
								page.rows[row].columns[column].items[item].item.yOffset += extraColumnHeight / 2
							}
						case AlignJustify:
							if len(page.rows[row].columns[column].items) == 1 {
								page.rows[row].columns[column].items[0].item.yOffset += extraColumnHeight / 2
							} else {
								interSpace := extraColumnHeight / float64(len(page.rows[row].columns[column].items)-1)
								for item := range page.rows[row].columns[column].items {
									page.rows[row].columns[column].items[item].item.yOffset += interSpace * float64(item)
								}
							}
						case AlignSpreadTop:
							interSpace := extraColumnHeight / float64(len(page.rows[row].columns[column].items))
							for item := range page.rows[row].columns[column].items {
								page.rows[row].columns[column].items[item].item.yOffset += interSpace * float64(item)
							}
						case AlignSpreadBottom:
							interSpace := extraColumnHeight / float64(len(page.rows[row].columns[column].items))
							for item := range page.rows[row].columns[column].items {
								page.rows[row].columns[column].items[item].item.yOffset += interSpace * float64(item+1)
							}
						case AlignSpreadMiddle:
							interSpace := extraColumnHeight / float64(len(page.rows[row].columns[column].items)+1)
							for item := range page.rows[row].columns[column].items {
								page.rows[row].columns[column].items[item].item.yOffset += interSpace * float64(item+1)
							}
						}
					}
				}

				extraRowWidth := page.availableWidth - page.rows[row].width()
				if extraRowWidth > 0 {
					rowDistribute := AlignLeft
					if len(page.rows[row].columns[0].items) > 0 {
						rowDistribute = page.rows[row].columns[0].items[0].item.Align("row-distribute")
					}
					switch BindingAlign(rowDistribute, binding, pp) {
					case AlignRight:
						for column := range page.rows[row].columns {
							for item := range page.rows[row].columns[column].items {
								page.rows[row].columns[column].items[item].item.xOffset += extraRowWidth
							}
						}
					case AlignCenter:
						for column := range page.rows[row].columns {
							for item := range page.rows[row].columns[column].items {
								page.rows[row].columns[column].items[item].item.xOffset += extraRowWidth / 2
							}
						}
					case AlignJustify:
						if len(page.rows[row].columns) == 1 {
							for item := range page.rows[row].columns[0].items {
								page.rows[row].columns[0].items[item].item.xOffset += extraRowWidth / 2
							}
						} else {
							interSpace := extraRowWidth / float64(len(page.rows[row].columns)-1)
							for column := range page.rows[row].columns {
								for item := range page.rows[row].columns[column].items {
									page.rows[row].columns[column].items[item].item.xOffset += interSpace * float64(column)
								}
							}
						}
					case AlignSpreadLeft:
						interSpace := extraRowWidth / float64(len(page.rows[row].columns))
						for column := range page.rows[row].columns {
							for item := range page.rows[row].columns[column].items {
								page.rows[row].columns[column].items[item].item.xOffset += interSpace * float64(column)
							}
						}
					case AlignSpreadRight:
						interSpace := extraRowWidth / float64(len(page.rows[row].columns))
						for column := range page.rows[row].columns {
							for item := range page.rows[row].columns[column].items {
								page.rows[row].columns[column].items[item].item.xOffset += interSpace * float64(column+1)
							}
						}
					case AlignSpreadCenter:
						interSpace := extraRowWidth / float64(len(page.rows[row].columns)+1)
						for column := range page.rows[row].columns {
							for item := range page.rows[row].columns[column].items {
								page.rows[row].columns[column].items[item].item.xOffset += interSpace * float64(column+1)
							}
						}
					}
				}
			}

			extraPageHeight := page.availableHeight - page.height()
			if extraPageHeight > 0 {
				pageDistribute := AlignTop
				if len(page.rows[0].columns[0].items) > 0 {
					pageDistribute = page.rows[0].columns[0].items[0].item.Align("page-distribute")
				}
				switch BindingAlign(pageDistribute, binding, pp) {
				case AlignBottom:
					for row := range page.rows {
						for column := range page.rows[row].columns {
							for item := range page.rows[row].columns[column].items {
								page.rows[row].columns[column].items[item].item.yOffset += extraPageHeight
							}
						}
					}
				case AlignMiddle:
					for row := range page.rows {
						for column := range page.rows[row].columns {
							for item := range page.rows[row].columns[column].items {
								page.rows[row].columns[column].items[item].item.yOffset += extraPageHeight / 2
							}
						}
					}
				case AlignJustify:
					if len(page.rows) == 1 {
						for column := range page.rows[0].columns {
							for item := range page.rows[0].columns[column].items {
								page.rows[0].columns[column].items[item].item.yOffset += extraPageHeight / 2
							}
						}
					} else {
						interSpace := extraPageHeight / float64(len(page.rows)-1)
						for row := range page.rows {
							for column := range page.rows[row].columns {
								for item := range page.rows[row].columns[column].items {
									page.rows[row].columns[column].items[item].item.yOffset += interSpace * float64(row)
								}
							}
						}
					}
				case AlignSpreadTop:
					interSpace := extraPageHeight / float64(len(page.rows))
					for row := range page.rows {
						for column := range page.rows[row].columns {
							for item := range page.rows[row].columns[column].items {
								page.rows[row].columns[column].items[item].item.yOffset += interSpace * float64(row)
							}
						}
					}
				case AlignSpreadBottom:
					interSpace := extraPageHeight / float64(len(page.rows))
					for row := range page.rows {
						for column := range page.rows[row].columns {
							for item := range page.rows[row].columns[column].items {
								page.rows[row].columns[column].items[item].item.yOffset += interSpace * float64(row+1)
							}
						}
					}
				case AlignSpreadMiddle:
					interSpace := extraPageHeight / float64(len(page.rows)+1)
					for row := range page.rows {
						for column := range page.rows[row].columns {
							for item := range page.rows[row].columns[column].items {
								page.rows[row].columns[column].items[item].item.yOffset += interSpace * float64(row+1)
							}
						}
					}
				}
			}

			page.updateOffsets()
		}
	}
}
