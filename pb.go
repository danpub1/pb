package main

import (
	"flag"
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
		log.Fatal(err)
	}

	thisModTime := fi.ModTime()

	return lastModTime != thisModTime, thisModTime
}

const (
	CacheModeNone = iota
	CacheModeDuring
	CacheModeFull
)

var globalVerboseFlag = 0
var cacheMode *int
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
	cacheMode = flag.Int("cache", CacheModeDuring, "cache mode")

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

			if globalVerboseFlag&4 != 0 {
				log.Printf("Resized pages")
			}
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
// Float
// Headers, Footers
// is there something wrong with settings that column settings for break, distribute, don't work?
//   (some) header settings for text seem to have to be on a row instead of on a column
//
// Output sigmoidal brightness/lightness?
// eyelevel, spreadeyelevel, mouthlevel, spreadmouthlevel?
// highlights, midtones, shadows
// Image, font zip:filename:filename - read directly from zip and/or cache
// Image, font https://... downloaded and then cached (in a zip file?)
// HSL Adjustment
// Frame
// Background
// Justify text
// Italics, Bold, Bold-Italics
