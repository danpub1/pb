package main

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"log"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"
)

func escapeValue(s string) string {
	s = strings.ReplaceAll(s, "`", "``")
	s = strings.ReplaceAll(s, " ", "`_")
	return s
}

func escapeText(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\t", "\\t")
	s = strings.ReplaceAll(s, "\x00", "\\0")
	s = strings.ReplaceAll(s, "\x01", "\\1")
	s = strings.ReplaceAll(s, "\x02", "\\2")
	s = strings.ReplaceAll(s, "\x03", "\\3")
	s = strings.ReplaceAll(s, "\x04", "\\4")
	s = strings.ReplaceAll(s, "\x05", "\\5")
	s = strings.ReplaceAll(s, "\x06", "\\6")
	s = strings.ReplaceAll(s, "\x07", "\\7")
	s = strings.ReplaceAll(s, "\x08", "\\8")
	s = strings.ReplaceAll(s, "\x0B", "\\9")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}

func printSettings(theItem *PbItem, o *strings.Builder) {
	var keys []string
	for ss := range theItem.settings {
		keys = append(keys, ss)
	}
	sort.Strings(keys)
	for _, ss := range keys {
		if len(ss) > 0 && ss != "image" && ss != "text" {
			vv := theItem.Setting(ss)
			if len(vv) > 0 {
				o.WriteString(fmt.Sprintf("%v:%v ", ss, escapeValue(vv)))
			}
		}
	}
}

func hasSettingsToPrint(theItem *PbItem) bool {
	for ss, vv := range theItem.settings {
		if len(ss) > 0 && ss != "image" && ss != "text" && len(vv) > 0 {
			return true
		}
	}
	return false
}

func printItems(items []PbItem, comments bool) string {
	var o strings.Builder

	for ii, theItem := range items {
		//o.WriteString(fmt.Sprintf("// %v\n", ii+1))
		switch theItem.itemType {
		case ItemTypeBook:
			o.WriteString("*** ")
			printSettings(&theItem, &o)
			o.WriteString("\n")
		case ItemTypePage:
			o.WriteString("+++ ")
			printSettings(&theItem, &o)
			o.WriteString("\n")
		case ItemTypeRow:
			o.WriteString("--- ")
			printSettings(&theItem, &o)
			o.WriteString("\n")
		case ItemTypeColumn:
			o.WriteString("... ")
			printSettings(&theItem, &o)
			o.WriteString("\n")
		case ItemTypeText:
			if hasSettingsToPrint(&theItem) {
				o.WriteString("$ ")
				printSettings(&theItem, &o)
			}
			o.WriteString("# ")
			o.WriteString(escapeText(theItem.Setting("text")))
			o.WriteString("\n")
		case ItemTypeImage:
			o.WriteString(theItem.Setting("image"))
			o.WriteString(" ")
			if hasSettingsToPrint(&theItem) {
				o.WriteString("$ ")
				printSettings(&theItem, &o)
			}
			if len(theItem.Setting("text")) > 0 {
				o.WriteString(" # ")
				o.WriteString(escapeText(theItem.Setting("text")))
			}
			o.WriteString("\n")
			if comments {
				o.WriteString(fmt.Sprintf("/// ImageWidth:%v, ImageHeight:%v, ExifDate:%v FileDate:%v\n", theItem.imageWidthPx, theItem.imageHeightPx, theItem.exifDate, theItem.fileDate))
			}
		}
		if comments {
			if theItem.textBlockLayouts != nil {
				o.WriteString(fmt.Sprintf("/// TextBlockLayouts: %v\n", theItem.textBlockLayouts))
			}
			o.WriteString(fmt.Sprintf("/// Item %v: Page:%v, Row:%v, Column:%v, TextWidth:%v, TextHeight:%v, ImageWidth:%v, ImageHeight:%v, X:%v, Y:%v\n\n", ii, theItem.page, theItem.row, theItem.column, theItem.textWidth, theItem.textHeight, theItem.imageWidth, theItem.imageHeight, theItem.xOffset, theItem.yOffset))
		}
	}

	return o.String()
}

var rxTextStyle, _ = regexp.Compile("^(#{1,9}|#[0-9])([CRLJBE]?)$")

func decodeTextDirective(directive string) string {
	parts := rxTextStyle.FindStringSubmatch(directive)
	if len(parts) < 2 {
		log.Print("Error parsing text directive \"" + directive + "\"")
		return ""
	}
	styleNum := ""
	if len(parts[1]) == 1 || len(parts[1]) > 2 || parts[1] == "##" {
		styleNum = fmt.Sprintf("%v", len(parts[1]))
	} else {
		styleNum, _ = strings.CutPrefix(parts[1], "#")
	}

	textAlign := ""

	if len(parts[2]) > 0 {
		switch parts[2][0] {
		case 'L':
			textAlign = " text-align:left caption-align:left"
		case 'R':
			textAlign = " text-align:right caption-align:right"
		case 'C':
			textAlign = " text-align:center caption-align:center"
		case 'J':
			textAlign = " text-align:justified caption-align:justified"
		case 'B':
			textAlign = " text-align:binding caption-align:binding"
		case 'E':
			textAlign = " text-align:edge caption-align:edge"
		}
	}

	return "{{" + styleNum + "}}" + textAlign
}

var rxEscapeSpace, _ = regexp.Compile("`_")
var rxEscape, _ = regexp.Compile("`(.)")

func unescape(line string) string {
	line = unescapeEscapee(line, "`_", " ")
	line = unescapeEscapee(line, "`[^`]", "$2")
	return strings.ReplaceAll(line, "``", "`")
}

func unescapeEscapee(text string, replacee string, replacement string) string {
	runes := strings.Split(replacee, "")
	if len(runes) < 2 {
		return text
	}
	escape := runes[0]
	if escape == `\` {
		escape = escape + escape
	}
	replacee = strings.Join(runes[1:], "")
	if rex, rerr := regexp.Compile(fmt.Sprintf(`^(%[1]v%[2]v)`, escape, replacee)); rerr == nil {
		for {
			if rex.MatchString(text) {
				text = rex.ReplaceAllString(text, replacement)
			} else {
				break
			}
		}
	}
	if rex, rerr := regexp.Compile(fmt.Sprintf(`([^%[1]v])(%[1]v%[2]v)`, escape, replacee)); rerr == nil {
		for {
			if rex.MatchString(text) {
				text = rex.ReplaceAllString(text, fmt.Sprintf("$1%v", replacement))
			} else {
				break
			}
		}
	}
	if rex, rerr := regexp.Compile(fmt.Sprintf(`((%[1]v%[1]v)+)(%[1]v%[2]v)`, escape, replacee)); rerr == nil {
		for {
			if rex.MatchString(text) {
				text = rex.ReplaceAllString(text, fmt.Sprintf("$1%v", replacement))
			} else {
				break
			}
		}
	}
	return text
}

func unescapeText(text string) string {
	text = unescapeEscapee(text, `\n`, "\n")
	text = unescapeEscapee(text, `\t`, "\t")
	text = unescapeEscapee(text, `\0`, "\x00")
	text = unescapeEscapee(text, `\1`, "\x01")
	text = unescapeEscapee(text, `\2`, "\x02")
	text = unescapeEscapee(text, `\3`, "\x03")
	text = unescapeEscapee(text, `\4`, "\x04")
	text = unescapeEscapee(text, `\5`, "\x05")
	text = unescapeEscapee(text, `\6`, "\x06")
	text = unescapeEscapee(text, `\7`, "\x07")
	text = unescapeEscapee(text, `\8`, "\x08")
	text = unescapeEscapee(text, `\9`, "\x0B")
	text = unescapeEscapee(text, `\[^\\]`, "$2")
	return strings.ReplaceAll(text, "\\\\", "\\")
}

// ***|+++|---|... [Settings]
// [Image][ $ [Settings]][ # Text]
// (Styles and includes are already processed)
func parse(line string, styles map[string]string) PbItem {
	var theItem PbItem
	theItem.settings = map[string]string{}
	settingsText := ""
	settingsStyle := ""
	textStyle := ""

	if strings.HasPrefix(line, "***") {
		theItem.itemType = ItemTypeBook
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			settingsText = unescape(strings.TrimSpace(parts[1]))
		}
	} else if strings.HasPrefix(line, "+++") {
		theItem.itemType = ItemTypePage
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			settingsText = unescape(strings.TrimSpace(parts[1]))
		}
	} else if strings.HasPrefix(line, "---") {
		theItem.itemType = ItemTypeRow
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			settingsText = unescape(strings.TrimSpace(parts[1]))
		}
	} else if strings.HasPrefix(line, "...") {
		theItem.itemType = ItemTypeColumn
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			settingsText = unescape(strings.TrimSpace(parts[1]))
		}
	} else {
		theItem.itemType = ItemTypeText
		if !strings.HasPrefix(line, "$") && !strings.HasPrefix(line, "#") {
			theItem.itemType = ItemTypeImage
			dollar := strings.Index(line, " $")
			hash := strings.Index(line, " #")
			if dollar >= 0 {
				theItem.Set("image", unescape(strings.TrimSpace(line[0:dollar])))
				line = line[dollar+1:]
			} else if hash >= 0 {
				theItem.Set("image", unescape(strings.TrimSpace(line[0:hash])))
				line = line[hash+1:]
			} else {
				theItem.Set("image", unescape(strings.TrimSpace(line)))
				line = ""
			}
		}

		if strings.HasPrefix(line, "$") {
			if strings.HasPrefix(line, "$ ") {
				line = strings.TrimSpace(line[2:])
				if !strings.HasPrefix(line, "#") {
					parts := strings.SplitN(line, " #", 2)
					settingsText = strings.TrimSpace(parts[0])
					if len(parts) > 1 {
						line = "#" + parts[1]
					} else {
						line = ""
					}
				}
			} else {
				parts := strings.SplitN(line, " ", 2)
				settingsStyle = "{{" + parts[0][1:] + "}}"
				if len(parts) > 1 {
					line = strings.TrimSpace(parts[1])
					if !strings.HasPrefix(line, "#") {
						parts := strings.SplitN(line, " #", 2)
						settingsText = strings.TrimSpace(parts[0])
						if len(parts) > 1 {
							line = "#" + parts[1]
						} else {
							line = ""
						}
					}
				} else {
					line = ""
				}
			}
		}

		if strings.HasPrefix(line, "#") {
			parts := strings.SplitN(line, " ", 2)
			textStyle = decodeTextDirective(parts[0])
			if len(parts) > 1 {
				theItem.Set("text", unescapeText(parts[1]))
			} else {
				theItem.Set("text", "")
			}
		}
	}

	settingsText = settingsStyle + " " + textStyle + " " + settingsText
	settingsText = applyStyles(settingsText, styles)
	parts := strings.Split(settingsText, " ")
	for ii := range parts {
		processSetting(parts[ii], &theItem)
	}

	return theItem
}

func processSetting(setting string, theItem *PbItem) {
	setting = strings.TrimSpace(setting)
	pieces := strings.SplitN(setting, ":", 2)
	if len(pieces) == 2 {
		switch pieces[0] {
		case "trim", "fit", "squish":
			pieces[1] = pieces[0] + "," + pieces[1]
			pieces[0] = "rect"
		case "crop":
			pieces[0] = "rect"
		case "scale":
			pieces[1] = pieces[0] + ":" + pieces[1]
			pieces[0] = "size"
		case "gutter":
			switch theItem.itemType {
			case ItemTypePage:
				pieces[0] = "row-gutter"
			case ItemTypeRow:
				pieces[0] = "column-gutter"
			case ItemTypeColumn:
				pieces[0] = "item-gutter"
			default:
				log.Print("Gutter shortcut on invalid item")
			}
		}
		theItem.Set(pieces[0], unescape(pieces[1]))
	} else if len(pieces) == 1 {
		switch pieces[0] {
		case "norender", "nolayout", "noresize", "row-break", "column-break", "page-break", "current-page", "deduplicate":
			theItem.Set(pieces[0], "true")
		case "nowatch", "norecurse":
			theItem.Set(pieces[0][2:], "false")
		case "smaller", "much-smaller", "much-much-smaller", "much-much-much-smaller",
			"larger", "much-larger", "much-much-larger", "much-much-much-larger":
			theItem.Set("size", pieces[0])
		case "spreadtop", "spreadmiddle", "spreadbottom", "spreadleft", "spreadcenter", "spreadright", "spreadbinding", "spreadedge",
			"top", "middle", "bottom", "left", "center", "right", "binding", "edge", "justify":
			switch theItem.itemType {
			case ItemTypePage:
				theItem.Set("distribute-rows", pieces[0])
			case ItemTypeRow:
				theItem.Set("distribute-columns", pieces[0])
			case ItemTypeColumn:
				theItem.Set("distribute-items", pieces[0])
			default:
				log.Print("Alignment shortcut on invalid item")
			}
		case "help":
		}
	}
}

func processAsLinesFromBasePath(lines []string, basePath string, styles map[string]string) ([]PbItem, map[string]string) {
	var items []PbItem
	for _, s := range lines {
		if strings.HasPrefix(s, "$$$ ") {
			style := strings.Replace(s, "$$$ ", "", 1)
			styles = defineStyle(style, styles)
		} else if strings.HasPrefix(s, "@@@ ") {
			file2 := strings.Replace(s, "@@@ ", "", 1)
			var includedItems []PbItem
			includedItems, styles = readInputFile(localizePath(file2, basePath), styles)
			for _, includedItem := range includedItems {
				items = append(items, includedItem)
			}
		} else {
			s = applyStyles(s, styles)
			theItem := parse(s, styles)
			localizePaths(&theItem, basePath)
			newItems := expandWild(&theItem)
			if newItems != nil {
				items = append(items, newItems...)
			} else {
				items = append(items, theItem)
			}
		}
	}

	return items, styles
}

var gExts = []string{".jpg", ".jpeg", ".png", ".zip"}

var zipHeaders map[string][]*zip.File = make(map[string][]*zip.File)

func openZip(filename string) ([]*zip.File, error) {
	if headers, exists := zipHeaders[filename]; exists {
		return headers, nil
	} else if zipReader, err := zip.OpenReader(filename); err == nil {
		zipHeaders[filename] = zipReader.File
		return zipReader.File, zipReader.Close()
	} else {
		return make([]*zip.File, 0), err
	}
}

func glob(path string, recurse bool) ([]string, error) {
	log.Printf("Expanding wildcarded filename: %v", path)

	exts := gExts
	for ii := len(exts); ii > 0; ii-- {
		exts = append(exts, strings.ToUpper(exts[ii-1]))
	}

	if zipParts := strings.SplitN(path, "::", 2); len(zipParts) == 2 {
		if zipParts[1] == "*" {
			sources := make([]string, 0)
			for _, ext := range exts {
				if strings.ToLower(ext) == ".zip" {
					continue
				}
				sources2, err := glob(zipParts[0]+"::"+zipParts[1]+ext, recurse)
				if err != nil {
					log.Print(err)
					return make([]string, 0), err
				}
				if len(sources2) > 0 {
					sources = append(sources, sources2...)
				}
			}
			return sources, nil
		} else {
			if zipReaderFile, err := openZip(zipParts[0]); err != nil {
				log.Print(err)
				return make([]string, 0), err
			} else {
				sources := make([]string, 0)
				for f := range zipReaderFile {
					if matched, err := filepath.Match(zipParts[1], zipReaderFile[f].Name); err == nil && matched {
						sources = append(sources, zipParts[0]+"::"+zipReaderFile[f].Name)
					} else if err != nil {
						log.Print(err)
						return make([]string, 0), err
					}
				}
				return sources, nil
			}
		}
	} else {
		filename := filepath.Base(path)
		rPath, _ := strings.CutSuffix(path, filename)

		if filename == "*" {
			sources := make([]string, 0)
			for _, ext := range exts {
				sources2, err := glob(rPath+filename+ext, recurse)
				if err != nil {
					log.Print(err)
					return make([]string, 0), err
				}
				if len(sources2) > 0 {
					sources = append(sources, sources2...)
				}
			}
			return sources, nil
		}

		sources, err := filepath.Glob(path)
		if err != nil {
			log.Print(err)
			return make([]string, 0), err
		}

		if strings.HasSuffix(strings.ToLower(path), ".zip") {
			zipSources := make([]string, 0)
			for ii := range sources {
				sources2, err := glob(sources[ii]+"::*", recurse)
				if err != nil {
					log.Print(err)
					return make([]string, 0), err
				}
				if len(sources2) > 0 {
					zipSources = append(zipSources, sources2...)
				}
			}
			sources = zipSources
		}

		rPath = filepath.Clean(rPath)
		if recurse {
			rErr := filepath.WalkDir(rPath, func(dirpath string, dirEntry fs.DirEntry, err error) error {
				if err != nil {
					return filepath.SkipDir
				}

				if dirEntry.IsDir() && dirpath != rPath {
					rSources, err2 := glob(dirpath+string(filepath.Separator)+filename, true)
					if err2 != nil {
						log.Print(err2)
						return err2
					}

					if len(rSources) > 0 {
						sources = append(sources, rSources...)
					}
				}

				return nil
			})

			if rErr != nil {
				log.Print(rErr)
				return make([]string, 0), rErr
			}
		}
		return sources, nil
	}
}

func expandWild(item *PbItem) []PbItem {
	var newItems []PbItem

	if item.itemType == ItemTypeImage {
		imageName := item.Setting("image")
		if strings.ContainsAny(imageName, "?*") {
			sources, err := glob(imageName, item.BoolSetting("recurse"))
			if err != nil {
				log.Print(err)
				return make([]PbItem, 0)
			}
			slices.Sort(sources)
			sources = slices.Compact(sources)
			slices.SortFunc(sources, compareFilenames)
			if len(sources) > 1 {
				newItems = make([]PbItem, len(sources))
				for jj := 0; jj < len(sources); jj++ {
					newItems[jj] = item.DeepCopy()
					newItems[jj].Set("image", sources[jj])
				}
			} else if len(sources) == 1 {
				newItems = make([]PbItem, 1)
				newItems[0] = item.DeepCopy()
				newItems[0].Set("image", sources[0])
			}
		}
	}

	return newItems
}

// split filename using path separator
// also split off extension as a separate part
// Causes "name (1).jpg" to sort after "name.jpg"
func splitFilename(filename string) []string {
	if len(filename) > 0 {
		rv := strings.Split(filename, string(os.PathSeparator))
		lastPart := rv[len(rv)-1]
		lastParts := strings.Split(lastPart, ".")
		if len(lastParts) > 1 {
			lastPart = lastPart[0 : len(lastPart)-len(lastParts[len(lastParts)-1])-1]
			rv = rv[0 : len(rv)-1]
			rv = append(rv, lastPart)
			rv = append(rv, lastParts[len(lastParts)-1])
		}
		return rv
	}

	return make([]string, 0)
}

func compareFilenames(a string, b string) int {
	aparts := splitFilename(a)
	bparts := splitFilename(b)
	alen := len(aparts) - 1
	blen := len(bparts) - 1
	ii := 0
	for {
		if alen >= ii && blen >= ii {
			laparts := strings.ToLower(aparts[ii])
			lbparts := strings.ToLower(bparts[ii])
			if laparts == lbparts {
				laparts = aparts[ii]
				lbparts = bparts[ii]
			}
			if laparts == lbparts {
				if ii == alen && ii == blen {
					return 0
				} else if ii == alen {
					return -1
				} else if ii == blen {
					return 1
				} else {
					ii++
					continue
				}
			} else {
				if ii == alen && ii == blen {
					if laparts > lbparts {
						return 1
					} else {
						return -1
					}
				} else if ii == alen {
					return -1
				} else if ii == blen {
					return 1
				} else {
					if laparts > lbparts {
						return 1
					} else {
						return -1
					}
				}
			}

		} else {
			if alen < ii && blen < ii {
				return 0
			} else if alen < ii {
				return -1
			} else if blen < ii {
				return 1
			}
		}
	}
}

func compareItemsByFilename(a PbItem, b PbItem) int {
	return compareFilenames(a.Setting("image"), b.Setting("image"))
}

var rxTimeName, _ = regexp.Compile(`([0-9]{4,4})([0-9]{2,2})([0-9]{2,2})_([0-9]{2,2})([0-9]{2,2})([0-9]{2,2})`)

func itemTime(item *PbItem) time.Time {
	itemTime := timeFromName(item)
	if itemTime.IsZero() {
		itemTime = item.exifDate
	}
	if itemTime.IsZero() {
		itemTime = time.Unix(item.fileDate, 0)
	}
	return itemTime
}

func timeFromName(item *PbItem) time.Time {
	rv := time.Time{}
	name := item.Setting("image")
	if strings.Contains(name, "::") {
		parts := strings.SplitN(name, "::", 2)
		name = parts[1]
	}

	if rxTimeName.MatchString(name) {
		parts := rxTimeName.FindStringSubmatch(name)
		if len(parts) == 7 {
			year := Atoi(parts[1])
			month := time.Month(Atoi(parts[2]))
			day := Atoi(parts[3])
			hour := Atoi(parts[4])
			minute := Atoi(parts[5])
			second := Atoi(parts[6])
			location := time.Now().Location()
			if !item.exifDate.IsZero() {
				location = item.exifDate.Location()
			}
			rv = time.Date(year, month, day, hour, minute, second, 0, location)
		}
	}

	return rv
}

func compareItemsByDate(a PbItem, b PbItem) int {
	aTime := itemTime(&a)
	bTime := itemTime(&b)

	ii := aTime.Compare(bTime)
	if ii != 0 {
		return ii
	}

	return compareItemsByFilename(a, b)
}

func sortItems(items []PbItem) {
	startItem := -1
	endItem := -1
	sortSetting := ""
	for ii := range items {
		if items[ii].itemType == ItemTypeImage && startItem == -1 {
			sortSetting = items[ii].ColumnSetting("sort")
			startItem = ii
			endItem = ii
		} else if items[ii].itemType == ItemTypeImage {
			endItem = ii
		}

		if items[ii].itemType != ItemTypeImage || ii+1 == len(items) {
			if endItem > startItem {
				switch sortSetting {
				case "filename":
					slices.SortFunc(items[startItem:endItem+1], compareItemsByFilename)
					if Opts.Verbose("D") {
						log.Printf("Sorted items[%v:%v]", startItem, endItem+1)
					}
				case "date":
					slices.SortFunc(items[startItem:endItem+1], compareItemsByDate)
					if Opts.Verbose("D") {
						log.Printf("Sorted items[%v:%v]", startItem, endItem+1)
					}
				}
			}
			startItem = -1
			endItem = -1
			sortSetting = ""
		}
	}
}

func defineStyle(line string, styles map[string]string) map[string]string {
	parts := strings.SplitN(line, " ", 2)
	styles[parts[0]] = applyStyles(parts[1], styles)
	return styles
}

func applyStyles(line string, styles map[string]string) string {
	// Replace {{style}} references with content
	for kk, vv := range styles {
		line = strings.ReplaceAll(line, fmt.Sprintf("{{%v}}", kk), vv)
	}

	return line
}

var rxRootPath, _ = regexp.Compile(`^([a-z]:|[/\\])`)

func localizePath(path string, basePath string) string {
	if !rxRootPath.MatchString(path) && !strings.HasPrefix(path, "::") {
		pieces := strings.Split(path, ",")
		for ii := range pieces {
			if len(pieces[ii]) > 0 {
				pieces[ii] = basePath + pieces[ii]
			}
		}
		path = strings.Join(pieces, ",")
	}

	return path
}

func localizePaths(theItem *PbItem, basePath string) {
	if image, imageExists := theItem.settings["image"]; imageExists {
		theItem.Set("image", localizePath(image, basePath))
	}
	if font, fontExists := theItem.settings["font"]; fontExists {
		theItem.Set("font", localizePath(font, basePath))
	}
}

func makeIntoLines(lines []string) []string {
	var f []string

	for ii, s := range lines {
		s = strings.TrimRight(s, " ")
		if len(s) != 0 {
			continuation := strings.HasPrefix(s, " ")
			if continuation {
				s = strings.TrimLeft(s, " ")
			}
			if !strings.HasPrefix(s, "/// ") && s != "///" && !(ii == 0 && !continuation && strings.HasPrefix(s, "#!")) {
				if continuation {
					if len(f) > 0 {
						f[len(f)-1] += " " + s
					}
				} else {
					f = append(f, s)
				}
			}
		}
	}

	return f
}

func ApplyItemSpecificStyles(items []PbItem) {
	today := time.Now()
	date := today.Format("02-Jan-2006")
	year := today.Format("2006")
	for ii := range items {
		if items[ii].itemType == ItemTypeImage || items[ii].itemType == ItemTypeText {
			text := items[ii].Setting("text")
			if items[ii].itemType == ItemTypeImage {
				imageActualName := items[ii].Setting("image")
				imageName := imageActualName
				if imageNameParts := strings.SplitN(imageName, "::", 2); len(imageNameParts) == 2 {
					imageName = imageNameParts[1]
				}
				text = strings.ReplaceAll(text, "{{Filename}}", filepath.Base(imageName))  // Deprecated
				text = strings.ReplaceAll(text, "{{Fullname}}", filepath.Clean(imageName)) // Deprecated
				text = strings.ReplaceAll(text, "{{FileName}}", filepath.Base(imageName))
				text = strings.ReplaceAll(text, "{{FullName}}", filepath.Clean(imageName))
				text = strings.ReplaceAll(text, "{{ImageName}}", imageActualName)
				text = strings.ReplaceAll(text, "{{ImageDate}}", itemTime(&items[ii]).Format(time.DateTime))
				text = strings.ReplaceAll(text, "{{FileDate}}", time.Unix(items[ii].fileDate, 0).Format(time.DateTime))
				text = strings.ReplaceAll(text, "{{ExifDate}}", items[ii].exifDate.Format(time.DateTime))
			}
			if items[ii].itemType == ItemTypeText {
				if strings.Contains(text, "{{NextImageDate}}") {
					replacement := "(Undated)"
					for jj := ii + 1; jj < len(items); jj++ {
						if items[jj].itemType == ItemTypeImage {
							imageDate := items[jj].exifDate
							if imageDate.IsZero() && items[jj].fileDate != 0 {
								imageDate = time.Unix(items[jj].fileDate, 0)
							}
							if !imageDate.IsZero() {
								replacement = imageDate.Format("Monday January 2, 2006")
							}
							break
						}
					}
					text = strings.ReplaceAll(text, "{{NextImageDate}}", replacement)
				}
			}

			text = strings.ReplaceAll(text, "{{Date}}", date)
			text = strings.ReplaceAll(text, "{{Year}}", year)
			items[ii].Set("text", text)
		}
	}
}

type dedupEntry struct {
	size     uint64
	filename string
	index    int
}

func deduplicate(items []PbItem) []PbItem {
	if len(items) > 0 {
		if items[0].BoolSetting("deduplicate") {
			if Opts.Verbose("D") {
				log.Printf("Deduplicating %v items", len(items))
			}

			// Make a list of everything sorted by size
			sizes := make([]dedupEntry, 0, len(items))
			for ii := range items {
				if items[ii].itemType == ItemTypeImage {
					filename := items[ii].Setting("image")
					sizes = append(sizes, dedupEntry{fileSize(filename), filename, ii})
				}
			}
			slices.SortFunc(sizes, func(a dedupEntry, b dedupEntry) int {
				if a.size > b.size {
					return 1
				} else if b.size > a.size {
					return -1
				} else if a.index > b.index {
					return 1
				} else if b.index > a.index {
					return -1
				} else {
					return 0
				}
			})

			// identify the duplicates by size in that list
			possibleDups := make([]dedupEntry, 0)
			var last *dedupEntry = &dedupEntry{0, "", -1}
			inDup := false
			for ii := range sizes {
				if sizes[ii].size == last.size {
					if !inDup {
						possibleDups = append(possibleDups, *last)
					}
					possibleDups = append(possibleDups, sizes[ii])
					inDup = true
				} else {
					inDup = false
					last = &sizes[ii]
				}
			}

			// identify the duplicates by hash of the ones that are duplicated by size
			hashes := make([]string, 0, len(possibleDups))
			filenames := make([]string, 0, len(possibleDups))
			actualDups := make([]int, 0)
			deduplicated := 0
			for ii := 0; ii < len(possibleDups); ii++ {
				hash := items[possibleDups[ii].index].HashImage()
				if !slices.Contains(hashes, hash) {
					hashes = append(hashes, hash)
					filenames = append(filenames, possibleDups[ii].filename)
				} else {
					actualDups = append(actualDups, possibleDups[ii].index)
					if Opts.Verbose("D") {
						log.Printf("Deduplicating %v, keeping %v", possibleDups[ii].filename, filenames[slices.Index(hashes, hash)])
					}
				}
			}

			slices.Sort(actualDups)
			slices.Reverse(actualDups)

			for _, ii := range actualDups {
				items = slices.Delete(items, ii, ii+1)
				deduplicated++
			}

			if deduplicated > 0 {
				for ii := range items {
					items[ii].pb = items
				}

				OptimizeSettings(items)

				if Opts.Verbose("D") {
					log.Printf("Deduplicated %v items", deduplicated)
				}
			}
		}
	}

	return items
}

func addDayHeaders(items []PbItem) []PbItem {

	if len(items) == 0 {
		return items
	}

	dayHeaders := items[0].BookSetting("day-headers")
	title := items[0].BookSetting("title")
	subtitle := items[0].BookSetting("subtitle")

	if dayHeaders == "" && title == "" && subtitle == "" {
		return items
	}

	var dayHeader PbItem
	var dayHeaderPage *PbItem
	var titleItem *PbItem
	var subtitleItem *PbItem
	found := false

	if dayHeaders != "auto" && dayHeaders != "" {
		for ii := range items {
			if items[ii].itemType == ItemTypeText && items[ii].Setting("name") == dayHeaders {
				dayHeader = items[ii].DeepCopy()
				found = true
			}
		}
	}

	if dayHeaders == "auto" || (dayHeaders != "" && !found) {
		dayHeader = PbItem{}
		dayHeader.itemType = ItemTypeText
		dayHeader.settings = map[string]string{}
		dayHeader.settings["text"] = "{{NextImageDate}}"
		dayHeader.settings["font"] = "::FiraSans-SemiBold.otf"
		dayHeader.settings["font-size"] = "20"
		dayHeader.settings["page-break"] = "true"
		dayHeader.pb = items

		if items[0].BookSetting("distribute-rows") != "spreadtop" {
			dayHeaderPage = &PbItem{}
			dayHeaderPage.itemType = ItemTypePage
			dayHeaderPage.settings = map[string]string{}
			dayHeaderPage.settings["distribute-rows"] = "spreadtop"
			dayHeaderPage.pb = items
		}
	} else {
		if dayHeader.Setting("distribute-rows") != "spreadtop" {
			dayHeaderPage = &PbItem{}
			dayHeaderPage.itemType = ItemTypePage
			dayHeaderPage.settings = map[string]string{}
			dayHeaderPage.settings["distribute-rows"] = "spreadtop"
			dayHeaderPage.pb = items
		}
	}

	if len(title) > 0 {
		titleItem = &PbItem{}
		titleItem.itemType = ItemTypeText
		titleItem.settings = map[string]string{}
		titleItem.settings["text"] = unescapeText(title)
		titleItem.settings["font"] = "::FiraSans-Heavy.otf"
		titleItem.settings["font-size"] = "36"
		titleItem.settings["text-align"] = "center"
		titleItem.pb = items
	}

	if len(subtitle) > 0 {
		subtitleItem = &PbItem{}
		subtitleItem.itemType = ItemTypeText
		subtitleItem.settings = map[string]string{}
		subtitleItem.settings["text"] = unescapeText(subtitle)
		subtitleItem.settings["font"] = "::FiraSans-Regular.otf"
		subtitleItem.settings["font-size"] = "14"
		subtitleItem.settings["text-align"] = "center"
		subtitleItem.pb = items
	}

	var lastDay time.Time
	firstHeader := true

	for ii := 0; ii < len(items); ii++ {
		if items[ii].itemType == ItemTypeImage {
			if firstHeader && titleItem != nil {
				items = slices.Insert(items, ii, titleItem.DeepCopy())
				ii++
			}
			if firstHeader && subtitleItem != nil {
				items = slices.Insert(items, ii, subtitleItem.DeepCopy())
				ii++
			}
			imageDate := itemTime(&items[ii])
			if firstHeader || imageDate.Year() != lastDay.Year() || imageDate.Month() != lastDay.Month() || imageDate.Day() != lastDay.Day() {
				if len(dayHeaders) > 0 {
					if dayHeaderPage != nil {
						items = slices.Insert(items, ii, dayHeaderPage.DeepCopy())
						ii++
					}
					items = slices.Insert(items, ii, dayHeader.DeepCopy())
					ii++
				} else if firstHeader && (titleItem != nil || subtitleItem != nil) {
					newPage := &PbItem{}
					newPage.itemType = ItemTypePage
					newPage.settings = map[string]string{}
					newPage.pb = items
					items = slices.Insert(items, ii, newPage.DeepCopy())
					ii++
				}
				lastDay = imageDate
				firstHeader = false
			}
		}
	}

	for ii := range items {
		items[ii].pb = items
	}

	OptimizeSettings(items)
	return items
}

func ApplyDefaultCaptions(items []PbItem) {
	for ii := range items {
		if items[ii].itemType == ItemTypeImage || items[ii].itemType == ItemTypeText {
			if items[ii].itemType == ItemTypeImage {
				caption := items[ii].Setting("caption")
				if len(caption) != 0 && len(items[ii].Setting("text")) == 0 {
					items[ii].Set("text", unescapeText(caption))
				}
			}
		}
	}
}

func readInputFile(inFile string, styles map[string]string) ([]PbItem, map[string]string) {
	var inBytes []byte
	var err error

	basePath := ""

	var inStrings []string
	var fi os.FileInfo
	fi, err = os.Stat(inFile)

	if err == nil && fi.IsDir() {
		if !strings.HasSuffix(inFile, string(filepath.Separator)) {
			inFile = inFile + string(filepath.Separator) + "*"
		}
	}

	lowerFile := strings.ToLower(inFile)

	if strings.HasSuffix(lowerFile, ".zip") {
		inFile = inFile + "::*"
		lowerFile = strings.ToLower(inFile)
	}

	foundPrefix := false
	if inFile, foundPrefix = strings.CutPrefix(inFile, "@"); foundPrefix {
		inFile = "@@@ " + inFile
		lowerFile = strings.ToLower(inFile)
	}

	foundExt := false
	for _, ext := range gExts {
		if strings.HasSuffix(lowerFile, ext) {
			foundExt = true
		}
	}

	if strings.Contains(inFile, "*") || strings.HasPrefix(inFile, "@@@ ") || foundExt {
		inStrings = make([]string, 0)
		inStrings = append(inStrings, inFile)
	} else {
		if inFile == "-" {
			inBytes, err = io.ReadAll(os.Stdin)
		} else {
			inBytes, err = os.ReadFile(inFile)
		}

		if err != nil {
			log.Print(err)
			return make([]PbItem, 0), map[string]string{}
		}

		inStrings = strings.Split(string(inBytes), "\n")
		basePath = BasePath(inFile)
	}

	inStrings = makeIntoLines(inStrings)

	var items []PbItem
	items, styles = processAsLinesFromBasePath(inStrings, basePath, styles)

	return items, styles
}

func BasePath(fileName string) string {
	basePath := filepath.Dir(fileName) + "/"
	if basePath == "./" {
		basePath = ""
	}
	return basePath
}

func ReadPbFile(inFiles []string, args []string) []PbItem {
	items := make([]PbItem, 0)
	for _, inFile := range inFiles {
		oneItems, _ := readInputFile(inFile, map[string]string{})
		items = append(items, oneItems...)
	}

	bookIdxs := make([]int, 0)
	for ii := range items {
		if items[ii].itemType == ItemTypeBook {
			bookIdxs = append(bookIdxs, ii)
			break
		}
	}

	// Need to have a book to put command line options
	// If no book or book is not first, create an empty book
	if len(bookIdxs) == 0 || bookIdxs[0] != 0 {
		thebook := PbItem{}
		thebook.itemType = ItemTypeBook
		thebook.settings = map[string]string{}
		newitems := make([]PbItem, 0)
		newitems = append(newitems, thebook)
		newitems = append(newitems, items...)
		items = newitems
	}

	// If multiple books, merge them into the first book, which is now the zeroth element
	if len(bookIdxs) > 1 || (len(bookIdxs) == 1 && bookIdxs[0] != 0) {
		for ii := range items {
			if ii > 0 {
				maps.Copy(items[0].settings, items[ii].settings)
			}
		}

		for ii := len(items) - 1; ii > 0; ii-- {
			if items[ii].itemType == ItemTypeBook {
				newitems := make([]PbItem, 0)
				newitems = append(newitems, items[0:ii]...)
				if ii < len(items)-1 {
					newitems = append(newitems, items[ii+1:]...)
				}
			}
		}
	}

	for _, arg := range args {
		if setting, found := strings.CutPrefix(arg, "--"); found {
			processSetting(setting, &items[0])
		}
	}

	for ii := range items {
		items[ii].pb = items
	}

	OptimizeSettings(items)

	ApplyDefaultCaptions(items)

	return items
}
