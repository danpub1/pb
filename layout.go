package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math"
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

			captionGutter := items[ii].CaptionGutter()

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

	pageHasContent bool

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

		pageHasContent: source.pageHasContent,

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
		captionGutter := items[ii].CaptionGutter()

		itemWidth := math.Max(items[ii].textWidth, items[ii].imageWidth)
		itemHeight := items[ii].imageHeight + captionGutter + items[ii].textHeight

		s.pageWidth, s.pageHeight = items[ii].pageDimensions()
		items[ii].inLayout = true

		if ii > 0 {
			items[ii].page = items[ii-1].page
			items[ii].row = items[ii-1].row
			items[ii].column = items[ii-1].column
			items[ii].xOffset = items[ii-1].xOffset
			items[ii].yOffset = items[ii-1].yOffset
		} else {
			items[ii].page = 0
			items[ii].row = 0
			items[ii].column = 0
			items[ii].xOffset = 0
			items[ii].yOffset = 0
		}

		if items[ii].itemType == ItemTypeBook && s.pagesInBook == 0 || (items[ii].itemType == ItemTypePage || items[ii].BoolSetting("page-break")) && (s.rowsOnPage > 0 || s.pageHasContent) {
			if ii > 0 {
				items[ii].page = items[ii-1].page + 1
			} else {
				items[ii].page = 0
			}
			items[ii].row = 0
			items[ii].column = 0
			items[ii].xOffset = 0
			items[ii].yOffset = 0
			s.rowsOnPage = 0
			s.columnsInRow = 0
			s.itemsInColumn = 0
			s.pageHasContent = false
			s.curRowYOffset = 0
			s.curRowHeight = 0
			s.curColumnXOffset = 0
			s.curColumnWidth = 0
			s.curColumnHeight = 0
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
			s.curRowYOffset = s.curRowYOffset + s.curRowHeight + items[ii].FloatPageSetting("row-gutter")
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
			s.curColumnXOffset = s.curColumnXOffset + s.curColumnWidth + items[ii].FloatRowSetting("column-gutter")
			s.curColumnWidth = 0
			s.curColumnHeight = 0
			items[ii].xOffset = s.curColumnXOffset
			items[ii].yOffset = s.curRowYOffset
		}

		if items[ii].itemType == ItemTypeImage || items[ii].itemType == ItemTypeText {
			isFloated := len(items[ii].Setting("float")) != 0
			isNotInLayout := isFloated || len(items[ii].Setting("name")) != 0

			columnItemGutter := 0.0
			if s.itemsInColumn > 0 && !isNotInLayout {
				columnItemGutter = items[ii].FloatColumnSetting("item-gutter")
			}
			rowColumnGutter := 0.0
			if s.columnsInRow > 0 && !isNotInLayout {
				rowColumnGutter = items[ii].FloatRowSetting("column-gutter")
			}
			pageRowGutter := 0.0
			if s.rowsOnPage > 0 && !isNotInLayout {
				pageRowGutter = items[ii].FloatPageSetting("row-gutter")
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

			items[ii].inLayout = true
			if isFloated {
				items[ii].inLayout = false
				s.pageHasContent = true
			} else if isNotInLayout {
				items[ii].inLayout = false
			} else if curItemYOffset+itemHeight <= s.pageHeight && s.curColumnXOffset+itemWidth <= s.pageWidth {
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
					VerboseLog(fmt.Sprintf("Column too wide, moving column to next row at %v\n", ii))
					items[startOfColumn].Set("row-break", "true")
					ii = startOfColumn - 1
					s = stateStack[ii].DeepCopy()
				} else { // Column is too wide and there is not room for it in the next
					if items[ii].BoolColumnSetting("keep-columns-together") {
						VerboseLog(fmt.Sprintf("Column too wide, Moving column to next page at %v\n", ii))
						items[startOfColumn].Set("page-break", "true")
						ii = startOfColumn - 1
						s = stateStack[ii].DeepCopy()
					} else { // just breaking at the item
						if s.curRowYOffset+s.curRowHeight+pageRowGutter+itemHeight < s.pageHeight {
							VerboseLog(fmt.Sprintf("Column too wide, breaking row at %v\n", ii))
							items[ii].Set("row-break", "true")
							ii = ii - 1
							s = stateStack[ii].DeepCopy()
						} else {
							VerboseLog(fmt.Sprintf("Column too wide, breaking page at %v\n", ii))
							items[ii].Set("page-break", "true")
							ii = ii - 1
							s = stateStack[ii].DeepCopy()
						}
					}
				}
			} else {
				// Is there room for another column?
				if s.itemsInColumn > 0 && s.curColumnXOffset+s.curColumnWidth+rowColumnGutter+itemWidth < s.pageWidth {
					VerboseLog(fmt.Sprintf("Column too tall, breaking column at %v\n", ii))
					items[ii].Set("column-break", "true")
					ii = ii - 1
					s = stateStack[ii].DeepCopy()
				} else {
					// TODO: If this is the only thing on the page and it's still too big? (Endless loop)
					VerboseLog(fmt.Sprintf("Column too tall, breaking page at %v\n", ii))
					items[ii].Set("page-break", "true")
					ii = ii - 1
					s = stateStack[ii].DeepCopy()
				}
			}
		}
	}

	pbBook := ToPbBook(items)

	for pp, page := range pbBook.pages {
		rowLengths := make([]float64, 0)
		for row := range page.rows {
			for column := range page.rows[row].columns {
				for item := range page.rows[row].columns[column].items {
					if page.rows[row].columns[column].items[item].item.inLayout {
						rowLengths = accrueRowLength(rowLengths, row, page.rows[row].columns[column].items[item].item)
					}
				}
			}
		}
		shortestRow := -1.0
		longestRow := -1.0
		for _, rowLength := range rowLengths {
			if shortestRow == -1.0 || shortestRow > rowLength {
				shortestRow = rowLength
			}
			if longestRow == -1.0 || longestRow < rowLength {
				longestRow = rowLength
			}
		}

		if Opts.Verbose("DD") {
			log.Printf("Page %v: %v Rows, Row Length Ratio: %v\n", pp+1, len(page.rows), shortestRow/longestRow)
		}

	}
	return pbBook
}

var firstTimeResizeCache bool = true

func loadResizeCache() map[string]string {
	cache := map[string]string{}

	// if Opts.Cache()&CacheModeResize == 0 || Opts.Cache()&CacheModeResizeDuring != 0 && firstTimeResizeCache {
	// 	firstTimeResizeCache = false
	// 	return cache
	// }

	// bytes, err := os.ReadFile(".pbresizecache")
	// if err == nil {
	// 	json.Unmarshal(bytes, &cache)
	// }

	return cache
}

func saveResizeCache(cache *map[string]string) {
	// bytes, err := json.Marshal(cache)
	// if err == nil {
	// 	os.WriteFile(".pbresizecache", bytes, 0666)
	// }
}

func checkResizeCacheEntry(_ /*cache*/ *map[string]string, jsonValue string) (string, string) {
	hashbytes := sha256.Sum256([]byte(jsonValue))
	jsonhash := hex.EncodeToString(hashbytes[:])
	// if entry, exists := (*cache)[jsonhash]; exists {
	// 	return entry, jsonhash
	// }
	return "", jsonhash
}

func updateResizeCacheEntry(cache *map[string]string, jsonValue string, jsonhash string) {
	// (*cache)[jsonhash] = jsonValue
}

func spaceToDistribute(page *PbPage, row *PbRow, column *PbColumn, item *PbColumnItem) (float64, float64, bool) {
	spareRowWidth := page.availableWidth - row.width()
	spareColumnWidth := column.width() - item.width()
	sparePageHeight := page.availableHeight - page.height()
	spareColumnHeight := row.height() - column.height()
	return math.Max(spareRowWidth, spareColumnWidth), math.Max(sparePageHeight, spareColumnHeight), (spareRowWidth > 0 || spareColumnWidth > 0) && (sparePageHeight > 0 || spareColumnHeight > 0)
}

func resizeItem(itemColumnItemNum int, itemColumnNum int, itemRowNum int, pbPage *PbPage, dx float64, dy float64) bool {
	pbRow := &pbPage.rows[itemRowNum]
	pbColumn := &pbRow.columns[itemColumnNum]
	pbColumnItem := &pbColumn.items[itemColumnItemNum]
	pbItem := pbColumnItem.item

	if pbItem.itemType != ItemTypeImage {
		return false
	}

	amount := 1.0 / pbItem.Density()

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

func accrueRowLength(rowLengths []float64, row int, item *PbItem) []float64 {
	if item == nil || (item.itemType != ItemTypeImage && item.itemType != ItemTypeText) {
		return rowLengths
	}

	thisRow := 0.0
	if item.itemType == ItemTypeText {
		thisRow = item.textWidth
	} else {
		thisRow = max(item.imageWidth, item.textWidth)
	}

	if row == len(rowLengths) {
		rowLengths = append(rowLengths, thisRow)
	} else {
		rowLengths[row] += thisRow
	}

	return rowLengths
}

func resizePages(pb *PbBook, outPageRange string, firstIteration bool) {
	resizeCache := loadResizeCache()

	for pp := range pb.pages {
		if isPageInRange(outPageRange, pp, firstIteration) || isCurrentPage(pb, pp) {
			changed := false
			if changed, _ = fileChanged(inFiles, lastModTime); changed {
				break
			}
			page := &pb.pages[pp]

			if item := page.PbItem(); item != nil && item.BoolPageSetting("noresize") {
				continue
			}

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
								if page.rows[row].columns[column].items[item].item.inLayout {
									if dx, dy, canResize := spaceToDistribute(page, &page.rows[row], &page.rows[row].columns[column], &page.rows[row].columns[column].items[item]); canResize {
										resizedOne := resizeItem(item, column, row, page, dx, dy)
										resized = resized || resizedOne
									}
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

func NumItemLayout(items []PbColumnItem) int {
	numLayout := 0
	for ii := range items {
		if items[ii].item.inLayout {
			numLayout++
		}
	}
	return numLayout
}

func NumColumnLayout(items []PbColumn) int {
	numLayout := 0
	for ii := range items {
		if NumItemLayout(items[ii].items) > 0 {
			numLayout++
		}
	}
	return numLayout
}

func NumRowLayout(items []PbRow) int {
	numLayout := 0
	for ii := range items {
		if NumColumnLayout(items[ii].columns) > 0 {
			numLayout++
		}
	}
	return numLayout
}

func layoutPages(pbBook *PbBook, outPageRange string, firstIteration bool) {
	binding := BindingUnknown
	if len(pbBook.pages) > 0 {
		if item := pbBook.PbItem(); item != nil {
			binding = item.Binding()
		}
	}

	for pp := range pbBook.pages {
		if isPageInRange(outPageRange, pp, firstIteration) || isCurrentPage(pbBook, pp) {
			page := &pbBook.pages[pp]

			if item := page.PbItem(); item != nil && item.BoolPageSetting("nolayout") {
				continue
			}

			// pageHeight := page.height()
			for row := range page.rows {
				rowHeight := page.rows[row].height()
				// rowWidth := page.rows[row].width()
				for column := range page.rows[row].columns {
					columnWidth := page.rows[row].columns[column].width() // distribute items across this width
					for item := range page.rows[row].columns[column].items {

						if !page.rows[row].columns[column].items[item].item.inLayout {
							continue
						}

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
						if NumItemLayout(page.rows[row].columns[column].items) > 0 {
							columnDistribute = page.rows[row].columns[column].items[0].item.Align("distribute-items")
						}
						switch BindingAlign(columnDistribute, binding, pp) {
						case AlignBottom:
							for item := range page.rows[row].columns[column].items {
								if page.rows[row].columns[column].items[item].item.inLayout {
									page.rows[row].columns[column].items[item].item.yOffset += extraColumnHeight
								}
							}
						case AlignMiddle:
							for item := range page.rows[row].columns[column].items {
								if page.rows[row].columns[column].items[item].item.inLayout {
									page.rows[row].columns[column].items[item].item.yOffset += extraColumnHeight / 2
								}
							}
						case AlignJustify:
							if NumItemLayout(page.rows[row].columns[column].items) == 1 {
								if page.rows[row].columns[column].items[0].item.inLayout {
									page.rows[row].columns[column].items[0].item.yOffset += extraColumnHeight / 2
								}
							} else {
								interSpace := extraColumnHeight / float64(NumItemLayout(page.rows[row].columns[column].items)-1)
								numItem := 0
								for item := range page.rows[row].columns[column].items {
									if page.rows[row].columns[column].items[item].item.inLayout {
										page.rows[row].columns[column].items[item].item.yOffset += interSpace * float64(numItem)
										numItem++
									}
								}
							}
						case AlignSpreadTop:
							interSpace := extraColumnHeight / float64(NumItemLayout(page.rows[row].columns[column].items))
							numItem := 0
							for item := range page.rows[row].columns[column].items {
								if page.rows[row].columns[column].items[item].item.inLayout {
									page.rows[row].columns[column].items[item].item.yOffset += interSpace * float64(numItem)
									numItem++
								}
							}
						case AlignSpreadBottom:
							interSpace := extraColumnHeight / float64(NumItemLayout(page.rows[row].columns[column].items))
							numItem := 0
							for item := range page.rows[row].columns[column].items {
								if page.rows[row].columns[column].items[item].item.inLayout {
									page.rows[row].columns[column].items[item].item.yOffset += interSpace * float64(numItem+1)
									numItem++
								}
							}
						case AlignSpreadMiddle:
							spreadPercent := 50.0
							if columnItem := page.rows[row].columns[column].PbItem(); columnItem != nil {
								spreadPercent = columnItem.FloatColumnSetting("spread-percent")
							}
							interSpace := extraColumnHeight / float64(NumItemLayout(page.rows[row].columns[column].items)+1)
							numItem := 0
							for item := range page.rows[row].columns[column].items {
								if page.rows[row].columns[column].items[item].item.inLayout {
									if item == 0 {
										page.rows[row].columns[column].items[item].item.yOffset += interSpace * float64(numItem+1) * spreadPercent / 50.0
									} else {
										page.rows[row].columns[column].items[item].item.yOffset += interSpace * float64(numItem+1)
									}
									numItem++
								}
							}
						case AlignSpreadPack:
							// Same as Justify except for single item in column
							// Single item in column in first row == top
							// Single item in column in last row == bottom
							numItems := NumItemLayout(page.rows[row].columns[column].items)

							if row == 0 && numItems == 1 { // Top
								// Do nothing for top
							} else if row == len(page.rows)-1 && numItems == 1 { // Bottom
								for item := range page.rows[row].columns[column].items {
									if page.rows[row].columns[column].items[item].item.inLayout {
										page.rows[row].columns[column].items[item].item.yOffset += extraColumnHeight
									}
								}
							} else { // Justify
								if NumItemLayout(page.rows[row].columns[column].items) == 1 {
									if page.rows[row].columns[column].items[0].item.inLayout {
										page.rows[row].columns[column].items[0].item.yOffset += extraColumnHeight / 2
									}
								} else {
									interSpace := extraColumnHeight / float64(NumItemLayout(page.rows[row].columns[column].items)-1)
									numItem := 0
									for item := range page.rows[row].columns[column].items {
										if page.rows[row].columns[column].items[item].item.inLayout {
											page.rows[row].columns[column].items[item].item.yOffset += interSpace * float64(numItem)
											numItem++
										}
									}
								}
							}
						}
					}
				}

				extraRowWidth := page.availableWidth - page.rows[row].width()
				if extraRowWidth > 0 {
					rowDistribute := AlignLeft
					if NumItemLayout(page.rows[row].columns[0].items) > 0 {
						rowDistribute = page.rows[row].columns[0].items[0].item.Align("distribute-columns")
					}
					switch BindingAlign(rowDistribute, binding, pp) {
					case AlignRight:
						for column := range page.rows[row].columns {
							for item := range page.rows[row].columns[column].items {
								if page.rows[row].columns[column].items[item].item.inLayout {
									page.rows[row].columns[column].items[item].item.xOffset += extraRowWidth
								}
							}
						}
					case AlignCenter:
						for column := range page.rows[row].columns {
							for item := range page.rows[row].columns[column].items {
								if page.rows[row].columns[column].items[item].item.inLayout {
									page.rows[row].columns[column].items[item].item.xOffset += extraRowWidth / 2
								}
							}
						}
					case AlignJustify, AlignSpreadPack:
						if NumColumnLayout(page.rows[row].columns) == 1 {
							for item := range page.rows[row].columns[0].items {
								if page.rows[row].columns[0].items[item].item.inLayout {
									page.rows[row].columns[0].items[item].item.xOffset += extraRowWidth / 2
								}
							}
						} else {
							interSpace := extraRowWidth / float64(NumColumnLayout(page.rows[row].columns)-1)
							numColumn := 0
							for column := range page.rows[row].columns {
								for item := range page.rows[row].columns[column].items {
									if page.rows[row].columns[column].items[item].item.inLayout {
										page.rows[row].columns[column].items[item].item.xOffset += interSpace * float64(numColumn)
									}
								}
								if NumItemLayout(page.rows[row].columns[column].items) > 0 {
									numColumn++
								}
							}
						}
					case AlignSpreadLeft:
						interSpace := extraRowWidth / float64(NumColumnLayout(page.rows[row].columns))
						numColumn := 0
						for column := range page.rows[row].columns {
							for item := range page.rows[row].columns[column].items {
								if page.rows[row].columns[column].items[item].item.inLayout {
									page.rows[row].columns[column].items[item].item.xOffset += interSpace * float64(numColumn)
								}
							}
							if NumItemLayout(page.rows[row].columns[column].items) > 0 {
								numColumn++
							}
						}
					case AlignSpreadRight:
						interSpace := extraRowWidth / float64(NumColumnLayout(page.rows[row].columns))
						numColumn := 0
						for column := range page.rows[row].columns {
							for item := range page.rows[row].columns[column].items {
								if page.rows[row].columns[column].items[item].item.inLayout {
									page.rows[row].columns[column].items[item].item.xOffset += interSpace * float64(numColumn+1)
								}
							}
							if NumItemLayout(page.rows[row].columns[column].items) > 0 {
								numColumn++
							}
						}
					case AlignSpreadCenter:
						spreadPercent := 50.0
						if rowItem := page.rows[row].PbItem(); rowItem != nil {
							spreadPercent = rowItem.FloatRowSetting("spread-percent")
						}
						interSpace := extraRowWidth / float64(NumColumnLayout(page.rows[row].columns)+1)
						numColumn := 0
						for column := range page.rows[row].columns {
							for item := range page.rows[row].columns[column].items {
								if page.rows[row].columns[column].items[item].item.inLayout {
									if column == 0 {
										page.rows[row].columns[column].items[item].item.xOffset += interSpace * float64(numColumn+1) * spreadPercent / 50.0
									} else {
										page.rows[row].columns[column].items[item].item.xOffset += interSpace * float64(numColumn+1)
									}
								}
							}
							if NumItemLayout(page.rows[row].columns[column].items) > 0 {
								numColumn++
							}
						}
					}
				}
			}

			extraPageHeight := page.availableHeight - page.height()
			if extraPageHeight > 0 {
				pageDistribute := AlignTop
				if len(page.rows) > 0 && len(page.rows[0].columns) > 0 && NumItemLayout(page.rows[0].columns[0].items) > 0 {
					pageDistribute = page.rows[0].columns[0].items[0].item.Align("distribute-rows")
				}
				switch BindingAlign(pageDistribute, binding, pp) {
				case AlignBottom:
					for row := range page.rows {
						for column := range page.rows[row].columns {
							for item := range page.rows[row].columns[column].items {
								if page.rows[row].columns[column].items[item].item.inLayout {
									page.rows[row].columns[column].items[item].item.yOffset += extraPageHeight
								}
							}
						}
					}
				case AlignMiddle:
					for row := range page.rows {
						for column := range page.rows[row].columns {
							for item := range page.rows[row].columns[column].items {
								if page.rows[row].columns[column].items[item].item.inLayout {
									page.rows[row].columns[column].items[item].item.yOffset += extraPageHeight / 2
								}
							}
						}
					}
				case AlignJustify, AlignSpreadPack:
					if NumRowLayout(page.rows) == 1 {
						for column := range page.rows[0].columns {
							for item := range page.rows[0].columns[column].items {
								if page.rows[0].columns[column].items[item].item.inLayout {
									page.rows[0].columns[column].items[item].item.yOffset += extraPageHeight / 2
								}
							}
						}
					} else {
						interSpace := extraPageHeight / float64(NumRowLayout(page.rows)-1)
						numRow := 0
						for row := range page.rows {
							for column := range page.rows[row].columns {
								for item := range page.rows[row].columns[column].items {
									if page.rows[row].columns[column].items[item].item.inLayout {
										page.rows[row].columns[column].items[item].item.yOffset += interSpace * float64(numRow)
									}
								}
							}
							if NumColumnLayout(page.rows[row].columns) > 0 {
								numRow++
							}
						}
					}
				case AlignSpreadTop:
					interSpace := extraPageHeight / float64(NumRowLayout(page.rows))
					numRow := 0
					for row := range page.rows {
						for column := range page.rows[row].columns {
							for item := range page.rows[row].columns[column].items {
								if page.rows[row].columns[column].items[item].item.inLayout {
									page.rows[row].columns[column].items[item].item.yOffset += interSpace * float64(numRow)
								}
							}
						}
						if NumColumnLayout(page.rows[row].columns) > 0 {
							numRow++
						}
					}
				case AlignSpreadBottom:
					interSpace := extraPageHeight / float64(NumRowLayout(page.rows))
					numRow := 0
					for row := range page.rows {
						for column := range page.rows[row].columns {
							for item := range page.rows[row].columns[column].items {
								if page.rows[row].columns[column].items[item].item.inLayout {
									page.rows[row].columns[column].items[item].item.yOffset += interSpace * float64(numRow+1)
								}
							}
						}
						if NumColumnLayout(page.rows[row].columns) > 0 {
							numRow++
						}
					}
				case AlignSpreadMiddle:
					spreadPercent := 50.0
					if pageItem := page.PbItem(); pageItem != nil {
						spreadPercent = pageItem.FloatPageSetting("spread-percent")
					}
					interSpace := extraPageHeight / float64(NumRowLayout(page.rows)+1)
					numRow := 0
					for row := range page.rows {
						for column := range page.rows[row].columns {
							for item := range page.rows[row].columns[column].items {
								if page.rows[row].columns[column].items[item].item.inLayout {
									if row == 0 {
										page.rows[row].columns[column].items[item].item.yOffset += interSpace * float64(numRow+1) * spreadPercent / 50.0
									} else {
										page.rows[row].columns[column].items[item].item.yOffset += interSpace * float64(numRow+1)
									}
								}
							}
						}
						if NumColumnLayout(page.rows[row].columns) > 0 {
							numRow++
						}
					}
				}
			}

			page.updateOffsets()
		}
	}
}

const (
	CanMoveTop    = 1
	CanMoveBottom = 2
	CanMoveLeft   = 4
	CanMoveRight  = 8
)

func overlaps(aa *PbItem, bb *PbItem) (bool, bool, float64, float64, float64, float64) {
	aaWidth, aaHeight, bbWidth, bbHeight := 0.0, 0.0, 0.0, 0.0

	if aa.itemType == ItemTypeImage {
		aaWidth = max(aa.imageWidth, aa.textWidth)
		aaHeight = aa.imageHeight + aa.textHeight + aa.CaptionGutter()
	} else {
		aaWidth = aa.textWidth
		aaHeight = aa.textHeight
	}
	aaTop := aa.yOffset
	aaLeft := aa.xOffset
	aaBottom := aaTop + aaHeight
	aaRight := aaLeft + aaWidth

	if bb.itemType == ItemTypeImage {
		bbWidth = max(bb.imageWidth, bb.textWidth)
		bbHeight = bb.imageHeight + bb.textHeight + bb.CaptionGutter()
	} else {
		bbWidth = bb.textWidth
		bbHeight = bb.textHeight
	}
	bbTop := bb.yOffset
	bbLeft := bb.xOffset
	bbBottom := bbTop + bbHeight
	bbRight := bbLeft + bbWidth

	overlapsV := (bbLeft >= aaLeft && bbLeft < aaRight) || (bbRight >= aaLeft && bbRight < aaRight) ||
		(bbLeft <= aaLeft && bbRight >= aaRight) || (aaLeft <= bbLeft && aaRight >= bbRight)

	overlapsH := (bbTop >= aaTop && bbTop < aaBottom) || (bbBottom >= aaTop && bbBottom < aaBottom) ||
		(bbTop <= aaTop && bbBottom >= aaBottom) || (aaTop <= bbTop && aaBottom >= bbBottom)

	return overlapsV, overlapsH, aaWidth, aaHeight, bbWidth, bbHeight
}

func canGrow(items []*PbItem, ii int, amount float64, gutter float64, page *PbPage) int {
	return canMoveGrow(items, ii, amount, gutter, page, false)
}

func canMove(items []*PbItem, ii int, amount float64, gutter float64, page *PbPage) int {
	return canMoveGrow(items, ii, amount, gutter, page, true)
}

func canMoveGrow(items []*PbItem, ii int, amount float64, gutter float64, page *PbPage, move bool) int {
	if items == nil || ii < 0 || ii > len(items) || (items[ii].itemType != ItemTypeImage && items[ii].itemType != ItemTypeText) {
		return 0
	}

	canMove := CanMoveTop | CanMoveBottom | CanMoveLeft | CanMoveRight

	// If moving, don't move things away from the edges
	if move {
		if items[ii].xOffset <= 0 {
			canMove &= ^CanMoveLeft
			canMove &= ^CanMoveRight
		}

		if items[ii].yOffset <= 0 {
			canMove &= ^CanMoveTop
			canMove &= ^CanMoveBottom
		}

		if items[ii].itemType == ItemTypeImage {
			if items[ii].xOffset+math.Max(items[ii].imageWidth, items[ii].textWidth) >= page.availableWidth {
				canMove &= ^CanMoveLeft
				canMove &= ^CanMoveRight
			}
			if items[ii].yOffset+items[ii].imageHeight+items[ii].textHeight+items[ii].CaptionGutter() >= page.availableHeight {
				canMove &= ^CanMoveTop
				canMove &= ^CanMoveBottom
			}
		} else {
			if items[ii].xOffset+items[ii].textWidth >= page.availableWidth {
				canMove &= ^CanMoveLeft
				canMove &= ^CanMoveRight
			}
			if items[ii].yOffset+items[ii].textHeight >= page.availableHeight {
				canMove &= ^CanMoveTop
				canMove &= ^CanMoveBottom
			}
		}
	}

	if items[ii].xOffset-amount <= 0 {
		canMove &= ^CanMoveLeft
	}

	if items[ii].yOffset-amount <= 0 {
		canMove &= ^CanMoveTop
	}

	if items[ii].itemType == ItemTypeImage {
		if items[ii].xOffset+math.Max(items[ii].imageWidth, items[ii].textWidth)+amount >= page.availableWidth {
			canMove &= ^CanMoveRight
		}
		if items[ii].yOffset+items[ii].imageHeight+items[ii].textHeight+items[ii].CaptionGutter()+amount >= page.availableHeight {
			canMove &= ^CanMoveBottom
		}
	} else {
		if items[ii].xOffset+items[ii].textWidth+amount >= page.availableWidth {
			canMove &= ^CanMoveRight
		}
		if items[ii].yOffset+items[ii].textHeight+amount >= page.availableHeight {
			canMove &= ^CanMoveBottom
		}
	}

	if items[ii].itemType == ItemTypeImage && move == false {
		landscapeMax, landscapeMin, portraitMax, portraitMin := items[ii].packAspects()
		aspect := items[ii].imageWidth / items[ii].imageHeight
		if items[ii].baseAspect > 1 {
			if aspect > landscapeMax || aspect < landscapeMin {
				canMove = 0
			}
		} else {
			if aspect > portraitMax || aspect < portraitMin {
				canMove = 0
			}
		}
	}

	if canMove != 0 {
		for jj := range items {
			if ii != jj {
				overlapsV, overlapsH, iiWidth, iiHeight, jjWidth, jjHeight := overlaps(items[ii], items[jj])

				if overlapsH && items[ii].xOffset > items[jj].xOffset && items[ii].xOffset-amount <= items[jj].xOffset+jjWidth+gutter {
					canMove &= ^CanMoveLeft
				}

				if overlapsV && items[ii].yOffset > items[jj].yOffset && items[ii].yOffset-amount <= items[jj].yOffset+jjHeight+gutter {
					canMove &= ^CanMoveTop
				}

				if overlapsH && items[jj].xOffset > items[ii].xOffset && items[ii].xOffset+iiWidth+amount >= items[jj].xOffset-gutter {
					canMove &= ^CanMoveRight
				}

				if overlapsV && items[jj].yOffset > items[ii].yOffset && items[ii].yOffset+iiHeight+amount >= items[jj].yOffset-gutter {
					canMove &= ^CanMoveBottom
				}

				if canMove == 0 {
					break
				}
			}
		}
	}

	return canMove
}

func packPages(pbBook *PbBook, outPageRange string, firstIteration bool) {
	for pp := range pbBook.pages {
		if isPageInRange(outPageRange, pp, firstIteration) || isCurrentPage(pbBook, pp) {
			page := &pbBook.pages[pp]

			if pageItem := page.PbItem(); pageItem == nil || (pageItem.BoolPageSetting("nolayout") || !pageItem.BoolPageSetting("pack-page")) {
				continue
			} else {
				items := make([]*PbItem, 0)
				foundInvalid := false

			findItems:
				for rowNum := range page.rows {
					for columnNum := range page.rows[rowNum].columns {
						for columnItemNum := range page.rows[rowNum].columns[columnNum].items {
							item := page.rows[rowNum].columns[columnNum].items[columnItemNum].item
							if item.itemType != ItemTypeImage && item.itemType != ItemTypeText {
								foundInvalid = true
								log.Printf("Found Invalid Item on page %v\n", pp)
								break findItems
							} else if item.itemType == ItemTypeImage {
								item.baseAspect = item.imageWidth / item.imageHeight
							}
							items = append(items, item)
						}
					}
				}

				if foundInvalid {
					continue
				}

				amount := 0.5 / pageItem.Density()
				gutter := pageItem.FloatPageSetting("pack-gutter")

				firstIteration := true

				for {
					VerboseLog(fmt.Sprintf("Growing Page %v", pp))
					grewItems := false
					for {
						grewAnItem := false

						for ii := range items {
							if items[ii].itemType == ItemTypeText || len(items[ii].Setting("text")) > 0 || !items[ii].BoolSetting("pack") {
								continue
							}

							canGrow := canGrow(items, ii, amount, gutter, page)

							if canGrow != 0 {
								if canGrow&CanMoveLeft != 0 {
									items[ii].xOffset -= amount
									items[ii].imageWidth += amount
								}
								if canGrow&CanMoveTop != 0 {
									items[ii].yOffset -= amount
									items[ii].imageHeight += amount
								}
								if canGrow&CanMoveRight != 0 {
									items[ii].imageWidth += amount
								}
								if canGrow&CanMoveBottom != 0 {
									items[ii].imageHeight += amount
								}

								rect := fmt.Sprintf("trim,%v:%v", int(math.Round(items[ii].imageWidth*1000)), int(math.Round(items[ii].imageHeight*1000)))
								items[ii].Set("rect", rect)
								grewAnItem = true
								grewItems = true
							}
						}

						if !grewAnItem {
							break
						}
					}

					if !grewItems && !firstIteration {
						break
					}

					VerboseLog(fmt.Sprintf("Moving Page %v", pp))
					movedItems := false

					for ii := range items {
						amountLeft, amountRight, amountUp, amountDown := 0.0, 0.0, 0.0, 0.0
						multiplier := 1
						for {
							thisAmount := amount * float64(multiplier)
							canMove := canMove(items, ii, thisAmount, gutter, page)
							if canMove == 0 {
								break
							}
							if canMove&CanMoveLeft != 0 {
								amountLeft = thisAmount
							}
							if canMove&CanMoveRight != 0 {
								amountRight = thisAmount
							}
							if canMove&CanMoveTop != 0 {
								amountUp = thisAmount
							}
							if canMove&CanMoveBottom != 0 {
								amountDown = thisAmount
							}
							multiplier++
						}

						if amountRight-amountLeft != 0 {
							items[ii].xOffset += (amountRight - amountLeft) / 2
							movedItems = true
							//VerboseLog(fmt.Sprintf("Moving %v.%v LR %v", pp+1, ii, (amountRight-amountLeft)/2))
						}

						if amountDown-amountUp != 0 {
							items[ii].yOffset += (amountDown - amountUp) / 2
							movedItems = true
							//VerboseLog(fmt.Sprintf("Moving %v.%v UD %v", pp+1, ii, (amountDown-amountUp)/2))
						}
					}

					if !grewItems && !movedItems {
						break
					}

					firstIteration = false
				}
			}

			page.updateOffsets()
		}
	}
}
