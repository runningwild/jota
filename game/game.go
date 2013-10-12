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
	"github.com/runningwild/magnus/champ"
	"github.com/runningwild/magnus/generator"
	"github.com/runningwild/magnus/gui"
	"github.com/runningwild/magnus/stats"
	"github.com/runningwild/magnus/texture"
	"math"
	"path/filepath"
)

// type Ability func(game *Game, player *PlayerEnt, params map[string]int) Process

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

type EffectMaker func(params map[string]int) Process

var effect_makers map[string]EffectMaker

func RegisterEffect(name string, maker EffectMaker) {
	if effect_makers == nil {
		effect_makers = make(map[string]EffectMaker)
	}
	effect_makers[name] = maker
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

	// Kills a process.  Any Killed process will return PhaseComplete on any
	// future calls to Phase().
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

var AllColors = []Color{ColorRed, ColorGreen, ColorBlue}

func init() {
}

type PlayerEnt struct {
	BaseEnt
	Champ int
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
		var p PlayerEnt
		p.StatsInst = stats.Make(stats.Base{
			Health: 1000,
			Mass:   750,
			Acc:    1000.0,
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
	gob.Register(&PlayerEnt{})
}

// func (p *PlayerEnt) ReleaseResources() {
// 	p.Los.ReleaseResources()
// }

func (p *PlayerEnt) Draw(game *Game, side int) {
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

func (p *PlayerEnt) Think(g *Game) {
	p.BaseEnt.Think(g)
}

func (p *PlayerEnt) Supply(supply Mana) Mana {
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
	Dead() bool

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

type SetupSideData struct {
	Side  int
	Champ int
}

// Used before the 'game' starts to choose sides and characters and whatnot.
type Setup struct {
	Mode      string                   // should be "moba" or "standard"
	EngineIds []int64                  // engine ids of the engines currently joined
	Sides     map[int64]*SetupSideData // map from engineid to side data
	Seed      int64                    // random seed
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
			g.Setup.Sides[id] = &SetupSideData{}
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
	g.Setup.Sides[s.EngineId].Side = s.Side
}
func init() {
	gob.Register(SetupChangeSides{})
}

type SetupChampSelect struct {
	EngineId int64
	Champ    int
}

func init() {
	gob.Register(SetupChampSelect{})
}
func (s SetupChampSelect) Apply(_g interface{}) {
	g := _g.(*Game)
	if g.Setup == nil {
		return
	}
	sideData := g.Setup.Sides[s.EngineId]
	if sideData == nil {
		return
	}
	sideData.Champ += s.Champ
	if sideData.Champ < 0 {
		sideData.Champ = 0
	}
	if sideData.Champ >= len(g.Champs) {
		sideData.Champ = len(g.Champs) - 1
	}
}

type SetupComplete struct {
	Seed int64
}

func (u SetupComplete) Apply(_g interface{}) {
	g := _g.(*Game)
	if g.Setup == nil {
		return
	}

	g.Engines = make(map[int64]*PlayerData)
	for _, id := range g.Setup.EngineIds {
		g.Engines[id] = &PlayerData{
			PlayerGid: Gid(fmt.Sprintf("Engine:%d", id)),
			Side:      g.Setup.Sides[id].Side,
		}
	}

	var room Room
	dx, dy := 1024, 1024
	generated := generator.GenerateRoom(float64(dx), float64(dy), 100, 64, u.Seed)
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
	for engineId, sideData := range g.Setup.Sides {
		sides[sideData.Side] = append(sides[sideData.Side], engineId)
	}
	for side, ids := range sides {
		gids := g.AddPlayers(ids, side)
		g.Moba.Sides[side] = &GameModeMobaSideData{}
		for i := range ids {
			player := g.Ents[gids[i]].(*PlayerEnt)
			player.Champ = g.Setup.Sides[ids[i]].Champ
		}
	}
	g.Moba.losCache = makeLosCache(dx, dy)

	g.MakeControlPoints()
	g.Init()
	base.Log().Printf("Nillifying g.Setup()")
	g.Setup = nil
}
func init() {
	gob.Register(SetupComplete{})
}

type PlayerData struct {
	PlayerGid Gid

	// If positive, this is the number of frames remaining until the player
	// respawns.
	CountdownFrames int

	Side int

	// If this is an ai controlled player then this will be non-nil.
	Ai *AiPlayerData
}

type AiPlayerData struct {
	N int
}

type Game struct {
	Setup *Setup

	Levels map[Gid]*Level

	Rng *cmwc.Cmwc

	Friction float64

	// Last Id assigned to anything
	NextIdValue int

	Ents map[Gid]Ent

	// List of data specific to players/computers
	Engines map[int64]*PlayerData

	GameThinks int

	// All effects that are not tied to a player.
	Processes []Process

	// Game Modes - Exactly one of these will be set
	Standard *GameModeStandard
	Moba     *GameModeMoba

	// Champion defs loaded from the data file.  These are set by the host and
	// sent to clients to make debugging and tuning easier.
	Champs []champ.Champion

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

func (g *Game) InitializeClientData() {
	// TODO: Do something useful with this.
}

type GameModeStandard struct {
	Architect architectData
	Invaders  invadersData
}
type GameModeMoba struct {
	// Map from side to the moba data for that side
	Sides    map[int]*GameModeMobaSideData
	losCache *losCache
}
type GameModeMobaSideData struct {
	AppeaseGob struct{}
}

func (g *Game) NextGid() Gid {
	g.NextIdValue++
	return Gid(fmt.Sprintf("%d", g.NextIdValue))
}

func (g *Game) NextId() int {
	g.NextIdValue++
	return g.NextIdValue
}

func MakeGame() *Game {
	var g Game
	g.Setup = &Setup{}
	g.Setup.Mode = "moba"
	g.Setup.Sides = make(map[int64]*SetupSideData)

	// NOTE: Obviously this isn't threadsafe, but I don't intend to be Init()ing
	// multiple game objects at the same time.
	base.RemoveRegistry("champs")
	base.RegisterRegistry("champs", make(map[string]*champ.ChampionDef))
	base.RegisterAllObjectsInDir("champs", filepath.Join(base.GetDataDir(), "champs"), ".json", "json")

	names := base.GetAllNamesInRegistry("champs")
	g.Champs = make([]champ.Champion, len(names))
	for i, name := range names {
		g.Champs[i].Defname = name
		base.GetObject("champs", &g.Champs[i])
	}

	return &g
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
	// Values less than this might be used for ability processes, ect...
	g.NextIdValue = 10000
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
		g.Moba.losCache.SetWallCache(g.temp.VisibleWallCache[GidInvadersStart])
	}

	// cache ent data
	for _, ent := range g.temp.AllEnts {
		if ent.Dead() {
			if _, ok := ent.(*PlayerEnt); ok {
				var id int64
				_, err := fmt.Sscanf(string(ent.Id()), "Engine:%d", &id)
				if err != nil {
					base.Error().Printf("Unable to parse player id '%v'", ent.Id())
				} else {
					if engineData, ok := g.Engines[id]; ok {
						if !ok {
							base.Error().Printf("Unable to find engine %d for player %v", id, ent.Id())
						} else {
							engineData.CountdownFrames = 60 * 10
						}
					}
				}
			}
			ent.OnDeath(g)
			g.RemoveEnt(ent.Id())
		}
	}

	// Death countdown
	for engineId, engineData := range g.Engines {
		if engineData.CountdownFrames > 0 {
			engineData.CountdownFrames--
			if engineData.CountdownFrames == 0 {
				// TODO: It's a bit janky to do it like this, right?
				g.AddPlayers([]int64{engineId}, engineData.Side)
			}
		}
	}

	if g.temp.AllEnts == nil || g.temp.AllEntsDirty {
		g.temp.AllEnts = g.temp.AllEnts[0:0]
		g.DoForEnts(func(gid Gid, ent Ent) {
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
		eps := 1.0e-3
		pos.X = clamp(pos.X, eps, float64(g.Levels[ent.Level()].Room.Dx)-eps)
		pos.Y = clamp(pos.Y, eps, float64(g.Levels[ent.Level()].Room.Dy)-eps)
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

// Returns true iff a has los to b, regardless of distance, except that nothing
// can ever have los to something that is beyond stats.LosPlayerHorizon.
func (g *Game) ExistsLos(a, b linear.Vec2) bool {
	if g.Moba == nil {
		panic("Not implemented except in mobas")
	}
	vps := g.Moba.losCache.Get(int(a.X), int(a.Y), stats.LosPlayerHorizon)
	x := int(b.X / LosGridSize)
	y := int(b.Y / LosGridSize)
	for _, vp := range vps {
		if vp.X == x && vp.Y == y {
			return true
		}
	}
	return false
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
	player, ok := g.Ents[t.Gid].(*PlayerEnt)
	if !ok {
		return
	}
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
	player, ok := g.Ents[a.Gid].(*PlayerEnt)
	if !ok {
		return
	}
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
