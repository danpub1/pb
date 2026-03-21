package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spakin/netpbm"

	"github.com/disintegration/imaging"
)

func writeHeader(outFilename string) (int, error) {
	ext := filepath.Ext(outFilename)
	format := strings.ToLower(ext)

	if format == ".pdf" {
		bytes := []byte("%PDF-1.7\n")
		err := os.WriteFile(outFilename, bytes, 0666)
		if err != nil {
			log.Print(err)
		}
		return len(bytes), err
	}

	return 0, nil
}

type PageInfo struct {
	offset int
	width  int
	height int
}

func writeFooter(outFilename string, bytesWritten int, pageInfo []PageInfo) (int, error) {
	ext := filepath.Ext(outFilename)
	format := strings.ToLower(ext)

	if format == ".pdf" {
		buffer := strings.Builder{}
		numPages := len(pageInfo)
		pageInfo = append(pageInfo, PageInfo{bytesWritten, 0, 0})
		n, err := buffer.WriteString(fmt.Sprintf("%v 0 obj\n<</Type/Catalog/Pages %v 0 R>>\nendobj\n", numPages+1, numPages+2))
		bytesWritten += n
		pageInfo = append(pageInfo, PageInfo{bytesWritten, 0, 0})
		n, err = buffer.WriteString(fmt.Sprintf("%v 0 obj\n<</Type/Pages/Count %v/Kids[", numPages+2, numPages))
		bytesWritten += n
		for ii := range numPages {
			space := " "
			if ii == 0 {
				space = ""
			}
			n, err = buffer.WriteString(fmt.Sprintf("%v%v 0 R", space, ii*3+numPages+3+2))
			bytesWritten += n
		}
		n, err = buffer.WriteString("]>>\nendobj\n")
		bytesWritten += n

		for ii := range numPages {
			pageInfo = append(pageInfo, PageInfo{bytesWritten, 0, 0})
			n, err = buffer.WriteString(fmt.Sprintf("%v 0 obj\n<</XObject<</I%v %v 0 R>>>>\nendobj\n", ii*3+numPages+3+0, ii+1, ii+1))
			bytesWritten += n
			pageInfo = append(pageInfo, PageInfo{bytesWritten, 0, 0})
			cmd := fmt.Sprintf("q %v 0 0 %v 0 0 cm /I%v Do Q", pageInfo[ii].width, pageInfo[ii].height, ii+1)
			n, err = buffer.WriteString(fmt.Sprintf("%v 0 obj\n<</Length %v>>\nstream\n%v\nendstream\nendobj\n", ii*3+numPages+3+1, len(cmd), cmd))
			bytesWritten += n
			pageInfo = append(pageInfo, PageInfo{bytesWritten, 0, 0})
			n, err = buffer.WriteString(fmt.Sprintf("%v 0 obj\n<</Type/Page/MediaBox[0 0 %v %v]/Rotate 0/Resources %v 0 R/Contents %v 0 R/Parent %v 0 R>>\nendobj\n", ii*3+numPages+3+2, pageInfo[ii].width, pageInfo[ii].height, ii*3+numPages+3+0, ii*3+numPages+3+1, numPages+2))
			bytesWritten += n
		}

		startOfXref := bytesWritten

		n, err = buffer.WriteString(fmt.Sprintf("xref\n0 %v\n0000000000 00001 f\n", numPages*4+3))
		for ii := range pageInfo {
			n, err = buffer.WriteString(fmt.Sprintf("%010d 00000 n\n", pageInfo[ii].offset))
		}

		n, err = buffer.WriteString(fmt.Sprintf("trailer\n<</Size %v/Root %v 0 R>>\nstartxref\n%v\n%%%%EOF", len(pageInfo)+1, numPages+1, startOfXref))
		out, err := os.OpenFile(outFilename, os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Print(err)
			return 0, err
		}
		defer out.Close()
		n, err = out.Write([]byte(buffer.String()))
		if err != nil {
			log.Print(err)
			return 0, err
		}
		return n, nil
	}

	return 0, nil
}

type PdfJpegObjectWriter struct {
	out    io.Writer
	mem    *[]byte
	objNum int
	width  int
	height int
}

// Write implements [io.Writer].
func (writer PdfJpegObjectWriter) Write(p []byte) (int, error) {
	*writer.mem = append(*writer.mem, p...)
	return len(p), nil
}

func (writer *PdfJpegObjectWriter) Start(out io.Writer, objNum int, width int, height int) {
	writer.out = out
	writer.objNum = objNum
	writer.width = width
	writer.height = height
	writer.mem = &[]byte{}
}

func (writer *PdfJpegObjectWriter) Finish() (int, error) {
	text := fmt.Sprintf("%v 0 obj\n<</Filter[/DCTDecode]/Type/XObject/Subtype/Image/BitsPerComponent 8/Width %v/Height %v/ColorSpace/DeviceRGB/Length %v>>\nstream\n", writer.objNum, writer.width, writer.height, len(*writer.mem))
	bytesWritten := 0
	n, err := writer.out.Write([]byte(text))
	if err != nil {
		log.Print(err)
		return bytesWritten, err
	}
	bytesWritten += n
	n, err = writer.out.Write(*writer.mem)
	if err != nil {
		log.Print(err)
		return bytesWritten, err
	}
	bytesWritten += n
	n, err = writer.out.Write([]byte("\nendstream\nendobj\n"))
	if err != nil {
		log.Print(err)
		return bytesWritten, err
	}
	bytesWritten += n
	return bytesWritten, err
}

func writePage(img image.Image, objNum int, curPage int, outFilename string, isPageRangeMulti bool, compressionLevel int) (int, error) {
	ext := filepath.Ext(outFilename)
	format := strings.ToLower(ext)
	if isPageRangeMulti && format != ".pdf" {
		outFilename = strings.TrimSuffix(outFilename, ext)
		outFilename = fmt.Sprintf(outFilename+"-%v"+ext, curPage)
	}

	var out *os.File
	var err error
	if format != ".pdf" {
		out, err = os.Create(outFilename)
	} else {
		out, err = os.OpenFile(outFilename, os.O_WRONLY|os.O_APPEND, 0666)
	}
	if err != nil {
		log.Print(err)
		return 0, err
	}
	defer out.Close()
	switch format {
	case ".png":
		if err := png.Encode(out, img); err != nil {
			log.Print(err)
			return 0, err
		}
		return 0, nil
	case ".jpg", ".jpeg":
		options := jpeg.Options{Quality: compressionLevel}
		if err := jpeg.Encode(out, img, &options); err != nil {
			log.Print(err)
			return 0, err
		}
		return 0, nil
		// return writeJPEG(img, out, compressionLevel)
	case ".pdf":
		writer := PdfJpegObjectWriter{}
		writer.Start(out, objNum, img.Bounds().Dx(), img.Bounds().Dy())

		options := jpeg.Options{Quality: compressionLevel}
		if err := jpeg.Encode(writer, img, &options); err != nil {
			log.Print(err)
			return 0, err
		}

		// if _, err := writeJPEG(img, writer, compressionLevel); err != nil {
		// 	log.Print(err)
		// 	return 0, err
		// }

		return writer.Finish()
	}

	return 0, nil
}

func scaleToRect(picture image.Image, item *PbItem) image.Image {
	zoom, dstAspect, xOffset, yOffset := item.ImageRectSetting()

	wr, hr, _, _ := calcStraighten(float64(item.imageWidthPx), float64(item.imageHeightPx), item.FloatSetting("straighten"))

	srcAspect := wr / hr

	dstWidth := int(math.Round(wr))
	dstHeight := int(math.Round(hr))
	dstXOffset := 0
	dstYOffset := 0
	switch zoom {
	case 0: // trim
		if dstAspect > srcAspect { // dst is wider than src, crop top & bottom
			dstHeight = int(math.Round(float64(dstWidth) / dstAspect))
			dstYOffset = int(math.Round(float64(int(math.Round(hr))-dstHeight) * float64(yOffset) / 100.0))
			return imaging.Crop(picture, image.Rectangle{image.Point{dstXOffset, dstYOffset}, image.Point{dstXOffset + dstWidth, dstYOffset + dstHeight}})
		} else if dstAspect < srcAspect { // dst is taller than src, crop left & right
			dstWidth = int(math.Round(float64(dstHeight) * dstAspect))
			dstXOffset = int(math.Round(float64(int(math.Round(wr))-dstWidth) * float64(xOffset) / 100.0))
			return imaging.Crop(picture, image.Rectangle{image.Point{dstXOffset, dstYOffset}, image.Point{dstXOffset + dstWidth, dstYOffset + dstHeight}})
		} else {
			return picture
		}
	case 1: // fit
		if dstAspect > srcAspect { // dst is wider than src, pad left & right
			dstWidth = int(math.Round(float64(int(math.Round(hr))) * dstAspect))
			dstXOffset = int(math.Round(float64(dstWidth-int(math.Round(wr))) * float64(xOffset) / 100.0))
			dst := image.NewRGBA(image.Rect(0, 0, dstWidth, dstHeight))
			backColor := colorToNRGBA(item.Setting("background"))
			draw.Draw(dst, dst.Bounds(), image.NewUniform(color.RGBA{backColor.R, backColor.G, backColor.B, backColor.A}), image.Point{}, draw.Src)
			return imaging.Paste(dst, picture, image.Point{dstXOffset, dstYOffset})
		} else if dstAspect < srcAspect { // dst is taller than src, pad top & bottom
			dstHeight = int(math.Round(float64(int(math.Round(wr))) / dstAspect))
			dstYOffset = int(math.Round(float64(dstHeight-int(math.Round(hr))) * float64(yOffset) / 100.0))
			dst := image.NewRGBA(image.Rect(0, 0, dstWidth, dstHeight))
			backColor := colorToNRGBA(item.Setting("background"))
			draw.Draw(dst, dst.Bounds(), image.NewUniform(color.RGBA{backColor.R, backColor.G, backColor.B, backColor.A}), image.Point{}, draw.Src)
			return imaging.Paste(dst, picture, image.Point{dstXOffset, dstYOffset})
		} else {
			return picture
		}
	default: // arbitrary zoom
	}

	return picture
}

func calcStraighten(width float64, height float64, angle float64) (float64, float64, float64, float64) {
	if angle == 0 {
		return width, height, 0, 0
	}
	aspectRatio := float64(width) / float64(height)
	angler := angle * 2 * 3.1415926535897932384626433 / 360
	sin_a := math.Abs(math.Sin(angler))
	cos_a := math.Abs(math.Cos(angler))

	var sideLong, sideShort float64
	if aspectRatio >= 1 {
		sideLong = float64(width)
		sideShort = float64(height)
	} else {
		sideLong = float64(height)
		sideShort = float64(width)
	}

	var wr, hr float64
	if sideShort <= 2*sin_a*cos_a*sideLong || math.Abs(sin_a-cos_a) < 0.0000000001 {
		xx := sideShort / 2
		if aspectRatio >= 1 {
			wr = xx / sin_a
			hr = xx / cos_a
		} else {
			wr = xx / cos_a
			hr = xx / sin_a
		}
	} else {
		cos2a := cos_a*cos_a - sin_a*sin_a
		wr = math.Abs(width*cos_a-height*sin_a) / cos2a
		hr = math.Abs(height*cos_a-width*sin_a) / cos2a
	}

	hOff := math.Abs((width*cos_a+height*sin_a)-wr) / 2
	vOff := math.Abs((width*sin_a+height*cos_a)-hr) / 2

	return wr, hr, hOff, vOff
}

func straighten(picture image.Image, angle float64) image.Image {

	wr, hr, hOff, vOff := calcStraighten(float64(picture.Bounds().Dx()), float64(picture.Bounds().Dy()), angle)

	picture = imaging.Rotate(picture, -angle, color.RGBA{127, 127, 127, 255})

	rect := image.Rectangle{
		image.Point{int(math.Round(hOff)), int(math.Round(vOff))},
		image.Point{int(math.Round(hOff + wr)), int(math.Round(vOff + hr))}}

	return imaging.Crop(picture, rect)
}

func convertImage(picture image.Image) image.Image {
	// this is too slow for regular use
	// may be able to adapt to use imagmagick or mozjpeg to create quality jpegs for final output
	log.Print("executing convert")
	cmd := exec.Command("convert", "-", "-adaptive-sharpen", "x5", "PNM:-")

	stdin, err1 := cmd.StdinPipe()
	if err1 != nil {
		log.Print("Error opening stdin")
		log.Print(err1)
		return picture
	}

	stdout, err2 := cmd.StdoutPipe()
	if err2 != nil {
		stdin.Close()
		log.Print("Error opening stdout")
		log.Print(err2)
		return picture
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer stdin.Close()
		defer wg.Done()
		err := netpbm.Encode(stdin, picture, &netpbm.EncodeOptions{})
		if err != nil {
			log.Print("Error encoding image")
			log.Print(err)
		}
	}()

	wg.Add(1)
	go func() {
		defer stdout.Close()
		defer wg.Done()
		newpicture, err := netpbm.Decode(stdout, &netpbm.DecodeOptions{})
		if err != nil {
			log.Print("Error decoding image")
			log.Print(err)
		}
		if newpicture != nil && newpicture.Bounds().Dx() == picture.Bounds().Dx() && newpicture.Bounds().Dy() == picture.Bounds().Dy() {
			picture = newpicture
		}
	}()

	err2 = cmd.Run()
	if err2 != nil {
		log.Print("Error running command")
		log.Print(err2)
	}

	wg.Wait()
	log.Print("executed convert")
	return picture
}

func writeJPEG(picture image.Image, out io.Writer, compressionLevel int) (int, error) {
	// this is too slow for regular use
	// may be able to adapt to use imagmagick or mozjpeg to create quality jpegs for final output
	cmd := exec.Command("convert", "PNM:-", "-quality", fmt.Sprintf("%v", compressionLevel), "-sampling-factor", "4:4:4", "JPEG:-")
	//cmd := exec.Command("cjpeg", "-quality", fmt.Sprintf("%v", compressionLevel), "-sample", "1x1")

	log.Print(cmd.String())
	bytesWritten := 0
	var errReturn error

	stdin, err1 := cmd.StdinPipe()
	if err1 != nil {
		log.Print("Error opening stdin")
		log.Print(err1)
		return 0, err1
	}

	stdout, err2 := cmd.StdoutPipe()
	if err2 != nil {
		stdin.Close()
		log.Print("Error opening stdout")
		log.Print(err2)
		return 0, err2
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer stdin.Close()
		defer wg.Done()
		err := netpbm.Encode(stdin, picture, &netpbm.EncodeOptions{})
		if err != nil {
			log.Print("Error encoding image")
			log.Print(err)
			errReturn = err
		}
	}()

	wg.Add(1)
	go func() {
		defer stdout.Close()
		defer wg.Done()
		p := make([]byte, 1024*64)
		for {
			n, err := stdout.Read(p)
			if n > 0 {
				m, err2 := out.Write(p[:n])
				if err2 != nil {
					log.Print("error writing output file")
					log.Print(err2)
					errReturn = err2
					break
				}
				bytesWritten += m
				if m != n {
					log.Print("truncated write")
					break
				}
			}
			if err != nil {
				if err != io.EOF {
					log.Print("Error readig input stream")
					log.Print(err)
					errReturn = err
				}
				break
			}
		}
	}()

	// err2 = cmd.Start()
	// if err2 != nil {
	// 	log.Print("Error running command")
	// 	log.Print(err2)
	// 	errReturn = err2
	// }
	// err2 = cmd.Wait()
	// if err2 != nil {
	// 	log.Print("Error waiting for command")
	// 	log.Print(err2)
	// 	errReturn = err2
	// }

	err2 = cmd.Run()
	if err2 != nil {
		log.Print("Error running command")
		log.Print(err2)
		errReturn = err2
	}

	// var bytes []byte
	// bytes, err2 = cmd.Output()
	// if err2 != nil {
	// 	log.Print("Error running command")
	// 	log.Print(err2)
	// 	errReturn = err2
	// }
	// log.Print(string(bytes))

	wg.Wait()

	if errReturn == nil {
		log.Printf("%v: %v bytes", cmd.String(), bytesWritten)
	} else {
		log.Printf("%v: %v bytes, %v", cmd.String(), bytesWritten, errReturn)
	}
	return bytesWritten, errReturn
}

func renderPages(pbBook *PbBook, outPageRange string, outFilename string) {
	n, err := writeHeader(outFilename)
	if err != nil {
		return
	}

	isPageRangeMulti := isPageRangeMulti(outPageRange, pbBook)

	offsets := []PageInfo{}
	objNum := 1
	for pp := range pbBook.pages {
		changed := false
		if changed, _ = fileChanged(*inFileFlag, lastModTime); changed {
			break
		}
		page := &pbBook.pages[pp]
		if isPageInRange(outPageRange, pp) || isCurrentPage(pbBook, pp) {
			var top float64
			var left float64
			var dst *image.RGBA = nil
			density := 1.0

			if len(page.rows[0].columns[0].items) == 0 {
				continue
			}
			item := page.rows[0].columns[0].items[0].item

			pageWidth, pageHeight := FloatSize(item.Setting("page-size"))
			density = item.Density()
			top, _, _, left = FourTwoOne(item.Setting("margin"))
			widthDots := int(math.Round(dotsFromUnitsFloat(pageWidth, density)))
			heightDots := int(math.Round(dotsFromUnitsFloat(pageHeight, density)))
			dst = image.NewRGBA(image.Rect(0, 0, widthDots, heightDots))

			// fill the destination with the background color
			backColor := colorToNRGBA(item.Setting("background"))
			draw.Draw(dst, dst.Bounds(), image.NewUniform(color.RGBA{backColor.R, backColor.G, backColor.B, backColor.A}), image.Point{}, draw.Src)

			for row := range page.rows {
				for column := range page.rows[row].columns {
					for columnItem := range page.rows[row].columns[column].items {
						item = page.rows[row].columns[column].items[columnItem].item

						if item.itemType == ItemTypeText && len(item.Setting("name")) == 0 {
							textImage := TextToImage(&item.textBlockLayouts[0], item.TextInfo())
							xDots := int(math.Round(dotsFromUnitsFloat(left+item.xOffset, density)))
							yDots := int(math.Round(dotsFromUnitsFloat(top+item.yOffset, density)))
							draw.Draw(dst, image.Rect(xDots, yDots, xDots+textImage.Bounds().Size().X, yDots+textImage.Bounds().Size().Y), textImage, image.Point{}, draw.Over)
						}

						if item.itemType == ItemTypeImage && len(item.Setting("name")) == 0 {
							picture, err := imaging.Open(item.Setting("image"))
							if err != nil {
								log.Printf("failed to open image: %v", err)
								continue
							}

							straightenAngle := item.FloatSetting("straighten")
							if straightenAngle != 0 {
								picture = straighten(picture, straightenAngle)
							}

							rotation := Atoi(item.Setting("rotate"))
							switch rotation {
							case -90, 270:
								picture = imaging.Rotate90(picture)
							case 90, -270:
								picture = imaging.Rotate270(picture)
							case 180, -180:
								picture = imaging.Rotate180(picture)
							}

							brightness := item.FloatSetting("brightness")
							if brightness != 0 {
								picture = imaging.AdjustBrightness(picture, brightness)
							}
							contrast := item.FloatSetting("contrast")
							if contrast != 0 {
								picture = imaging.AdjustContrast(picture, contrast)
							}
							gamma := item.FloatSetting("gamma")
							if gamma != 1.0 {
								picture = imaging.AdjustGamma(picture, gamma)
							}
							saturation := item.FloatSetting("saturation")
							if saturation != 0 {
								picture = imaging.AdjustSaturation(picture, saturation)
							}
							factor, midpoint := item.SigmoidalSetting()
							if factor != 0 {
								picture = imaging.AdjustSigmoid(picture, midpoint, factor)
							}

							picture = scaleToRect(picture, item)

							// Resize the cropped image to width = 200px preserving the aspect ratio.
							imageWidthDots := int(math.Round(dotsFromUnitsFloat(item.imageWidth, density)))
							imageHeightDots := int(math.Round(dotsFromUnitsFloat(item.imageHeight, density)))

							picture = imaging.Resize(picture, imageWidthDots, imageHeightDots, imaging.Lanczos)

							blur := item.FloatSetting("blur")
							if blur != 0 {
								picture = imaging.Blur(picture, blur)
							}
							sharpen := item.FloatSetting("sharpen")
							if sharpen != 0 {
								picture = imaging.Sharpen(picture, sharpen)
							}

							textWidthDots := int(math.Round(dotsFromUnitsFloat(item.textWidth, density)))
							textHeightDots := int(math.Round(dotsFromUnitsFloat(item.textHeight, density)))
							xDots := int(math.Round(dotsFromUnitsFloat(left+item.xOffset, density)))
							yDots := int(math.Round(dotsFromUnitsFloat(top+item.yOffset, density)))

							if len(item.textBlockLayouts) > 0 {
								captionGutterDots := int(math.Round(dotsFromUnitsFloat(item.FloatSetting("caption-gutter"), density)))
								textBlockLayout := TextBlockLayout{}
								if item.bestTextBlockLayout >= 0 {
									textBlockLayout = item.textBlockLayouts[item.bestTextBlockLayout]
								}
								textImage := TextToImage(&textBlockLayout, item.TextInfo())
								captionWidthDots := int(math.Round(dotsFromUnitsFloat(textBlockLayout.width, density)))

								xDotsImage := xDots
								if captionWidthDots > imageWidthDots {
									xDotsImage = xDots + (captionWidthDots-imageWidthDots)/2
								}
								draw.Draw(dst, image.Rect(xDotsImage, yDots, xDotsImage+imageWidthDots, yDots+imageHeightDots), picture, image.Point{}, draw.Over)

								xDotsText := xDots
								if captionWidthDots < imageWidthDots {
									xDotsText = xDots + (imageWidthDots-captionWidthDots)/2
								}

								yDots += imageHeightDots + captionGutterDots
								draw.Draw(dst, image.Rect(xDotsText, yDots, xDotsText+textWidthDots, yDots+textHeightDots), textImage, image.Point{}, draw.Over)
							} else {
								draw.Draw(dst, image.Rect(xDots, yDots, xDots+imageWidthDots, yDots+imageHeightDots), picture, image.Point{}, draw.Over)
							}
						}
					}
				}
			}

			w, h := item.PageSizePts()
			offsets = append(offsets, PageInfo{n, w, h})
			thisn, thisErr := writePage(dst, objNum, pp, outFilename, isPageRangeMulti, item.IntBookSetting("output-compression"))
			if thisErr != nil {
				return
			}

			if globalVerboseFlag&4 != 0 {
				log.Printf("Rendered Page %v / %v", pp+1, len(pbBook.pages))
			}

			n += thisn
			objNum++
		}
	}
	_, endErr := writeFooter(outFilename, n, offsets)
	if endErr != nil {
		return
	}
}
