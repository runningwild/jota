package ability

import (
	"encoding/gob"
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/cgf"
	"github.com/runningwild/linear"
	"github.com/runningwild/magnus/base"
	"github.com/runningwild/magnus/game"
	"github.com/runningwild/magnus/texture"
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
	base.Log().Printf("Append")

	g.Processes = append(g.Processes, &fireProcess{
		BasicPhases: BasicPhases{game.PhaseRunning},
		Frames:      100,
		Pos:         pos,
		Inner:       100,
		Outer:       200,
		Angle:       0.1,
		Heading:     float32(player.Angle),
	})
}

type fireProcess struct {
	BasicPhases
	NullCondition
	Frames  int32
	Pos     linear.Vec2
	Inner   float32
	Outer   float32
	Angle   float32
	Heading float32
}

func (f *fireProcess) Supply(supply game.Mana) game.Mana {
	return supply
}

func (f *fireProcess) Think(g *game.Game) {
	f.Frames--
	if f.Frames == 0 {
		f.BasicPhases.The_phase = game.PhaseComplete
	}
}

func (f *fireProcess) Draw(gid game.Gid, g *game.Game) {
	base.EnableShader("fire")
	gl.Color4ub(255, 255, 255, 255)
	base.SetUniformF("fire", "inner", f.Inner/f.Outer*0.5)
	base.SetUniformF("fire", "outer", 0.5)
	base.SetUniformF("fire", "frac", f.Angle)
	base.SetUniformF("fire", "heading", f.Heading)
	texture.Render(
		f.Pos.X-float64(f.Outer),
		f.Pos.Y-float64(f.Outer),
		2*float64(f.Outer),
		2*float64(f.Outer))
	base.EnableShader("")
}
