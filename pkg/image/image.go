package image

import (
	"bytes"
	"image"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"strings"

	"golang.org/x/image/bmp"

	"github.com/nfnt/resize"
)

const (
	PNG  = "png"
	JPEG = "jpeg"
	GIF  = "gif"
	BMP  = "bmp"
)

const (
	Center     = "center"
	AspectFill = "fill"
)

type Object interface {
	Resize(width, height int, mode string) ([]byte, error)
}

type object struct {
	img    image.Image
	format string
}

func New(format string, body []byte) (Object, error) {
	img, _, err := image.Decode(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	return &object{img: img, format: format}, err
}

func (o *object) Resize(width, height int, mode string) ([]byte, error) {
	var img image.Image
	switch mode {
	case Center:
		img = o.Center(width, height)
	case AspectFill:
		img = o.AspectFill(width, height)
	}

	return o.ImageToBytes(img)
}

func (o *object) Center(width, height int) image.Image {
	// 创建目标图像
	target := image.NewRGBA(image.Rect(0, 0, width, height))

	// 计算裁剪位置
	x, y := 0, 0
	x = (o.img.Bounds().Dx() - width) / 2
	y = (o.img.Bounds().Dy() - height) / 2

	// 裁剪图像
	draw.Draw(target, target.Bounds(), o.img, image.Point{X: x, Y: y}, draw.Src)

	// 调整图像大小
	return resize.Resize(uint(width), uint(height), target, resize.Lanczos3)
}

func (o *object) AspectFill(width, height int) image.Image {
	// 获取原始图像的宽高
	srcWidth := o.img.Bounds().Dx()
	srcHeight := o.img.Bounds().Dy()

	// 计算原始图像的宽高比和目标宽高比
	srcAspectRatio := float64(srcWidth) / float64(srcHeight)
	dstAspectRatio := float64(width) / float64(height)

	// 计算裁剪区域的起始点和大小
	var cropX, cropY, cropWidth, cropHeight int
	if srcAspectRatio > dstAspectRatio {
		// 原始图像更宽，裁剪高度
		cropWidth = int(float64(srcHeight) * dstAspectRatio)
		cropHeight = srcHeight
		cropX = (srcWidth - cropWidth) / 2
		cropY = 0
	} else {
		// 原始图像更高，裁剪宽度
		cropWidth = srcWidth
		cropHeight = int(float64(srcWidth) / dstAspectRatio)
		cropX = 0
		cropY = (srcHeight - cropHeight) / 2
	}

	// 创建裁剪后的图像
	croppedImg := image.NewRGBA(image.Rect(0, 0, cropWidth, cropHeight))
	draw.Draw(croppedImg, croppedImg.Bounds(), o.img, image.Point{X: cropX, Y: cropY}, draw.Over)

	// 缩放图像
	return resize.Resize(uint(width), uint(height), croppedImg, resize.Lanczos3)
}

func (o *object) ImageToBytes(img image.Image) ([]byte, error) {
	var (
		err    error
		buffer strings.Builder
	)

	switch o.format {
	case PNG:
		err = png.Encode(&buffer, img)
	case JPEG:
		err = jpeg.Encode(&buffer, img, nil)
	case GIF:
		err = gif.Encode(&buffer, img, nil)
	case BMP:
		err = bmp.Encode(&buffer, img)
	default:
		err = png.Encode(&buffer, img)
	}
	if err != nil {
		return nil, err
	}
	return []byte(buffer.String()), nil
}
