package main

import (
	"bufio"
	"encoding/json"
	"image"
	"log"
	"os"
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
