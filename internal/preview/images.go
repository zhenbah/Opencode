package preview

import (
	"fmt"
	"github.com/muesli/termenv"
	"github.com/nfnt/resize"
	_ "golang.org/x/image/webp"
	"golang.org/x/term"
	"image"
	"image/color"
	"image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
)

type RenderMode int

const (
	RenderAuto RenderMode = iota
	RenderUnicode
	RenderAscii
	RenderDots
)

const (
	maxWidth  = 30
	maxHeight = 20
)

func detectRenderMode() RenderMode {
	colorSupport := termenv.ColorProfile()

	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width < 40 || height < 20 {
		return RenderAscii
	}

	switch colorSupport {
	case termenv.TrueColor:
		return RenderUnicode
	case termenv.ANSI256:
		return RenderDots
	default:
		return RenderAscii
	}
}

func ConvertImageToANSI(img image.Image, defaultBGColor color.Color) string {
	renderMode := detectRenderMode()

	width := img.Bounds().Dx()
	height := img.Bounds().Dy()
	var outputLines []string

	for y := 0; y < height; y += 2 {
		var line strings.Builder
		for x := range width {
			upperColor := rgbToTermenv(img.At(x, y))
			lowerColor := rgbToTermenv(defaultBGColor)

			if y+1 < height {
				lowerColor = rgbToTermenv(img.At(x, y+1))
			}

			// Render based on mode
			var cell string
			switch renderMode {
			case RenderUnicode:
				// Using the "▄" character which fills the lower half
				cell = termenv.String("▄").Foreground(lowerColor).Background(upperColor).String()
			case RenderDots:
				// Use dots for less color-intensive terminals
				cell = termenv.String("•").Foreground(lowerColor).Background(upperColor).String()
			case RenderAscii:
				// Simple ASCII representation
				brightness := calculateBrightness(img.At(x, y))
				cell = getAscIIChar(brightness)
			default:
				cell = " "
			}

			line.WriteString(cell)
		}
		outputLines = append(outputLines, line.String())
	}

	return strings.Join(outputLines, "\n")
}

func calculateBrightness(c color.Color) float64 {
	rgba := color.RGBAModel.Convert(c).(color.RGBA)
	return (0.299*float64(rgba.R) + 0.587*float64(rgba.G) + 0.114*float64(rgba.B)) / 255.0
}

func getAscIIChar(brightness float64) string {
	asciiChars := " .:-=+*#%@"
	index := int(brightness * float64(len(asciiChars)-1))
	return string(asciiChars[index])
}

func ImagePreview(path string, defaultBGColor string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if strings.ToLower(filepath.Ext(path)) == ".gif" {
		gifImg, err := gif.DecodeAll(file)
		if err != nil {
			return "", err
		}

		img := gifImg.Image[0]

		resizedImg := resize.Thumbnail(uint(maxWidth), uint(maxHeight), img, resize.Lanczos3)

		bgColor, err := hexToColor(defaultBGColor)
		if err != nil {
			return "", fmt.Errorf("invalid background color: %w", err)
		}

		ansiImage := ConvertImageToANSI(resizedImg, bgColor)

		return ansiImage, nil
	}

	img, _, err := image.Decode(file)
	if err != nil {
		return "", err
	}

	resizedImg := resize.Thumbnail(uint(maxWidth), uint(maxHeight), img, resize.Lanczos3)

	bgColor, err := hexToColor(defaultBGColor)
	if err != nil {
		return "", fmt.Errorf("invalid background color: %w", err)
	}

	ansiImage := ConvertImageToANSI(resizedImg, bgColor)

	return ansiImage, nil
}

func rgbToTermenv(col color.Color) termenv.RGBColor {
	rgba := color.RGBAModel.Convert(col).(color.RGBA)
	return termenv.RGBColor(fmt.Sprintf("#%02x%02x%02x", rgba.R, rgba.G, rgba.B))
}

func hexToColor(hex string) (color.RGBA, error) {
	if len(hex) != 7 || hex[0] != '#' {
		return color.RGBA{}, fmt.Errorf("invalid hex color format")
	}
	var r, g, b uint8
	_, err := fmt.Sscanf(hex[1:], "%02x%02x%02x", &r, &g, &b)
	if err != nil {
		return color.RGBA{}, err
	}
	return color.RGBA{R: r, G: g, B: b, A: 255}, nil
}
