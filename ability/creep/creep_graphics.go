// +build !nographics

package creep

import (
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/jota/base"
	"github.com/runningwild/jota/game"
	"github.com/runningwild/jota/texture"
)

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
