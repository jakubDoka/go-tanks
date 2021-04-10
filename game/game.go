package game

import (
	"fmt"
	"os"
	"strings"

	"github.com/jakubDoka/mlok/ggl"
	"github.com/jakubDoka/mlok/ggl/ui"
	"github.com/jakubDoka/mlok/load"
	"github.com/jakubDoka/mlok/mat/rgba"
	"github.com/jakubDoka/tanks/game/assets"
)

type Game struct {
	*ggl.Window
	*assets.Assets
	*World
	*Net

	Closed bool
}

func NGame() *Game {
	g := &Game{}

	win, err := ggl.NWindow(&ggl.WindowConfig{
		Width:     1000,
		Height:    600,
		Resizable: true,
		Title:     "Tanks",
	})
	if err != nil {
		panic(err)
	}
	g.Window = win

	g.Assets = assets.NAssets()
	g.Load("assets", assets.RawAssets)

	for _, p := range g.Assets.Mods {
		g.Load(p, load.OS)
	}

	g.Compile()

	for _, e := range g.Assets.Errors {
		fmt.Println(e)
	}

	g.World = NWorld(g.Assets)

	g.SetupMainMenu()
	g.SetupEndScreen()
	g.SetupSinglePlayer()

	return g
}

func (g *Game) SetupMainMenu() {
	scene := g.Assets.UIScenes["main_menu"]

	main := scene.ID("main")
	maps := scene.ID("maps")
	mapList := scene.ID("map_list")
	back := scene.ID("Back")
	errors := scene.ID("Errors").Module.(*ui.Button)
	errors_page := scene.ID("errors")
	mods := scene.ID("mods")
	mod_list := scene.ID("mod_list")
	mod_status := scene.ID("mod_status").Module.(*ui.Text)
	mod_input := scene.ID("mod_input").Module.(*ui.Area)

	var update_list func()

	update_list = func() {
		for mod_list.ChildCount() != 0 {
			mod_list.PopChild(0)
		}

		for _, m := range g.Assets.Mods {
			err := mod_list.AddGoml(gomlTemp(`<option name=%q button_text="Remove"/>`, m))
			if err != nil {
				panic(err)
			}
			scene.ID(m).Listen(ui.Click, func(i interface{}) {
				for i, o := range g.Assets.Mods {
					if o == m {
						g.Assets.Mods = append(g.Assets.Mods[:i], g.Assets.Mods[i+1:]...)
						g.Assets.SaveGameConfig()
						update_list()
						break
					}
				}
			})
		}
	}

	update_list()

	change := func(e *ui.Element, showback bool) {
		scene.Root.ForChild(func(ch *ui.Element) { ch.SetHidden(true) })
		e.SetHidden(false)
		back.SetHidden(!showback)
	}

	scene.ID("Exit").Listen(ui.Click, func(i interface{}) {
		g.Closed = true
	})
	back.Listen(ui.Click, func(i interface{}) {
		change(main, false)
	})
	scene.ID("Singleplayer").Listen(ui.Click, func(i interface{}) {
		change(maps, true)
	})
	scene.ID("Mods").Listen(ui.Click, func(i interface{}) {
		change(mods, true)
	})

	scene.ID("mod_submit").Listen(ui.Click, func(i interface{}) {
		p := string(mod_input.Content)

		if p == "" {
			mod_status.SetText("nothing to submit")
			return
		}

		if _, err := os.Stat(p); os.IsNotExist(err) {
			mod_status.SetText("directory does not exist")
			return
		}

		mod_status.SetText("reboot the game to apply changes")
		g.Assets.Mods = append(g.Assets.Mods, p)
		g.Assets.SaveGameConfig()
		update_list()
	})

	s := g.Assets.Worlds.Slice()
	for i := range s {
		c := &s[i]
		err := mapList.AddGoml(gomlTemp(`<option name="%s" button_text="Play"/>`, c.K))
		if err != nil {
			panic(err)
		}
		scene.ID(c.K).Listen(ui.Click, func(i interface{}) {
			g.LoadMap(true, &c.V)
		})
	}

	if len(g.Assets.Errors) == 0 {
		errors.Hidden()
	} else {
		var all string
		for _, e := range g.Errors {
			all += e.Error() + "\n"
		}

		scene.ID("error_log").Module.(*ui.Text).SetText(all)
		if strings.Contains(all, "[fatal]") {
			errors.States[ui.Idle].Mask = rgba.Red
		} else {
			errors.States[ui.Idle].Mask = rgba.Yellow
		}

		errors.Listen(ui.Click, func(i interface{}) {
			change(errors_page, true)
		})
	}

}

func (g *Game) SetupEndScreen() {
	scene := g.Assets.UIScenes["end_screen"]

	scene.ID("Exit").Listen(ui.Click, func(i interface{}) {
		g.SetScene("main_menu")
	})

	scene.ID("Retry").Listen(ui.Click, func(i interface{}) {
		g.LoadMap(true, g.Original)
	})
}

func (g *Game) SetupSinglePlayer() {
	scene := g.Assets.UIScenes["singleplayer"]

	scene.ID("Menu").Listen(ui.Click, func(i interface{}) {
		scene.ID("poppup").SetHidden(false)
		g.GameState = Menu
	})

	scene.ID("Exit").Listen(ui.Click, func(i interface{}) {
		g.EndGame(false)
	})

	scene.ID("Resume").Listen(ui.Click, func(i interface{}) {
		scene.ID("poppup").SetHidden(true)
		g.GameState = Singleplayer
	})
}

func gomlTemp(str string, args ...interface{}) []byte {
	return []byte(fmt.Sprintf(str, args...))
}

func (g *Game) ShouldClose() bool {
	return g.Window.ShouldClose() || g.Closed
}
