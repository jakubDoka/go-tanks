package main

import (
	"github.com/jakubDoka/mlok/logic/frame"
	"github.com/jakubDoka/tanks/game"
)

func main() {
	game := game.NGame()

	ticker := frame.Delta{}

	for !game.ShouldClose() {
		delta := ticker.Tick()
		game.World.Update(game.Window, delta)
	}
}
