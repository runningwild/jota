package game

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/cgf"
	"github.com/runningwild/cmwc"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/util/algorithm"
	"github.com/runningwild/linear"
	"github.com/runningwild/magnus/base"
	"github.com/runningwild/magnus/generator"
	"github.com/runningwild/magnus/gui"
	"github.com/runningwild/magnus/stats"
	"github.com/runningwild/magnus/texture"
	"math"
	"path/filepath"
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
	Draw(gid Gid, game *Game, side int)
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

type Process interface {
	Drain
	Thinker
	stats.Condition
	Draw(id Gid, game *Game, side int)
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
}

// AddPlayers adds numPlayers to the specified side.  In standard game mode side
// should be zero, otherwise it should be between 0 and number of side - 1,
// inclusive.
func (g *Game) AddPlayers(engineIds []int64, side int) []Gid {
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
	for i, engineId := range engineIds {
		var p Player
		p.StatsInst = stats.Make(stats.Base{
			Health: 1000,
			Mass:   750,
			Acc:    0.2,
			Turn:   0.07,
			Rate:   0.5,
			Size:   12,
			Vision: 600,
		})
		p.CurrentLevel = GidInvadersStart

		// Evenly space the players on a circle around the starting position.
		rot := (linear.Vec2{25, 0}).Rotate(float64(i) * 2 * 3.1415926535 / float64(len(engineIds)))
		p.Position = g.Levels[GidInvadersStart].Room.Starts[side].Add(rot)

		// NEXT: REthing Gids and how the levels are laid out - should they just
		// be indexed by gids?
		p.Side_ = side
		p.Gid = Gid(fmt.Sprintf("Engine:%d", engineId))
		p.Processes = make(map[int]Process)
		p.SetLevel(GidInvadersStart)
		g.AddEnt(&p)
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

func (p *Player) Draw(game *Game, side int) {
	var t *texture.Data
	var alpha gl.Ubyte
	if side == p.Side() {
		alpha = gl.Ubyte(255.0 * (1.0 - p.Stats().Cloaking()/2))
	} else {
		alpha = gl.Ubyte(255.0 * (1.0 - p.Stats().Cloaking()))
	}
	gl.Color4ub(255, 255, 255, alpha)
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
		proc.Draw(p.Id(), game, side)
	}
	base.EnableShader("status_bar")
	base.SetUniformF("status_bar", "inner", 0.08)
	base.SetUniformF("status_bar", "outer", 0.09)
	base.SetUniformF("status_bar", "buffer", 0.01)

	base.SetUniformF("status_bar", "frac", 1.0)
	gl.Color4ub(125, 125, 125, alpha/2)
	texture.Render(p.Position.X-100, p.Position.Y-100, 200, 200)

	health_frac := float32(p.Stats().HealthCur() / p.Stats().HealthMax())
	if health_frac > 0.5 {
		color_frac := 1.0 - (health_frac-0.5)*2.0
		gl.Color4ub(gl.Ubyte(255.0*color_frac), 255, 0, alpha)
	} else {
		color_frac := health_frac * 2.0
		gl.Color4ub(255, gl.Ubyte(255.0*color_frac), 0, alpha)
	}
	base.SetUniformF("status_bar", "frac", health_frac)
	texture.Render(p.Position.X-100, p.Position.Y-100, 200, 200)
	base.EnableShader("")
}

func (p *Player) Think(g *Game) {
	p.BaseEnt.Think(g)
}

func (p *Player) Supply(supply Mana) Mana {
	for _, process := range p.Processes {
		supply = process.Supply(supply)
	}
	return supply
}

type Ent interface {
	Draw(g *Game, side int)
	Think(game *Game)
	ApplyForce(force linear.Vec2)

	// If this Ent is immovable it may provide walls that will be considered just
	// like normal walls.
	// TODO: Decide whether or not to actually support this
	// Walls() [][]linear.Vec2

	// Stats based methods
	OnDeath(g *Game)
	Stats() *stats.Inst
	Mass() float64
	Vel() linear.Vec2

	Id() Gid
	SetId(Gid)
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

// Used before the 'game' starts to choose sides and characters and whatnot.
type Setup struct {
	Mode      string        // should be "moba" or "standard"
	EngineIds []int64       // engine ids of the engines currently joined
	Sides     map[int64]int // map from engineid to side
}

type SetupSetEngineIds struct {
	EngineIds []int64
}

func (s SetupSetEngineIds) Apply(_g interface{}) {
	g := _g.(*Game)
	if g.Setup == nil {
		return
	}
	g.Setup.EngineIds = s.EngineIds
	for _, id := range g.Setup.EngineIds {
		if _, ok := g.Setup.Sides[id]; !ok {
			g.Setup.Sides[id] = 0
		}
	}
}
func init() {
	gob.Register(SetupSetEngineIds{})
}

type SetupChangeSides struct {
	EngineId int64
	Side     int
}

func (s SetupChangeSides) Apply(_g interface{}) {
	g := _g.(*Game)
	if g.Setup == nil {
		return
	}
	g.Setup.Sides[s.EngineId] = s.Side
}
func init() {
	gob.Register(SetupChangeSides{})
}

type SetupComplete struct{}

func (u SetupComplete) Apply(_g interface{}) {
	g := _g.(*Game)
	if g.Setup == nil {
		return
	}

	var room Room
	dx, dy := 1024, 2048
	generated := generator.GenerateRoom(float64(dx), float64(dy), 100, 64, 64522029961391019)
	data, err := json.Marshal(generated)
	if err != nil {
		base.Error().Fatalf("%v", err)
	}
	err = json.Unmarshal(data, &room)
	// err = base.LoadJson(filepath.Join(base.GetDataDir(), "rooms/basic.json"), &room)
	if err != nil {
		base.Error().Fatalf("%v", err)
	}
	g.Levels = make(map[Gid]*Level)
	g.Levels[GidInvadersStart] = &Level{}
	g.Levels[GidInvadersStart].Room = room
	g.Rng = cmwc.MakeGoodCmwc()
	g.Rng.Seed(12313131)
	g.Ents = make(map[Gid]Ent)
	g.Friction = 0.97
	// g.Standard = &GameModeStandard{}
	g.Moba = &GameModeMoba{
		Sides: make(map[int]*GameModeMobaSideData),
	}
	sides := make(map[int][]int64)
	for engineId, side := range g.Setup.Sides {
		sides[side] = append(sides[side], engineId)
	}
	for side, ids := range sides {
		g.AddPlayers(ids, side)
		g.Moba.Sides[side] = &GameModeMobaSideData{
			losCache: makeLosCache(dx, dy),
		}
	}
	g.MakeControlPoints()
	g.Init()
	g.Setup = nil
}
func init() {
	gob.Register(SetupComplete{})
}

type Game struct {
	Setup *Setup

	Levels map[Gid]*Level

	Rng *cmwc.Cmwc

	Friction float64

	// Last Id assigned to an entity
	NextGidValue int

	Ents map[Gid]Ent

	GameThinks int

	// All effects that are not tied to a player.
	Processes []Process

	// Game Modes - Exactly one of these will be set
	Standard *GameModeStandard
	Moba     *GameModeMoba

	temp struct {
		// This include all room walls for each room, and all walls declared by any
		// ents in that room.
		AllWalls map[Gid][]linear.Seg2
		// It looks silly, but this is...
		// map[LevelId][x/cacheGridSize][y/cacheGridSize] slice of linear.Poly
		// AllWallsGrid  map[Gid][][][][]linear.Vec2
		AllWallsDirty bool
		WallCache     map[Gid]*wallCache
		// VisibleWallCache is like WallCache but returns all segments visible from
		// a location, rather than all segments that should be checked for
		// collision.
		VisibleWallCache map[Gid]*wallCache

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
	// Map from side to the moba data for that side
	Sides map[int]*GameModeMobaSideData
}
type GameModeMobaSideData struct {
	AppeaseGob struct{}
	losCache   *losCache
}

func (g *Game) NextGid() Gid {
	g.NextGidValue++
	return Gid(fmt.Sprintf("%d", g.NextGidValue))
}

func (g *Game) Init() {
	g.DoForLevels(func(gid Gid, level *Level) {
		msOptions := ManaSourceOptions{
			NumSeeds:    20,
			NumNodeRows: level.Room.Dy / 32,
			NumNodeCols: level.Room.Dx / 32,

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
	g.temp.AllEntsDirty = true
}

func (g *Game) AddEnt(ent Ent) {
	if ent.Id() == "" {
		ent.SetId(g.NextGid())
	}
	g.Ents[ent.Id()] = ent
	g.temp.AllEntsDirty = true
}

func (g *Game) Think() {
	g.GameThinks++
	if g.Setup != nil {
		return
	}
	defer base.StackCatcher()

	// cache wall data
	if g.temp.AllWalls == nil || g.temp.AllWallsDirty {
		g.temp.AllWalls = make(map[Gid][]linear.Seg2)
		g.temp.WallCache = make(map[Gid]*wallCache)
		g.temp.VisibleWallCache = make(map[Gid]*wallCache)
		for gid := range g.Levels {
			var allWalls []linear.Seg2
			base.DoOrdered(g.Levels[gid].Room.Walls, func(a, b string) bool { return a < b }, func(_ string, walls linear.Poly) {
				for i := range walls {
					allWalls = append(allWalls, walls.Seg(i))
				}
			})
			// g.DoForEnts(func(entGid Gid, ent Ent) {
			// 	if ent.Level() == gid {
			// 		for _, walls := range ent.Walls() {
			// 			for i := range walls {
			// 				allWalls = append(allWalls, walls.Seg(i))
			// 			}
			// 		}
			// 	}
			// })
			g.temp.AllWalls[gid] = allWalls
			g.temp.WallCache[gid] = &wallCache{}
			g.temp.WallCache[gid].SetWalls(g.Levels[gid].Room.Dx, g.Levels[gid].Room.Dy, allWalls, 100)
			g.temp.VisibleWallCache[gid] = &wallCache{}
			g.temp.VisibleWallCache[gid].SetWalls(g.Levels[gid].Room.Dx, g.Levels[gid].Room.Dy, allWalls, stats.LosPlayerHorizon)
			base.Log().Printf("WallCache: %v", g.temp.WallCache)
		}
		for _, data := range g.Moba.Sides {
			data.losCache.SetWallCache(g.temp.VisibleWallCache[GidInvadersStart])
		}
	}

	// cache ent data
	for _, ent := range g.temp.AllEnts {
		if ent.Stats().HealthCur() <= 0 {
			ent.OnDeath(g)
			g.RemoveEnt(ent.Id())
		}
	}
	if g.temp.AllEnts == nil || g.temp.AllEntsDirty {
		g.temp.AllEnts = g.temp.AllEnts[0:0]
		g.DoForEnts(func(gid Gid, ent Ent) {
			base.Log().Printf("Appending %v", ent.Id())
			g.temp.AllEnts = append(g.temp.AllEnts, ent)
		})
		g.temp.AllEntsDirty = false
	}

	for _, proc := range g.Processes {
		proc.Think(g)
	}
	algorithm.Choose(&g.Processes, func(proc Process) bool { return proc.Phase() != PhaseComplete })

	// Advance players, check for collisions, add segments
	for _, ent := range g.temp.AllEnts {
		ent.Think(g)
		pos := ent.Pos()
		pos.X = clamp(pos.X, 0, float64(g.Levels[ent.Level()].Room.Dx))
		pos.Y = clamp(pos.Y, 0, float64(g.Levels[ent.Level()].Room.Dy))
		ent.SetPos(pos)
	}

	for i := 0; i < len(g.temp.AllEnts); i++ {
		for j := i + 1; j < len(g.temp.AllEnts); j++ {
			outerEnt := g.temp.AllEnts[i]
			innerEnt := g.temp.AllEnts[j]
			distSq := outerEnt.Pos().Sub(innerEnt.Pos()).Mag2()
			colDist := outerEnt.Stats().Size() + innerEnt.Stats().Size()
			if distSq > colDist*colDist {
				continue
			}
			if distSq < 0.0001 {
				continue
			}
			if distSq <= 0.25 {
				distSq = 0.25
			}
			dist := math.Sqrt(distSq)
			force := 50.0 * (colDist - dist)
			outerEnt.ApplyForce(outerEnt.Pos().Sub(innerEnt.Pos()).Scale(force / dist))
			innerEnt.ApplyForce(innerEnt.Pos().Sub(outerEnt.Pos()).Scale(force / dist))
		}
	}

	switch {
	case g.Moba != nil:
		g.ThinkMoba()
	case g.Standard != nil:
		panic("Thinkgs aren't implemented, like thinking on mana sources")
		// Do standard thinking
	default:
		panic("Game mode not set")
	}
}

func (g *Game) ThinkMoba() {
	g.Levels[GidInvadersStart].ManaSource.Think(g.Ents)
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
	Engine *cgf.Engine
	Local  *LocalData
	Dims   gui.Dims
	game   *Game
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
	game := gw.Engine.GetState().(*Game)
	if game.Setup != nil {
		gw.Local.Setup(game)
	} else {
		gw.Local.Think(game)
	}
	gw.Engine.Unpause()
}
func (gw *GameWindow) Respond(group gin.EventGroup) {
}
func (gw *GameWindow) RequestedDims() gui.Dims {
	return gw.Dims
}

func (gw *GameWindow) Draw(region gui.Region, style gui.StyleStack) {
	defer base.StackCatcher()
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
	game := gw.Engine.GetState().(*Game)
	game.RenderLocal(region, gw.Local)
	gw.Engine.Unpause()
}
func (gw *GameWindow) DrawFocused(region gui.Region) {}
