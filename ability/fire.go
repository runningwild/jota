package ability

import (
	"encoding/gob"
	"github.com/runningwild/jota/game"
	"github.com/runningwild/jota/stats"
	"github.com/runningwild/linear"
	"math"
	"math/rand"
)

type fireRegion int

const (
	fireRegionFront fireRegion = iota
	fireRegionFlank
	fireRegionBack
)

func makeFire(params map[string]float64) game.Ability {
	var f fire
	f.id = NextAbilityId()
	switch params["region"] {
	case 1:
		f.region = fireRegionFront
	case 2:
		f.region = fireRegionFlank
	case 3:
		f.region = fireRegionBack
	default:
		panic("Unexpected value for 'region' parameter of fire ability")
	}
	f.distToCenter = params["distToCenter"]
	f.deviance = params["deviance"]
	f.startRadius = params["startRadius"]
	f.endRadius = params["endRadius"]
	f.durationThinks = int(params["durationThinks"])
	f.dps = params["dps"]
	f.xps = params["xps"]
	f.cost = params["cost"]
	return &f
}

func init() {
	game.RegisterAbility("fire", makeFire)
	gob.Register(&fire{})
}

type fire struct {
	id             int
	region         fireRegion
	cost           float64
	distToCenter   float64
	deviance       float64
	startRadius    float64
	endRadius      float64
	durationThinks int
	dps            float64
	xps            float64
	draw           bool
	active         bool
	trigger        bool
	draining       bool

	// Used to carry over a fraction of an explosion from one frame to the next
	// so that we can accurately hit the xps (explisions per second).
	xFrac float64
}

func (f *fire) Input(ent game.Ent, g *game.Game, pressAmt float64, trigger bool) {
	f.draw = pressAmt > 0
	if !trigger || pressAmt == 0.0 {
		f.trigger = false
		f.draining = false
	}
	if !f.trigger {
		f.trigger = trigger
		f.draining = true
		player := ent.(*game.PlayerEnt)
		if pressAmt == 0 {
			delete(player.Processes, f.id)
			return
		}
		_, ok := player.Processes[f.id].(*multiDrain)
		if !ok {
			player.Processes[f.id] = &multiDrain{Gid: player.Gid, Unit: game.Mana{f.cost, 0, 0}}
			return
		}
	}
}

func (f *fire) getFrontPos(ent game.Ent, g *game.Game) linear.Vec2 {
	r := rand.New(g.Rng)
	theta := r.Float64() * math.Pi * 2
	dist := math.Abs(r.NormFloat64() * f.deviance)
	if dist > f.deviance*4 {
		dist = f.deviance * 4
	}
	dist = dist + dist*math.Cos(theta)
	center := (linear.Vec2{f.distToCenter, 0}).Rotate(ent.Angle()).Add(ent.Pos())
	return (linear.Vec2{0, dist}).Rotate(ent.Angle() - math.Pi/2 + theta).Add(center)
}

func (f *fire) getFlankPos(ent game.Ent, g *game.Game) linear.Vec2 {
	r := rand.New(g.Rng)
	theta := r.Float64() * math.Pi * 2
	dist := math.Abs(r.NormFloat64() * f.deviance)
	if dist > f.deviance*4 {
		dist = f.deviance * 4
	}
	dist = dist + dist*math.Cos(theta)
	center := (linear.Vec2{f.distToCenter, 0}).Rotate(ent.Angle()).Add(ent.Pos())
	return (linear.Vec2{0, dist}).Rotate(ent.Angle() - math.Pi/2 + theta).Add(center)
}

func (f *fire) getBackPos(ent game.Ent, g *game.Game) linear.Vec2 {
	r := rand.New(g.Rng)
	theta := r.Float64() * math.Pi * 2
	dist := math.Abs(r.NormFloat64() * f.deviance)
	if dist > f.deviance*4 {
		dist = f.deviance * 4
	}
	dist = dist + dist*math.Cos(theta)
	center := (linear.Vec2{f.distToCenter, 0}).Rotate(ent.Angle() + math.Pi).Add(ent.Pos())
	return (linear.Vec2{0, dist}).Rotate(ent.Angle() + math.Pi/2 + theta).Add(center)
}

func (f *fire) getPos(ent game.Ent, g *game.Game) linear.Vec2 {
	switch f.region {
	case fireRegionFront:
		return f.getFrontPos(ent, g)
	case fireRegionFlank:
		return f.getFlankPos(ent, g)
	case fireRegionBack:
		return f.getBackPos(ent, g)
	default:
		return linear.Vec2{}
	}
}

func (f *fire) Think(ent game.Ent, g *game.Game) {
	player := ent.(*game.PlayerEnt)
	proc, ok := player.Processes[f.id].(*multiDrain)
	if !ok {
		return
	}
	if f.trigger && f.draining && proc.Stored > 1 {
		proc.Stored -= 0.1

		// TODO: This is assuming 60fps - maybe that should be checked somewhere?
		for f.xFrac += f.xps / 60.0; f.xFrac > 0.0; f.xFrac-- {
			g.Processes = append(g.Processes, &asplosionProc{
				StartRadius:    f.startRadius,
				EndRadius:      f.endRadius,
				DurationThinks: f.durationThinks,
				Dps:            f.dps,
				Pos:            f.getPos(ent, g),
			})
		}

		if proc.Stored <= 1.0 {
			f.draining = false
			f.xFrac = 0
		}
	}
}
func (f *fire) Draw(ent game.Ent, g *game.Game) {
}
func (f *fire) IsActive() bool {
	return false
}

// COPY PASTED - SHARE WITH RED.GO
type asplosionProc struct {
	NullCondition
	DurationThinks int
	NumThinks      int
	StartRadius    float64
	EndRadius      float64
	CurrentRadius  float64
	Dps            float64
	Pos            linear.Vec2
	Killed         bool
}

func (p *asplosionProc) Supply(mana game.Mana) game.Mana {
	return game.Mana{}
}
func (p *asplosionProc) Think(g *game.Game) {
	p.NumThinks++
	p.CurrentRadius = float64(p.NumThinks)/float64(p.DurationThinks)*(p.EndRadius-p.StartRadius) + p.StartRadius
	for _, ent := range g.Ents {
		if ent.Pos().Sub(p.Pos).Mag2() <= p.CurrentRadius*p.CurrentRadius {
			ent.Stats().ApplyDamage(stats.Damage{stats.DamageFire, p.Dps})
		}
	}
}
func (p *asplosionProc) Kill(g *game.Game) {
	p.Killed = true
}
func (p *asplosionProc) Dead() bool {
	return p.Killed || p.NumThinks > p.DurationThinks
}
