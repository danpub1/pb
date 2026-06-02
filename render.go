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
	"slices"
	"strings"
	"sync"

	"github.com/spakin/netpbm"

	"github.com/disintegration/imaging"
)

type Rounded struct {
	width  int
	height int
	power  float64 // -10 .. +10
	radius int     // 0 <= radius <= min(width, height)

	wmr  int
	hmr  int
	wmrr int
	hmrr int
	p    float64
	rsq  float64
}

func NewRounded(width int, height int, power float64, radius int) Rounded {
	r := Rounded{width, height, power, radius, 0, 0, 0, 0, 0, 0}

	r.radius = int(math.Max(math.Min(float64(r.radius), math.Min(float64(r.width), float64(r.height))), 0))
	r.wmr = r.width - r.radius
	r.hmr = r.height - r.radius
	r.wmrr = r.width - 2*r.radius
	r.hmrr = r.height - 2*r.radius
	r.p = math.Pow(2, r.power)
	r.rsq = math.Abs(math.Pow(float64(r.radius), r.p))
	return r
}

func (r Rounded) ColorModel() color.Model {
	return color.AlphaModel
}

func (r Rounded) Bounds() image.Rectangle {
	return image.Rect(0, 0, r.width, r.height)
}

func (r Rounded) At(x, y int) color.Color {
	if x > r.radius && x < r.wmr || y > r.radius && y < r.hmr {
		return color.Alpha{255}
	}

	if x >= r.wmr {
		x -= r.wmrr
	}

	if y >= r.hmr {
		y -= r.hmrr
	}

	x = int(math.Abs(float64(x - r.radius + 1)))
	y = int(math.Abs(float64(y - r.radius + 1)))
	if math.Pow(float64(x), r.p)+math.Pow(float64(y), r.p) <= r.rsq {
		return color.Alpha{255}
	}

	return color.Alpha{0}
}

func GetMask(width int, height int, power float64, sCornerRadius string, density float64) *image.NRGBA {
	cornerRadius := 0
	if prefix, ok := strings.CutSuffix(sCornerRadius, "%"); ok {
		cornerPercent := Atoi(prefix)
		cornerRadius = int(math.Round(math.Min(float64(width), float64(height)) / 2 * float64(cornerPercent) / 100.0))
	} else {
		cornerRadius = int(math.Round(dotsFromUnitsFloat(Atof(sCornerRadius), density)))
		cornerRadius = int(math.Min(math.Min(float64(width), float64(height))/2, float64(cornerRadius)))
	}
	return imaging.Resize(NewRounded(width*5, height*5, power, cornerRadius*5), width, height, imaging.Box)
}

func ApplyMask(picture image.Image, mask image.Image) image.Image {
	dst := image.NewNRGBA(image.Rect(0, 0, picture.Bounds().Size().X, picture.Bounds().Size().Y))
	draw.DrawMask(dst, image.Rect(0, 0, picture.Bounds().Size().X, picture.Bounds().Size().Y), picture, image.Point{}, mask, image.Point{}, draw.Over)
	return dst
}

func ApplyRoundedCorners(picture image.Image, sCornerRadius string, density float64) image.Image {
	power := 1.0
	parts := strings.SplitN(sCornerRadius, ",", 2)
	if len(parts) > 0 && len(parts[0]) > 0 {
		sCornerRadius = parts[0]
	}
	if len(parts) > 1 && len(parts[1]) > 0 {
		power = Atof(parts[1])
	}
	if len(sCornerRadius) > 0 && sCornerRadius != "0" {
		mask := GetMask(picture.Bounds().Dx(), picture.Bounds().Dy(), power, sCornerRadius, density)
		return ApplyMask(picture, mask)
	}

	return picture
}

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

func writeNewline(outFilename string) (int, error) {
	ext := filepath.Ext(outFilename)
	format := strings.ToLower(ext)

	if format == ".pdf" {
		out, err := os.OpenFile(outFilename, os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Print(err)
			return 0, err
		}
		defer out.Close()
		var n int
		n, err = out.Write([]byte("\n"))
		if err != nil {
			log.Print(err)
			return 0, err
		}

		return n, nil
	}

	return 0, nil
}

type PageInfo struct {
	offset   int
	width    int
	height   int
	pageNum  int
	objNum   int
	pageHash string
}

func writeFooter(outFilename string, bytesWritten int, pageInfo []PageInfo) (int, error) {
	ext := filepath.Ext(outFilename)
	format := strings.ToLower(ext)

	if format == ".pdf" {
		buffer := strings.Builder{}
		numPages := len(pageInfo)

		objList := make([]PageInfo, 0)
		objList = append(objList, pageInfo[:]...)
		slices.SortFunc(objList, func(a PageInfo, b PageInfo) int { return a.objNum - b.objNum })

		objList = append(objList, PageInfo{bytesWritten, 0, 0, 0, 0, ""})
		n, err := buffer.WriteString(fmt.Sprintf("%v 0 obj\n<</Type/Catalog/Pages %v 0 R>>\nendobj\n", numPages+1, numPages+2))
		bytesWritten += n
		objList = append(objList, PageInfo{bytesWritten, 0, 0, 0, 0, ""})
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
			objList = append(objList, PageInfo{bytesWritten, 0, 0, 0, 0, ""})
			n, err = buffer.WriteString(fmt.Sprintf("%v 0 obj\n<</XObject<</I%v %v 0 R>>>>\nendobj\n", ii*3+numPages+3+0, pageInfo[ii].objNum, pageInfo[ii].objNum))
			bytesWritten += n
			objList = append(objList, PageInfo{bytesWritten, 0, 0, 0, 0, ""})
			cmd := fmt.Sprintf("q %v 0 0 %v 0 0 cm /I%v Do Q", pageInfo[ii].width, pageInfo[ii].height, pageInfo[ii].objNum)
			n, err = buffer.WriteString(fmt.Sprintf("%v 0 obj\n<</Length %v>>\nstream\n%v\nendstream\nendobj\n", ii*3+numPages+3+1, len(cmd), cmd))
			bytesWritten += n
			objList = append(objList, PageInfo{bytesWritten, 0, 0, 0, 0, ""})
			n, err = buffer.WriteString(fmt.Sprintf("%v 0 obj\n<</Type/Page/MediaBox[0 0 %v %v]/Rotate 0/Resources %v 0 R/Contents %v 0 R/Parent %v 0 R>>\nendobj\n", ii*3+numPages+3+2, pageInfo[ii].width, pageInfo[ii].height, ii*3+numPages+3+0, ii*3+numPages+3+1, numPages+2))
			bytesWritten += n
		}

		startOfXref := bytesWritten

		n, err = buffer.WriteString(fmt.Sprintf("xref\n0 %v\n0000000000 00001 f\n", numPages*4+3))
		bytesWritten += n
		for ii := range objList {
			n, err = buffer.WriteString(fmt.Sprintf("%010d 00000 n\n", objList[ii].offset))
			bytesWritten += n
		}

		n, err = buffer.WriteString(fmt.Sprintf("trailer\n<</Size %v/Root %v 0 R>>\nstartxref\n%v\n%%%%EOF", len(objList)+1, numPages+1, startOfXref))
		bytesWritten += n
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

		return bytesWritten, nil
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

func writePage(img image.Image, objNum int, curPage int, outFilename string, isPageRangeMulti bool, compressionLevel int, useMozJpeg bool, samplingFactor string, cjpegCmd string) (int, error) {
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
		if useMozJpeg {
			bytesWritten := 0
			var err error
			if bytesWritten, err = writeJPEG(img, out, compressionLevel, samplingFactor, cjpegCmd); err != nil {
				log.Print(err)
				return 0, err
			}
			return bytesWritten, nil
		} else {
			if Opts.Verbose("D") {
				log.Print("Writing JPEG")
			}
			options := jpeg.Options{Quality: compressionLevel}
			if err := jpeg.Encode(out, img, &options); err != nil {
				log.Print(err)
				return 0, err
			}
			if Opts.Verbose("D") {
				log.Print("Wrote JPEG")
			}
			return 0, nil
		}
	case ".pdf":
		writer := PdfJpegObjectWriter{}
		writer.Start(out, objNum, img.Bounds().Dx(), img.Bounds().Dy())

		if useMozJpeg {
			var err error
			if _, err = writeJPEG(img, writer, compressionLevel, samplingFactor, cjpegCmd); err != nil {
				log.Print(err)
				return 0, err
			}
		} else {
			if Opts.Verbose("D") {
				log.Print("Writing JPEG")
			}
			options := jpeg.Options{Quality: compressionLevel}
			if err := jpeg.Encode(writer, img, &options); err != nil {
				log.Print(err)
				return 0, err
			}
			if Opts.Verbose("D") {
				log.Print("Wrote JPEG")
			}
		}

		return writer.Finish()
	}

	return 0, nil
}

func scaleToRect(picture image.Image, item *PbItem) image.Image {
	zoom, zoomXOffset, zoomYOffset, dstAspect, offset := item.ImageRectSetting()

	if zoom == -1 { // squish
		return picture
	}

	wr, hr, _, _ := calcStraighten(float64(item.imageWidthPx), float64(item.imageHeightPx), item.FloatSetting("straighten"))

	srcAspect := wr / hr

	if zoom > 0 && zoom < 100 {
		newWr := wr
		newHr := hr
		if dstAspect > srcAspect { // dst is wider than src
			newHr = hr * float64(zoom) / 100.0
			newWr = newHr * dstAspect
			if newWr > wr {
				newWr = wr
			}
		} else { // dst is taller than src
			newWr = wr * float64(zoom) / 100.0
			newHr = newWr / dstAspect
			if newHr > hr {
				newHr = hr
			}
		}
		xOffset := int(math.Round((wr - newWr) * float64(zoomXOffset) / 100.0))
		yOffset := int(math.Round((hr - newHr) * float64(zoomYOffset) / 100.0))
		picture = imaging.Crop(picture, image.Rectangle{image.Point{xOffset, yOffset}, image.Point{xOffset + int(math.Round(newWr)), yOffset + int(math.Round(newHr))}})
		wr = newWr
		hr = newHr
		srcAspect = wr / hr
		zoom = 1
	}

	dstWidth := int(math.Round(wr))
	dstHeight := int(math.Round(hr))
	dstXOffset := 0
	dstYOffset := 0

	switch zoom {
	case 0: // zoom == 0 == trim
		if dstAspect > srcAspect { // dst is wider than src, crop top & bottom
			dstHeight = int(math.Round(float64(dstWidth) / dstAspect))
			dstYOffset = int(math.Round(float64(int(math.Round(hr))-dstHeight) * float64(offset) / 100.0))
			return imaging.Crop(picture, image.Rectangle{image.Point{dstXOffset, dstYOffset}, image.Point{dstXOffset + dstWidth, dstYOffset + dstHeight}})
		} else if dstAspect < srcAspect { // dst is taller than src, crop left & right
			dstWidth = int(math.Round(float64(dstHeight) * dstAspect))
			dstXOffset = int(math.Round(float64(int(math.Round(wr))-dstWidth) * float64(offset) / 100.0))
			return imaging.Crop(picture, image.Rectangle{image.Point{dstXOffset, dstYOffset}, image.Point{dstXOffset + dstWidth, dstYOffset + dstHeight}})
		} else {
			return picture
		}
	case 100: // zoom == 100 == fit
		if dstAspect > srcAspect { // dst is wider than src, pad left & right
			dstWidth = int(math.Round(float64(int(math.Round(hr))) * dstAspect))
			dstXOffset = int(math.Round(float64(dstWidth-int(math.Round(wr))) * float64(offset) / 100.0))
			dst := image.NewNRGBA(image.Rect(0, 0, dstWidth, dstHeight))
			backColor := colorToNRGBA(item.Setting("image-background"))
			draw.Draw(dst, dst.Bounds(), image.NewUniform(color.NRGBA{backColor.R, backColor.G, backColor.B, backColor.A}), image.Point{}, draw.Src)
			return imaging.Paste(dst, picture, image.Point{dstXOffset, dstYOffset})
		} else if dstAspect < srcAspect { // dst is taller than src, pad top & bottom
			dstHeight = int(math.Round(float64(int(math.Round(wr))) / dstAspect))
			dstYOffset = int(math.Round(float64(dstHeight-int(math.Round(hr))) * float64(offset) / 100.0))
			dst := image.NewNRGBA(image.Rect(0, 0, dstWidth, dstHeight))
			backColor := colorToNRGBA(item.Setting("image-background"))
			draw.Draw(dst, dst.Bounds(), image.NewUniform(color.NRGBA{backColor.R, backColor.G, backColor.B, backColor.A}), image.Point{}, draw.Src)
			return imaging.Paste(dst, picture, image.Point{dstXOffset, dstYOffset})
		} else {
			return picture
		}
	default:
		return picture
	}
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

func tilt(picture image.Image, angle float64) (image.Image, int, int) {

	orgWidth := picture.Bounds().Dx()
	orgHeight := picture.Bounds().Dy()

	picture = imaging.Rotate(picture, -angle, color.NRGBA{127, 127, 127, 0})

	newWidth := picture.Bounds().Dx()
	newHeight := picture.Bounds().Dy()

	return picture, (newWidth - orgWidth) / 2, (newHeight - orgHeight) / 2
}

func convertImage(picture image.Image) image.Image {
	// this is too slow for regular use
	// may be able to adapt to use imagmagick or mozjpeg to create quality jpegs for final output
	log.Print("executing convert")
	cmd := exec.Command("convert", "-", "-adaptive-sharpen", "x5", "PPM:-")

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
		err := netpbm.Encode(stdin, picture, &netpbm.EncodeOptions{Format: netpbm.PPM})
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

func writeJPEG(picture image.Image, out io.Writer, compressionLevel int, samplingFactor string, cjpegCmd string) (int, error) {
	if len(cjpegCmd) == 0 {
		cjpegCmd = "/home/dms/programming/mozjpeg-4.1.1/mozjpeg-4.1.1/cjpeg-static"
	}
	cmd := exec.Command(cjpegCmd, "-quality", fmt.Sprintf("%v", compressionLevel), "-sample", samplingFactor)

	bytesWritten := 0
	var errReturn error

	log.Print(cmd.String())

	stdin, err1 := cmd.StdinPipe()
	if err1 != nil {
		log.Print("Error opening stdin")
		log.Print(err1)
		return 0, err1
	}

	stdout, err2 := cmd.StdoutPipe()
	if err2 != nil {
		log.Print("Error opening stdout")
		log.Print(err2)
		return 0, err2
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer stdin.Close()
		defer wg.Done()
		err := netpbm.Encode(stdin, picture, &netpbm.EncodeOptions{Format: netpbm.PPM})
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

	err2 = cmd.Run()
	if err2 != nil {
		log.Print("Error running command")
		log.Print(err2)
		errReturn = err2
	}

	wg.Wait()

	if errReturn == nil {
		log.Printf("%v: %v bytes", cmd.String(), bytesWritten)
	} else {
		log.Printf("%v: %v bytes, %v", cmd.String(), bytesWritten, errReturn)
	}
	return bytesWritten, errReturn
}

func renderText(item *PbItem, textBlockLayouts []TextBlockLayout, left float64, top float64, density float64) (image.Image, int, int) {

	if len(textBlockLayouts) == 0 {
		return nil, 0, 0
	}

	var textImage image.Image
	rotation := Atoi(item.Setting("rotate"))
	switch rotation {
	case -90, 270:
		temp := textBlockLayouts[0].height
		textBlockLayouts[0].height = textBlockLayouts[0].width
		textBlockLayouts[0].width = temp
		textImage = TextToImage(&textBlockLayouts[0], item.TextInfo())
		temp = textBlockLayouts[0].height
		textBlockLayouts[0].height = textBlockLayouts[0].width
		textBlockLayouts[0].width = temp
		textImage = imaging.Rotate90(textImage)
	case 90, -270:
		temp := textBlockLayouts[0].height
		textBlockLayouts[0].height = textBlockLayouts[0].width
		textBlockLayouts[0].width = temp
		textImage = TextToImage(&textBlockLayouts[0], item.TextInfo())
		temp = textBlockLayouts[0].height
		textBlockLayouts[0].height = textBlockLayouts[0].width
		textBlockLayouts[0].width = temp
		textImage = imaging.Rotate270(textImage)
	case 180, -180:
		textImage = TextToImage(&textBlockLayouts[0], item.TextInfo())
		textImage = imaging.Rotate180(textImage)
	default:
		textImage = TextToImage(&textBlockLayouts[0], item.TextInfo())
	}

	textImage = ApplyRoundedCorners(textImage, item.Setting("corner-radius"), density)

	tiltAngle := item.FloatSetting("tilt")
	deltaXtilt := 0
	deltaYtilt := 0
	var picture image.Image
	if tiltAngle != 0 {
		picture, deltaXtilt, deltaYtilt = tilt(textImage, tiltAngle)
	} else {
		picture = textImage
	}

	if sSize := item.Setting("float"); sSize != "" {
		if sSizeParts := strings.SplitN(sSize, ",", 2); len(sSizeParts) == 2 {
			item.xOffset = Atof(sSizeParts[0]) - left
			item.yOffset = Atof(sSizeParts[1]) - top
		}
	}

	if sOutline := item.Setting("text-outline"); len(sOutline) > 0 {
		xOffset := 0
		yOffset := 0
		picture, _, _, xOffset, yOffset = Outline(picture, picture.Bounds().Size().X, picture.Bounds().Size().Y, density, sOutline)
		deltaXtilt -= xOffset
		deltaYtilt -= yOffset
	}

	if sDropShadow := item.Setting("text-shadow"); len(sDropShadow) > 0 {
		xOffset := 0
		yOffset := 0
		picture, _, _, xOffset, yOffset = DropShadow(picture, picture.Bounds().Size().X, picture.Bounds().Size().Y, density, sDropShadow)
		deltaXtilt -= xOffset
		deltaYtilt -= yOffset
	}

	return picture, deltaXtilt, deltaYtilt
}

func renderHeader(dst *image.NRGBA, header string, curPage int, totalPages int, left float64, margin float64, marginOffsetSign int, density float64, namedItems []PbItem) {
	parts := strings.SplitN(header, ",", 5)
	offset := 0.0
	leadingPages := 0
	trailingPages := 0
	headerName := ""
	if (curPage+1)%2 == 0 {
		if len(parts) > 0 {
			headerName = parts[0]
		}
	} else {
		if len(parts) > 1 {
			headerName = parts[1]
		} else {
			headerName = parts[0]
		}
	}
	if len(parts) > 2 {
		offset = Atof(parts[2])
	}
	if len(parts) > 3 {
		leadingPages = Atoi(parts[3])
	}
	if len(parts) > 4 {
		trailingPages = Atoi(parts[4])
	}

	var headerItem *PbItem
	if len(headerName) > 0 {
		for ii := range namedItems {
			if namedItems[ii].Setting("name") == headerName {
				headerItem = &namedItems[ii]
				break
			}
		}
	}

	if headerItem != nil {
		headerItemText := headerItem.Setting("text")
		headerItemText = strings.ReplaceAll(headerItemText, "{{PageNumber}}", fmt.Sprintf("%v", curPage+1-leadingPages))
		headerItemText = strings.ReplaceAll(headerItemText, "{{TotalPages}}", fmt.Sprintf("%v", totalPages-leadingPages-trailingPages))
		textBlockLayouts := getOneTextDimensions(headerItem, headerItemText)
		textImage, deltaXtilt, deltaYtilt := renderText(headerItem, textBlockLayouts, left, margin, density)
		if textImage != nil {
			xDots := int(math.Round(dotsFromUnitsFloat(left, density))) - deltaXtilt
			yDots := int(math.Round(dotsFromUnitsFloat(margin+offset*float64(marginOffsetSign), density))) - deltaYtilt - textImage.Bounds().Size().Y/2
			draw.Draw(dst, image.Rect(xDots, yDots, xDots+textImage.Bounds().Size().X, yDots+textImage.Bounds().Size().Y), textImage, image.Point{}, draw.Over)
		}
	}
}

func DropShadow(picture image.Image, width int, height int, density float64, sDropShadow string) (image.Image, int, int, int, int) {
	parts := strings.SplitN(sDropShadow, ",", 4) // color, blur, x, y

	bcolor := color.NRGBA{0, 0, 0, 0}
	if len(parts[0]) > 0 {
		bcolor = colorToNRGBA(parts[0])
	}

	blur := 0.0
	if len(parts) > 1 && len(parts[1]) > 0 {
		blur = Atof(parts[1])
	}

	xOffset := 0
	if len(parts) > 2 && len(parts[2]) > 0 {
		x := Atof(parts[2])
		xOffset = int(math.Round(dotsFromUnitsFloat(x, density)))
	}

	yOffset := 0
	if len(parts) > 3 && len(parts[3]) > 0 {
		y := Atof(parts[3])
		yOffset = int(math.Round(dotsFromUnitsFloat(y, density)))
	}

	newWidth := int(math.Round(blur*2*3 + math.Abs(float64(xOffset)) + float64(width)))
	newHeight := int(math.Round(blur*2*3 + math.Abs(float64(yOffset)) + float64(height)))

	newPicture := image.NewNRGBA(image.Rect(0, 0, newWidth, newHeight))
	draw.Draw(newPicture, newPicture.Bounds(), image.NewUniform(color.NRGBA{0, 0, 0, 0}), image.Point{}, draw.Src)

	dXOffset := int(math.Round(math.Max(0, float64(xOffset)) + blur*3))
	dYOffset := int(math.Round(math.Max(0, float64(yOffset)) + blur*3))
	draw.Draw(newPicture, image.Rect(dXOffset, dYOffset, dXOffset+width, dYOffset+height), picture, image.Point{}, draw.Over)

	setShadowColor := func(c color.NRGBA) color.NRGBA {
		return color.NRGBA{bcolor.R, bcolor.G, bcolor.B, uint8(math.Round(float64(c.A) * float64(bcolor.A) / 255.0))}
	}
	newPicture = imaging.AdjustFunc(newPicture, setShadowColor)

	newPicture = imaging.Blur(newPicture, blur)

	newXOffset := int(math.Round(math.Max(0, blur*3-float64(xOffset))))
	newYOffset := int(math.Round(math.Max(0, blur*3-float64(yOffset))))
	draw.Draw(newPicture, image.Rect(newXOffset, newYOffset, newXOffset+width, newYOffset+height), picture, image.Point{}, draw.Over)

	return newPicture, newWidth, newHeight, newXOffset, newYOffset
}

func Outline(picture image.Image, width int, height int, density float64, sOutline string) (image.Image, int, int, int, int) {
	parts := strings.SplitN(sOutline, ",", 2) // color,amount

	bcolor := color.NRGBA{0, 0, 0, 0}
	if len(parts[0]) > 0 {
		bcolor = colorToNRGBA(parts[0])
	}

	offset := 0
	if len(parts) > 1 && len(parts[1]) > 0 {
		x := Atof(parts[1])
		offset = int(math.Abs(math.Round(dotsFromUnitsFloat(x, density))))
	}

	newWidth := int(math.Round(math.Abs(float64(offset*2)) + float64(width)))
	newHeight := int(math.Round(math.Abs(float64(offset*2)) + float64(height)))

	newPicture := image.NewNRGBA(image.Rect(0, 0, newWidth, newHeight))
	draw.Draw(newPicture, newPicture.Bounds(), image.NewUniform(color.NRGBA{0, 0, 0, 0}), image.Point{}, draw.Src)

	// for ii := 0; ii <= offset*2; ii++ {
	// 	for jj := 0; jj <= offset*2; jj++ {
	// 		draw.Draw(newPicture, image.Rect(ii, jj, ii+width, jj+height), picture, image.Point{}, draw.Over)
	// 	}
	// }

	for ii := 0; ii <= offset*2; ii++ {
		draw.Draw(newPicture, image.Rect(0, ii, 0+width, ii+height), picture, image.Point{}, draw.Over)
		draw.Draw(newPicture, image.Rect(offset*2, ii, offset*2+width, ii+height), picture, image.Point{}, draw.Over)
		if ii != 0 {
			draw.Draw(newPicture, image.Rect(ii, 0, ii+width, 0+height), picture, image.Point{}, draw.Over)
		}
		if ii != offset*2 {
			draw.Draw(newPicture, image.Rect(ii, offset*2, ii+width, offset*2+height), picture, image.Point{}, draw.Over)
		}
	}

	setOutlineColor := func(c color.NRGBA) color.NRGBA {
		return color.NRGBA{bcolor.R, bcolor.G, bcolor.B, uint8(math.Round(float64(c.A) * float64(bcolor.A) / 255.0))}
	}
	newPicture = imaging.AdjustFunc(newPicture, setOutlineColor)

	draw.Draw(newPicture, image.Rect(offset, offset, offset+width, offset+height), picture, image.Point{}, draw.Over)

	return newPicture, newWidth, newHeight, -offset, -offset
}

func GetNamedImage(sImage string, left float64, top float64, density float64, pbBook *PbBook, cache []BackgroundCacheItem) (*BackgroundCacheItem, []BackgroundCacheItem) {
	cacheItem := FindBackgroundCacheItem(cache, sImage)
	if cacheItem == nil {
		var backgroundItem *PbItem
		for ii := range pbBook.namedItems {
			if pbBook.namedItems[ii].Setting("name") == sImage {
				backgroundItem = &pbBook.namedItems[ii]
				break
			}
		}
		if backgroundItem != nil {
			picture, xDots, yDots, deltaXtilt, deltaYtilt, imageWidthDots, imageHeightDots, newCache := renderImage(backgroundItem, left, top, density, pbBook, cache)
			if picture != nil {
				cache = newCache
				cacheItem = &BackgroundCacheItem{sImage, picture, xDots, yDots, deltaXtilt, deltaYtilt, imageWidthDots, imageHeightDots}
				cache = AddBackgourndCacheItem(cache, cacheItem)
			}
		}
	}
	return cacheItem, cache
}

func ApplyFrame(picture image.Image, item *PbItem, density float64, pbBook *PbBook, cache []BackgroundCacheItem) (image.Image, int, int, []BackgroundCacheItem) {
	frameInfo := item.ImageFrame()
	if frameInfo.color.A == 0 && len(frameInfo.name) == 0 {
		return picture, 0, 0, cache
	}

	addedHeight := dotsFromUnitsFloat(frameInfo.size.top+frameInfo.size.bottom, density)
	addedWidth := dotsFromUnitsFloat(frameInfo.size.left+frameInfo.size.right, density)
	newWidth := int(math.Round(float64(picture.Bounds().Dx()) + addedWidth))
	newHeight := int(math.Round(float64(picture.Bounds().Dy()) + addedHeight))
	newPicture := image.NewNRGBA(image.Rect(0, 0, newWidth, newHeight))

	if frameInfo.color.A > 0 {
		draw.Draw(newPicture, newPicture.Bounds(), image.NewUniform(color.NRGBA{frameInfo.color.R, frameInfo.color.G, frameInfo.color.B, frameInfo.color.A}), image.Point{}, draw.Src)
	}

	if !frameInfo.above && len(frameInfo.name) > 0 {
		var cacheItem *BackgroundCacheItem
		cacheItem, cache = GetNamedImage(frameInfo.name, 0, 0, density, pbBook, cache)
		if cacheItem != nil {
			framePic := imaging.Resize(cacheItem.picture, newWidth, newHeight, imaging.Lanczos)
			draw.Draw(newPicture, image.Rect(0, 0, newWidth, newHeight), framePic, image.Point{}, draw.Over)
		}
	}

	xOffset := int(math.Round(dotsFromUnitsFloat(frameInfo.size.left, density)))
	yOffset := int(math.Round(dotsFromUnitsFloat(frameInfo.size.top, density)))
	draw.Draw(newPicture, image.Rect(xOffset, yOffset, picture.Bounds().Dx()+xOffset, picture.Bounds().Dy()+yOffset), picture, image.Point{}, draw.Src)

	if frameInfo.above && len(frameInfo.name) > 0 {
		var cacheItem *BackgroundCacheItem
		cacheItem, cache = GetNamedImage(frameInfo.name, 0, 0, density, pbBook, cache)
		if cacheItem != nil {
			framePic := imaging.Resize(cacheItem.picture, newWidth, newHeight, imaging.Lanczos)
			draw.Draw(newPicture, image.Rect(0, 0, newWidth, newHeight), framePic, image.Point{}, draw.Over)
		}
	}

	return newPicture, xOffset, yOffset, cache
}

func renderImage(item *PbItem, left float64, top float64, density float64, pbBook *PbBook, cache []BackgroundCacheItem) (image.Image, int, int, int, int, int, int, []BackgroundCacheItem) {
	picture := item.GetImage()
	if picture == nil {
		return nil, 0, 0, 0, 0, 0, 0, cache
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

	flip := strings.ToLower(item.Setting("flip"))
	switch flip {
	case "h":
		picture = imaging.FlipH(picture)
	case "v":
		picture = imaging.FlipV(picture)
	}

	if sSize := item.Setting("float"); sSize != "" {
		if sSizeParts := strings.SplitN(sSize, ",", 4); len(sSizeParts) == 4 {
			item.xOffset = Atof(sSizeParts[0]) - left
			item.yOffset = Atof(sSizeParts[1]) - top
			item.imageWidth = Atof(sSizeParts[2])
			item.imageHeight = Atof(sSizeParts[3])
		}
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

	picture = ApplyRoundedCorners(picture, item.Setting("corner-radius"), density)

	frameXOffset, frameYOffset := 0, 0
	picture, frameXOffset, frameYOffset, cache = ApplyFrame(picture, item, density, pbBook, cache)
	imageWidthDots = picture.Bounds().Dx()
	imageHeightDots = picture.Bounds().Dy()

	outlineXOffset, outlineYOffset := 0, 0
	if sOutline := item.Setting("image-outline"); len(sOutline) > 0 {
		picture, _, _, outlineXOffset, outlineYOffset = Outline(picture, picture.Bounds().Size().X, picture.Bounds().Size().Y, density, sOutline)
		imageWidthDots = picture.Bounds().Dx()
		imageHeightDots = picture.Bounds().Dy()
	}

	itemXOffset := item.xOffset - unitsFromDots(float64(frameXOffset+outlineXOffset), density)
	itemYOffset := item.yOffset - unitsFromDots(float64(frameYOffset+outlineYOffset), density)

	if len(item.textBlockLayouts) > 0 {
		captionGutterDots := int(math.Round(dotsFromUnitsFloat(item.FloatSetting("caption-gutter"), density)))
		textBlockLayout := TextBlockLayout{}
		if item.bestTextBlockLayout >= 0 {
			textBlockLayout = item.textBlockLayouts[item.bestTextBlockLayout]
		}

		var textImage image.Image = TextToImage(&textBlockLayout, item.TextInfo())
		textImage = ApplyRoundedCorners(textImage, item.Setting("corner-radius"), density)

		captionWidthDots := int(math.Round(dotsFromUnitsFloat(textBlockLayout.width, density)))
		captionHeightDots := int(math.Round(dotsFromUnitsFloat(textBlockLayout.height, density)))

		combinedWidthDots := int(math.Max(float64(captionWidthDots), float64(imageWidthDots)))
		combinedHeightDots := imageHeightDots + captionGutterDots + captionHeightDots
		combined := image.NewNRGBA(image.Rect(0, 0, combinedWidthDots, combinedHeightDots))

		captionX := 0
		captionY := 0
		imageX := 0
		imageY := 0
		if captionWidthDots < imageWidthDots {
			captionX = (imageWidthDots - captionWidthDots) / 2
		} else if imageWidthDots < captionWidthDots {
			imageX = (captionWidthDots - imageWidthDots) / 2
		}

		if item.Setting("caption-position") == "below" {
			captionY = imageHeightDots + captionGutterDots
		} else {
			imageY = captionHeightDots + captionGutterDots
		}

		draw.Draw(combined, image.Rect(captionX, captionY, captionX+captionWidthDots, captionY+captionHeightDots), textImage, image.Point{}, draw.Over)
		draw.Draw(combined, image.Rect(imageX, imageY, imageX+imageWidthDots, imageY+imageHeightDots), picture, image.Point{}, draw.Over)

		picture = combined
		imageWidthDots = combinedWidthDots
		imageHeightDots = combinedHeightDots
	}

	tiltAngle := item.FloatSetting("tilt")
	deltaXtilt := 0
	deltaYtilt := 0
	if tiltAngle != 0 {
		picture, deltaXtilt, deltaYtilt = tilt(picture, tiltAngle)
	}

	xDots := int(math.Round(dotsFromUnitsFloat(left+itemXOffset, density)))
	yDots := int(math.Round(dotsFromUnitsFloat(top+itemYOffset, density)))

	dropXOff, dropYOff := 0, 0
	if sDropShadow := item.Setting("image-shadow"); len(sDropShadow) > 0 {
		picture, imageWidthDots, imageHeightDots, dropXOff, dropYOff = DropShadow(picture, imageWidthDots, imageHeightDots, density, sDropShadow)
	}

	return picture, xDots, yDots, deltaXtilt + dropXOff, deltaYtilt + dropYOff, imageWidthDots, imageHeightDots, cache
}

type BackgroundCacheItem struct {
	name            string
	picture         image.Image
	xDots           int
	yDots           int
	deltaXtilt      int
	deltaYtilt      int
	imageWidthDots  int
	imageHeightDots int
}

func FindBackgroundCacheItem(cache []BackgroundCacheItem, name string) *BackgroundCacheItem {
	for ii := range cache {
		if cache[ii].name == name {
			return &cache[ii]
		}
	}
	return nil
}

func AddBackgourndCacheItem(cache []BackgroundCacheItem, item *BackgroundCacheItem) []BackgroundCacheItem {
	if foundItem := FindBackgroundCacheItem(cache, item.name); foundItem == nil {
		cache = append(cache, *item)
	}
	return cache
}

// This is what is needed for writing header, footer, pages
type OutFileInfo struct {
	offsets []PageInfo
	n       int
}

var lastOutFileInfo map[string]OutFileInfo

func renderPages(pbBook *PbBook, outPageRange string, firstIteration bool, pageHashes []string) {
	outFileInfo := make(map[string]OutFileInfo, 0)

	isPageRangeMulti := isPageRangeMulti(outPageRange, firstIteration, pbBook)

	backgroundCache := make([]BackgroundCacheItem, 0)

	usedObjs := make([]int, 0)
	if !firstIteration && lastOutFileInfo != nil {
		for pp := range pbBook.pages {
			page := &pbBook.pages[pp]

			if item := page.PbItem(); item != nil && item.BoolPageSetting("norender") {
				continue
			}

			if len(page.rows) == 0 || len(page.rows[0].columns) == 0 || len(page.rows[0].columns[0].items) == 0 {
				continue
			}

			if isPageInRange(outPageRange, pp, firstIteration) || isCurrentPage(pbBook, pp) {
				continue
			}

			item := page.rows[0].columns[0].items[0].item

			thisOutFilename := item.PageSetting("output-file")

			if _, exists := lastOutFileInfo[thisOutFilename]; exists {
				for ii := range lastOutFileInfo[thisOutFilename].offsets {
					if lastOutFileInfo[thisOutFilename].offsets[ii].pageHash == pageHashes[pp] {
						usedObjs = append(usedObjs, lastOutFileInfo[thisOutFilename].offsets[ii].objNum)
						break
					}
				}
			}
		}
	}

	slices.Sort(usedObjs)

	for pp := range pbBook.pages {
		changed := false
		if changed, _ = fileChanged(inFiles, lastModTime); changed {
			lastOutFileInfo = nil
			return
		}
		page := &pbBook.pages[pp]

		if item := page.PbItem(); item != nil && item.BoolPageSetting("norender") {
			continue
		}

		if len(page.rows) == 0 || len(page.rows[0].columns) == 0 || len(page.rows[0].columns[0].items) == 0 {
			continue
		}
		item := page.rows[0].columns[0].items[0].item

		thisOutFilename := item.PageSetting("output-file")

		if isPageInRange(outPageRange, pp, firstIteration) || isCurrentPage(pbBook, pp) {
			var top float64
			var bottom float64
			var left float64
			var dst *image.NRGBA = nil
			density := 1.0

			pageWidth, pageHeight := FloatSize(item.PageSetting("page-size"))
			density = item.Density()
			top, _, bottom, left = FourTwoOne(item.PageSetting("margin"))
			widthDots := int(math.Round(dotsFromUnitsFloat(pageWidth, density)))
			heightDots := int(math.Round(dotsFromUnitsFloat(pageHeight, density)))
			dst = image.NewNRGBA(image.Rect(0, 0, widthDots, heightDots))

			// fill the destination with the background
			sBackground := item.PageSetting("background")
			if strings.HasPrefix(sBackground, "#") {
				backColor := colorToNRGBA(sBackground)
				draw.Draw(dst, dst.Bounds(), image.NewUniform(color.NRGBA{backColor.R, backColor.G, backColor.B, backColor.A}), image.Point{}, draw.Src)
			} else {
				if len(sBackground) > 0 {
					var cacheItem *BackgroundCacheItem
					cacheItem, backgroundCache = GetNamedImage(sBackground, left, top, density, pbBook, backgroundCache)
					if cacheItem != nil {
						draw.Draw(dst, image.Rect(cacheItem.xDots-cacheItem.deltaXtilt, cacheItem.yDots-cacheItem.deltaYtilt, cacheItem.xDots+cacheItem.imageWidthDots+cacheItem.deltaXtilt, cacheItem.yDots+cacheItem.imageHeightDots+cacheItem.deltaYtilt), cacheItem.picture, image.Point{}, draw.Over)
					}
				}
			}

			header := item.PageSetting("header")
			if len(header) > 0 {
				renderHeader(dst, header, pp, len(pbBook.pages), left, top, -1, density, pbBook.namedItems)
			}

			footer := item.PageSetting("footer")
			if len(footer) > 0 {
				renderHeader(dst, footer, pp, len(pbBook.pages), left, pageHeight-bottom, 1, density, pbBook.namedItems)
			}

			for row := range page.rows {
				for column := range page.rows[row].columns {
					for columnItem := range page.rows[row].columns[column].items {
						item = page.rows[row].columns[column].items[columnItem].item

						if item.itemType == ItemTypeText {
							textImage, deltaXtilt, deltaYtilt := renderText(item, item.textBlockLayouts, left, top, density)
							if textImage != nil {
								xDots := int(math.Round(dotsFromUnitsFloat(left+item.xOffset, density))) - deltaXtilt
								yDots := int(math.Round(dotsFromUnitsFloat(top+item.yOffset, density))) - deltaYtilt
								draw.Draw(dst, image.Rect(xDots, yDots, xDots+textImage.Bounds().Size().X, yDots+textImage.Bounds().Size().Y), textImage, image.Point{}, draw.Over)
							}
						}

						if item.itemType == ItemTypeImage {
							picture, xDots, yDots, deltaXtilt, deltaYtilt, imageWidthDots, imageHeightDots, newCache := renderImage(item, left, top, density, pbBook, backgroundCache)
							backgroundCache = newCache
							if picture != nil {
								draw.Draw(dst, image.Rect(xDots-deltaXtilt, yDots-deltaYtilt, xDots+imageWidthDots+deltaXtilt, yDots+imageHeightDots+deltaYtilt), picture, image.Point{}, draw.Over)
							}
						}
					}
				}
			}

			if outputGamma := item.FloatPageSetting("output-gamma"); outputGamma != 1.0 {
				dst = imaging.AdjustGamma(dst, outputGamma)
			}

			if outputSharpen := item.FloatPageSetting("output-sharpen"); outputSharpen != 0 {
				dst = imaging.Sharpen(dst, outputSharpen)
			}

			var info OutFileInfo
			var exists bool
			if info, exists = outFileInfo[thisOutFilename]; !exists {
				info = OutFileInfo{}
				found := false
				if !firstIteration && lastOutFileInfo != nil {
					if _, exists := lastOutFileInfo[thisOutFilename]; exists {
						info.n = lastOutFileInfo[thisOutFilename].n
						found = true
						n, err := writeNewline(thisOutFilename)
						if err != nil {
							lastOutFileInfo = nil
							return
						}
						info.n += n
					}
				}
				if !found {
					n, err := writeHeader(thisOutFilename)
					if err != nil {
						lastOutFileInfo = nil
						return
					}
					info.n = n
				}
				outFileInfo[thisOutFilename] = info
			}

			objNum := 1
			for _, ii := range usedObjs {
				if ii != objNum {
					break
				}
				objNum++
			}
			usedObjs = append(usedObjs, objNum)
			slices.Sort(usedObjs)

			w, h := item.PageSizePts()
			info.offsets = append(info.offsets, PageInfo{info.n, w, h, pp, objNum, pageHashes[pp]})

			thisn, thisErr := writePage(dst, objNum, pp, thisOutFilename, isPageRangeMulti, item.IntPageSetting("output-compression"), item.BoolPageSetting("output-mozjpeg"), item.PageSetting("output-mozjpeg-sampling"), item.PageSetting("cjpeg-command"))
			if thisErr != nil {
				lastOutFileInfo = nil
				return
			}

			if Opts.Verbose("D") {
				log.Printf("Rendered Page %v / %v", pp+1, len(pbBook.pages))
			}

			info.n += thisn

			outFileInfo[thisOutFilename] = info
		} else if !firstIteration && lastOutFileInfo != nil {
			if _, exists := lastOutFileInfo[thisOutFilename]; exists {
				for ii, lastOffset := range lastOutFileInfo[thisOutFilename].offsets {
					if lastOutFileInfo[thisOutFilename].offsets[ii].pageHash == pageHashes[pp] {
						var info OutFileInfo
						if _, exists := outFileInfo[thisOutFilename]; !exists {
							info = OutFileInfo{}
							info.n = lastOutFileInfo[thisOutFilename].n
							n, err := writeNewline(thisOutFilename)
							if err != nil {
								lastOutFileInfo = nil
								return
							}
							info.n += n
						} else {
							info = outFileInfo[thisOutFilename]
						}
						info.offsets = append(info.offsets, PageInfo{lastOffset.offset, lastOffset.width, lastOffset.height, pp, lastOffset.objNum, pageHashes[pp]})
						outFileInfo[thisOutFilename] = info
						break
					}
				}
			}
		}
	}

	for outFilename, info := range outFileInfo {
		bytesWritten, endErr := writeFooter(outFilename, info.n, info.offsets)
		if endErr != nil {
			lastOutFileInfo = nil
			return
		}
		info.n = bytesWritten
		outFileInfo[outFilename] = info
	}

	lastOutFileInfo = outFileInfo
}
