// +build !nographics

package game

import (
	"fmt"
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/jota/base"
	g2 "github.com/runningwild/jota/gui"
	"github.com/runningwild/jota/stats"
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

func (camera *cameraInfo) FocusRegion(g *Game, side int) {
	min := linear.Vec2{1e9, 1e9}
	max := linear.Vec2{-1e9, -1e9}
	hits := 0
	for _, ent := range g.temp.AllEnts {
		if ent.Side() != side {
			continue
		}
		if player, ok := ent.(*PlayerEnt); ok {
			hits++
			pos := player.Pos()
			if pos.X < min.X {
				min.X = pos.X
			}
			if pos.Y < min.Y {
				min.Y = pos.Y
			}
			if pos.X > max.X {
				max.X = pos.X
			}
			if pos.Y > max.Y {
				max.Y = pos.Y
			}
		}
	}
	if hits == 0 {
		min.X = 0
		min.Y = 0
		max.X = float64(g.Levels[GidInvadersStart].Room.Dx)
		max.Y = float64(g.Levels[GidInvadersStart].Room.Dy)
	} else {
		min.X -= stats.LosPlayerHorizon + 50
		min.Y -= stats.LosPlayerHorizon + 50
		if min.X < 0 {
			min.X = 0
		}
		if min.Y < 0 {
			min.Y = 0
		}
		max.X += stats.LosPlayerHorizon + 50
		max.Y += stats.LosPlayerHorizon + 50
		if max.X > float64(g.Levels[GidInvadersStart].Room.Dx) {
			max.X = float64(g.Levels[GidInvadersStart].Room.Dx)
		}
		if max.Y > float64(g.Levels[GidInvadersStart].Room.Dy) {
			max.Y = float64(g.Levels[GidInvadersStart].Room.Dy)
		}
	}

	mid := min.Add(max).Scale(0.5)
	dims := max.Sub(min)
	if dims.X/dims.Y < camera.regionDims.X/camera.regionDims.Y {
		dims.X = dims.Y * camera.regionDims.X / camera.regionDims.Y
	} else {
		dims.Y = dims.X * camera.regionDims.Y / camera.regionDims.X
	}
	camera.target.dims = dims
	camera.target.mid = mid

	if camera.current.mid.X == 0 && camera.current.mid.Y == 0 {
		// On the very first frame the current midpoint will be (0,0), which should
		// never happen after the game begins.  In this one case we'll immediately
		// set current to target so we don't start off by approaching it from the
		// origin.
		camera.current = camera.target
	} else {
		// speed is in (0, 1), the higher it is, the faster current approaches target.
		speed := 0.1
		camera.current.dims = camera.current.dims.Scale(1 - speed).Add(camera.target.dims.Scale(speed))
		camera.current.mid = camera.current.mid.Scale(1 - speed).Add(camera.target.mid.Scale(speed))
	}
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
		dataStr := fmt.Sprintf("Engine %d, Side %d, %s", id, g.Setup.Players[id].Side, g.Champs[g.Setup.Players[id].ChampIndex].Name)
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

func expandPoly(in linear.Poly, out *linear.Poly) {
	if len(*out) < len(in) {
		*out = make(linear.Poly, len(in))
	}
	for i := range *out {
		(*out)[i] = linear.Vec2{}
	}
	for i, v := range in {
		segi := in.Seg(i)
		(*out)[i] = (*out)[i].Add(v.Add(segi.Ray().Cross().Norm().Scale(8.0)))
		j := (i - 1 + len(in)) % len(in)
		segj := in.Seg(j)
		(*out)[i] = (*out)[i].Add(v.Add(segj.Ray().Cross().Norm().Scale(8.0)))
	}
	for i := range *out {
		(*out)[i] = (*out)[i].Scale(0.5)
	}
}

func (g *Game) RenderLosMask() {
	ent := g.Ents[g.local.Gid]
	if ent == nil {
		return
	}
	walls := g.temp.VisibleWallCache[GidInvadersStart].GetWalls(int(ent.Pos().X), int(ent.Pos().Y))
	gl.Disable(gl.TEXTURE_2D)
	gl.Color4ub(0, 0, 0, 255)
	gl.Begin(gl.TRIANGLES)
	for _, wall := range walls {
		if wall.Right(ent.Pos()) {
			continue
		}
		a := wall.P
		b := ent.Pos().Sub(wall.P).Norm().Scale(-10000.0).Add(wall.P)
		mid := wall.P.Add(wall.Q).Scale(0.5)
		c := ent.Pos().Sub(mid).Norm().Scale(-10000.0).Add(mid)
		d := ent.Pos().Sub(wall.Q).Norm().Scale(-10000.0).Add(wall.Q)
		e := wall.Q
		gl.Vertex2d(gl.Double(a.X), gl.Double(a.Y))
		gl.Vertex2d(gl.Double(b.X), gl.Double(b.Y))
		gl.Vertex2d(gl.Double(c.X), gl.Double(c.Y))

		gl.Vertex2d(gl.Double(a.X), gl.Double(a.Y))
		gl.Vertex2d(gl.Double(c.X), gl.Double(c.Y))
		gl.Vertex2d(gl.Double(d.X), gl.Double(d.Y))

		gl.Vertex2d(gl.Double(a.X), gl.Double(a.Y))
		gl.Vertex2d(gl.Double(d.X), gl.Double(d.Y))
		gl.Vertex2d(gl.Double(e.X), gl.Double(e.Y))
	}
	gl.End()
	base.EnableShader("horizon")
	base.SetUniformV2("horizon", "center", ent.Pos())
	base.SetUniformF("horizon", "horizon", LosMaxDist)
	gl.Begin(gl.QUADS)
	dx := gl.Int(g.Levels[GidInvadersStart].Room.Dx)
	dy := gl.Int(g.Levels[GidInvadersStart].Room.Dy)
	gl.Vertex2i(0, 0)
	gl.Vertex2i(dx, 0)
	gl.Vertex2i(dx, dy)
	gl.Vertex2i(0, dy)
	gl.End()
	base.EnableShader("")
}

func (g *Game) RenderLocalGame(region g2.Region) {
	// func (g *Game) renderLocalHelper(region g2.Region, local *LocalData, camera *cameraInfo, side int) {
	g.local.Camera.FocusRegion(g, 0)
	gl.MatrixMode(gl.PROJECTION)
	gl.PushMatrix()
	gl.LoadIdentity()
	// Set the viewport so that we only render into the region that we're supposed
	// to render to.
	// TODO: Check if this works on all graphics cards - I've heard that the opengl
	// spec doesn't actually require that viewport does any clipping.
	gl.PushAttrib(gl.VIEWPORT_BIT)
	gl.Viewport(gl.Int(region.X), gl.Int(region.Y), gl.Sizei(region.Dx), gl.Sizei(region.Dy))
	defer gl.PopAttrib()

	current := &g.local.Camera.current
	gl.Ortho(
		gl.Double(current.mid.X-current.dims.X/2),
		gl.Double(current.mid.X+current.dims.X/2),
		gl.Double(current.mid.Y+current.dims.Y/2),
		gl.Double(current.mid.Y-current.dims.Y/2),
		gl.Double(1000),
		gl.Double(-1000),
	)
	defer func() {
		gl.MatrixMode(gl.PROJECTION)
		gl.PopMatrix()
		gl.MatrixMode(gl.MODELVIEW)
	}()
	gl.MatrixMode(gl.MODELVIEW)

	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	level := g.Levels[GidInvadersStart]
	zoom := current.dims.X / float64(region.Dims.Dx)
	level.ManaSource.Draw(zoom, float64(level.Room.Dx), float64(level.Room.Dy))

	gl.Color4d(1, 1, 1, 1)
	var expandedPoly linear.Poly
	for _, poly := range g.Levels[GidInvadersStart].Room.Walls {
		// Don't draw counter-clockwise polys, specifically this means don't draw
		// the boundary of the level.
		if poly.IsCounterClockwise() {
			continue
		}
		// KLUDGE: This will expand the polygon slightly so that it actually shows
		// up when the los shadows are drawn over it.  Eventually there should be
		// separate los polys, colision polys, and draw polys so that this isn't
		// necessary.
		gl.Begin(gl.TRIANGLE_FAN)
		expandPoly(poly, &expandedPoly)
		for _, v := range expandedPoly {
			gl.Vertex2d(gl.Double(v.X), gl.Double(v.Y))
		}
		gl.End()
	}

	gui.SetFontColor(0, 255, 0, 255)
	for side, pos := range g.Levels[GidInvadersStart].Room.Starts {
		base.GetDictionary("luxisr").RenderString(fmt.Sprintf("S%d", side), pos.X, pos.Y, 0, 100, gui.Center)
	}

	gl.Color4d(1, 1, 1, 1)
	for _, ent := range g.temp.AllEnts {
		ent.Draw(g)
	}
	gl.Disable(gl.TEXTURE_2D)

	// TODO: figure out how to draw abilities.
	// for i := range local.moba.players {
	// 	p := &local.moba.players[i]
	// 	if p.abs.activeAbility != nil {
	// 		p.abs.activeAbility.Draw(p.gid, g, side)
	// 	}
	// }
	// for _, proc := range g.Processes {
	// 	proc.Draw(Gid(""), g, side)
	// }

	gl.Color4ub(0, 0, 255, 200)
	g.RenderLosMask()
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
	g.RenderLocalGame(region)
}
