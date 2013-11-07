// +build !nographics

package game

import (
	"fmt"
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/jota/base"
	g2 "github.com/runningwild/jota/gui"
	"github.com/runningwild/jota/stats"
	"github.com/runningwild/jota/texture"
	"github.com/runningwild/linear"
	"path/filepath"
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
	player := g.Ents[g.local.Gid]
	if player == nil {
		min.X = 0
		min.Y = 0
		max.X = float64(g.Level.Room.Dx)
		max.Y = float64(g.Level.Room.Dy)
	} else {
		min.X = player.Pos().X - (stats.LosPlayerHorizon + 50)
		min.Y = player.Pos().Y - (stats.LosPlayerHorizon + 50)
		if min.X < 0 {
			min.X = 0
		}
		if min.Y < 0 {
			min.Y = 0
		}
		max.X = player.Pos().X + (stats.LosPlayerHorizon + 50)
		max.Y = player.Pos().Y + (stats.LosPlayerHorizon + 50)
		if max.X > float64(g.Level.Room.Dx) {
			max.X = float64(g.Level.Room.Dx)
		}
		if max.Y > float64(g.Level.Room.Dy) {
			max.Y = float64(g.Level.Room.Dy)
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
	g.Setup.local.RLock()
	defer g.Setup.local.RUnlock()
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
		if g.local.Engine.Id() == 1 && i == g.Setup.local.Index {
			dict.RenderString(">", 50, y, 0, size, gui.Right)
		}
	}
	y += size
	gui.SetFontColor(0.7, 0.7, 0.7, 1)
	if g.local.Engine.Id() == 1 {
		dict.RenderString("Start!", size, y, 0, size, gui.Left)
		if g.Setup.local.Index == len(g.Setup.EngineIds) {
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
	walls := g.temp.VisibleWallCache.GetWalls(int(ent.Pos().X), int(ent.Pos().Y))
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
	dx := gl.Int(g.Level.Room.Dx)
	dy := gl.Int(g.Level.Room.Dy)
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

	level := g.Level
	zoom := current.dims.X / float64(region.Dims.Dx)
	level.ManaSource.Draw(zoom, float64(level.Room.Dx), float64(level.Room.Dy))

	gl.Color4d(1, 1, 1, 1)
	var expandedPoly linear.Poly
	for _, poly := range g.Level.Room.Walls {
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
	for side, data := range g.Level.Room.SideData {
		base.GetDictionary("luxisr").RenderString(fmt.Sprintf("S%d", side), data.Base.X, data.Base.Y, 0, 100, gui.Center)
	}

	gl.Color4d(1, 1, 1, 1)
	for _, ent := range g.temp.AllEnts {
		ent.Draw(g)
	}
	gl.Disable(gl.TEXTURE_2D)

	// TODO: figure out how to draw abilities.
	// for gid, ent:=range g.Ents {
	// 	for _, proc := range ent.
	// }
	// for i := range local.moba.players {
	// 	p := &local.moba.players[i]
	// 	if p.abs.activeAbility != nil {
	// 		p.abs.activeAbility.Draw(p.gid, g, side)
	// 	}
	// }
	for _, proc := range g.Processes {
		proc.Draw(Gid(""), g.local.Gid, g)
	}

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

func (p *PlayerEnt) Draw(game *Game) {
	var t *texture.Data
	var alpha gl.Ubyte
	if game.local.Side == p.Side() {
		alpha = gl.Ubyte(255.0 * (1.0 - p.Stats().Cloaking()/2))
	} else {
		alpha = gl.Ubyte(255.0 * (1.0 - p.Stats().Cloaking()))
	}
	gl.Color4ub(255, 255, 255, alpha)
	// if p.Id() == 1 {
	t = texture.LoadFromPath(filepath.Join(base.GetDataDir(), "ships/ship.png"))
	// } else if p.Id() == 2 {
	// 	t = texture.LoadFromPath(filepath.Join(base.GetDataDir(), "ships/ship3.png"))
	// } else {
	// 	t = texture.LoadFromPath(filepath.Join(base.GetDataDir(), "ships/ship2.png"))
	// }
	t.RenderAdvanced(
		p.Position.X-float64(t.Dx())/2,
		p.Position.Y-float64(t.Dy())/2,
		float64(t.Dx()),
		float64(t.Dy()),
		p.Angle_,
		false)

	for _, proc := range p.Processes {
		proc.Draw(p.Id(), game.local.Gid, game)
	}
	base.EnableShader("status_bar")
	base.SetUniformF("status_bar", "inner", 0.08)
	base.SetUniformF("status_bar", "outer", 0.09)
	base.SetUniformF("status_bar", "buffer", 0.01)

	base.SetUniformF("status_bar", "frac", 1.0)
	gl.Color4ub(125, 125, 125, alpha/2)
	texture.Render(p.Position.X-100, p.Position.Y-100, 200, 200)

	health_frac := float32(p.Stats().HealthCur() / p.Stats().HealthMax())
	if health_frac > 0.5 {
		color_frac := 1.0 - (health_frac-0.5)*2.0
		gl.Color4ub(gl.Ubyte(255.0*color_frac), 255, 0, alpha)
	} else {
		color_frac := health_frac * 2.0
		gl.Color4ub(255, gl.Ubyte(255.0*color_frac), 0, alpha)
	}
	base.SetUniformF("status_bar", "frac", health_frac)
	texture.Render(p.Position.X-100, p.Position.Y-100, 200, 200)
	base.EnableShader("")
}

func (gw *GameWindow) Draw(region g2.Region, style g2.StyleStack) {
	defer base.StackCatcher()
	defer func() {
		// gl.Translated(gl.Double(gw.region.X), gl.Double(gw.region.Y), 0)
		gl.Disable(gl.TEXTURE_2D)
		gl.Color4ub(255, 255, 255, 255)
		gl.LineWidth(3)
		gl.Begin(gl.LINES)
		bx, by := gl.Int(region.X), gl.Int(region.Y)
		bdx, bdy := gl.Int(region.Dx), gl.Int(region.Dy)
		gl.Vertex2i(bx, by)
		gl.Vertex2i(bx, by+bdy)
		gl.Vertex2i(bx, by+bdy)
		gl.Vertex2i(bx+bdx, by+bdy)
		gl.Vertex2i(bx+bdx, by+bdy)
		gl.Vertex2i(bx+bdx, by)
		gl.Vertex2i(bx+bdx, by)
		gl.Vertex2i(bx, by)
		gl.End()
		gl.LineWidth(1)
	}()

	gw.Engine.Pause()
	game := gw.Engine.GetState().(*Game)
	// Note that since we do a READER lock on game.local we cannot do any writes
	// to local data while rendering.
	game.local.RLock()
	game.RenderLocal(region)
	game.local.RUnlock()
	gw.Engine.Unpause()
}
