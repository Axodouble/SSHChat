package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func main() {
	style := tcell.StyleDefault.Background(tcell.ColorBlack)

	box := tview.NewBox().SetBorder(true).SetBorder(true).SetBorderStyle(style).SetTitle("Box")

	if err := tview.NewApplication().SetRoot(box, true).Run(); err != nil {
		panic(err)
	}
}
