package game

import (
	"github.com/runningwild/linear"
	"github.com/runningwild/magnus/los"
	"github.com/runningwild/magnus/stats"
	"math"
	"sort"
	"sync"
)

const LosGridSize = 16

type losCache struct {
	losBuffers      []*los.Los
	losBuffersMutex sync.Mutex

	cache      map[losCacheViewerPos][]visiblePos
	cacheMutex sync.Mutex

	// These values are read-only after creation so not mutexes are needed.
	wallCache *wallCache
	dx, dy    int
}

func makeLosCache(dx, dy int) *losCache {
	var lc losCache
	lc.cache = make(map[losCacheViewerPos][]visiblePos)
	lc.wallCache = &wallCache{}
	lc.dx = dx / LosGridSize
	lc.dy = dy / LosGridSize
	return &lc
}

type losCacheViewerPos struct {
	X, Y int
}

type visiblePos struct {
	X, Y int
	Val  byte
	Dist float64
}

type visiblePosInternal struct {
	X, Y  int
	Dist  float64
	Angle float64
	Index int
}

type visiblePosInternalSlice []visiblePosInternal

func (vps visiblePosInternalSlice) Len() int           { return len(vps) }
func (vps visiblePosInternalSlice) Less(i, j int) bool { return vps[i].Dist < vps[j].Dist }
func (vps visiblePosInternalSlice) Swap(i, j int)      { vps[i], vps[j] = vps[j], vps[i] }

func (lc *losCache) SetWallCache(wc *wallCache) {
	lc.wallCache = wc
	lc.cache = make(map[losCacheViewerPos][]visiblePos)
}

// Includes all possible visiblePosInternal values in order of distance
var maxVps visiblePosInternalSlice

// Set up maxVps
func init() {
	max := stats.LosPlayerHorizon / LosGridSize
	for x := -max; x <= max; x++ {
		for y := -max; y <= max; y++ {
			dist := math.Sqrt(float64(x*x + y*y))
			if dist > float64(max) {
				continue
			}
			angle := math.Atan2(float64(y), float64(x))
			maxVps = append(maxVps, visiblePosInternal{
				X:     x,
				Y:     y,
				Dist:  dist,
				Angle: angle,
				Index: int(((angle/(2*math.Pi))+0.5)*float64(los.Resolution)) % los.Resolution,
			})
		}
	}
	sort.Sort(maxVps)
}

// losCache.Get() is Thread-Safe
func (lc *losCache) Get(i, j int, maxDist float64) []visiblePos {
	vp := losCacheViewerPos{i, j}
	var vps []visiblePos
	var ok bool
	lc.cacheMutex.Lock()
	vps, ok = lc.cache[vp]
	lc.cacheMutex.Unlock()
	if ok {
		var max int
		for max = 0; max < len(vps) && vps[max].Dist <= maxDist; max++ {
		}
		return vps[0:max]
	}

	lc.losBuffersMutex.Lock()
	var losBuffer *los.Los
	if len(lc.losBuffers) > 0 {
		losBuffer = lc.losBuffers[0]
		lc.losBuffers = lc.losBuffers[1:]
	} else {
		losBuffer = los.Make(stats.LosPlayerHorizon)
	}
	lc.losBuffersMutex.Unlock()

	pos := linear.Vec2{float64(i) + 0.5, float64(j) + 0.5}
	losBuffer.Reset(pos)
	vps = make([]visiblePos, len(maxVps))[0:0]
	for _, wall := range lc.wallCache.GetWalls(i, j) {
		mid := wall.P.Add(wall.Q).Scale(0.5)
		if mid.Sub(pos).Mag() < stats.LosPlayerHorizon+wall.Ray().Mag() {
			losBuffer.DrawSeg(wall, "")
		}
	}
	ix := int(pos.X / LosGridSize)
	iy := int(pos.Y / LosGridSize)
	for _, v := range maxVps {
		x := ix + v.X
		y := iy + v.Y
		if x < 0 || x >= lc.dx {
			continue
		}
		if y < 0 || y >= lc.dy {
			continue
		}
		raw := losBuffer.RawAccess()
		val := 255.0
		dist := v.Dist * LosGridSize
		distSq := dist * dist
		if distSq < float64(raw[v.Index]) {
			val = 0
		} else if distSq < float64(raw[(v.Index+1)%len(raw)]) ||
			distSq < float64(raw[(v.Index+len(raw)-1)%len(raw)]) {
			val = 100
		} else if distSq < float64(raw[(v.Index+2)%len(raw)]) ||
			distSq < float64(raw[(v.Index+len(raw)-2)%len(raw)]) {
			val = 200
		}
		fade := 100.0
		if dist > stats.LosPlayerHorizon-fade {
			val = 255 - (255-val)*(1.0-(fade-(stats.LosPlayerHorizon-dist))/fade)
		}
		if val < 255 {
			vps = append(vps, visiblePos{
				X:    x,
				Y:    y,
				Val:  byte(val),
				Dist: dist,
			})
		}
	}
	lc.cacheMutex.Lock()
	lc.cache[vp] = vps
	lc.cacheMutex.Unlock()

	lc.losBuffersMutex.Lock()
	lc.losBuffers = append(lc.losBuffers, losBuffer)
	lc.losBuffersMutex.Unlock()

	// Call this function again because now the values are cached.  We do this
	// because we also need to slice the return value appropriately for maxDist
	// and shouldn't have that logic in two places.
	return lc.Get(i, j, maxDist)
}
