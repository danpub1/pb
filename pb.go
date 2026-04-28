package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

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

	fi, err := os.Stat(filename)
	if err == nil {
		rv = fi.ModTime().Unix()
	}

	return rv
}

func fileChanged(filename string, lastModTime time.Time) (bool, time.Time) {
	fi, err := os.Stat(filename)
	if err != nil {
		log.Print(err)
		return false, lastModTime
	}

	thisModTime := fi.ModTime()

	return lastModTime != thisModTime, thisModTime
}

const (
	CacheModeUnknown      = 0
	CacheModeImageNone    = 1
	CacheModeImageDuring  = 2
	CacheModeImageFull    = 4
	CacheModeResizeNone   = 8
	CacheModeResizeDuring = 16
	CacheModeResizeFull   = 32
	CacheModeAll          = 63
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

var inFile = ""

func main() {
	args := os.Args[1:]

	for _, arg := range args {
		if !strings.HasPrefix(arg, "--") {
			if inFile == "" {
				inFile = arg
			} else {
				log.Printf("Only one input file is allowed, %v ignored", arg)
			}
		}
	}

	if inFile == "" {
		log.Print("No input file specified")
		return
	}

	_, lastModTime = fileChanged(inFile, time.Time{})

	for {
		items := ReadPbFile(inFile, args)

		Opts.Set(items)

		if Opts.Verbose("D") {
			log.Printf("Read input file")
		}

		if Opts.Verbose("P") {
			fmt.Println(printItems(items, false))
		}

		numImages := getImageDimensions(items)
		numTexts := getTextDimensions(items)

		if Opts.Verbose("D") {
			log.Printf("Got Dimensions for %v Images and Measured %v Texts", numImages, numTexts)
		}

		// break into columns, rows
		pbBook := breakIntoPages(items)

		if Opts.Verbose("D") {
			log.Printf("Paginated: %v pages", len(pbBook.pages))
		}

		// calculate sizes that fills available space
		resizePages(pbBook, Opts.PageRange())

		if Opts.Verbose("D") {
			log.Printf("Resized pages")
		}

		// determine positions on page
		layoutPages(pbBook, Opts.PageRange())

		if Opts.Verbose("D") {
			log.Printf("Laid out pages")
		}

		renderPages(pbBook, Opts.PageRange())

		if Opts.Verbose("X") {
			fmt.Println(printItems(items, true))
		}

		if !Opts.Watch() {
			break
		}

		if Opts.Verbose("D") {
			log.Printf("Refreshed")
		}

		changed := false
		for changed, lastModTime = fileChanged(inFile, lastModTime); !changed; changed, lastModTime = fileChanged(inFile, lastModTime) {
			time.Sleep(time.Duration(int64(1) * 1000 * 1000 * 1000))
		}
	}
}
