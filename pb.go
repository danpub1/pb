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
	CacheModeUnknown      = 0
	CacheModeImageNone    = 1
	CacheModeImageDuring  = 2
	CacheModeImageFull    = 4
	CacheModeResizeNone   = 8
	CacheModeResizeDuring = 16
	CacheModeResizeFull   = 32
	CacheModeAll          = 63
)
const (
	WatchModeUnknown = -1
	WatchModeOff     = 0
	WatchModeOn      = 1
)
const (
	NoresizeModeUnknown = -1
	NoresizeModeOff     = 0
	NoresizeModeOn      = 1
)
const (
	NolayoutModeUnknown = -1
	NolayoutModeOff     = 0
	NolayoutModeOn      = 1
)
const (
	NorenderModeUnknown = -1
	NorenderModeOff     = 0
	NorenderModeOn      = 1
)

var lastModTime time.Time

type OptionSet struct {
	inFile    string
	outFile   string
	verbose   string
	pageRange string
	cjpegCmd  string
	watch     int
	cacheMode int
	noresize  int
	nolayout  int
	norender  int
}

type Options struct {
	fileOptions OptionSet
	argsOptions OptionSet
}

var Opts Options

func (this *Options) ParseFileOptions(options map[string]string) {
	if val, exists := options["o"]; exists {
		this.fileOptions.outFile = val
	}

	if val, exists := options["v"]; exists {
		this.fileOptions.verbose = val
	}

	if val, exists := options["p"]; exists {
		this.fileOptions.pageRange = val
	}

	if val, exists := options["w"]; exists {
		this.fileOptions.watch = Atoi(val)
	}

	if val, exists := options["cache"]; exists {
		this.fileOptions.cacheMode = Atoi(val)
	}

	if val, exists := options["noresize"]; exists {
		this.fileOptions.noresize = Atoi(val)
	}

	if val, exists := options["nolayout"]; exists {
		this.fileOptions.nolayout = Atoi(val)
	}

	if val, exists := options["norender"]; exists {
		this.fileOptions.norender = Atoi(val)
	}

	if val, exists := options["cjpeg"]; exists {
		this.fileOptions.cjpegCmd = val
	}
}

func (this *Options) InFile() string {
	return this.argsOptions.inFile
}

func (this *Options) OutFile() string {
	if len(this.argsOptions.outFile) > 0 {
		return this.argsOptions.outFile
	} else {
		return this.fileOptions.outFile
	}
}

func (this *Options) Verbose(level string) bool {
	return (len(this.argsOptions.verbose) > 0 && strings.Contains(this.argsOptions.verbose, level)) ||
		(len(this.argsOptions.verbose) == 0 && len(this.fileOptions.verbose) > 0 && strings.Contains(this.fileOptions.verbose, level))
}

func (this *Options) PageRange() string {
	if len(this.argsOptions.pageRange) > 0 {
		return this.argsOptions.pageRange
	} else {
		return this.fileOptions.pageRange
	}
}

func (this *Options) Watch() bool {
	if this.argsOptions.watch != WatchModeUnknown {
		return this.argsOptions.watch == WatchModeOn
	} else {
		return this.fileOptions.watch == WatchModeOn
	}
}

func (this *Options) Cache() int {
	if this.argsOptions.cacheMode != CacheModeUnknown {
		return this.argsOptions.cacheMode
	} else {
		return this.fileOptions.cacheMode
	}
}

func (this *Options) Noresize() bool {
	if this.argsOptions.noresize != NoresizeModeUnknown {
		return this.argsOptions.noresize == NoresizeModeOn
	} else {
		return this.fileOptions.noresize == NoresizeModeOn
	}
}

func (this *Options) Nolayout() bool {
	if this.argsOptions.nolayout != NolayoutModeUnknown {
		return this.argsOptions.nolayout == NolayoutModeOn
	} else {
		return this.fileOptions.nolayout == NolayoutModeOn
	}
}

func (this *Options) Norender() bool {
	if this.argsOptions.norender != NorenderModeUnknown {
		return this.argsOptions.norender == NorenderModeOn
	} else {
		return this.fileOptions.norender == NorenderModeOn
	}
}

func (this *Options) CjpegCmd() string {
	if len(this.argsOptions.cjpegCmd) > 0 {
		return this.argsOptions.cjpegCmd
	} else {
		return this.fileOptions.cjpegCmd
	}
}

func main() {
	Opts.fileOptions = OptionSet{
		inFile:    "",
		outFile:   "",
		verbose:   "",
		pageRange: "-",
		cacheMode: CacheModeImageNone | CacheModeResizeNone,
		watch:     WatchModeOff,
		noresize:  NoresizeModeOff,
		nolayout:  NolayoutModeOff,
		norender:  NorenderModeOff,
		cjpegCmd:  "",
	}

	flag.StringVar(&(Opts.argsOptions.inFile), "i", "", "Input File")
	flag.StringVar(&(Opts.argsOptions.outFile), "o", "", "Output File")
	flag.StringVar(&(Opts.argsOptions.verbose), "v", "", "Verbose Level")
	flag.StringVar(&(Opts.argsOptions.pageRange), "p", "", "Page Range")
	flag.IntVar(&(Opts.argsOptions.watch), "w", WatchModeUnknown, "Watch Level")
	flag.IntVar(&(Opts.argsOptions.cacheMode), "cache", CacheModeUnknown, "Cache Level")
	flag.IntVar(&(Opts.argsOptions.noresize), "noresize", NoresizeModeUnknown, "No Resize")
	flag.IntVar(&(Opts.argsOptions.nolayout), "nolayout", NolayoutModeUnknown, "No Layout")
	flag.IntVar(&(Opts.argsOptions.norender), "norender", NorenderModeUnknown, "No Render")
	flag.StringVar(&(Opts.argsOptions.cjpegCmd), "cjpeg", "", "cjpeg path")
	flag.Parse()

	if Opts.InFile() == "" {
		flag.Usage()
		log.Fatal("no input file specified")
	}

	_, lastModTime = fileChanged(Opts.InFile(), time.Time{})

	for {
		items, options := ReadPbFile(Opts.InFile())

		Opts.ParseFileOptions(options)

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
		if !Opts.Noresize() {
			resizePages(pbBook, Opts.PageRange())

			if Opts.Verbose("D") {
				log.Printf("Resized pages")
			}
		}

		// determine positions on page
		if !Opts.Nolayout() {
			layoutPages(pbBook, Opts.PageRange())

			if Opts.Verbose("D") {
				log.Printf("Laid out pages")
			}
		}

		if !Opts.Norender() {
			renderPages(pbBook, Opts.PageRange(), Opts.OutFile())
		}

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
		for changed, lastModTime = fileChanged(Opts.InFile(), lastModTime); !changed; changed, lastModTime = fileChanged(Opts.InFile(), lastModTime) {
			time.Sleep(time.Duration(int64(1) * 1000 * 1000 * 1000))
		}
	}
}
