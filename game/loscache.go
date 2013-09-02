package game

import (
	"github.com/runningwild/linear"
	"github.com/runningwild/magnus/los"
	"github.com/runningwild/magnus/stats"
	"math"
	"sort"
)

type losCache struct {
	losBuffer *los.Los
	wallCache *wallCache
	cache     map[losCacheViewerPos]visiblePosSlice
	dx, dy    int
}

func makeLosCache(dx, dy int) *losCache {
	var lc losCache
	lc.losBuffer = los.Make(stats.LosPlayerHorizon)
	lc.cache = make(map[losCacheViewerPos]visiblePosSlice)
	lc.wallCache = &wallCache{}
	lc.dx = dx
	lc.dy = dy
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

type visiblePosSlice []visiblePos

func (vps visiblePosSlice) Len() int           { return len(vps) }
func (vps visiblePosSlice) Less(i, j int) bool { return vps[i].Dist < vps[j].Dist }
func (vps visiblePosSlice) Swap(i, j int)      { vps[i], vps[j] = vps[j], vps[i] }

func (lc *losCache) SetWallCache(wc *wallCache) {
	lc.wallCache = wc
	lc.cache = make(map[losCacheViewerPos]visiblePosSlice)
}

func (lc *losCache) Get(i, j int, maxDist float64) []visiblePos {
	vp := losCacheViewerPos{i, j}
	if vps, ok := lc.cache[vp]; ok {
		var max int
		for max = 0; max < len(vps) && vps[max].Dist <= maxDist; max++ {
		}
		return vps[0:max]
	}
	pos := linear.Vec2{float64(i), float64(j)}
	lc.losBuffer.Reset(pos)
	var vps visiblePosSlice
	for _, wall := range lc.wallCache.GetWalls(i, j) {
		mid := wall.P.Add(wall.Q).Scale(0.5)
		if mid.Sub(pos).Mag() < stats.LosPlayerHorizon+wall.Ray().Mag() {
			lc.losBuffer.DrawSeg(wall, "")
		}
	}
	dx0 := (int(pos.X+0.5) - stats.LosPlayerHorizon) / LosGridSize
	dx1 := (int(pos.X+0.5) + stats.LosPlayerHorizon) / LosGridSize
	dy0 := (int(pos.Y+0.5) - stats.LosPlayerHorizon) / LosGridSize
	dy1 := (int(pos.Y+0.5) + stats.LosPlayerHorizon) / LosGridSize
	for x := dx0; x <= dx1; x++ {
		if x < 0 || x >= lc.dx {
			continue
		}
		for y := dy0; y <= dy1; y++ {
			if y < 0 || y >= lc.dy {
				continue
			}
			seg := linear.Seg2{
				pos,
				linear.Vec2{(float64(x) + 0.5) * LosGridSize, (float64(y) + 0.5) * LosGridSize},
			}
			dist2 := seg.Ray().Mag2()
			if dist2 > stats.LosPlayerHorizon*stats.LosPlayerHorizon {
				continue
			}
			raw := lc.losBuffer.RawAccess()
			angle := math.Atan2(seg.Ray().Y, seg.Ray().X)
			index := int(((angle/(2*math.Pi))+0.5)*float64(len(raw))) % len(raw)
			if dist2 < stats.LosPlayerHorizon*stats.LosPlayerHorizon {
				val := 255.0
				if dist2 < float64(raw[index]) {
					val = 0
				} else if dist2 < float64(raw[(index+1)%len(raw)]) ||
					dist2 < float64(raw[(index+len(raw)-1)%len(raw)]) {
					val = 100
				} else if dist2 < float64(raw[(index+2)%len(raw)]) ||
					dist2 < float64(raw[(index+len(raw)-2)%len(raw)]) {
					val = 200
				}
				fade := 100.0
				if dist2 > (stats.LosPlayerHorizon-fade)*(stats.LosPlayerHorizon-fade) {
					val = 255 - (255-val)*(1.0-(fade-(stats.LosPlayerHorizon-math.Sqrt(dist2)))/fade)
				}
				if val < 255 {
					vps = append(vps, visiblePos{
						X:    x,
						Y:    y,
						Val:  byte(val),
						Dist: math.Sqrt(dist2),
					})
				}
			}
		}
	}
	sort.Sort(vps)
	lc.cache[vp] = vps
	// Call this function again because now the values are cached.  We do this
	// because we also need to slice the return value appropriately for maxDist
	// and shouldn't have that logic in two places.
	return lc.Get(i, j, maxDist)
}
