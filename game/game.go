package game

import (
	"encoding/gob"
	"fmt"
	"github.com/runningwild/cgf"
	"github.com/runningwild/cmwc"
	"github.com/runningwild/glop/util/algorithm"
	"github.com/runningwild/jota/base"
	"github.com/runningwild/jota/champ"
	"github.com/runningwild/jota/stats"
	"github.com/runningwild/linear"
	"math"
	"path/filepath"
	"sync"
)

const LosMaxDist = 1000

type EffectMaker func(params map[string]float64) Process

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

type Thinker interface {
	Think(game *Game)

	// Kills a process.  Any Killed process will return PhaseComplete on any
	// future calls to Phase().
	Kill(game *Game)

	Dead() bool
}

type Process interface {
	Drain
	Thinker
	stats.Condition
	Draw(source, observer Gid, game *Game)
}

type Color int

const (
	ColorRed Color = iota
	ColorGreen
	ColorBlue
)

var AllColors = []Color{ColorRed, ColorGreen, ColorBlue}

type Ent interface {
	Draw(g *Game)
	Think(game *Game)
	ApplyForce(force linear.Vec2)

	// Stats based methods
	OnDeath(g *Game)
	Stats() *stats.Inst
	Mass() float64
	Vel() linear.Vec2
	Dead() bool

	// Suicide should circumvent all conditions/buffs/etc... and just kill the
	// ent.  After Suicide() all calls to Dead() should return true.
	Suicide()

	// For applying move events
	Move(angle, magnitude float64)

	Id() Gid
	SetId(Gid)
	Pos() linear.Vec2
	Angle() float64
	Side() int // which side the ent belongs to
	Type() EntType

	// Need to have a SetPos method because we don't want ents moving through
	// walls.
	SetPos(pos linear.Vec2)

	Supply(mana Mana) Mana

	Abilities() []Ability

	BindAi(name string, engine *cgf.Engine)
}

type EntType int

const (
	EntTypePlayer EntType = iota
	EntTypeControlPoint
	EntTypeObstacle
	EntTypeProjectile
	EntTypeCreep
	EntTypeOther
)

type NonManaUser struct{}

func (NonManaUser) Supply(mana Mana) Mana { return mana }

type Gid string

type Level struct {
	ManaSource ManaSource
	Room       Room
}

type SetupPlayerData struct {
	Side       int
	ChampIndex int
}

type localSetupData struct {
	// Position of the cursor.
	Index int

	// Event handling and engine thinking can happen concurrently, so we need to
	// be able to lock the local data.  Embedded for convenience.
	sync.RWMutex
}

// Used before the 'game' starts to choose sides and characters and whatnot.
type SetupData struct {
	EngineIds []int64                    // engine ids of the engines currently joined
	Players   map[int64]*SetupPlayerData // map from engineid to player data
	Seed      int64                      // random seed

	local localSetupData
}

type SetupSetEngineIds struct {
	EngineIds []int64
}

func (s SetupSetEngineIds) Apply(_g interface{}) {
	g := _g.(*Game)
	if g.Setup == nil {
		return
	}
	g.Manager = -1
	g.Setup.EngineIds = s.EngineIds
	for _, id := range g.Setup.EngineIds {
		if g.Manager == -1 || id < g.Manager {
			g.Manager = id
		}
		if _, ok := g.Setup.Players[id]; !ok {
			g.Setup.Players[id] = &SetupPlayerData{}
		}
	}
	if len(g.Setup.EngineIds) > len(s.EngineIds) {
		unused := make(map[int64]bool)
		for _, id := range g.Setup.EngineIds {
			unused[id] = true
		}
		for _, id := range s.EngineIds {
			delete(unused, id)
		}
		var usedEngineIds []int64
		for _, id := range g.Setup.EngineIds {
			if !unused[id] {
				usedEngineIds = append(usedEngineIds, id)
			}
		}
		g.Setup.EngineIds = usedEngineIds
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
	g.Setup.Players[s.EngineId].Side = s.Side
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
	sideData := g.Setup.Players[s.EngineId]
	if sideData == nil {
		return
	}
	sideData.ChampIndex += s.Champ
	if sideData.ChampIndex < 0 {
		sideData.ChampIndex = 0
	}
	if sideData.ChampIndex >= len(g.Champs) {
		sideData.ChampIndex = len(g.Champs) - 1
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
	sideCount := make(map[int]int)
	// Must have at least two sides
	sideCount[0] = 0
	sideCount[1] = 0
	for _, spd := range g.Setup.Players {
		sideCount[spd.Side]++
	}
	g.Engines = make(map[int64]*PlayerData)
	for id, player := range g.Setup.Players {
		var gid Gid
		if id < 0 {
			gid = Gid(fmt.Sprintf("Ai:%d", -id))
		} else {
			gid = Gid(fmt.Sprintf("Engine:%d", id))
		}
		g.Engines[id] = &PlayerData{
			PlayerGid:  Gid(gid),
			Side:       player.Side,
			ChampIndex: player.ChampIndex,
		}
	}

	// Now that we have the information we can set up a lot of the local data for
	// this engine's player.

	if g.IsPlaying() {
		g.local.Side = g.Engines[g.local.Engine.Id()].Side
		g.local.Gid = g.Engines[g.local.Engine.Id()].PlayerGid
		g.local.Data = g.Engines[g.local.Engine.Id()]
	}

	var room Room
	err := base.LoadJson(filepath.Join(base.GetDataDir(), "rooms/basic.json"), &room)
	if err != nil {
		base.Error().Fatalf("%v", err)
	}
	errs := room.Validate()
	for _, err := range errs {
		base.Error().Printf("%v", err)
	}
	if len(errs) > 0 {
		base.Error().Fatalf("Errors with the level, bailing...")
	}
	g.Level = &Level{}
	g.Level.Room = room
	g.Rng = cmwc.MakeGoodCmwc()
	g.Rng.Seed(u.Seed)
	g.Ents = make(map[Gid]Ent)
	g.Friction = 0.97
	g.losCache = makeLosCache(g.Level.Room.Dx, g.Level.Room.Dy)
	sides := make(map[int][]int64)
	var playerDatas []*PlayerData
	base.DoOrdered(g.Engines, func(a, b int64) bool { return a < b }, func(id int64, data *PlayerData) {
		sides[data.Side] = append(sides[data.Side], id)
		playerDatas = append(playerDatas, data)
	})
	for id, ed := range g.Engines {
		base.Log().Printf("%v -> %v", id, *ed)
	}
	g.AddPlayers(playerDatas)

	g.MakeControlPoints()
	g.Init()
	base.Log().Printf("Nillifying g.Setup()")
	g.Setup = nil
}
func init() {
	gob.Register(SetupComplete{})
}

type PlayerData struct {
	// If the player's Ent is in the game then its Gid will match this one.
	PlayerGid Gid

	// If positive, this is the number of frames remaining until the player
	// respawns.
	CountdownFrames int

	Side int

	// Index into the champs array of the champion that this player is using.
	ChampIndex int
}

// All of these values apply to the local player only
type localGameData struct {
	Engine    *cgf.Engine
	Camera    cameraInfo
	Gid       Gid
	Side      int
	Abilities []Ability

	// As long as we're using the keyboard, this is to make sure we track the
	// value of each key properly, so that you don't stop rotating left if you
	// release the right key, for example.
	Up, Down, Left, Right float64

	// This is just a convenience, it points to the PlayerData in
	// Game.Engines[thisEngineId]
	Data *PlayerData

	pathingData *PathingData

	// Event handling and engine thinking can happen concurrently, so we need to
	// be able to lock the local data.  Embedded for convenience.
	sync.RWMutex

	temp struct {
		// This include all room walls for each room, and all walls declared by any
		// ents in that room.
		AllWalls []linear.Seg2
		// It looks silly, but this is...
		// map[LevelId][x/cacheGridSize][y/cacheGridSize] slice of linear.Poly
		// AllWallsGrid  [][][][]linear.Vec2
		AllWallsDirty bool
		WallCache     *wallCache
		// VisibleWallCache is like WallCache but returns all segments visible from
		// a location, rather than all segments that should be checked for
		// collision.
		VisibleWallCache *wallCache

		// List of all ents, in the order that they should be iterated in.
		AllEnts      []Ent
		AllEntsDirty bool
		EntGrid      *entGridCache
	}
}

type Game struct {
	// Engine id of the engine managing the game - this is distinct from the host
	// who is just doing networking.
	Manager int64

	Setup *SetupData

	Level *Level

	Rng *cmwc.Cmwc

	Friction float64

	// Last Id assigned to anything
	NextIdValue int

	Ents map[Gid]Ent

	// List of data specific to players/computers
	Engines map[int64]*PlayerData

	// All effects that are not tied to a player.
	Processes []Process

	losCache *losCache

	// Champion defs loaded from the data file.  These are set by the host and
	// sent to clients to make debugging and tuning easier.
	Champs []champ.Champion

	local  localGameData
	editor editorData
}

func (g *Game) InitializeClientData() {
	// TODO: Do something useful with this.
}

// IsPlaying() returns true if this is a human player playing the game.  This
// functions returns false if this is a host binary that is simply connecting
// players together.
func (g *Game) IsPlaying() bool {
	return !g.local.Engine.IsHost() || g.IsManaging()
}

func (g *Game) IsManaging() bool {
	return g.local.Engine.Id() == g.Manager
}

func (g *Game) PathingData() *PathingData {
	return g.local.pathingData
}

func (g *Game) EntsInRange(pos linear.Vec2, dist float64) []Ent {
	g.local.RLock()
	defer g.local.RUnlock()
	var ents []Ent
	g.local.temp.EntGrid.EntsInRange(pos, dist, &ents)
	return ents
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
	g.Setup = &SetupData{}
	g.Setup.Players = make(map[int64]*SetupPlayerData)

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
		base.Log().Printf("Champ %v has %v", name, g.Champs[i].Abilities)
	}
	return &g
}

func (g *Game) Init() {
	msOptions := ManaSourceOptions{
		NumSeeds:    20,
		NumNodeRows: g.Level.Room.Dy / 32,
		NumNodeCols: g.Level.Room.Dx / 32,

		BoardLeft:   0,
		BoardTop:    0,
		BoardRight:  float64(g.Level.Room.Dx),
		BoardBottom: float64(g.Level.Room.Dy),

		MaxDrainDistance: 120.0,
		MaxDrainRate:     5.0,

		RegenPerFrame:     0.002,
		NodeMagnitude:     100,
		MinNodeBrightness: 20,
		MaxNodeBrightness: 150,

		Rng: g.Rng,
	}
	g.Level.ManaSource.Init(&msOptions)
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

func (g *Game) RemoveEnt(gid Gid) {
	delete(g.Ents, gid)
	g.local.temp.AllEntsDirty = true
}

func (g *Game) AddEnt(ent Ent) {
	if ent.Id() == "" {
		ent.SetId(g.NextGid())
	}
	g.Ents[ent.Id()] = ent
	g.local.temp.AllEntsDirty = true
}

// Give the gid of a Player ent (ai or engine) this will return that player's
// side, or -1 if such a player does not exist.
func (g *Game) GidToSide(gid Gid) int {
	for _, p := range g.Engines {
		if p.PlayerGid == gid {
			return p.Side
		}
	}
	return -1
}

func (g *Game) ThinkSetup() {
	g.local.Lock()
	defer g.local.Unlock()
	if g.local.Engine.IsHost() {
		// Update the list of ids in case it's changed
		ids := g.local.Engine.Ids()
		for i := range ids {
			if ids[i] == g.local.Engine.Id() {
				ids[i] = ids[len(ids)-1]
				ids = ids[0 : len(ids)-1]
				break
			}
		}
		g.local.Engine.ApplyEvent(SetupSetEngineIds{ids})
	} else if !g.IsManaging() {
		for i, v := range g.Setup.EngineIds {
			if v == g.local.Engine.Id() {
				g.Setup.local.Index = i
			}
		}
	}
}

func (g *Game) ThinkGame() {
	// cache wall data
	if g.local.temp.AllWalls == nil || g.local.temp.AllWallsDirty {
		g.local.temp.AllWallsDirty = false
		g.local.temp.AllWalls = nil
		g.local.temp.WallCache = nil
		g.local.temp.VisibleWallCache = nil

		// Can't use a nil slice, otherwise we'll run this block every Think for levels
		// with no walls.
		allWalls := make([]linear.Seg2, 0)
		base.DoOrdered(g.Level.Room.Walls, func(a, b string) bool { return a < b }, func(_ string, walls linear.Poly) {
			for i := range walls {
				allWalls = append(allWalls, walls.Seg(i))
			}
		})
		g.local.temp.AllWalls = allWalls
		g.local.temp.WallCache = &wallCache{}
		g.local.temp.WallCache.SetWalls(g.Level.Room.Dx, g.Level.Room.Dy, allWalls, 100)
		g.local.temp.VisibleWallCache = &wallCache{}
		g.local.temp.VisibleWallCache.SetWalls(g.Level.Room.Dx, g.Level.Room.Dy, allWalls, stats.LosPlayerHorizon)
		g.local.pathingData = makePathingData(&g.Level.Room)
	}

	// cache ent data
	for _, ent := range g.local.temp.AllEnts {
		if ent.Dead() {
			if _, ok := ent.(*PlayerEnt); ok {
				var id int64
				_, err := fmt.Sscanf(string(ent.Id()), "Engine:%d", &id)
				if err != nil {
					_, err = fmt.Sscanf(string(ent.Id()), "Ai:%d", &id)
					id = -id // Ai's engine ids are negative
				}
				if err != nil {
					base.Error().Printf("Unable to parse player id '%v'", ent.Id())
				} else {
					if engineData, ok := g.Engines[id]; ok {
						if !ok {
							base.Error().Printf("Unable to find engine %d for player %v", id, ent.Id())
						} else {
							engineData.CountdownFrames = 60 * 10
							base.Log().Printf("%v died, counting down....", *engineData)
						}
					}
				}
			}
			ent.OnDeath(g)
			g.RemoveEnt(ent.Id())
		}
	}

	// Death countdown
	base.DoOrdered(g.Engines, func(a, b int64) bool { return a < b }, func(_ int64, engineData *PlayerData) {
		if engineData.CountdownFrames > 0 {
			engineData.CountdownFrames--
			if engineData.CountdownFrames == 0 {
				g.AddPlayers([]*PlayerData{engineData})
			}
		}
	})

	if g.local.temp.AllEnts == nil || g.local.temp.AllEntsDirty {
		g.local.temp.AllEnts = g.local.temp.AllEnts[0:0]
		g.DoForEnts(func(gid Gid, ent Ent) {
			g.local.temp.AllEnts = append(g.local.temp.AllEnts, ent)
		})
		g.local.temp.AllEntsDirty = false
	}
	if g.local.temp.EntGrid == nil {
		g.local.temp.EntGrid = MakeEntCache(g.Level.Room.Dx, g.Level.Room.Dy)
	}
	g.local.temp.EntGrid.SetEnts(g.local.temp.AllEnts)

	for _, proc := range g.Processes {
		proc.Think(g)
	}
	algorithm.Choose(&g.Processes, func(proc Process) bool { return !proc.Dead() })

	// Advance players, check for collisions, add segments
	eps := 1.0e-3
	for _, ent := range g.local.temp.AllEnts {
		ent.Think(g)
		for _, ab := range ent.Abilities() {
			ab.Think(ent, g)
		}
		pos := ent.Pos()
		pos.X = clamp(pos.X, eps, float64(g.Level.Room.Dx)-eps)
		pos.Y = clamp(pos.Y, eps, float64(g.Level.Room.Dy)-eps)
		ent.SetPos(pos)
	}

	var nearby []Ent
	for i := 0; i < len(g.local.temp.AllEnts); i++ {
		outerEnt := g.local.temp.AllEnts[i]
		outerSize := outerEnt.Stats().Size()
		if outerSize == 0 {
			continue
		}
		g.local.temp.EntGrid.EntsInRange(outerEnt.Pos(), 100, &nearby)
		for _, innerEnt := range nearby {
			innerSize := innerEnt.Stats().Size()
			if innerSize == 0 {
				continue
			}
			distSq := outerEnt.Pos().Sub(innerEnt.Pos()).Mag2()
			colDist := innerSize + outerSize
			if distSq > colDist*colDist {
				continue
			}
			if distSq < 0.015625 { // this means that dist < 0.125
				continue
			}
			dist := math.Sqrt(distSq)
			force := 50.0 * (colDist - dist)
			innerEnt.ApplyForce(innerEnt.Pos().Sub(outerEnt.Pos()).Scale(force / dist))
		}
	}

	g.Level.ManaSource.Think(g.Ents)
}

func (g *Game) Think() {
	defer base.StackCatcher()
	switch {
	case g.Setup != nil:
		g.ThinkSetup()
	default:
		g.ThinkGame()
	}
}

// Returns true iff a has los to b, regardless of distance, except that nothing
// can ever have los to something that is beyond stats.LosPlayerHorizon.
func (g *Game) ExistsLos(a, b linear.Vec2) bool {
	return g.Level.Room.ExistsLos(a, b)
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

type Move struct {
	Gid       Gid
	Angle     float64
	Magnitude float64
}

func init() {
	gob.Register(Move{})
}

func (m Move) Apply(_g interface{}) {
	g := _g.(*Game)
	ent := g.Ents[m.Gid]
	if ent == nil {
		return
	}
	if ent.Id() != m.Gid {
		base.Error().Printf("Move.Apply(): %v %v")
	}
	ent.Move(m.Angle, m.Magnitude)
}

// Utils

func expandPoly(in linear.Poly, out *linear.Poly) {
	if len(*out) < len(in) {
		*out = make(linear.Poly, len(in))
	}
	for i := range *out {
		(*out)[i] = linear.Vec2{}
	}
	for i, v := range in {
		segi := in.Seg(i)
		(*out)[i] = (*out)[i].Add(v.Add(segi.Ray().Cross().Norm().Scale(8.0)))
		j := (i - 1 + len(in)) % len(in)
		segj := in.Seg(j)
		(*out)[i] = (*out)[i].Add(v.Add(segj.Ray().Cross().Norm().Scale(8.0)))
	}
	for i := range *out {
		(*out)[i] = (*out)[i].Scale(0.5)
	}
}
