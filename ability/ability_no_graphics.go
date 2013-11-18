// +build nographics

package ability

import (
	"github.com/runningwild/jota/game"
)

func (p *cloakProc) Draw(src, obs game.Gid, game *game.Game)         {}
func (p *asplosionProc) Draw(src, obs game.Gid, game *game.Game)     {}
func (f *lightning) Draw(ent game.Ent, g *game.Game)                 {}
func (p *lightningBoltProc) Draw(src, obs game.Gid, game *game.Game) {}
func (p *multiDrain) Draw(src, obs game.Gid, game *game.Game)        {}
func (p *nitroProc) Draw(src, obs game.Gid, game *game.Game)         {}
func (p *pull) Draw(ent game.Ent, g *game.Game)                      {}
func (p *shieldProc) Draw(src, obs game.Gid, game *game.Game)        {}
