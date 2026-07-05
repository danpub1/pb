package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type ImageCacheEntry struct {
	FileDate      int64
	ImageWidthPx  int
	ImageHeightPx int
	Orientation   int
	ExifDate      time.Time
}

var firstTimeImageCache = true

func loadImageCache() map[string]ImageCacheEntry {
	cache := map[string]ImageCacheEntry{}

	if Opts.Cache()&CacheModeImage == 0 || Opts.Cache()&CacheModeImageDuring != 0 && firstTimeImageCache {
		firstTimeImageCache = false
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

func checkCacheEntry(cache *map[string]ImageCacheEntry, filename string) (int, int, int, bool, int64) {
	if entry, exists := (*cache)[filename]; exists {
		fileDate := fileDate(filename)
		if entry.FileDate == fileDate {
			return entry.ImageWidthPx, entry.ImageHeightPx, entry.Orientation, true, fileDate
		}
	}
	return 0, 0, 0, false, fileDate(filename)
}

func updateCacheEntry(cache *map[string]ImageCacheEntry, filename string, imageWidthPx int, imageHeightPx int, orientation int, exifDate time.Time) {
	(*cache)[filename] = ImageCacheEntry{fileDate(filename), imageWidthPx, imageHeightPx, orientation, exifDate}
}

func exifRotate(orientation int) (int, bool) {
	rotation := 0
	flip := false
	switch orientation {
	default:
	case 0:
	case 1:
		flip = false
		rotation = 0
	case 2:
		flip = true
		rotation = 0
	case 3:
		flip = false
		rotation = 180
	case 4:
		flip = true
		rotation = 180
	case 5:
		flip = true
		rotation = 90
	case 6:
		flip = false
		rotation = 90
	case 7:
		flip = true
		rotation = 270
	case 8:
		flip = false
		rotation = 270
	}
	return rotation, flip
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

			rotation := items[ii].IntSetting("rotate")

			if imageWidthPx, imageHeightPx, orientation, exists, fileDate := checkCacheEntry(&cache, filename); exists {
				items[ii].exifOrientation = orientation
				items[ii].fileDate = fileDate
				exifRotation, _ := exifRotate(orientation)
				rotation = (rotation + exifRotation) % 360
				if rotation == 90 || rotation == -90 || rotation == 270 || rotation == -270 {
					items[ii].imageWidthPx = imageHeightPx
					items[ii].imageHeightPx = imageWidthPx
				} else {
					items[ii].imageWidthPx = imageWidthPx
					items[ii].imageHeightPx = imageHeightPx
					items[ii].exifOrientation = orientation
				}
				continue
			} else {
				items[ii].fileDate = fileDate
			}

			config, orientation, exifDate := items[ii].GetImageConfig()
			items[ii].exifOrientation = orientation
			items[ii].exifDate = exifDate

			if config.Width > 0 && config.Height > 0 {
				exifRotation, _ := exifRotate(orientation)
				rotation = (rotation + exifRotation) % 360
				if rotation == 90 || rotation == -90 || rotation == 270 || rotation == -270 {
					items[ii].imageWidthPx = config.Height
					items[ii].imageHeightPx = config.Width
				} else {
					items[ii].imageWidthPx = config.Width
					items[ii].imageHeightPx = config.Height
				}
			}

			updateCacheEntry(&cache, filename, config.Width, config.Height, orientation, exifDate)
		}
	}

	saveImageCache(&cache)
	return numImages
}

func MeasureText2(text string, maxWidth float64, maxHeight float64, item *PbItem) []TextBlockLayout {
	textInfo := item.TextInfo()
	_, pageHeight := ContainerSize(item.PageSetting("page-size"), item.PageSetting("margin"))

	textBlockLayouts := MeasureText(text, maxWidth, maxHeight, textInfo)

	textHeight := item.FloatSetting("text-height")

	fontSizeMin := item.FloatSetting("font-size-min")

	availableHeight := pageHeight
	if textHeight > 0 && textHeight < availableHeight {
		availableHeight = textHeight
	}

	if maxHeight == 0 && len(textBlockLayouts) == 1 && textHeight > textBlockLayouts[0].height {
		extra := textHeight - textBlockLayouts[0].height
		textInfo.padding.top += extra / 2.0
		textInfo.padding.bottom += extra / 2.0
		textBlockLayouts = MeasureText(text, maxWidth, maxHeight, textInfo)
	} else if fontSizeMin > 0 && maxHeight == 0 && len(textBlockLayouts) == 1 && textBlockLayouts[0].height > availableHeight {
		decrement := 1 / item.Density()
		for textBlockLayouts[0].height > availableHeight && textInfo.height > fontSizeMin {
			textInfo.height -= decrement
			if textInfo.height < fontSizeMin {
				textInfo.height = fontSizeMin
			}
			textBlockLayouts = MeasureText(text, maxWidth, maxHeight, textInfo)
		}
		item.Set("font-size", fmt.Sprintf("%v", textInfo.height))
	}

	return textBlockLayouts
}

func getOneTextDimensions(item *PbItem, text string) []TextBlockLayout {
	rotate := item.IntSetting("rotate")
	var width float64
	bRotate := rotate == 90 || rotate == -90 || rotate == 270 || rotate == -270
	if bRotate {
		width = HeightForContainer(item.Setting("text-width"), item.PageSetting("page-size"), item.PageSetting("margin"))
	} else {
		width = WidthForContainer(item.Setting("text-width"), item.PageSetting("page-size"), item.PageSetting("margin"))
	}
	textBlockLayouts := MeasureText2(text, width, 0.0, item)
	if bRotate {
		for jj := range textBlockLayouts {
			temp := textBlockLayouts[jj].height
			textBlockLayouts[jj].height = textBlockLayouts[jj].width
			textBlockLayouts[jj].width = temp
		}
	}
	return textBlockLayouts
}

func getTextDimensions(items []PbItem) int {
	if len(items) == 0 {
		return 0
	}

	numTexts := 0
	for ii := range items {
		if items[ii].itemType == ItemTypeText && len(items[ii].Setting("text")) > 0 {
			items[ii].textBlockLayouts = getOneTextDimensions(&items[ii], items[ii].Setting("text"))
			numTexts++
		} else if items[ii].itemType == ItemTypeImage && len(items[ii].Setting("text")) > 0 {
			maxWidth := ContainerWidth(items[ii].PageSetting("page-size"), items[ii].PageSetting("margin"))
			maxHeight := ContainerHeight(items[ii].PageSetting("page-size"), items[ii].PageSetting("margin"))
			items[ii].textBlockLayouts = MeasureText2(items[ii].Setting("text"), maxWidth, maxHeight, &items[ii])
			numTexts++
		}
	}

	return numTexts
}
