package main

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"log"
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

func printItems(items []PbItem) string {
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
			o.WriteString(fmt.Sprintf("/// ImageWidth:%v, ImageHeight:%v\n", theItem.imageWidthPx, theItem.imageHeightPx))
		}
		if theItem.textBlockLayouts != nil {
			o.WriteString(fmt.Sprintf("/// TextBlockLayouts: %v\n", theItem.textBlockLayouts))
		}
		o.WriteString(fmt.Sprintf("/// Item %v: Page:%v, Row:%v, Column:%v, TextWidth:%v, TextHeight:%v, ImageWidth:%v, ImageHeight:%v, X:%v, Y:%v\n\n", ii, theItem.page, theItem.row, theItem.column, theItem.textWidth, theItem.textHeight, theItem.imageWidth, theItem.imageHeight, theItem.xOffset, theItem.yOffset))
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
			textAlign = " text-align:left"
		case 'R':
			textAlign = " text-align:right"
		case 'C':
			textAlign = " text-align:center"
		case 'J':
			textAlign = " text-align:justified"
		case 'B':
			textAlign = " text-align:binding"
		case 'E':
			textAlign = " text-align:edge"
		}
	}

	return "{{" + styleNum + "}}" + textAlign
}

var rxEscapeSpace, _ = regexp.Compile("`_")
var rxEscape, _ = regexp.Compile("`(.)")

func unescape(line string) string {
	line = rxEscapeSpace.ReplaceAllString(line, " ")
	return rxEscape.ReplaceAllString(line, "$1")
}

func unescapeText(text string) string {
	text = strings.ReplaceAll(text, "\\t", "\t")
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
		parts[ii] = strings.TrimSpace(parts[ii])
		pieces := strings.SplitN(parts[ii], ":", 2)
		if len(pieces) == 2 {
			if pieces[0] == "trim" || pieces[0] == "fit" {
				pieces[1] = pieces[0] + "," + pieces[1]
				pieces[0] = "rect"
			}
			theItem.Set(pieces[0], unescape(pieces[1]))
		} else if len(pieces) == 1 {
			switch pieces[0] {
			case "row-break":
				theItem.Set("row-break", "true")
			case "column-break":
				theItem.Set("column-break", "true")
			case "page-break":
				theItem.Set("page-break", "true")
			case "current-page":
				theItem.Set("current-page", "true")
			case "smaller", "much-smaller", "much-much-smaller", "much-much-much-smaller",
				"larger", "much-larger", "much-much-larger", "much-much-much-larger":
				theItem.Set("size", pieces[0])
			}
		}
	}

	return theItem
}

func processAsLinesFromBasePath(lines []string, basePath string, styles map[string]string, options map[string]string) ([]PbItem, map[string]string, map[string]string) {
	var items []PbItem
	for _, s := range lines {
		if strings.HasPrefix(s, "$$$ ") {
			style := strings.Replace(s, "$$$ ", "", 1)
			styles = defineStyle(style, styles)
		} else if strings.HasPrefix(s, ">>> ") {
			option := strings.Replace(s, ">>> ", "", 1)
			options = defineOption(option, options)
		} else if strings.HasPrefix(s, "@@@ ") {
			file2 := strings.Replace(s, "@@@ ", "", 1)
			var includedItems []PbItem
			includedItems, styles, options = readInputFile(localizePath(file2, basePath), styles, options)
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

	return items, styles, options
}

func glob(path string, recurse bool) ([]string, error) {
	exts := []string{".jpg", ".jpeg", ".png"}
	for ii := len(exts); ii > 0; ii-- {
		exts = append(exts, strings.ToUpper(exts[ii-1]))
	}

	if zipParts := strings.SplitN(path, "::", 2); len(zipParts) == 2 {
		if zipParts[1] == "*" {
			sources := make([]string, 0)
			for _, ext := range exts {
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
			if zipReader, err := zip.OpenReader(zipParts[0]); err != nil {
				log.Print(err)
				return make([]string, 0), err
			} else {
				sources := make([]string, 0)
				for f := range zipReader.File {
					if matched, err := filepath.Match(zipParts[1], zipReader.File[f].Name); err == nil && matched {
						sources = append(sources, zipParts[0]+"::"+zipReader.File[f].Name)
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
				log.Fatal(err)
			}
			slices.Sort(sources)
			sources = slices.Compact(sources)
			slices.SortFunc(sources, func(a string, b string) int {
				aparts := strings.Split(a, string(os.PathSeparator))
				bparts := strings.Split(b, string(os.PathSeparator))
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
			})
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

func defineOption(line string, options map[string]string) map[string]string {
	parts := strings.SplitN(line, " ", 2)
	options[parts[0]] = parts[1]
	return options
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
	if !rxRootPath.MatchString(path) {
		path = basePath + path
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
				imageName := items[ii].Setting("image")
				if imageNameParts := strings.SplitN(imageName, "::", 2); len(imageNameParts) == 2 {
					imageName = imageNameParts[1]
				}
				text = strings.ReplaceAll(text, "{{Filename}}", filepath.Base(imageName))
				text = strings.ReplaceAll(text, "{{Fullname}}", filepath.Clean(imageName))
			}
			text = strings.ReplaceAll(text, "{{Date}}", date)
			text = strings.ReplaceAll(text, "{{Year}}", year)
			items[ii].Set("text", text)
		}
	}
}

func readInputFile(inFile string, styles map[string]string, options map[string]string) ([]PbItem, map[string]string, map[string]string) {
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
	items, styles, options = processAsLinesFromBasePath(inStrings, basePath, styles, options)
	ApplyItemSpecificStyles(items)

	return items, styles, options
}

func BasePath(fileName string) string {
	basePath := filepath.Dir(fileName) + "/"
	if basePath == "./" {
		basePath = ""
	}
	return basePath
}

func ReadPbFile(inFileFlag string) ([]PbItem, map[string]string) {
	items, _, options := readInputFile(inFileFlag, map[string]string{}, map[string]string{})

	for ii := range items {
		items[ii].pb = items
	}

	OptimizeSettings(items)

	return items, options
}
