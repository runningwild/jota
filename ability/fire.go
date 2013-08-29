package ability

import (
	"encoding/gob"
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/cgf"
	"github.com/runningwild/cmwc"
	"github.com/runningwild/linear"
	"github.com/runningwild/magnus/base"
	"github.com/runningwild/magnus/game"
	"github.com/runningwild/magnus/texture"
	"math"
	"math/rand"
)

func makeFire(params map[string]int) game.Ability {
	var f fire
	f.id = nextAbilityId()
	return &f
}

func init() {
	game.RegisterAbility("fire", makeFire)
}

type fire struct {
	nonResponder
	nonThinker
	nonRendering

	id int
}

func (f *fire) Activate(gid game.Gid, keyPress bool) ([]cgf.Event, bool) {
	if !keyPress {
		return nil, false
	}
	ret := []cgf.Event{
		addFireEvent{
			PlayerGid: gid,
			Id:        f.id,
		},
	}
	base.Log().Printf("Activate")
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
	player := g.Ents[e.PlayerGid].(*game.Player)
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
	proj   []linear.Seg2
	thinks int
	rng    *cmwc.Cmwc
}

func (f *fireProcess) Supply(supply game.Mana) game.Mana {
	f.Stored += supply[game.ColorRed]
	supply[game.ColorRed] = 0
	return supply
}

func (f *fireProcess) Think(g *game.Game) {
	player := g.Ents[f.Gid].(*game.Player)
	f.Pos = player.Position
	f.Stored *= 0.97
	if f.rng == nil {
		f.rng = cmwc.MakeGoodCmwc()
		f.rng.SeedWithDevRand()
	}
	max := 10
	if len(f.proj) > 0 && f.thinks%3 == 0 {
		f.proj = f.proj[1:]
	}
	for len(f.proj) < max {
		f.proj = append(f.proj, fireDoLine(f.rng, player.Position, player.Angle, f.Stored, g.Levels[player.CurrentLevel]))
	}
	f.thinks++
}

func fireDoLine(c *cmwc.Cmwc, pos linear.Vec2, angle float64, stored float64, level *game.Level) linear.Seg2 {
	rng := rand.New(c)
	ray := (linear.Vec2{1, 0})
	ray.Scale(math.Abs(rng.NormFloat64() / 2))
	ray = ray.Rotate(angle).Rotate(rng.NormFloat64() / 5).Scale(stored)
	seg := linear.Seg2{pos, pos.Add(ray)}
	base.DoOrdered(level.Room.Walls, func(a, b string) bool { return a < b }, func(_ string, poly linear.Poly) {
		for i := range poly {
			if seg.DoesIsect(poly.Seg(i)) {
				isect := seg.Isect(poly.Seg(i))
				seg.Q = isect
			}
		}
	})
	return seg
}

func (f *fireProcess) Draw(gid game.Gid, g *game.Game) {
	gl.Disable(gl.TEXTURE_2D)
	gl.Begin(gl.TRIANGLES)
	for i, line := range f.proj {
		angle := 0.02
		v1 := line.Ray().Rotate(-angle).Add(f.Pos)
		v2 := line.Ray().Rotate(angle).Add(f.Pos)
		gl.Color4d(0, 0, 0, 0)
		gl.Vertex2i(gl.Int(line.P.X), gl.Int(line.P.Y))
		gl.Color4d(0.7, 0.7, 0.7, gl.Double(float64(i)/float64(len(f.proj))))
		gl.Vertex2i(gl.Int(v1.X), gl.Int(v1.Y))
		gl.Vertex2i(gl.Int(v2.X), gl.Int(v2.Y))
	}
	gl.End()
	player := g.Ents[gid].(*game.Player)
	ray := (linear.Vec2{1, 0}).Rotate(player.Angle).Scale(f.Stored)
	center := ray.Add(player.Position)

	return
	base.EnableShader("fire")
	gl.Color4ub(255, 255, 255, 255)
	base.SetUniformF("fire", "inner", f.Inner/f.Outer*0.5)
	base.SetUniformF("fire", "outer", 0.5)
	base.SetUniformF("fire", "frac", 1)
	base.SetUniformF("fire", "heading", 0)
	texture.Render(
		center.X-float64(f.Outer),
		center.Y-float64(f.Outer),
		2*float64(f.Outer),
		2*float64(f.Outer))
	base.EnableShader("")
}
