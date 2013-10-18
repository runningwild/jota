// +build !nographics

package game

import (
	"fmt"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/jota/base"
	g2 "github.com/runningwild/jota/gui"
	"github.com/runningwild/linear"
)

type cameraInfo struct {
	regionPos  linear.Vec2
	regionDims linear.Vec2
	// Camera positions.  target is used for the invaders so that the camera can
	// follow the players without being too jerky.  limit is used by the architect
	// so we have a constant reference point.
	current, target, limit struct {
		mid, dims linear.Vec2
	}
	zoom         float64
	cursorHidden bool
}

type gameLocalData struct {
	camera cameraInfo
}

func (g *Game) RenderLocalSetup(region g2.Region) {
	dict := base.GetDictionary("luxisr")
	size := 60.0
	y := 100.0
	dict.RenderString("Engines:", size, y, 0, size, gui.Left)
	for i, id := range g.Setup.EngineIds {
		y += size
		if id == g.local.Engine.Id() {
			gui.SetFontColor(0.7, 0.7, 1, 1)
		} else {
			gui.SetFontColor(0.7, 0.7, 0.7, 1)
		}
		dataStr := fmt.Sprintf("Engine %d, Side %d, %s", id, g.Setup.Sides[id].Side, g.Champs[g.Setup.Sides[id].Champ].Name)
		dict.RenderString(dataStr, size, y, 0, size, gui.Left)
		if g.local.Engine.Id() == 1 && i == g.Setup.local.index {
			dict.RenderString(">", 50, y, 0, size, gui.Right)
		}
	}
	y += size
	gui.SetFontColor(0.7, 0.7, 0.7, 1)
	if g.local.Engine.Id() == 1 {
		dict.RenderString("Start!", size, y, 0, size, gui.Left)
		if g.Setup.local.index == len(g.Setup.EngineIds) {
			dict.RenderString(">", 50, y, 0, size, gui.Right)
		}
	}
}

func (g *Game) RenderLocalMoba(region g2.Region) {
	// g.renderLocalHelper(region, local, &local.moba.currentPlayer.camera, local.moba.currentPlayer.side)
	// if g.Ents[local.moba.currentPlayer.gid] == nil {
	//   var id int64
	//   fmt.Sscanf(string(local.moba.currentPlayer.gid), "Engine:%d", &id)
	//   seconds := float64(g.Engines[id].CountdownFrames) / 60.0
	//   dict := base.GetDictionary("luxisr")
	//   gui.SetFontColor(0.7, 0.7, 1, 1)
	//   dict.RenderString(fmt.Sprintf("%2.3f", seconds), 300, 300, 0, 100, gui.Left)
	// }
}

// Draws everything that is relevant to the players on a computer, but not the
// players across the network.  Any ui used to determine how to place an object
// or use an ability, for example.
func (g *Game) RenderLocal(region g2.Region) {
	if g.Setup != nil {
		g.RenderLocalSetup(region)
		return
	}
	g.local.Camera.regionPos = linear.Vec2{float64(region.X), float64(region.Y)}
	g.local.Camera.regionDims = linear.Vec2{float64(region.Dx), float64(region.Dy)}
	g.RenderLocalMoba(region)
}
