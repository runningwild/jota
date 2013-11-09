package game

import (
	"github.com/runningwild/linear"
	"sort"
)

type entsByGid []Ent

func (ebg entsByGid) Len() int           { return len(ebg) }
func (ebg entsByGid) Less(i, j int) bool { return ebg[i].Id() < ebg[j].Id() }
func (ebg entsByGid) Swap(i, j int)      { ebg[i], ebg[j] = ebg[j], ebg[i] }

const entGridSize = 64

type entGridCache struct {
	grid   [][][]Ent
	dx, dy int
}

func MakeEntCache(dx, dy int) *entGridCache {
	var cache entGridCache
	cache.dx = (dx + entGridSize - 1) / entGridSize
	cache.dy = (dy + entGridSize - 1) / entGridSize
	cache.grid = make([][][]Ent, cache.dx)
	for i := range cache.grid {
		cache.grid[i] = make([][]Ent, cache.dy)
	}
	return &cache
}

func (cache *entGridCache) SetEnts(ents []Ent) {
	for i := range cache.grid {
		for j := range cache.grid[i] {
			cache.grid[i][j] = cache.grid[i][j][0:0]
		}
	}
	for _, ent := range ents {
		x := int(ent.Pos().X+entGridSize/2) / entGridSize
		y := int(ent.Pos().Y+entGridSize/2) / entGridSize
		if x < 0 || x >= cache.dx || y < 0 || y >= cache.dy {
			continue
		}
		cache.grid[x][y] = append(cache.grid[x][y], ent)
	}
}

func (cache *entGridCache) EntsInRange(pos linear.Vec2, dist float64, ents *[]Ent) {
	*ents = (*ents)[0:0]
	gridDist := 2 + int(dist/entGridSize)
	minx := int(pos.X)/entGridSize - gridDist
	if minx < 0 {
		minx = 0
	}
	maxx := int(pos.X)/entGridSize + gridDist
	if maxx > cache.dx {
		maxx = cache.dx
	}
	miny := int(pos.Y)/entGridSize - gridDist
	if miny < 0 {
		miny = 0
	}
	maxy := int(pos.Y)/entGridSize + gridDist
	if maxy > cache.dy {
		maxy = cache.dy
	}
	for x := minx; x < maxx; x++ {
		for y := miny; y < maxy; y++ {
			for _, ent := range cache.grid[x][y] {
				if pos.Sub(ent.Pos()).Mag()-ent.Stats().Size() <= dist {
					*ents = append(*ents, ent)
				}
			}
		}
	}
	sort.Sort(entsByGid(*ents))
}
