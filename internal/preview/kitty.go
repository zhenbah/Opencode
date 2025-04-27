package preview

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/opencode-ai/opencode/internal/logging"
	"golang.org/x/image/draw"
	"image"
	"image/png"
	"os"
	"strings"
)

const chunkSize = 4096 // Size of each chunk
func PreviewImage(path string, width, height int) string {
	if path == "" {
		logging.Error("Error whilr rendering preview")
		return ""
	}
	var imageData []byte
	var err error
	if width > 0 || height > 0 {
		imageData, err = resizeImage(path, width, height)
	} else {
		imageData, err = os.ReadFile(path)
	}
	if err != nil {
		logging.Error("Error whilr rendering preview")
		return ""
	}
	base64Data := base64.StdEncoding.EncodeToString(imageData)
	return sendImageToKitty(base64Data)
}

func resizeImage(filepath string, targetWidth, targetHeight int) ([]byte, error) {

	input, err := os.Open(filepath)

	if err != nil {
		logging.Error("Error whilr rendering preview")
		return nil, err

	}

	defer input.Close()
	src, _, err := image.Decode(input)
	if err != nil {
		logging.Error("Error whilr rendering preview")
		return nil, err
	}

	bounds := src.Bounds()
	origWidth := bounds.Dx()
	origHeight := bounds.Dy()
	if targetWidth == 0 && targetHeight == 0 {
		targetWidth = origWidth
		targetHeight = origHeight
	} else if targetWidth == 0 {
		targetWidth = int(float64(targetHeight) * float64(origWidth) / float64(origHeight))
	} else if targetHeight == 0 {
		targetHeight = int(float64(targetWidth) * float64(origHeight) / float64(origWidth))
	} else {
		ratioWidth := float64(targetWidth) / float64(origWidth)
		ratioHeight := float64(targetHeight) / float64(origHeight)
		ratio := min(ratioHeight, ratioWidth)
		targetWidth = int(float64(origWidth) * ratio)
		targetHeight = int(float64(origHeight) * ratio)
	}
	dst := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))
	draw.BiLinear.Scale(dst, dst.Rect, src, src.Bounds(), draw.Over, nil)
	var buf bytes.Buffer
	err = png.Encode(&buf, dst)
	if err != nil {
		return nil, fmt.Errorf("error encoding image: %v", err)
	}

	return buf.Bytes(), nil
}

func sendImageToKitty(base64Data string) string {

	pos := 0
	total := len(base64Data)
	var imageString strings.Builder
	for pos < total {
		end := min(pos+chunkSize, total)
		chunk := base64Data[pos:end]
		imageString.WriteString("\033_G")
		if pos == 0 {
			imageString.WriteString("a=T,f=100")
			if end < total {
				imageString.WriteString(",m=1")
			}
		} else {
			imageString.WriteString("m=1")
			if end == total {
				imageString.WriteString(",m=0")
			}
		}
		imageString.WriteString(";")
		imageString.WriteString(chunk)
		imageString.WriteString("\033\\")
		pos = end
	}
	fmt.Println("tst")
	return imageString.String()
}
