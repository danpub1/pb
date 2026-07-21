package main

import (
	"archive/zip"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"
	"time"
)

func isPageInRange(pageRange string, pageNum int, firstIteration bool) bool {
	if len(pageRange) == 0 || pageRange == "-" || (pageRange == "$" && firstIteration) {
		return true
	}

	// internally the page numbering is zero based, the external page number is one based
	pageNum += 1

	pageRanges := strings.SplitSeq(pageRange, ",")

	for aRange := range pageRanges {
		if aRange == "*" || (aRange == "$" && !firstIteration) {
			return true
		} else if strings.HasPrefix(aRange, "-") {
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

func isPageRangeMulti(pageRange string, firstIteration bool, pbBook *PbBook) bool {
	pageCount := 0
	for pp := range pbBook.pages {
		if isPageInRange(pageRange, pp, firstIteration) {
			pageCount++
			if pageCount > 1 {
				return true
			}
		}
	}

	return false
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

func fileDate(filename string) int64 {
	rv := time.Time{}.Unix()

	if strings.Contains(filename, "::") {
		parts := strings.SplitN(filename, "::", 2)
		if zipFiles, err := openZip(parts[0]); err == nil {
			idx := slices.IndexFunc(zipFiles, func(zipFile *zip.File) bool { return zipFile.Name == parts[1] })
			if idx != -1 {
				// log.Printf("Filename: %v, Time: %v", filename, zipFiles[idx].Modified.Unix())
				rv = zipFiles[idx].Modified.Unix()
			} else {
				// log.Printf("%v\n", filename+" not found")
				fi, err := os.Stat(parts[0])
				if err == nil {
					rv = fi.ModTime().Unix()
				}
			}
		} else {
			// log.Printf("Error %v opening %v\n", err, filename)
			fi, err := os.Stat(parts[0])
			if err == nil {
				rv = fi.ModTime().Unix()
			}
		}
	} else {
		fi, err := os.Stat(filename)
		if err == nil {
			rv = fi.ModTime().Unix()
		}
	}

	return rv
}

func fileSize(filename string) uint64 {
	var rv uint64 = 0

	if strings.Contains(filename, "::") {
		parts := strings.SplitN(filename, "::", 2)
		if zipFiles, err := openZip(parts[0]); err == nil {
			idx := slices.IndexFunc(zipFiles, func(zipFile *zip.File) bool { return zipFile.Name == parts[1] })
			if idx != -1 {
				// log.Printf("Filename: %v, Time: %v", filename, zipFiles[idx].Modified.Unix())
				rv = zipFiles[idx].UncompressedSize64
			} else {
				// log.Printf("%v\n", filename+" not found")
				fi, err := os.Stat(parts[0])
				if err == nil {
					rv = uint64(fi.Size())
				}
			}
		} else {
			// log.Printf("Error %v opening %v\n", err, filename)
			fi, err := os.Stat(parts[0])
			if err == nil {
				rv = uint64(fi.Size())
			}
		}
	} else {
		fi, err := os.Stat(filename)
		if err == nil {
			rv = uint64(fi.Size())
		}
	}

	return rv
}

func hasFilesToWatch(filenames []string) bool {
	for _, filename := range filenames {
		if !strings.Contains(filename, "*") && !strings.Contains(filename, "::") {
			if _, err := os.Stat(filename); err == nil {
				return true
			}
		}
	}

	return false
}

func fileChanged(filenames []string, lastModTime time.Time) (bool, time.Time) {

	newModTime := time.Time{}
	for _, filename := range filenames {
		if !strings.Contains(filename, "*") && !strings.Contains(filename, "::") {
			filename, _ = strings.CutPrefix(filename, "@")
			fi, err := os.Stat(filename)
			if err != nil {
				log.Print(err)
				return false, lastModTime
			}

			thisModTime := fi.ModTime()
			if thisModTime.After(newModTime) {
				newModTime = thisModTime
			}
		}
	}

	return lastModTime.Before(newModTime), newModTime

}

const (
	CacheModeImage        = 3
	CacheModeImageDuring  = 1
	CacheModeImageFull    = 2
	CacheModeResize       = 12
	CacheModeResizeDuring = 4
	CacheModeResizeFull   = 8
)

var lastModTime time.Time

type Options struct {
	book *PbItem
}

var Opts Options

func (this *Options) Set(items []PbItem) {
	this.book = nil
	for ii := range items {
		items[ii].pb = items
		if items[ii].itemType == ItemTypeBook && this.book == nil {
			this.book = &items[ii]
		}
	}
}

func (this *Options) Verbose(level string) bool {
	return strings.Contains(this.book.BookSetting("verbose"), level)
}

func (this *Options) PageRange() string {
	return this.book.BookSetting("page-range")
}

func (this *Options) Watch() bool {
	return this.book.BoolBookSetting("watch")
}

func (this *Options) Cache() int {
	return this.book.IntBookSetting("cache-mode")
}

var inFiles []string

func main() {
	args := os.Args[1:]

	inFiles = make([]string, 0)

	for _, arg := range args {
		if !strings.HasPrefix(arg, "--") {
			inFiles = append(inFiles, arg)
		}
		if arg == "--help" {
			fmt.Println(printHelp())
			return
		}
	}

	_, lastModTime = fileChanged(inFiles, time.Time{})

	var lastIteration []string = nil
	var firstIteration = true

	for {
		items := ReadPbFile(inFiles, args)

		Opts.Set(items)

		if firstIteration && len(inFiles) == 0 && (len(items) == 0 || len(items[0].BookSetting("assemble")) == 0) {
			log.Print("No input specified")
		}

		if Opts.Verbose("D") {
			log.Printf("Read input file")
		}

		numImages := getImageDimensions(items)
		sortItems(items)
		items = deduplicate(items)
		items = addDayHeaders(items)

		ApplyItemSpecificStyles(items) // Needs exifDate & fileDate from getImageDimensions
		numTexts := getTextDimensions(items)

		if Opts.Verbose("D") {
			log.Printf("Got Dimensions for %v Images and Measured %v Texts", numImages, numTexts)
		}

		if Opts.Verbose("P") {
			fmt.Println(printItems(items, false))
		}

		// break into columns, rows
		pbBook := breakIntoPages(items)

		if Opts.Verbose("D") {
			log.Printf("Paginated: %v pages", len(pbBook.pages))
		}

		flat := pbBook.Flatten()
		pageRange := Opts.PageRange()
		includePages := "0"
		if lastIteration != nil {
			for ii, pp := range flat {
				if !slices.Contains(lastIteration, pp) {
					includePages = fmt.Sprintf("%v,%v", includePages, ii+1)
				}
			}
		}
		if !firstIteration {
			pageRange = strings.ReplaceAll(pageRange, "$", includePages)
		}
		pageRange = strings.ReplaceAll(pageRange, "*", includePages)
		lastIteration = flat

		// calculate sizes that fills available space
		resizePages(pbBook, pageRange, firstIteration)

		if Opts.Verbose("D") {
			log.Printf("Resized pages")
		}

		// determine positions on page
		layoutPages(pbBook, pageRange, firstIteration)

		if Opts.Verbose("D") {
			log.Printf("Laid out pages")
		}

		renderTextImages(pbBook)
		renderPages(pbBook, pageRange, firstIteration, flat)
		assemble(items)

		if Opts.Verbose("X") {
			fmt.Println(printItems(items, true))
		}

		if !hasFilesToWatch(inFiles) || !Opts.Watch() {
			break
		}

		if Opts.Verbose("D") {
			log.Printf("Refreshed")
		}

		changed := false
		for changed, lastModTime = fileChanged(inFiles, lastModTime); !changed; changed, lastModTime = fileChanged(inFiles, lastModTime) {
			time.Sleep(time.Duration(int64(1) * 1000 * 1000 * 1000))
		}

		if pageRange == "$" {
			firstIteration = false
		}
	}
}
