package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"sweep/internal/app"
)

func main() {
	a := app.New(app.LoadSettings())

	mygo.App.WhenReady(func() {
		win := mygo.NewWindow(mygo.WindowOptions{
			Title:     "Sweep",
			Width:     980,
			Height:    720,
			MinWidth:  640,
			MinHeight: 480,
			// Opens where the user left it last time.
			StateKey: "main",
			// The window shows the interface MyGo draws, not a web page.
			Content: ui.View(a.View),
		})
		a.SetWindow(win)
		a.Start()
	})

	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
