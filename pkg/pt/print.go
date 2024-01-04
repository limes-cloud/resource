package pt

import (
	"fmt"

	"github.com/gookit/color"
	"github.com/mbndr/figlet4go"
)

func Cyan(str string) {
	color.Cyanln(str)
}

func ArtFont(str string) {
	ascii := figlet4go.NewAsciiRender()
	options := figlet4go.NewRenderOptions()
	hexColor, _ := figlet4go.NewTrueColorFromHexString("885DBA")
	options.FontColor = []figlet4go.Color{
		// Colors can be given by default ansi color codes...
		figlet4go.ColorGreen,
		figlet4go.ColorYellow,
		figlet4go.ColorCyan,
		hexColor,
	}

	renderStr, _ := ascii.RenderOpts(str, options)
	fmt.Println(renderStr)
}
