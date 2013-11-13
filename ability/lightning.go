package ability

import (
	"encoding/gob"
	gl "github.com/chsc/gogl/gl21"
	// "github.com/runningwild/jota/base"
	"github.com/runningwild/jota/game"
	"github.com/runningwild/jota/stats"
	// "github.com/runningwild/jota/texture"
	"github.com/runningwild/linear"
	"math"
	// "math/rand"
)

func makeLightning(params map[string]int) game.Ability {
	var l lightning
	l.id = NextAbilityId()
	l.cost = float64(params["cost"])
	l.width = float64(params["width"])
	l.buildThinks = params["buildThinks"]
	l.durationThinks = params["durationThinks"]
	l.dps = float64(params["dps"])
	return &l
}

func init() {
	game.RegisterAbility("lightning", makeLightning)
	gob.Register(&lightning{})
}

type lightning struct {
	id int

	// Params
	cost           float64
	width          float64
	buildThinks    int
	durationThinks int
	dps            float64

	draw    bool
	trigger bool
}

func (l *lightning) Input(ent game.Ent, g *game.Game, pressAmt float64, trigger bool) {
	l.draw = pressAmt > 0
	if !trigger || pressAmt == 0.0 {
		l.trigger = false
	}
	if !l.trigger {
		l.trigger = trigger
		player := ent.(*game.PlayerEnt)
		if pressAmt == 0 {
			delete(player.Processes, l.id)
			return
		}
		_, ok := player.Processes[l.id].(*multiDrain)
		if !ok {
			player.Processes[l.id] = &multiDrain{Gid: player.Gid, Unit: game.Mana{0, l.cost, 0}}
			return
		}
	}
}

func (l *lightning) Think(ent game.Ent, g *game.Game) {
	player := ent.(*game.PlayerEnt)
	proc, ok := player.Processes[l.id].(*multiDrain)
	if !ok {
		return
	}
	if l.trigger && proc.Stored > 1 {
		delete(player.Processes, l.id)
		// find the endpoits of the lightning
		forward := (linear.Vec2{1, 0}).Rotate(player.Angle()).Scale(10000)
		bounds := [2]linear.Seg2{
			linear.Seg2{
				player.Pos(),
				player.Pos().Add(forward),
			},
			linear.Seg2{
				player.Pos(),
				player.Pos().Sub(forward),
			},
		}
		mag2s := [2]float64{-1.0, -1.0}
		var isects [2]linear.Vec2
		isects[0] = bounds[0].Q
		isects[1] = bounds[1].Q
		for _, wall := range g.Level.Room.Walls {
			for i := range wall {
				seg := wall.Seg(i)
				for j := range bounds {
					if bounds[j].DoesIsect(seg) {
						isect := bounds[j].Isect(seg)
						isectMag2 := isect.Sub(player.Pos()).Mag2()
						if isectMag2 < mag2s[j] || mag2s[j] == -1 {
							mag2s[j] = isectMag2
							isects[j] = isect
						}
					}
				}
			}
		}
		g.Processes = append(g.Processes, &lightningBoltProc{
			BuildThinks:    l.buildThinks,
			DurationThinks: l.durationThinks,
			Width:          l.width * math.Sqrt(proc.Stored),
			Dps:            l.dps,
			Power:          proc.Stored,
			Seg:            linear.Seg2{isects[0], isects[1]},
		})
	}
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
func (f *lightning) IsActive() bool {
	return false
}

type lightningBoltProc struct {
	NullCondition
	BuildThinks    int
	DurationThinks int
	NumThinks      int
	Width          float64
	Dps            float64
	Power          float64
	Seg            linear.Seg2
	Killed         bool
}

func (p *lightningBoltProc) Draw(src, obs game.Gid, game *game.Game) {
	if p.NumThinks < p.BuildThinks {
		return
	}
	gl.Disable(gl.TEXTURE_2D)
	gl.Color4ub(255, 255, 255, 255)
	perp := p.Seg.Ray().Cross().Norm().Scale(p.Width / 2)
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
func (p *lightningBoltProc) Supply(mana game.Mana) game.Mana {
	return game.Mana{}
}
func (p *lightningBoltProc) Think(g *game.Game) {
	p.NumThinks++
	if p.NumThinks < p.BuildThinks {
		return
	}
	perp := p.Seg.Ray().Cross().Norm().Scale(p.Width / 2)
	for _, ent := range g.Ents {
		entSeg := linear.Seg2{
			ent.Pos().Sub(perp),
			ent.Pos().Add(perp),
		}
		if entSeg.DoesIsect(p.Seg) {
			ent.Stats().ApplyDamage(stats.Damage{stats.DamageFire, p.Dps * p.Power})
		}
	}
	// for _, ent := range g.Ents {
	// 	if ent.Pos().Sub(p.Pos).Mag2() <= p.CurrentRadius*p.CurrentRadius {
	// 		ent.Stats().ApplyDamage(stats.Damage{stats.DamageFire, p.Dps})
	// 	}
	// }
}
func (p *lightningBoltProc) Kill(g *game.Game) {
	p.Killed = true
}
func (p *lightningBoltProc) Dead() bool {
	return p.Killed || p.NumThinks > (p.BuildThinks+p.DurationThinks)
}
