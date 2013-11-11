package game

import (
	"github.com/runningwild/linear"
)

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
		if x < 0 {
			x = 0
		}
		if x >= cache.dx {
			x = cache.dx - 1
		}
		if y < 0 {
			y = 0
		}
		if y >= cache.dy {
			y = cache.dy - 1
		}
		cache.grid[x][y] = append(cache.grid[x][y], ent)
	}
}

func (cache *entGridCache) EntsInRange(pos linear.Vec2, dist float64, ents *[]Ent) {
	*ents = (*ents)[0:0]
	gridDist := 1 + int(dist/entGridSize)
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
				*ents = append(*ents, ent)
			}
		}
	}
}
