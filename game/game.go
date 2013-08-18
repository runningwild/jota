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
	"github.com/runningwild/linear"
	"github.com/runningwild/magnus/base"
	"github.com/runningwild/magnus/gui"
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

// AddPlayers adds numPlayers to the specified side.  In standard game mode side
// should be zero, otherwise it should be between 0 and number of side - 1,
// inclusive.
func (g *Game) AddPlayers(numPlayers int, side int) []Gid {
	switch {
	case g.Standard != nil:
		if side != 0 {
			base.Error().Fatalf("AddPlayers expects side == 0 for Standard game mode.")
		}
	case g.Moba != nil:
		if side < 0 || side >= len(g.Levels[GidInvadersStart].Room.Starts) {
			base.Error().Fatalf("Got side %d, but this level only supports sides from 0 to %d.", len(g.Levels[GidInvadersStart].Room.Starts)-1)
		}
	default:
		base.Error().Fatalf("Cannot add players without first specifying a game mode.")
	}
	var gids []Gid
	for i := 0; i < numPlayers; i++ {
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
	    `))).Decode(&p.BaseEnt.StatsInst)
		if err != nil {
			base.Log().Fatalf("%v", err)
		}
		p.CurrentLevel = GidInvadersStart

		// Evenly space the players on a circle around the starting position.
		rot := (linear.Vec2{25, 0}).Rotate(float64(i) * 2 * 3.1415926535 / float64(numPlayers))
		p.Position = g.Levels[GidInvadersStart].Room.Starts[side].Add(rot)

		// NEXT: REthing Gids and how the levels are laid out - should they just
		// be indexed by gids?
		p.Side_ = side
		p.Gid = g.NextGid()
		p.Processes = make(map[int]Process)
		p.Los = los.Make(LosPlayerHorizon)
		p.SetLevel(GidInvadersStart)
		g.Ents[p.Gid] = &p
		gids = append(gids, p.Gid)
	}
	return gids
}

func init() {
	gob.Register(&Player{})
}

// func (p *Player) ReleaseResources() {
// 	p.Los.ReleaseResources()
// }

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

	health_frac := float32(p.Stats().HealthCur() / p.Stats().HealthMax())
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
	for _, _poly := range g.temp.AllWalls[p.Level()] {
		poly := linear.Poly(_poly)
		// maxDistSq := 0.0
		// for i := 1; i < len(poly); i++ {
		// 	distSq := poly[i].Sub(poly[0]).Mag2()
		// 	if distSq > maxDistSq {
		// 		maxDistSq = distSq
		// 	}
		// }
		// if
		for i := range poly {
			seg := poly.Seg(i)
			if p.Position.Sub(seg.P).Mag2()-seg.Ray().Mag2() > 4*LosPlayerHorizon*LosPlayerHorizon {
				continue
			}
			if seg.Right(p.Position) {
				continue
			}
			p.Los.DrawSeg(seg, "")

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

	// If this Ent is immovable it may provide walls that will be considered just
	// like normal walls.
	Walls() [][]linear.Vec2

	// Stats based methods
	OnDeath(g *Game)
	Stats() *stats.Inst
	Mass() float64
	Vel() linear.Vec2

	Id() Gid
	Pos() linear.Vec2
	Level() Gid
	Side() int // which side the ent belongs to

	// Need to have a SetPos method because we don't want ents moving through
	// walls.
	SetPos(pos linear.Vec2)
	SetLevel(Gid)

	Supply(mana Mana) Mana
}

type NonManaUser struct{}

func (NonManaUser) Supply(mana Mana) Mana { return mana }

type Gid string

const (
	GidInvadersStart Gid = "invaders start"
)

type Level struct {
	ManaSource ManaSource
	Room       Room
}

type Game struct {
	Levels map[Gid]*Level

	Rng *cmwc.Cmwc

	Friction      float64
	Friction_lava float64

	// Last Id assigned to an entity
	NextGidValue int

	Ents map[Gid]Ent

	GameThinks int

	// Game Modes - Exactly one of these will be set
	Standard *GameModeStandard
	Moba     *GameModeMoba

	temp struct {
		// This include all room walls for each room, and all walls declared by any
		// ents in that room.
		AllWalls      map[Gid][][]linear.Vec2
		AllWallsDirty bool

		// Map from level to list of ents on that level, in the order that they
		// should be iterated in.
		AllEntsByLevel      map[Gid][]Ent
		AllEntsByLevelDirty bool

		// List of all ents, in the order that they should be iterated in.
		AllEnts      []Ent
		AllEntsDirty bool

		// All levels, in the order that they should be iterated in.
		AllLevels      []*Level
		AllLevelsDirty bool
	}
}

type GameModeStandard struct {
	Architect architectData
	Invaders  invadersData
}
type GameModeMoba struct {
}

func (g *Game) NextGid() Gid {
	g.NextGidValue++
	return Gid(fmt.Sprintf("%d", g.NextGidValue))
}

func (g *Game) Init() {
	g.DoForLevels(func(gid Gid, level *Level) {
		msOptions := ManaSourceOptions{
			NumSeeds:    20,
			NumNodeRows: level.Room.Dy / 10,
			NumNodeCols: level.Room.Dx / 10,

			BoardLeft:   0,
			BoardTop:    0,
			BoardRight:  float64(level.Room.Dx),
			BoardBottom: float64(level.Room.Dy),

			MaxDrainDistance: 120.0,
			MaxDrainRate:     5.0,

			RegenPerFrame:     0.002,
			NodeMagnitude:     100,
			MinNodeBrightness: 20,
			MaxNodeBrightness: 150,

			Rng: g.Rng,
		}
		level.ManaSource.Init(&msOptions)
	})
}

func init() {
	gob.Register(&Game{})
}

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

func lessGids(a, b Gid) bool {
	return a < b
}

func (g *Game) DoForEnts(f func(Gid, Ent)) {
	base.DoOrdered(g.Ents, lessGids, f)
}

func (g *Game) DoForLevels(f func(Gid, *Level)) {
	base.DoOrdered(g.Levels, lessGids, f)
}

func (g *Game) RemoveEnt(gid Gid) {
	delete(g.Ents, gid)
}

func (g *Game) AddEnt(ent Ent) Gid {
	gid := g.NextGid()
	g.Ents[gid] = ent
	return gid
}

func (g *Game) Think() {
	defer func() {
		if r := recover(); r != nil {
			base.Error().Printf("Panic: %v", r)
			base.Error().Fatalf("Stack:\n%s", debug.Stack())
		}
	}()
	g.GameThinks++

	if g.temp.AllEntsByLevel == nil || g.temp.AllEntsByLevelDirty {
		g.temp.AllEntsByLevel = make(map[Gid][]Ent)
	}
	for gid := range g.temp.AllEntsByLevel {
		g.temp.AllEntsByLevel[gid] = g.temp.AllEntsByLevel[gid][0:0]
	}
	g.temp.AllEnts = g.temp.AllEnts[0:0]
	g.DoForEnts(func(gid Gid, ent Ent) {
		if ent.Stats().HealthCur() <= 0 {
			ent.OnDeath(g)
			g.RemoveEnt(gid)
		} else {
			g.temp.AllEntsByLevel[gid] = append(g.temp.AllEntsByLevel[gid], ent)
			g.temp.AllEnts = append(g.temp.AllEnts, ent)
		}
	})

	g.DoForLevels(func(gid Gid, level *Level) {
		level.ManaSource.Think(g.Ents)
	})

	// Advance players, check for collisions, add segments
	for _, ent := range g.temp.AllEnts {
		ent.Think(g)
		pos := ent.Pos()
		pos.X = clamp(pos.X, 0, float64(g.Levels[ent.Level()].Room.Dx))
		pos.Y = clamp(pos.Y, 0, float64(g.Levels[ent.Level()].Room.Dy))
		ent.SetPos(pos)
	}

	if g.temp.AllWalls == nil || g.temp.AllWallsDirty {
		g.temp.AllWalls = make(map[Gid][][]linear.Vec2)
		for gid := range g.Levels {
			var allWalls [][]linear.Vec2
			base.DoOrdered(g.Levels[gid].Room.Walls, func(a, b string) bool { return a < b }, func(_ string, walls linear.Poly) {
				allWalls = append(allWalls, []linear.Vec2(walls))
			})
			g.DoForEnts(func(entGid Gid, ent Ent) {
				if ent.Level() == gid {
					for _, walls := range ent.Walls() {
						allWalls = append(allWalls, walls)
					}
				}
			})
			g.temp.AllWalls[gid] = allWalls
		}
	}

	for _, outerEnt := range g.temp.AllEnts {
		for _, innerEnt := range g.temp.AllEnts {
			if innerEnt == outerEnt {
				continue
			}
			dist := outerEnt.Pos().Sub(innerEnt.Pos()).Mag()
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
			outerEnt.ApplyForce(outerEnt.Pos().Sub(innerEnt.Pos()).Norm().Scale(force))
		}
	}

	switch {
	case g.Moba != nil:
		// Do moba thinking
	case g.Standard != nil:
		// Do standard thinking
	default:
		panic("Game mode not set")
	}
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
	Local     *LocalData
	Dims      gui.Dims
	game      *Game
	prev_game *Game
}

func (gw *GameWindow) String() string {
	return "game window"
}
func (gw *GameWindow) Expandable() (bool, bool) {
	return false, false
}
func (gw *GameWindow) Requested() gui.Dims {
	return gui.Dims{800, 600}
}
func (gw *GameWindow) Think(g *gui.Gui) {
	gw.Engine.Pause()
	gw.Local.Think(gw.Engine.GetState().(*Game))
	gw.Engine.Unpause()
}
func (gw *GameWindow) Respond(group gin.EventGroup) {
}
func (gw *GameWindow) RequestedDims() gui.Dims {
	return gw.Dims
}

func (gw *GameWindow) Draw(region gui.Region, style gui.StyleStack) {
	defer func() {
		if r := recover(); r != nil {
			base.Error().Printf("Panic: %v", r)
			base.Error().Fatalf("Stack:\n%s", debug.Stack())
		}
	}()
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
	gw.Engine.GetState().(*Game).RenderLocal(region, gw.Local)
	gw.Engine.Unpause()
}
func (gw *GameWindow) DrawFocused(region gui.Region) {}
