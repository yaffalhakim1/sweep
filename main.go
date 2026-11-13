package main

import (
	"log"

	"github.com/egoist/mygo"

	"github.com/oneo-me/onevra/source/shells/sidebar"

	"sweep/internal/app"
)

func main() {
	a := app.New(app.LoadSettings())

	mygo.App.WhenReady(func() {
		// The window is Onevra's sidebar Shell: it draws the navigation
		// column, the floating header the window is dragged by, and the
		// resize handle, and it installs the theme before any content is
		// built. The app supplies only the two regions.
		shell := sidebar.NewShell(sidebar.ShellOptions{
			Title:        "Sweep",
			Width:        1180,
			Height:       800,
			MinWidth:     820,
			MinHeight:    560,
			StateKey:     "main",
			LeftWidth:    248,
			LeftMinWidth: 208,
			LeftMaxWidth: 320,
			Left:         a.ViewLeft,
			Content:      a.ViewContent,
			LeftHeader:   a.ViewLeftHeader,
			Theme:        a.ViewTheme,
		})
		a.SetWindow(shell.Window())
		// The sidebar paints its own surface, so the window does not need a
		// native material behind it. Turning Mica off is what keeps the
		// column opaque: with it on, the Shell leaves the sidebar transparent
		// for the material to show through, and the page's rows appear behind
		// the navigation.
		shell.Window().SetVibrancy(mygo.VibrancyNone)
		// The remembered appearance is what the native chrome follows, not
		// just the drawn content.
		if a.Dark() {
			mygo.Theme.SetSource(mygo.ThemeDark)
		}
		a.Start()
	})

	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
