package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/spakin/netpbm"

	"github.com/disintegration/imaging"
)

type ImageCacheEntry struct {
	Filedate      int64
	ImageWidthPx  int
	ImageHeightPx int
}

func loadImageCache() map[string]ImageCacheEntry {
	cache := map[string]ImageCacheEntry{}

	if *nocacheFlag {
		return cache
	}

	bytes, err := os.ReadFile(".pbimagecache")
	if err == nil {
		json.Unmarshal(bytes, &cache)
	}

	return cache
}

func saveImageCache(cache *map[string]ImageCacheEntry) {
	bytes, err := json.Marshal(cache)
	if err == nil {
		os.WriteFile(".pbimagecache", bytes, 0666)
	}
}

func checkCacheEntry(cache *map[string]ImageCacheEntry, filename string) (int, int, bool) {
	if entry, exists := (*cache)[filename]; exists {
		filedate := fileDate(filename)
		if entry.Filedate == filedate {
			return entry.ImageWidthPx, entry.ImageHeightPx, true
		}
	}
	return 0, 0, false
}

func updateCacheEntry(cache *map[string]ImageCacheEntry, filename string, imageWidthPx int, imageHeightPx int) {
	(*cache)[filename] = ImageCacheEntry{fileDate(filename), imageWidthPx, imageHeightPx}
}

func getImageDimensions(items []PbItem) int {
	if len(items) == 0 {
		return 0
	}

	numImages := 0
	cache := loadImageCache()

	for ii, theItem := range items {
		if theItem.itemType == ItemTypeImage {
			numImages++
			items[ii].imageWidthPx = 1
			items[ii].imageHeightPx = 1

			filename := theItem.Setting("image")
			rotation := Atoi(items[ii].Setting("rotate"))

			if imageWidthPx, imageHeightPx, exists := checkCacheEntry(&cache, filename); exists {
				if rotation == 90 || rotation == -90 || rotation == 270 || rotation == -270 {
					items[ii].imageWidthPx = imageHeightPx
					items[ii].imageHeightPx = imageWidthPx
				} else {
					items[ii].imageWidthPx = imageWidthPx
					items[ii].imageHeightPx = imageHeightPx
				}
				continue
			}

			imageFile, err := os.Open(filename)
			if err != nil {
				log.Print(err)
				continue
			}
			imageReader := bufio.NewReader(imageFile)
			config, _, err := image.DecodeConfig(imageReader)
			if err != nil {
				imageFile.Close()
				log.Print(err)
				continue
			}
			if err := imageFile.Close(); err != nil {
				log.Print(err)
				continue
			}

			if rotation == 90 || rotation == -90 || rotation == 270 || rotation == -270 {
				items[ii].imageWidthPx = config.Height
				items[ii].imageHeightPx = config.Width
			} else {
				items[ii].imageWidthPx = config.Width
				items[ii].imageHeightPx = config.Height
			}

			updateCacheEntry(&cache, filename, config.Width, config.Height)
		}
	}

	saveImageCache(&cache)
	return numImages
}

func getTextDimensions(items []PbItem) int {
	if len(items) == 0 {
		return 0
	}

	numTexts := 0
	for ii := range items {
		if items[ii].itemType == ItemTypeText && len(items[ii].Setting("text")) > 0 {
			width := WidthForContainer(items[ii].Setting("text-width"), items[ii].PageSetting("page-size"), items[ii].PageSetting("margin"))
			items[ii].textBlockLayouts = MeasureText(items[ii].Setting("text"), width, 0.0, items[ii].TextInfo())
			// TextToImage(&items[ii].textBlockLayouts[0], items[ii].TextInfo())
			numTexts++
		} else if items[ii].itemType == ItemTypeImage && len(items[ii].Setting("text")) > 0 {
			maxWidth := ContainerWidth(items[ii].PageSetting("page-size"), items[ii].PageSetting("margin"))
			maxHeight := ContainerHeight(items[ii].PageSetting("page-size"), items[ii].PageSetting("margin"))
			items[ii].textBlockLayouts = MeasureText(items[ii].Setting("text"), maxWidth, maxHeight, items[ii].TextInfo())
			numTexts++
		}
	}

	return numTexts
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

func writeHeader(outFilename string) (int, error) {
	ext := filepath.Ext(outFilename)
	format := strings.ToLower(ext)

	if format == ".pdf" {
		bytes := []byte("%PDF-1.7\n")
		err := os.WriteFile(outFilename, bytes, 0666)
		if err != nil {
			log.Print(err)
		}
		return len(bytes), err
	}

	return 0, nil
}

type PageInfo struct {
	offset int
	width  int
	height int
}

func writeFooter(outFilename string, bytesWritten int, pageInfo []PageInfo) (int, error) {
	ext := filepath.Ext(outFilename)
	format := strings.ToLower(ext)

	if format == ".pdf" {
		buffer := strings.Builder{}
		numPages := len(pageInfo)
		pageInfo = append(pageInfo, PageInfo{bytesWritten, 0, 0})
		n, err := buffer.WriteString(fmt.Sprintf("%v 0 obj\n<</Type/Catalog/Pages %v 0 R>>\nendobj\n", numPages+1, numPages+2))
		bytesWritten += n
		pageInfo = append(pageInfo, PageInfo{bytesWritten, 0, 0})
		n, err = buffer.WriteString(fmt.Sprintf("%v 0 obj\n<</Type/Pages/Count %v/Kids[", numPages+2, numPages))
		bytesWritten += n
		for ii := range numPages {
			space := " "
			if ii == 0 {
				space = ""
			}
			n, err = buffer.WriteString(fmt.Sprintf("%v%v 0 R", space, ii*3+numPages+3+2))
			bytesWritten += n
		}
		n, err = buffer.WriteString("]>>\nendobj\n")
		bytesWritten += n

		for ii := range numPages {
			pageInfo = append(pageInfo, PageInfo{bytesWritten, 0, 0})
			n, err = buffer.WriteString(fmt.Sprintf("%v 0 obj\n<</XObject<</I%v %v 0 R>>>>\nendobj\n", ii*3+numPages+3+0, ii+1, ii+1))
			bytesWritten += n
			pageInfo = append(pageInfo, PageInfo{bytesWritten, 0, 0})
			cmd := fmt.Sprintf("q %v 0 0 %v 0 0 cm /I%v Do Q", pageInfo[ii].width, pageInfo[ii].height, ii+1)
			n, err = buffer.WriteString(fmt.Sprintf("%v 0 obj\n<</Length %v>>\nstream\n%v\nendstream\nendobj\n", ii*3+numPages+3+1, len(cmd), cmd))
			bytesWritten += n
			pageInfo = append(pageInfo, PageInfo{bytesWritten, 0, 0})
			n, err = buffer.WriteString(fmt.Sprintf("%v 0 obj\n<</Type/Page/MediaBox[0 0 %v %v]/Rotate 0/Resources %v 0 R/Contents %v 0 R/Parent %v 0 R>>\nendobj\n", ii*3+numPages+3+2, pageInfo[ii].width, pageInfo[ii].height, ii*3+numPages+3+0, ii*3+numPages+3+1, numPages+2))
			bytesWritten += n
		}

		startOfXref := bytesWritten

		n, err = buffer.WriteString(fmt.Sprintf("xref\n0 %v\n0000000000 00001 f\n", numPages*4+3))
		for ii := range pageInfo {
			n, err = buffer.WriteString(fmt.Sprintf("%010d 00000 n\n", pageInfo[ii].offset))
		}

		n, err = buffer.WriteString(fmt.Sprintf("trailer\n<</Size %v/Root %v 0 R>>\nstartxref\n%v\n%%%%EOF", len(pageInfo)+1, numPages+1, startOfXref))
		out, err := os.OpenFile(outFilename, os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Print(err)
			return 0, err
		}
		defer out.Close()
		n, err = out.Write([]byte(buffer.String()))
		if err != nil {
			log.Print(err)
			return 0, err
		}
		return n, nil
	}

	return 0, nil
}

type PdfJpegObjectWriter struct {
	out    io.Writer
	mem    *[]byte
	objNum int
	width  int
	height int
}

// Write implements [io.Writer].
func (writer PdfJpegObjectWriter) Write(p []byte) (int, error) {
	*writer.mem = append(*writer.mem, p...)
	return len(p), nil
}

func (writer *PdfJpegObjectWriter) Start(out io.Writer, objNum int, width int, height int) {
	writer.out = out
	writer.objNum = objNum
	writer.width = width
	writer.height = height
	writer.mem = &[]byte{}
}

func (writer *PdfJpegObjectWriter) Finish() (int, error) {
	text := fmt.Sprintf("%v 0 obj\n<</Filter[/DCTDecode]/Type/XObject/Subtype/Image/BitsPerComponent 8/Width %v/Height %v/ColorSpace/DeviceRGB/Length %v>>\nstream\n", writer.objNum, writer.width, writer.height, len(*writer.mem))
	bytesWritten := 0
	n, err := writer.out.Write([]byte(text))
	if err != nil {
		log.Print(err)
		return bytesWritten, err
	}
	bytesWritten += n
	n, err = writer.out.Write(*writer.mem)
	if err != nil {
		log.Print(err)
		return bytesWritten, err
	}
	bytesWritten += n
	n, err = writer.out.Write([]byte("\nendstream\nendobj\n"))
	if err != nil {
		log.Print(err)
		return bytesWritten, err
	}
	bytesWritten += n
	return bytesWritten, err
}

func writePage(img image.Image, objNum int, curPage int, outFilename string, isPageRangeMulti bool, compressionLevel int) (int, error) {
	ext := filepath.Ext(outFilename)
	format := strings.ToLower(ext)
	if isPageRangeMulti && format != ".pdf" {
		outFilename = strings.TrimSuffix(outFilename, ext)
		outFilename = fmt.Sprintf(outFilename+"-%v"+ext, curPage)
	}

	var out *os.File
	var err error
	if format != ".pdf" {
		out, err = os.Create(outFilename)
	} else {
		out, err = os.OpenFile(outFilename, os.O_WRONLY|os.O_APPEND, 0666)
	}
	if err != nil {
		log.Print(err)
		return 0, err
	}
	defer out.Close()
	switch format {
	case ".png":
		if err := png.Encode(out, img); err != nil {
			log.Print(err)
			return 0, err
		}
		return 0, nil
	case ".jpg", ".jpeg":
		options := jpeg.Options{Quality: compressionLevel}
		if err := jpeg.Encode(out, img, &options); err != nil {
			log.Print(err)
			return 0, err
		}
		return 0, nil
		// return writeJPEG(img, out, compressionLevel)
	case ".pdf":
		writer := PdfJpegObjectWriter{}
		writer.Start(out, objNum, img.Bounds().Dx(), img.Bounds().Dy())

		options := jpeg.Options{Quality: compressionLevel}
		if err := jpeg.Encode(writer, img, &options); err != nil {
			log.Print(err)
			return 0, err
		}

		// if _, err := writeJPEG(img, writer, compressionLevel); err != nil {
		// 	log.Print(err)
		// 	return 0, err
		// }

		return writer.Finish()
	}

	return 0, nil
}

func scaleToRect(picture image.Image, item *PbItem) image.Image {
	zoom, dstAspect, xOffset, yOffset := item.ImageRectSetting()

	wr, hr, _, _ := calcStraighten(float64(item.imageWidthPx), float64(item.imageHeightPx), item.FloatSetting("straighten"))

	srcAspect := wr / hr

	dstWidth := int(math.Round(wr))
	dstHeight := int(math.Round(hr))
	dstXOffset := 0
	dstYOffset := 0
	switch zoom {
	case 0: // trim
		if dstAspect > srcAspect { // dst is wider than src, crop top & bottom
			dstHeight = int(math.Round(float64(dstWidth) / dstAspect))
			dstYOffset = int(math.Round(float64(int(math.Round(hr))-dstHeight) * float64(yOffset) / 100.0))
			return imaging.Crop(picture, image.Rectangle{image.Point{dstXOffset, dstYOffset}, image.Point{dstXOffset + dstWidth, dstYOffset + dstHeight}})
		} else if dstAspect < srcAspect { // dst is taller than src, crop left & right
			dstWidth = int(math.Round(float64(dstHeight) * dstAspect))
			dstXOffset = int(math.Round(float64(int(math.Round(wr))-dstWidth) * float64(xOffset) / 100.0))
			return imaging.Crop(picture, image.Rectangle{image.Point{dstXOffset, dstYOffset}, image.Point{dstXOffset + dstWidth, dstYOffset + dstHeight}})
		} else {
			return picture
		}
	case 1: // fit
		if dstAspect > srcAspect { // dst is wider than src, pad left & right
			dstWidth = int(math.Round(float64(int(math.Round(hr))) * dstAspect))
			dstXOffset = int(math.Round(float64(dstWidth-int(math.Round(wr))) * float64(xOffset) / 100.0))
			dst := image.NewRGBA(image.Rect(0, 0, dstWidth, dstHeight))
			backColor := colorToNRGBA(item.Setting("background"))
			draw.Draw(dst, dst.Bounds(), image.NewUniform(color.RGBA{backColor.R, backColor.G, backColor.B, backColor.A}), image.Point{}, draw.Src)
			return imaging.Paste(dst, picture, image.Point{dstXOffset, dstYOffset})
		} else if dstAspect < srcAspect { // dst is taller than src, pad top & bottom
			dstHeight = int(math.Round(float64(int(math.Round(wr))) / dstAspect))
			dstYOffset = int(math.Round(float64(dstHeight-int(math.Round(hr))) * float64(yOffset) / 100.0))
			dst := image.NewRGBA(image.Rect(0, 0, dstWidth, dstHeight))
			backColor := colorToNRGBA(item.Setting("background"))
			draw.Draw(dst, dst.Bounds(), image.NewUniform(color.RGBA{backColor.R, backColor.G, backColor.B, backColor.A}), image.Point{}, draw.Src)
			return imaging.Paste(dst, picture, image.Point{dstXOffset, dstYOffset})
		} else {
			return picture
		}
	default: // arbitrary zoom
	}

	return picture
}

func calcStraighten(width float64, height float64, angle float64) (float64, float64, float64, float64) {
	if angle == 0 {
		return width, height, 0, 0
	}
	aspectRatio := float64(width) / float64(height)
	angler := angle * 2 * 3.1415926535897932384626433 / 360
	sin_a := math.Abs(math.Sin(angler))
	cos_a := math.Abs(math.Cos(angler))

	var sideLong, sideShort float64
	if aspectRatio >= 1 {
		sideLong = float64(width)
		sideShort = float64(height)
	} else {
		sideLong = float64(height)
		sideShort = float64(width)
	}

	var wr, hr float64
	if sideShort <= 2*sin_a*cos_a*sideLong || math.Abs(sin_a-cos_a) < 0.0000000001 {
		xx := sideShort / 2
		if aspectRatio >= 1 {
			wr = xx / sin_a
			hr = xx / cos_a
		} else {
			wr = xx / cos_a
			hr = xx / sin_a
		}
	} else {
		cos2a := cos_a*cos_a - sin_a*sin_a
		wr = math.Abs(width*cos_a-height*sin_a) / cos2a
		hr = math.Abs(height*cos_a-width*sin_a) / cos2a
	}

	hOff := math.Abs((width*cos_a+height*sin_a)-wr) / 2
	vOff := math.Abs((width*sin_a+height*cos_a)-hr) / 2

	return wr, hr, hOff, vOff
}

func straighten(picture image.Image, angle float64) image.Image {

	wr, hr, hOff, vOff := calcStraighten(float64(picture.Bounds().Dx()), float64(picture.Bounds().Dy()), angle)

	picture = imaging.Rotate(picture, -angle, color.RGBA{127, 127, 127, 255})

	rect := image.Rectangle{
		image.Point{int(math.Round(hOff)), int(math.Round(vOff))},
		image.Point{int(math.Round(hOff + wr)), int(math.Round(vOff + hr))}}

	return imaging.Crop(picture, rect)
}

func convertImage(picture image.Image) image.Image {
	// this is too slow for regular use
	// may be able to adapt to use imagmagick or mozjpeg to create quality jpegs for final output
	log.Print("executing convert")
	cmd := exec.Command("convert", "-", "-adaptive-sharpen", "x5", "PNM:-")

	stdin, err1 := cmd.StdinPipe()
	if err1 != nil {
		log.Print("Error opening stdin")
		log.Print(err1)
		return picture
	}

	stdout, err2 := cmd.StdoutPipe()
	if err2 != nil {
		stdin.Close()
		log.Print("Error opening stdout")
		log.Print(err2)
		return picture
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer stdin.Close()
		defer wg.Done()
		err := netpbm.Encode(stdin, picture, &netpbm.EncodeOptions{})
		if err != nil {
			log.Print("Error encoding image")
			log.Print(err)
		}
	}()

	wg.Add(1)
	go func() {
		defer stdout.Close()
		defer wg.Done()
		newpicture, err := netpbm.Decode(stdout, &netpbm.DecodeOptions{})
		if err != nil {
			log.Print("Error decoding image")
			log.Print(err)
		}
		if newpicture != nil && newpicture.Bounds().Dx() == picture.Bounds().Dx() && newpicture.Bounds().Dy() == picture.Bounds().Dy() {
			picture = newpicture
		}
	}()

	err2 = cmd.Run()
	if err2 != nil {
		log.Print("Error running command")
		log.Print(err2)
	}

	wg.Wait()
	log.Print("executed convert")
	return picture
}

func writeJPEG(picture image.Image, out io.Writer, compressionLevel int) (int, error) {
	// this is too slow for regular use
	// may be able to adapt to use imagmagick or mozjpeg to create quality jpegs for final output
	cmd := exec.Command("convert", "PNM:-", "-quality", fmt.Sprintf("%v", compressionLevel), "-sampling-factor", "4:4:4", "JPEG:-")
	//cmd := exec.Command("cjpeg", "-quality", fmt.Sprintf("%v", compressionLevel), "-sample", "1x1")

	log.Print(cmd.String())
	bytesWritten := 0
	var errReturn error

	stdin, err1 := cmd.StdinPipe()
	if err1 != nil {
		log.Print("Error opening stdin")
		log.Print(err1)
		return 0, err1
	}

	stdout, err2 := cmd.StdoutPipe()
	if err2 != nil {
		stdin.Close()
		log.Print("Error opening stdout")
		log.Print(err2)
		return 0, err2
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer stdin.Close()
		defer wg.Done()
		err := netpbm.Encode(stdin, picture, &netpbm.EncodeOptions{})
		if err != nil {
			log.Print("Error encoding image")
			log.Print(err)
			errReturn = err
		}
	}()

	wg.Add(1)
	go func() {
		defer stdout.Close()
		defer wg.Done()
		p := make([]byte, 1024*64)
		for {
			n, err := stdout.Read(p)
			if n > 0 {
				m, err2 := out.Write(p[:n])
				if err2 != nil {
					log.Print("error writing output file")
					log.Print(err2)
					errReturn = err2
					break
				}
				bytesWritten += m
				if m != n {
					log.Print("truncated write")
					break
				}
			}
			if err != nil {
				if err != io.EOF {
					log.Print("Error readig input stream")
					log.Print(err)
					errReturn = err
				}
				break
			}
		}
	}()

	// err2 = cmd.Start()
	// if err2 != nil {
	// 	log.Print("Error running command")
	// 	log.Print(err2)
	// 	errReturn = err2
	// }
	// err2 = cmd.Wait()
	// if err2 != nil {
	// 	log.Print("Error waiting for command")
	// 	log.Print(err2)
	// 	errReturn = err2
	// }

	err2 = cmd.Run()
	if err2 != nil {
		log.Print("Error running command")
		log.Print(err2)
		errReturn = err2
	}

	// var bytes []byte
	// bytes, err2 = cmd.Output()
	// if err2 != nil {
	// 	log.Print("Error running command")
	// 	log.Print(err2)
	// 	errReturn = err2
	// }
	// log.Print(string(bytes))

	wg.Wait()

	if errReturn == nil {
		log.Printf("%v: %v bytes", cmd.String(), bytesWritten)
	} else {
		log.Printf("%v: %v bytes, %v", cmd.String(), bytesWritten, errReturn)
	}
	return bytesWritten, errReturn
}

func renderPages(pbBook *PbBook, outPageRange string, outFilename string) {
	n, err := writeHeader(outFilename)
	if err != nil {
		return
	}

	isPageRangeMulti := isPageRangeMulti(outPageRange, pbBook)

	offsets := []PageInfo{}
	objNum := 1
	for pp := range pbBook.pages {
		changed := false
		if changed, _ = fileChanged(*inFileFlag, lastModTime); changed {
			break
		}
		page := &pbBook.pages[pp]
		if isPageInRange(outPageRange, pp) || isCurrentPage(pbBook, pp) {
			var top float64
			var left float64
			var dst *image.RGBA = nil
			density := 1.0

			if len(page.rows[0].columns[0].items) == 0 {
				continue
			}
			item := page.rows[0].columns[0].items[0].item

			pageWidth, pageHeight := FloatSize(item.Setting("page-size"))
			density = item.Density()
			top, _, _, left = FourTwoOne(item.Setting("margin"))
			widthDots := int(math.Round(dotsFromUnitsFloat(pageWidth, density)))
			heightDots := int(math.Round(dotsFromUnitsFloat(pageHeight, density)))
			dst = image.NewRGBA(image.Rect(0, 0, widthDots, heightDots))

			// fill the destination with the background color
			backColor := colorToNRGBA(item.Setting("background"))
			draw.Draw(dst, dst.Bounds(), image.NewUniform(color.RGBA{backColor.R, backColor.G, backColor.B, backColor.A}), image.Point{}, draw.Src)

			for row := range page.rows {
				for column := range page.rows[row].columns {
					for columnItem := range page.rows[row].columns[column].items {
						item = page.rows[row].columns[column].items[columnItem].item

						if item.itemType == ItemTypeText && len(item.Setting("name")) == 0 {
							textImage := TextToImage(&item.textBlockLayouts[0], item.TextInfo())
							xDots := int(math.Round(dotsFromUnitsFloat(left+item.xOffset, density)))
							yDots := int(math.Round(dotsFromUnitsFloat(top+item.yOffset, density)))
							draw.Draw(dst, image.Rect(xDots, yDots, xDots+textImage.Bounds().Size().X, yDots+textImage.Bounds().Size().Y), textImage, image.Point{}, draw.Over)
						}

						if item.itemType == ItemTypeImage && len(item.Setting("name")) == 0 {
							picture, err := imaging.Open(item.Setting("image"))
							if err != nil {
								log.Printf("failed to open image: %v", err)
								continue
							}

							straightenAngle := item.FloatSetting("straighten")
							if straightenAngle != 0 {
								picture = straighten(picture, straightenAngle)
							}

							rotation := Atoi(item.Setting("rotate"))
							switch rotation {
							case -90, 270:
								picture = imaging.Rotate90(picture)
							case 90, -270:
								picture = imaging.Rotate270(picture)
							case 180, -180:
								picture = imaging.Rotate180(picture)
							}

							brightness := item.FloatSetting("brightness")
							if brightness != 0 {
								picture = imaging.AdjustBrightness(picture, brightness)
							}
							contrast := item.FloatSetting("contrast")
							if contrast != 0 {
								picture = imaging.AdjustContrast(picture, contrast)
							}
							gamma := item.FloatSetting("gamma")
							if gamma != 1.0 {
								picture = imaging.AdjustGamma(picture, gamma)
							}
							saturation := item.FloatSetting("saturation")
							if saturation != 0 {
								picture = imaging.AdjustSaturation(picture, saturation)
							}
							factor, midpoint := item.SigmoidalSetting()
							if factor != 0 {
								picture = imaging.AdjustSigmoid(picture, midpoint, factor)
							}

							picture = scaleToRect(picture, item)

							// Resize the cropped image to width = 200px preserving the aspect ratio.
							imageWidthDots := int(math.Round(dotsFromUnitsFloat(item.imageWidth, density)))
							imageHeightDots := int(math.Round(dotsFromUnitsFloat(item.imageHeight, density)))

							picture = imaging.Resize(picture, imageWidthDots, imageHeightDots, imaging.Lanczos)

							blur := item.FloatSetting("blur")
							if blur != 0 {
								picture = imaging.Blur(picture, blur)
							}
							sharpen := item.FloatSetting("sharpen")
							if sharpen != 0 {
								picture = imaging.Sharpen(picture, sharpen)
							}

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
				}
			}

			w, h := item.PageSizePts()
			offsets = append(offsets, PageInfo{n, w, h})
			thisn, thisErr := writePage(dst, objNum, pp, outFilename, isPageRangeMulti, item.IntBookSetting("output-compression"))
			if thisErr != nil {
				return
			}

			if globalVerboseFlag&4 != 0 {
				log.Printf("Rendered Page %v / %v", pp+1, len(pbBook.pages))
			}

			n += thisn
			objNum++
		}
	}
	_, endErr := writeFooter(outFilename, n, offsets)
	if endErr != nil {
		return
	}
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

func isPageInRange(pageRange string, pageNum int) bool {
	if len(pageRange) == 0 || pageRange == "-" {
		return true
	}

	// internally the page numbering is zero based, the external page number is one based
	pageNum += 1

	pageRanges := strings.Split(pageRange, ",")

	for _, aRange := range pageRanges {
		if strings.HasPrefix(aRange, "-") {
			if end, _ := strings.CutPrefix(aRange, "-"); pageNum <= Atoi(end) {
				return true
			}
		} else if strings.HasSuffix(aRange, "-") {
			if start, _ := strings.CutSuffix(aRange, "-"); pageNum >= Atoi(start) {
				return true
			}
		} else {
			parts := strings.SplitN(aRange, "-", 2)
			switch len(parts) {
			case 1:
				if Atoi(parts[0]) == pageNum {
					return true
				}
			case 2:
				start := Atoi(parts[0])
				end := Atoi(parts[1])

				if pageNum >= start && pageNum <= end {
					return true
				}
			}
		}
	}
	return false
}

func isPageRangeMulti(pageRange string, pbBook *PbBook) bool {
	pageCount := 0
	for pp := range pbBook.pages {
		if isPageInRange(pageRange, pp) {
			pageCount++
			if pageCount > 1 {
				return true
			}
		}
	}

	return false
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

func isCurrentPage(pb *PbBook, pp int) bool {
	rv := false
	if pb != nil && len(pb.pages) > pp {
		if len(pb.pages[pp].rows) > 0 && len(pb.pages[pp].rows[0].columns) > 0 && len(pb.pages[pp].rows[0].columns[0].items) > 0 && pb.pages[pp].rows[0].columns[0].items[0].item != nil {
			rv = pb.pages[pp].rows[0].columns[0].items[0].item.BoolPageSetting("current-page")
		}
	}
	return rv
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

func fileDate(filename string) int64 {
	rv := time.Time{}.Unix()

	fi, err := os.Stat(filename)
	if err == nil {
		rv = fi.ModTime().Unix()
	}

	return rv
}

func fileChanged(filename string, lastModTime time.Time) (bool, time.Time) {
	fi, err := os.Stat(filename)
	if err != nil {
		log.Fatal(err)
	}

	thisModTime := fi.ModTime()

	return lastModTime != thisModTime, thisModTime
}

var globalVerboseFlag = 0
var nocacheFlag *bool
var lastModTime time.Time
var inFileFlag *string

func main() {
	var (
		pageFlag     = flag.String("p", "", "page range")
		verboseFlag  = flag.Int("v", 0, "verbose")
		watchFlag    = flag.Bool("w", false, "watch")
		outFileFlag  = flag.String("o", "out.png", "output filename")
		noresizeFlag = flag.Bool("noresize", false, "noresize")
		nolayoutFlag = flag.Bool("nolayout", false, "nolayout")
		norenderFlag = flag.Bool("norender", false, "norender")
	)

	inFileFlag = flag.String("i", "", "input filename")
	nocacheFlag = flag.Bool("nocache", false, "do not use caches")

	flag.Parse()
	globalVerboseFlag = *verboseFlag

	if *inFileFlag == "" {
		flag.Usage()
		log.Fatal("no input file specified")
	}

	_, lastModTime = fileChanged(*inFileFlag, time.Time{})

	for {
		items, options := ReadPbFile(*inFileFlag)

		globalVerboseFlag = *verboseFlag
		if val, exists := options["v"]; exists {
			globalVerboseFlag = Atoi(val)
		}

		bNoResize := *noresizeFlag
		if val, exists := options["noresize"]; exists {
			bNoResize = Atob(val)
		}

		bNoLayout := *nolayoutFlag
		if val, exists := options["nolayout"]; exists {
			bNoLayout = Atob(val)
		}

		bNoRender := *norenderFlag
		if val, exists := options["norender"]; exists {
			bNoRender = Atob(val)
		}

		globalVerboseFlag = *verboseFlag
		if val, exists := options["v"]; exists {
			globalVerboseFlag = Atoi(val)
		}

		bWatch := *watchFlag
		if val, exists := options["w"]; exists {
			bWatch = Atob(val)
		}

		outputPageRange := *pageFlag
		if val, exists := options["p"]; exists {
			outputPageRange = val
		}

		outputFile := *outFileFlag
		if val, exists := options["o"]; exists {
			outputFile = val
		}

		if globalVerboseFlag&4 != 0 {
			log.Printf("Read input file")
		}

		numImages := getImageDimensions(items)
		numTexts := getTextDimensions(items)

		if globalVerboseFlag&4 != 0 {
			log.Printf("Got Dimensions for %v Images and Measured %v Texts", numImages, numTexts)
		}

		// break into columns, rows
		pbBook := breakIntoPages(items)

		if globalVerboseFlag&4 != 0 {
			log.Printf("Paginated: %v pages", len(pbBook.pages))
		}

		// calculate sizes that fills available space
		if !bNoResize {
			resizePages(pbBook, outputPageRange)
		}

		// determine positions on page
		if !bNoLayout {
			layoutPages(pbBook, outputPageRange)

			if globalVerboseFlag&4 != 0 {
				log.Printf("Laid out pages")
			}
		}

		if !bNoRender {
			renderPages(pbBook, outputPageRange, outputFile)
		}

		if globalVerboseFlag&1 != 0 {
			fmt.Println(printItems(items))
		}

		if !bWatch {
			break
		}

		if globalVerboseFlag&4 != 0 {
			log.Printf("Refreshed")
		}

		changed := false
		for changed, lastModTime = fileChanged(*inFileFlag, lastModTime); !changed; changed, lastModTime = fileChanged(*inFileFlag, lastModTime) {
			time.Sleep(time.Duration(int64(1) * 1000 * 1000 * 1000))
		}
	}
}

// TTD:
// spreadeyelevel
// HSL Adjustment
// highlights, midtones, shadows
// is there something wrong with settings that column settings for break, distribute, don't work?
