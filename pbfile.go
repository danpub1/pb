package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
)

func quote(s string) string {
	if strings.Contains(s, " ") || strings.Contains(s, "`") {
		return fmt.Sprintf("`%v`", strings.ReplaceAll(strings.ReplaceAll(s, "\\", "\\\\"), "`", "\\`"))
	}
	return s
}

func printSettings(theItem PbItem, o *strings.Builder) {
	var keys []string
	for ss := range theItem.settings {
		keys = append(keys, ss)
	}
	sort.Strings(keys)
	for _, ss := range keys {
		if len(ss) > 0 {
			vv := theItem.Setting(ss)
			if len(vv) > 0 {
				o.WriteString(fmt.Sprintf("%v:%v ", ss, quote(vv)))
			}
		}
	}
}

func printItems(items []PbItem, depth int) string {
	var o strings.Builder

	for _, theItem := range items {
		//o.WriteString(fmt.Sprintf("// %v\n", ii+1))
		if depth > 0 {
			o.WriteString(fmt.Sprintf("%v ", strings.Repeat(">", depth)))
		}
		switch theItem.itemType {
		case ItemTypeBook:
			o.WriteString("*** ")
			printSettings(theItem, &o)
		case ItemTypePage:
			o.WriteString("+++ ")
			printSettings(theItem, &o)
		case ItemTypeRow:
			o.WriteString("--- ")
			printSettings(theItem, &o)
		case ItemTypeText:
			o.WriteString("# ")
			printSettings(theItem, &o)
		case ItemTypeImage:
			o.WriteString("! ")
			printSettings(theItem, &o)
		case ItemTypeColumn:
			o.WriteString(fmt.Sprintf("| %v", theItem.Setting("column-width")))
		}
		o.WriteString("\n")
	}

	return o.String()
}

var rxValidDirective, _ = regexp.Compile(`^((#{1,9}|#[1-9])[CRLJBE]?)|!|---|\+\+\+|\*\*\*$`)
var rxTextDirective, _ = regexp.Compile("^(#{1,9}|#[1-9])([CRLJBE]?)$")
var rxChildDirective, _ = regexp.Compile("^>+$")
var rxSetting, _ = regexp.Compile("^([a-z-]+):(.+)$")

func finishAccumulating(accumulating strings.Builder, theItem PbItem) (strings.Builder, PbItem) {
	if accumulating.Len() > 0 {
		switch theItem.itemType {
		case ItemTypeUnknown:
			theItem.itemType = ItemTypeImage
			theItem.Set("image", accumulating.String())
		case ItemTypeImage:
			theItem.Set("image", accumulating.String())
		case ItemTypeText:
			theItem.Set("text", accumulating.String())
		}
		accumulating.Reset()
	} else {
		if theItem.itemType == ItemTypeUnknown {
			theItem.itemType = ItemTypeText
		}
	}

	return accumulating, theItem
}

func handleTextDirective(directive string, theItem PbItem, styles map[string]string) PbItem {
	parts := rxTextDirective.FindStringSubmatch(directive)
	styleNum := ""
	if len(parts[1]) == 1 || len(parts[1]) > 2 || parts[1] == "##" {
		styleNum = fmt.Sprintf("%v", len(parts[1]))
	} else {
		styleNum, _ = strings.CutPrefix(parts[1], "#")
	}

	if style, exists := styles[styleNum]; exists {
		for _, stylePart := range tokenize(style) {
			if rxSetting.MatchString(stylePart) {
				styleParts := rxSetting.FindStringSubmatch(stylePart)
				theItem.Set(styleParts[1], styleParts[2])
			}
		}
	}

	if len(parts[2]) > 0 {
		switch parts[2][0] {
		case 'L':
			theItem.Set("text-align", "left")
		case 'R':
			theItem.Set("text-align", "right")
		case 'C':
			theItem.Set("text-align", "center")
		case 'J':
			theItem.Set("text-align", "justified")
		case 'B':
			theItem.Set("text-align", "binding")
		case 'E':
			theItem.Set("text-align", "edge")
		}
	}

	return theItem
}

// > Optional Child Directive
// ***, +++, --- Book, Page, Row + [Settings]
// [!] + Image + [Settings] + [[# + [Settings]] + Text]
// [#] + [Settings] + Text
// (Styles and includes are already processed)
func parse(line string, styles map[string]string) PbItem {
	var theItem PbItem
	theItem.settings = map[string]string{}
	var accumulating strings.Builder
	accumulatingImage := true // assume accumulating an image until a settings is found
	gotTextDirective := false

	for ii, part := range tokenize(line) {

		// Leading child directive
		if ii == 0 && rxChildDirective.MatchString(part) {
			theItem.depth = len(part)
			continue
		}

		// Leading directive, possibly after a child directive
		if (ii == 1 && theItem.depth > 0 || ii == 0) && rxValidDirective.MatchString(part) {
			switch part[0] {
			case '*':
				theItem.itemType = ItemTypeBook
			case '+':
				theItem.itemType = ItemTypePage
			case '-':
				theItem.itemType = ItemTypeRow
			case '!':
				theItem.itemType = ItemTypeImage
			case '#':
				theItem.itemType = ItemTypeText
				theItem = handleTextDirective(part, theItem, styles)
				accumulatingImage = false
				gotTextDirective = true
			}
			continue
		}

		// Setting
		if rxSetting.MatchString(part) {
			finishAccumulating(accumulating, theItem)
			parts := rxSetting.FindStringSubmatch(part)
			theItem.Set(parts[1], parts[2])
			accumulatingImage = false
			continue
		}

		// Text directive in an image (caption)
		if theItem.itemType == ItemTypeImage && rxTextDirective.MatchString(part) {
			finishAccumulating(accumulating, theItem)
			theItem = handleTextDirective(part, theItem, styles)
			accumulatingImage = false
			gotTextDirective = true
			continue
		}

		// Accumulating a string
		if accumulating.Len() > 0 {
			accumulating.WriteString(" ")
		}
		accumulating.WriteString(part)

		if accumulatingImage && looksLikeImage(accumulating.String()) {
			theItem.Set("image", accumulating.String())
			if theItem.itemType == ItemTypeUnknown {
				theItem.itemType = ItemTypeImage
			}
			accumulating.Reset()
			accumulatingImage = false
		}
	}

	if accumulating.Len() > 0 {
		if accumulatingImage && looksLikeImage(accumulating.String()) {
			theItem.Set("image", accumulating.String())
			if theItem.itemType == ItemTypeUnknown {
				theItem.itemType = ItemTypeImage
			}
		} else {
			theItem.Set("text", accumulating.String())
			if theItem.itemType == ItemTypeUnknown {
				theItem.itemType = ItemTypeText
			}
			if !gotTextDirective {
				theItem = handleTextDirective("#", theItem, styles)
			}
		}
	}

	return theItem
}

func tokenize(line string) []string {
	var parts []string
	inSpace := false
	inEscape := false
	inQuote := false
	var part strings.Builder

	for _, c := range line {
		switch c {
		case ' ':
			if inQuote {
				part.WriteRune(c)
			} else {
				if !inSpace {
					inSpace = true
					if part.Len() > 0 {
						parts = append(parts, part.String())
						part.Reset()
					}
				}
			}
		case '`':
			inSpace = false
			if inEscape {
				part.WriteRune(c)
				inEscape = false
			} else {
				inQuote = !inQuote
			}
		case '\\':
			inSpace = false
			if inQuote {
				if inEscape {
					part.WriteRune(c)
				}
				inEscape = !inEscape
			} else {
				part.WriteRune(c)
			}
		default:
			inSpace = false
			if inQuote {
				if inEscape {
					inEscape = false
				} else {
					part.WriteRune(c)
				}
			} else {
				part.WriteRune(c)
			}
		}
	}

	if part.Len() > 0 {
		parts = append(parts, part.String())
	}

	return parts
}

func tokenize_new(line string) []string {
	var parts []string
	quoteCount := 0
	inSpace := false
	var part strings.Builder

	for _, c := range line {
		switch c {
		case ' ':
			if quoteCount == 2 {
				part.WriteRune('\'')
				quoteCount = 0
			}
			if quoteCount == 1 {
				part.WriteRune(c)
			} else {
				if !inSpace {
					inSpace = true
					if part.Len() > 0 {
						parts = append(parts, part.String())
						part.Reset()
					}
				}
			}
		case '\'':
			inSpace = false
			quoteCount++
			if quoteCount == 3 {
				part.WriteRune('\'')
				quoteCount -= 2
			}
		default:
			inSpace = false
			if quoteCount == 2 {
				part.WriteRune('\'')
				quoteCount = 0
			}
			part.WriteRune(c)
		}
	}

	if part.Len() > 0 {
		parts = append(parts, part.String())
	}

	return parts
}

func tokenize_old(line string) []string {
	var parts []string
	prev := '\000'
	inQuote := false
	inSpace := false
	var part strings.Builder

	for _, c := range line {
		if c == ' ' {
			if inQuote {
				part.WriteRune(c)
			} else {
				if !inSpace {
					inSpace = true
					if part.Len() > 0 {
						parts = append(parts, part.String())
						part.Reset()
					}
				}
			}
		} else {
			inSpace = false
			if c == '\'' {
				if inQuote {
					if prev == '\'' {
						part.WriteRune(c)
						c = '\000'
					} else {
						inQuote = false
					}
				} else {
					inQuote = true
				}
			} else {
				part.WriteRune(c)
			}
		}

		prev = c
	}

	if part.Len() > 0 {
		parts = append(parts, part.String())
	}

	return parts
}

func looksLikeImage(s string) bool {
	suffixes := []string{".jpeg", ".jpg", ".png", ".bmp", ".tiff", ".tif"}
	s = strings.ToLower(s)

	for _, suffix := range suffixes {
		if strings.HasSuffix(s, suffix) {
			return true
		}
	}

	return false
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
			if !strings.HasPrefix(s, "// ") && s != "//" && !(ii == 0 && !continuation && strings.HasPrefix(s, "#!")) {
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

func processAsLinesFromBasePath(lines []string, basePath string, styles map[string]string) ([]PbItem, map[string]string) {
	var items []PbItem
	for _, s := range lines {
		if strings.HasPrefix(s, "$ ") {
			styles = consumeStyle(s, styles)
		} else if strings.HasPrefix(s, "@ ") {
			file2 := strings.Replace(s, "@ ", "", 1)
			var includedItems []PbItem
			includedItems, styles = readInputFile(localizePath(file2, basePath), styles)
			for _, includedItem := range includedItems {
				includedItem.pb = items
				items = append(items, includedItem)
			}
		} else {
			theItem := applyStyles(s, styles)
			theItem = localizePaths(theItem, basePath)
			theItem.pb = items
			newItems := expandWild(theItem)
			if newItems != nil {
				items = append(items, newItems...)
			} else {
				items = append(items, theItem)
			}
		}
	}
	return items, styles
}

func expandWild(item PbItem) []PbItem {
	var newItems []PbItem

	if item.itemType == ItemTypeImage {
		if strings.ContainsAny(item.Setting("image"), "?*") {
			sources, err := filepath.Glob(item.Setting("image"))
			if err != nil {
				log.Fatal(err)
			}
			slices.Sort(sources)
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

func consumeStyle(line string, styles map[string]string) map[string]string {
	parts := strings.SplitN(line, " ", 3)
	styles[parts[1]] = parts[2]
	return styles
}

func applyStyles(line string, styles map[string]string) PbItem {
	// Replace {{style}} references with content
	for kk, vv := range styles {
		line = strings.ReplaceAll(line, fmt.Sprintf("{{%v}}", kk), vv)
	}

	// Replace text directives while parsing
	theItem := parse(line, styles)
	return theItem
}

var rxRootPath, _ = regexp.Compile(`^([a-z]:|[/\\])`)

func localizePath(path string, basePath string) string {
	if !rxRootPath.MatchString(path) {
		path = basePath + path
	}

	return path
}

func localizePaths(theItem PbItem, basePath string) PbItem {
	if image, imageExists := theItem.settings["image"]; imageExists {
		theItem.Set("image", localizePath(image, basePath))
	}
	if font, fontExists := theItem.settings["font"]; fontExists {
		theItem.Set("font", localizePath(font, basePath))
	}
	return theItem
}

func readInputFile(inFile string, styles map[string]string) ([]PbItem, map[string]string) {
	var inBytes []byte
	var err error

	basePath := BasePath(inFile)

	if inFile == "-" {
		inBytes, err = io.ReadAll(os.Stdin)
	} else {
		inBytes, err = os.ReadFile(inFile)
	}

	if err != nil {
		log.Fatal(err)
	}

	inStrings := strings.Split(string(inBytes), "\n")
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

func ReadPbFile(inFileFlag string) []PbItem {
	styles := map[string]string{}
	items, _ := readInputFile(inFileFlag, styles)

	return items
}
