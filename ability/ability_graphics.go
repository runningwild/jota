// +build !nographics

package ability

import (
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/jota/base"
	"github.com/runningwild/jota/game"
	"github.com/runningwild/jota/texture"
	"github.com/runningwild/linear"
	"math"
)

func (p *cloakProc) Draw(src, obs game.Gid, game *game.Game) {
	if src != obs {
		return
	}
	ent := game.Ents[src]
	if ent == nil {
		return
	}
	gl.Color4ub(0, 0, 255, 255)
	frac := float32(p.Cloak / p.MaxCloak)
	base.EnableShader("status_bar")
	base.SetUniformF("status_bar", "frac", frac)
	base.SetUniformF("status_bar", "inner", 0.2)
	base.SetUniformF("status_bar", "outer", 0.23)
	base.SetUniformF("status_bar", "buffer", 0.01)
	texture.Render(ent.Pos().X-100, ent.Pos().Y-100, 200, 200)
	base.EnableShader("")
}
func (p *asplosionProc) Draw(src, obs game.Gid, game *game.Game) {
	base.EnableShader("circle")
	base.SetUniformF("circle", "edge", 0.7)
	gl.Color4ub(255, 50, 10, gl.Ubyte(150))
	texture.Render(
		p.Pos.X-p.CurrentRadius,
		p.Pos.Y-p.CurrentRadius,
		2*p.CurrentRadius,
		2*p.CurrentRadius)
	base.EnableShader("")
}
func (f *lightning) Draw(ent game.Ent, g *game.Game) {
	if !f.draw {
		return
	}
	gl.Disable(gl.TEXTURE_2D)
	gl.Color4ub(255, 255, 255, 255)
	forward := (linear.Vec2{1, 0}).Rotate(ent.Angle()).Scale(100000.0)
	gl.Begin(gl.LINES)
	v := ent.Pos().Add(forward)
	gl.Vertex2d(gl.Double(v.X), gl.Double(v.Y))
	v = ent.Pos().Sub(forward)
	gl.Vertex2d(gl.Double(v.X), gl.Double(v.Y))
	gl.End()
}
func (p *lightningBoltProc) Draw(src, obs game.Gid, game *game.Game) {
	if p.NumThinks < p.BuildThinks {
		return
	}
	base.EnableShader("lightning")
	base.SetUniformV2("lightning", "dir", p.Seg.Ray().Norm())
	base.SetUniformV2("lightning", "bolt_root", p.Seg.P.Add(p.Seg.Q).Scale(0.5))

	base.SetUniformF("lightning", "bolt_thickness", 1.1)
	gl.Disable(gl.TEXTURE_2D)
	displayWidth := p.Width * 10
	perp := p.Seg.Ray().Cross().Norm().Scale(displayWidth / 2)
	move := float32(p.NumThinks) / float32(60) / 10.0
	for i := 0; i < 3; i++ {
		base.SetUniformF("lightning", "rand_offset", float32(i)+move)
		if i == 2 {
			base.SetUniformF("lightning", "bolt_thickness", 1.3)
		}
		switch i {
		case 0:
			gl.Color4ub(255, 200, 200, 200)
		case 1:
			gl.Color4ub(255, 255, 200, 200)
		case 2:
			gl.Color4ub(255, 255, 230, 225)
		}
		gl.Begin(gl.QUADS)
		v := p.Seg.P.Add(perp)
		gl.Vertex2d(gl.Double(v.X), gl.Double(v.Y))
		v = p.Seg.Q.Add(perp)
		gl.Vertex2d(gl.Double(v.X), gl.Double(v.Y))
		v = p.Seg.Q.Sub(perp)
		gl.Vertex2d(gl.Double(v.X), gl.Double(v.Y))
		v = p.Seg.P.Sub(perp)
		gl.Vertex2d(gl.Double(v.X), gl.Double(v.Y))
		gl.End()
	}
	base.EnableShader("")
}
func (p *multiDrain) Draw(src, obs game.Gid, game *game.Game) {
	if src != obs {
		return
	}
	ent := game.Ents[src]
	if ent == nil {
		return
	}
	base.EnableShader("status_bar")
	frac := p.Stored
	ready := math.Floor(frac)
	if ready == 0 {
		gl.Color4ub(255, 0, 0, 255)
	} else {
		gl.Color4ub(0, 255, 0, 255)
	}
	outer := 0.2
	increase := 0.01
	base.SetUniformF("status_bar", "frac", float32(frac-ready))
	base.SetUniformF("status_bar", "inner", float32(outer-increase*(ready+1)))
	base.SetUniformF("status_bar", "outer", float32(outer))
	base.SetUniformF("status_bar", "buffer", 0.01)
	texture.Render(ent.Pos().X-100, ent.Pos().Y-100, 200, 200)
	if ready > 0 {
		base.SetUniformF("status_bar", "frac", 1.0)
		base.SetUniformF("status_bar", "inner", float32(outer-ready*increase))
		base.SetUniformF("status_bar", "outer", float32(outer))
		texture.Render(ent.Pos().X-100, ent.Pos().Y-100, 200, 200)
	}
	base.EnableShader("")
}
func (p *nitroProc) Draw(src, obs game.Gid, game *game.Game) {
	if src != obs {
		return
	}
	ent := game.Ents[src]
	if ent == nil {
		return
	}
	gl.Color4ub(255, 0, 0, 255)
	frac := float32(p.Nitro / p.MaxNitro)
	base.EnableShader("status_bar")
	base.SetUniformF("status_bar", "frac", frac)
	base.SetUniformF("status_bar", "inner", 0.2)
	base.SetUniformF("status_bar", "outer", 0.23)
	base.SetUniformF("status_bar", "buffer", 0.01)
	texture.Render(ent.Pos().X-100, ent.Pos().Y-100, 200, 200)
	base.EnableShader("")
}
func (p *pull) Draw(ent game.Ent, g *game.Game) {
	if !p.draw {
		return
	}
	player, ok := ent.(*game.PlayerEnt)
	if !ok {
		return
	}
	// TODO: Don't draw for enemies?
	gl.Color4d(1, 1, 1, 1)
	gl.Disable(gl.TEXTURE_2D)
	v1 := player.Pos()
	v2 := v1.Add(linear.Vec2{1000, 0})
	v3 := v2.RotateAround(v1, player.Angle()-p.angle/2)
	v4 := v2.RotateAround(v1, player.Angle()+p.angle/2)
	gl.Begin(gl.LINES)
	vs := []linear.Vec2{v3, v4, player.Pos()}
	for i := range vs {
		gl.Vertex2d(gl.Double(vs[i].X), gl.Double(vs[i].Y))
		gl.Vertex2d(gl.Double(vs[(i+1)%len(vs)].X), gl.Double(vs[(i+1)%len(vs)].Y))
	}
	gl.End()
}
func (p *shieldProc) Draw(src, obs game.Gid, game *game.Game) {
	if src != obs {
		return
	}
	ent := game.Ents[src]
	if ent == nil {
		return
	}
	gl.Color4ub(0, 255, 0, 255)
	frac := float32(p.Shield / p.MaxShield)
	base.EnableShader("status_bar")
	base.SetUniformF("status_bar", "frac", frac)
	base.SetUniformF("status_bar", "inner", 0.2)
	base.SetUniformF("status_bar", "outer", 0.23)
	base.SetUniformF("status_bar", "buffer", 0.01)
	texture.Render(ent.Pos().X-100, ent.Pos().Y-100, 200, 200)
	base.EnableShader("")
}
