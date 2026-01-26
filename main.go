package main

import (
    "fyne.io/fyne/v2/app"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/widget"
)

func main() {
    myApp := app.New()
    myWindow := myApp.NewWindow("Hello Fyne")

    myWindow.SetContent(
        container.NewVBox(
            widget.NewLabel("Welcome to Fyne!"),
            widget.NewButton("Click Me", func() {
                myWindow.SetContent(widget.NewLabel("Button Clicked!"))
            }),
        ),
    )

    myWindow.ShowAndRun()
}