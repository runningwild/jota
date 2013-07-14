package game

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/cgf"
	"github.com/runningwild/cmwc"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/linear"
	"github.com/runningwild/magnus/base"
	"github.com/runningwild/magnus/los"
	"github.com/runningwild/magnus/stats"
	"github.com/runningwild/magnus/texture"
	"path/filepath"
	"runtime/debug"
)

// type Ability func(game *Game, player *Player, params map[string]int) Process

// An Ability represents something a player can do that does not directly affect
// the game state.
type Ability interface {
	// Called when a player selects this Ability.  Returns any number of events to
	// apply, as well as a bool that is true iff this Ability should become the
	// active Ability.
	Activate(gid Gid, keyPress bool) ([]cgf.Event, bool)

	// Called when this Ability is deselected as a result of another ability being
	// selected.  For some abilities this might not do anything, but certain
	// abilities may want to
	Deactivate(gid Gid) []cgf.Event

	// The active Ability will receive all of the events from the player.  It
	// should return true iff it consumes the event.
	Respond(gid Gid, group gin.EventGroup) bool

	// Returns any number of events to apply, as well as a bool that is true iff
	// this Ability should be deactivated.  Typically this will include an event
	// that will add a Process to this player.
	Think(gid Gid, game *Game, mouse linear.Vec2) ([]cgf.Event, bool)

	// If it is the active Ability it might want to draw some Ui stuff.
	Draw(gid Gid, game *Game)
}

type AbilityMaker func(params map[string]int) Ability

var ability_makers map[string]AbilityMaker

func RegisterAbility(name string, maker AbilityMaker) {
	if ability_makers == nil {
		ability_makers = make(map[string]AbilityMaker)
	}
	ability_makers[name] = maker
}

type Drain interface {
	// Supplies mana to the Process and returns the unused portion.
	Supply(Mana) Mana
}

type Phase int

const (
	// This phase is for any process that needs a ui before.  A player can only
	// have one Process in PhaseUi at a time.  If a player tries to use an ability
	// while a Process is in PhaseUi the process in PhaseUi will be killed.
	PhaseUi Phase = iota

	// Once a Process hits PhaseRunning it will remain here until it is complete.
	// A process should not reach this phase until it is done with player
	// interaction.
	PhaseRunning

	// Once a Process returns PhaseComplete it will always return PhaseComplete.
	PhaseComplete
)

type Thinker interface {
	Think(game *Game)

	// Kills a process.  Any Killed process will return true on any future
	// calls to Complete().
	Kill(game *Game)

	Phase() Phase
}

// TODO: Might want to be able to respond to events directly for Ui stuff
type Responder interface {
}

type Process interface {
	Drain
	Thinker
	Responder
	stats.Condition
	Draw(id Gid, game *Game)
	Copy() Process
}

type Color int

const (
	ColorRed Color = iota
	ColorGreen
	ColorBlue
)

func init() {
}

type Player struct {
	BaseEnt
	Los *los.Los
}

func (g *Game) AddPlayer(pos linear.Vec2) Ent {
	var p Player
	err := json.NewDecoder(bytes.NewBuffer([]byte(`
      {
        "Base": {
          "Max_turn": 0.07,
          "Max_acc": 0.2,
          "Mass": 750,
          "Max_rate": 10,
          "Influence": 75,
          "Health": 1000
        },
        "Dynamic": {
          "Health": 1000
        }
      }
    `))).Decode(&p.BaseEnt.Stats)
	if err != nil {
		base.Log().Fatalf("%v", err)
	}
	p.Position = pos
	p.Gid = g.NextGid()
	p.Processes = make(map[int]Process)
	p.Los = los.Make(LosMaxDist)
	g.Ents[p.Gid] = &p
	return &p
}

func (p *Player) Copy() Ent {
	p2 := *p
	p2.Processes = make(map[int]Process)
	for k, v := range p.Processes {
		p2.Processes[k] = v.Copy()
		if v == nil {
			panic("ASDF")
		}
	}
	p2.Los = p.Los.Copy()
	return &p2
}

func init() {
	gob.Register(&Player{})
}

func (p *Player) ReleaseResources() {
	p.Los.ReleaseResources()
}

func (p *Player) Draw(game *Game) {
	var t *texture.Data
	gl.Color4ub(255, 255, 255, 255)
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
		p.Angle,
		false)

	for _, proc := range p.Processes {
		proc.Draw(p.Id(), game)
	}
	base.EnableShader("status_bar")
	base.SetUniformF("status_bar", "inner", 0.08)
	base.SetUniformF("status_bar", "outer", 0.09)
	base.SetUniformF("status_bar", "buffer", 0.01)

	base.SetUniformF("status_bar", "frac", 1.0)
	gl.Color4ub(125, 125, 125, 100)
	texture.Render(p.Position.X-100, p.Position.Y-100, 200, 200)

	health_frac := float32(p.Stats.HealthCur() / p.Stats.HealthMax())
	if health_frac > 0.5 {
		color_frac := 1.0 - (health_frac-0.5)*2.0
		gl.Color4ub(gl.Ubyte(255.0*color_frac), 255, 0, 255)
	} else {
		color_frac := health_frac * 2.0
		gl.Color4ub(255, gl.Ubyte(255.0*color_frac), 0, 255)
	}
	base.SetUniformF("status_bar", "frac", health_frac)
	texture.Render(p.Position.X-100, p.Position.Y-100, 200, 200)
	base.EnableShader("")
}

func (p *Player) Think(g *Game) {
	p.Los.Reset(p.Pos())
	for polyIndex, poly := range g.Room.Walls {
		for i := range poly {
			p.Los.DrawSeg(poly.Seg(i), polyIndex)
		}
	}
	p.BaseEnt.Think(g)
}

func (p *Player) Supply(supply Mana) Mana {
	for _, process := range p.Processes {
		supply = process.Supply(supply)
	}
	return supply
}

type Ent interface {
	Draw(g *Game)
	Think(game *Game)
	ApplyForce(force linear.Vec2)

	// Stats based methods
	Alive() bool
	OnDeath(g *Game)
	ApplyDamage(damage stats.Damage)

	Id() Gid
	Pos() linear.Vec2

	// Need to have a SetPos method because we don't want ents moving through
	// walls.
	SetPos(pos linear.Vec2)

	Mass() float64
	Vel() linear.Vec2

	Supply(mana Mana) Mana
	Copy() Ent
}

type NonManaUser struct{}

func (NonManaUser) Supply(mana Mana) Mana { return mana }

type Gid string

type Game struct {
	ManaSource ManaSource

	Room Room

	Rng *cmwc.Cmwc

	// Dimensions of the board
	Dx, Dy int

	Friction      float64
	Friction_lava float64

	// Last Id assigned to an entity
	NextGidValue int

	Ents map[Gid]Ent

	GameThinks int

	Architect architectData
}

func (g *Game) NextGid() Gid {
	g.NextGidValue++
	return Gid(fmt.Sprintf("%d", g.NextGidValue))
}

func (g *Game) Init() {
	msOptions := ManaSourceOptions{
		NumSeeds:    20,
		NumNodeRows: 60,
		NumNodeCols: 90,

		BoardLeft:   0,
		BoardTop:    0,
		BoardRight:  float64(g.Dx),
		BoardBottom: float64(g.Dy),

		MaxDrainDistance: 120.0,
		MaxDrainRate:     5.0,

		RegenPerFrame:     0.002,
		NodeMagnitude:     100,
		MinNodeBrightness: 20,
		MaxNodeBrightness: 150,
	}
	g.ManaSource.Init(&msOptions)
}

func init() {
	gob.Register(&Game{})
}

func (g *Game) ReleaseResources() {
	for _, e := range g.Ents {
		if p, ok := e.(*Player); ok {
			p.ReleaseResources()
		}
	}
}

type gameResponderWrapper struct {
	l *localData
}

func (grw *gameResponderWrapper) HandleEventGroup(group gin.EventGroup) {
	grw.l.HandleEventGroup(group)
}

func (grw *gameResponderWrapper) Think(int64) {}

func invSquareDist(dist_sq float64) float64 {
	return 1.0 / (dist_sq + 1)
}

func getWeights(distance_squares []float64, value_sum float64, transform func(float64) float64) []float64 {
	weights := make([]float64, len(distance_squares))

	weight_sum := 0.0
	for i, dist_sq := range distance_squares {
		if dist_sq >= 0 {
			weights[i] = transform(dist_sq)
		} else {
			weights[i] = 0
		}
		weight_sum += weights[i]
	}

	for i, w := range weights {
		weights[i] = value_sum * w / weight_sum
	}
	return weights
}

// func (g *Game) Merge(g2 *Game) {
// 	frac := 0.0 // i.e. don't merge
// 	for i := range g.Ents {
// 		_p1 := g.Ents[i]
// 		var p1 *Player
// 		var ok bool
// 		if p1, ok = _p1.(*Player); !ok {
// 			continue
// 		}
// 		p2, ok := g2.GetEnt(p1.Id()).(*Player)
// 		if p2 == nil || !ok {
// 			continue
// 		}
// 		p1.Position.X = frac*p2.Position.X + (1-frac)*p1.Position.X
// 		p1.Position.Y = frac*p2.Position.Y + (1-frac)*p1.Position.Y
// 		p1.Angle = frac*p2.Angle + (1-frac)*p1.Angle
// 	}
// }

func (g *Game) Copy() interface{} {
	var g2 Game

	g2.ManaSource = g.ManaSource.Copy()

	g2.Room = g.Room

	g2.Rng = g.Rng.Copy()

	g2.Dx = g.Dx
	g2.Dy = g.Dy
	g2.Friction = g.Friction
	g2.Friction_lava = g.Friction_lava
	g2.NextGidValue = g.NextGidValue
	g2.GameThinks = g.GameThinks

	g2.Ents = make(map[Gid]Ent, len(g.Ents))
	for gid, ent := range g.Ents {
		g2.Ents[gid] = ent.Copy()
	}
	return &g2
}

func (g *Game) OverwriteWith(_g2 interface{}) {
	g2 := _g2.(*Game)
	g.ManaSource.OverwriteWith(&g2.ManaSource)
	g.Rng.OverwriteWith(g2.Rng)
	g.Dx = g2.Dx
	g.Dy = g2.Dy
	g.Friction = g2.Friction
	g.Room.Walls = g2.Room.Walls
	g.NextGidValue = g2.NextGidValue
	g.GameThinks = g2.GameThinks

	g.Ents = make(map[Gid]Ent, len(g2.Ents))
	for gid, ent := range g2.Ents {
		g.Ents[gid] = ent.Copy()
	}
}

func (g *Game) Think() {
	defer func() {
		if r := recover(); r != nil {
			base.Error().Printf("Panic: %v", r)
			base.Error().Printf("Stack:\n%s", debug.Stack())
			panic(r)
		}
	}()
	g.GameThinks++

	for i := range g.Ents {
		if !g.Ents[i].Alive() {
			g.Ents[i].OnDeath(g)
		}
	}
	var dead []Gid
	for gid, ent := range g.Ents {
		if !ent.Alive() {
			dead = append(dead, gid)
		}
	}
	for _, gid := range dead {
		delete(g.Ents, gid)
	}

	g.ManaSource.Think(g.Ents)
	g.Architect.Think(g)

	// Advance players, check for collisions, add segments
	for i := range g.Ents {
		if !g.Ents[i].Alive() {
			continue
		}
		g.Ents[i].Think(g)
		pos := g.Ents[i].Pos()
		pos.X = clamp(pos.X, 0, float64(g.Dx))
		pos.Y = clamp(pos.Y, 0, float64(g.Dy))
		g.Ents[i].SetPos(pos)
	}
	moved := make(map[Gid]bool)
	for i := range g.Ents {
		for j := range g.Ents {
			if i == j {
				continue
			}
			dist := g.Ents[i].Pos().Sub(g.Ents[j].Pos()).Mag()
			if dist > 25 {
				continue
			}
			if dist < 0.01 {
				continue
			}
			if dist <= 0.5 {
				dist = 0.5
			}
			force := 20.0 * (25 - dist)
			g.Ents[i].ApplyForce(g.Ents[i].Pos().Sub(g.Ents[j].Pos()).Norm().Scale(force))
			moved[i] = true
		}
	}

	// This must always be the very last thing
	localThink(g)
}

func clamp(v, low, high float64) float64 {
	if v < low {
		return low
	}
	if v > high {
		return high
	}
	return v
}

type Turn struct {
	Gid   Gid
	Delta float64
}

func init() {
	gob.Register(Turn{})
}

func (t Turn) Apply(_g interface{}) {
	g := _g.(*Game)
	player := g.Ents[t.Gid].(*Player)
	player.Delta.Angle = t.Delta
}

type Accelerate struct {
	Gid   Gid
	Delta float64
}

func init() {
	gob.Register(Accelerate{})
}

func (a Accelerate) Apply(_g interface{}) {
	g := _g.(*Game)
	player := g.Ents[a.Gid].(*Player)
	player.Delta.Speed = a.Delta / 2
}

type GameWindow struct {
	Engine    *cgf.Engine
	game      *Game
	prev_game *Game
	region    gui.Region

	nodeTextureId      gl.Uint
	nodeTextureData    []byte
	nodeWarpingTexture gl.Uint
	nodeWarpingData    []byte
}

func (gw *GameWindow) String() string {
	return "game window"
}
func (gw *GameWindow) Expandable() (bool, bool) {
	return false, false
}
func (gw *GameWindow) Requested() gui.Dims {
	if gw.game == nil {
		return gui.Dims{}
	}
	return gui.Dims{gw.game.Dx, gw.game.Dy}
}
func (gw *GameWindow) Rendered() gui.Region {
	return gw.region
}
func (gw *GameWindow) Think(g *gui.Gui, t int64) {
	// if gw.game == nil {
	old_game := gw.game
	gw.game = gw.Engine.CopyState().(*Game)
	if old_game != nil {
		old_game.ReleaseResources()
	}
	// gw.prev_game = gw.game.Copy().(*Game)
	// } else {
	// 	gw.Engine.UpdateState(gw.game)
	// 	gw.game.Merge(gw.prev_game)
	// 	gw.prev_game.OverwriteWith(gw.game)
	// }
}
func (gw *GameWindow) Respond(g *gui.Gui, group gui.EventGroup) bool {
	return false
}

var latest_region gui.Region

// Returns the most recent region used when rendering the game.
func (g *Game) Region() gui.Region {
	return latest_region
}

func (gw *GameWindow) Draw(region gui.Region) {
	defer func() {
		if r := recover(); r != nil {
			base.Error().Printf("Panic: %v", r)
			base.Error().Printf("Stack:\n%s", debug.Stack())
			panic(r)
		}
	}()
	gw.region = region
	latest_region = region
	gl.PushMatrix()
	defer gl.PopMatrix()
	gl.Translated(gl.Double(gw.region.X), gl.Double(gw.region.Y), 0)

	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	gw.game.ManaSource.Draw(gw, float64(gw.game.Dx), float64(gw.game.Dy))

	gl.Begin(gl.LINES)
	gl.Color4d(1, 1, 1, 1)
	for _, poly := range gw.game.Room.Walls {
		for i := range poly {
			seg := poly.Seg(i)
			gl.Vertex2d(gl.Double(seg.P.X), gl.Double(seg.P.Y))
			gl.Vertex2d(gl.Double(seg.Q.X), gl.Double(seg.Q.Y))
		}
	}
	gl.End()

	gl.Color4d(1, 0, 0, 1)
	for _, poly := range gw.game.Room.Lava {
		gl.Begin(gl.TRIANGLE_FAN)
		for _, v := range poly {
			gl.Vertex2d(gl.Double(v.X), gl.Double(v.Y))
		}
		gl.End()
	}

	gl.Color4d(1, 1, 1, 1)
	losCount := 0
	for _, ent := range gw.game.Ents {
		p, ok := ent.(*Player)
		if !ok {
			continue
		}
		p.Los.WriteDepthBuffer(local.los.texData[losCount], LosMaxDist)
		losCount++
	}
	gl.Color4d(1, 1, 1, 1)
	for _, ent := range gw.game.Ents {
		ent.Draw(gw.game)
	}
	gl.Disable(gl.TEXTURE_2D)
	gw.game.RenderLocal(region)
}
func (gw *GameWindow) DrawFocused(region gui.Region) {}
