package main

import (
	"encoding/json"
	"os"
)

type ImageCacheEntry struct {
	Filedate      int64
	ImageWidthPx  int
	ImageHeightPx int
}

func loadImageCache() map[string]ImageCacheEntry {
	cache := map[string]ImageCacheEntry{}

	if Opts.Cache()&CacheModeImageNone != 0 || Opts.Cache()&CacheModeImageDuring != 0 {
		if Opts.Cache()&CacheModeImageDuring != 0 {
			Opts.argsOptions.cacheMode = (Opts.argsOptions.cacheMode & (CacheModeAll ^ CacheModeImageDuring)) | CacheModeImageFull
		}
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

			config := items[ii].GetImageConfig()

			if config.Width > 0 && config.Height > 0 {
				if rotation == 90 || rotation == -90 || rotation == 270 || rotation == -270 {
					items[ii].imageWidthPx = config.Height
					items[ii].imageHeightPx = config.Width
				} else {
					items[ii].imageWidthPx = config.Width
					items[ii].imageHeightPx = config.Height
				}
			}

			updateCacheEntry(&cache, filename, config.Width, config.Height)
		}
	}

	saveImageCache(&cache)
	return numImages
}

func MeasureText2(text string, maxWidth float64, maxHeight float64, item *PbItem) []TextBlockLayout {
	textInfo := item.TextInfo()
	textBlockLayouts := MeasureText(text, maxWidth, maxHeight, textInfo)

	textHeight := item.FloatSetting("text-height")
	if maxHeight == 0 && len(textBlockLayouts) == 1 && textHeight > textBlockLayouts[0].height {
		extra := textHeight - textBlockLayouts[0].height
		textInfo.padding.top += extra / 2.0
		textInfo.padding.bottom += extra / 2.0
		textBlockLayouts = MeasureText(text, maxWidth, maxHeight, textInfo)
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
