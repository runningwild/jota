package ability

import (
	"encoding/gob"
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/cgf"
	"github.com/runningwild/cmwc"
	"github.com/runningwild/glop/util/algorithm"
	"github.com/runningwild/linear"
	"github.com/runningwild/magnus/base"
	"github.com/runningwild/magnus/game"
	"github.com/runningwild/magnus/stats"
	"github.com/runningwild/magnus/texture"
	// "math"
	"math/rand"
)

func makeFire(params map[string]int) game.Ability {
	var f fire
	f.id = NextAbilityId()
	return &f
}

func init() {
	game.RegisterAbility("fire", makeFire)
}

type fire struct {
	NonResponder
	NonThinker
	NonRendering

	id int
}

func (f *fire) Activate(gid game.Gid, keyPress bool) ([]cgf.Event, bool) {
	var ret []cgf.Event
	if keyPress {
		ret = append(ret, addFireEvent{
			PlayerGid: gid,
			Id:        f.id,
		})
	} else {
		ret = append(ret, addFireExplodeEvent{
			PlayerGid: gid,
			Id:        f.id,
		})
	}
	return ret, false
}

func (f *fire) Deactivate(gid game.Gid) []cgf.Event {
	return nil
}

type addFireEvent struct {
	PlayerGid game.Gid
	Id        int
}

func init() {
	gob.Register(addFireEvent{})
}

func (e addFireEvent) Apply(_g interface{}) {
	g := _g.(*game.Game)
	player, ok := g.Ents[e.PlayerGid].(*game.Player)
	if !ok {
		return
	}
	pos := player.Position.Add((linear.Vec2{40, 0}).Rotate(player.Angle))
	player.Processes[100+e.Id] = &fireProcess{
		BasicPhases: BasicPhases{game.PhaseRunning},
		Gid:         player.Gid,
		Frames:      100,
		Pos:         pos,
		Inner:       50,
		Outer:       100,
		Angle:       0.1,
		Heading:     float32(player.Angle),
	}
}

type fireExplosion struct {
	Pos    linear.Vec2
	Radius float64
	Timer  int
	Start  int
	Peak   int
	End    int
}

func (fe fireExplosion) Draw(test bool) {
	base.EnableShader("circle")
	base.SetUniformF("circle", "edge", 0.7)
	if test {
		gl.Color4ub(200, 200, 200, gl.Ubyte(150*fe.Alpha()))
	} else {
		gl.Color4ub(255, 50, 10, gl.Ubyte(150*fe.Alpha()))
	}
	texture.Render(
		fe.Pos.X-fe.Size(),
		fe.Pos.Y-fe.Size(),
		2*fe.Size(),
		2*fe.Size())
	base.EnableShader("")
}

func (f *fireExplosion) Think() {
	f.Timer++
}

func (f *fireExplosion) Done() bool {
	return f.Timer > f.End
}

func (fe fireExplosion) Size() float64 {
	if fe.Timer < fe.Start || fe.Timer > fe.End {
		return 0.0
	}
	if fe.Timer > fe.Peak {
		return fe.Radius
	}
	return fe.Radius * (0.5 + 0.5*float64(fe.Timer-fe.Start)/float64(fe.Peak-fe.Start))
}

func (fe fireExplosion) Alpha() float64 {
	if fe.Timer < fe.Peak {
		return 1.0
	}
	if fe.Timer > fe.End {
		return 0.0
	}
	return 1.0 - float64(fe.Timer-fe.Peak)/float64(fe.End-fe.Peak)/2.0
}

type fireProcess struct {
	BasicPhases
	NullCondition
	Gid     game.Gid
	Frames  int32
	Pos     linear.Vec2
	Inner   float32
	Outer   float32
	Angle   float32
	Heading float32
	Stored  float64

	// Sample projectiles - just for display, the real ones are calculated when
	// the process is actually complete.
	explosions []fireExplosion
	thinks     int
	rng        *cmwc.Cmwc
}

func (f *fireProcess) Supply(supply game.Mana) game.Mana {
	f.Stored += supply[game.ColorRed]
	supply[game.ColorRed] = 0
	return supply
}

func (f *fireProcess) Think(g *game.Game) {
	player, ok := g.Ents[f.Gid].(*game.Player)
	if !ok {
		return
	}
	f.Pos = player.Position
	f.Stored *= 0.97
	if f.rng == nil {
		f.rng = cmwc.MakeGoodCmwc()
		f.rng.SeedWithDevRand()
	}
	max := int(f.Stored / 15)
	algorithm.Choose(&f.explosions, func(e fireExplosion) bool { return !e.Done() })
	if len(f.explosions) < max {
		f.explosions = append(f.explosions, fireDoLine(f.rng, player.Position, player.Angle, f.Stored, 3, g.Levels[player.CurrentLevel]))
	}
	for i := range f.explosions {
		f.explosions[i].Think()
	}
	f.thinks++
}

func fireDoLine(c *cmwc.Cmwc, pos linear.Vec2, angle, stored float64, speed int, level *game.Level) fireExplosion {
	rng := rand.New(c)
	ray := (linear.Vec2{1, 0})
	// ray.Scale(math.Abs(rng.NormFloat64()/10) + 50)
	scale := (stored/5 + 50) * (1 + rng.Float64()*(0.2+stored/2000))
	ray = ray.Rotate(angle).Rotate(rng.NormFloat64() * (0.2 + stored/7500)).Scale(scale)
	seg := linear.Seg2{pos, pos.Add(ray)}
	base.DoOrdered(level.Room.Walls, func(a, b string) bool { return a < b }, func(_ string, poly linear.Poly) {
		for i := range poly {
			if seg.DoesIsect(poly.Seg(i)) {
				isect := seg.Isect(poly.Seg(i))
				seg.Q = isect
			}
		}
	})
	p1 := rng.Intn(speed)
	p2 := rng.Intn(speed)
	p3 := rng.Intn(speed)
	return fireExplosion{
		Pos:    seg.Q,
		Radius: rng.Float64()*40 + 30,
		Timer:  0,
		Start:  1*speed + p1,
		Peak:   4*speed + p1 + p2,
		End:    5*speed + p1 + p2 + p3,
	}
}

func (f *fireProcess) Draw(gid game.Gid, g *game.Game, side int) {
	player, ok := g.Ents[f.Gid].(*game.Player)
	if !ok {
		return
	}
	if side != player.Side() {
		return
	}
	for _, expl := range f.explosions {
		expl.Draw(true)
	}
}

type addFireExplodeEvent struct {
	PlayerGid game.Gid
	Id        int
}

func init() {
	gob.Register(addFireExplodeEvent{})
}

func (e addFireExplodeEvent) Apply(_g interface{}) {
	g := _g.(*game.Game)
	player, ok := g.Ents[e.PlayerGid].(*game.Player)
	if !ok {
		return
	}
	prevProc := player.Processes[100+e.Id].(*fireProcess)
	var fpe fireProcessExplosion
	fpe.The_phase = game.PhaseRunning
	delete(player.Processes, 100+e.Id)
	if int(prevProc.Stored/10) == 0 {
		return
	}
	num := int(g.Rng.Int63()%int64(prevProc.Stored/10)) + int(prevProc.Stored/10)
	for i := 0; i < num; i++ {
		fpe.Explosions = append(fpe.Explosions,
			fireDoLine(g.Rng, player.Position, player.Angle, prevProc.Stored, 10, g.Levels[player.CurrentLevel]))
	}
	g.Processes = append(g.Processes, &fpe)
}

type fireProcessExplosion struct {
	BasicPhases
	NullCondition
	Explosions []fireExplosion
}

func (f *fireProcessExplosion) Supply(supply game.Mana) game.Mana {
	return supply
}

func (f *fireProcessExplosion) Think(g *game.Game) {
	g.DoForEnts(func(gid game.Gid, ent game.Ent) {
		for _, expl := range f.Explosions {
			if expl.Size() == 0 {
				continue
			}
			if expl.Pos.Sub(ent.Pos()).Mag() <= expl.Size() {
				ent.Stats().ApplyDamage(stats.Damage{stats.DamageFire, 1})
			}
		}
	})
	done := true
	for i := range f.Explosions {
		f.Explosions[i].Think()
		if !f.Explosions[i].Done() {
			done = false
		}
	}
	if done {
		f.The_phase = game.PhaseComplete
	}
}

func (f *fireProcessExplosion) Draw(gid game.Gid, g *game.Game, side int) {
	base.EnableShader("circle")
	base.SetUniformF("circle", "edge", 0.7)
	for _, expl := range f.Explosions {
		expl.Draw(false)
	}
	base.EnableShader("")
}
