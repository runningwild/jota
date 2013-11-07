package game

import (
	"github.com/runningwild/linear"
)

const wallGridSize = 100

type wallCache struct {
	walls [][][]linear.Seg2
}

func (wc *wallCache) GetWalls(x, y int) []linear.Seg2 {
	if len(wc.walls) == 0 {
		return nil
	}
	x /= wallGridSize
	y /= wallGridSize
	if x < 0 {
		x = 0
	}
	if x >= len(wc.walls) {
		x = len(wc.walls) - 1
	}
	if len(wc.walls[x]) == 0 {
		return nil
	}
	if y < 0 {
		y = 0
	}
	if y >= len(wc.walls[x]) {
		y = len(wc.walls[x]) - 1
	}
	return wc.walls[x][y]
}

func (wc *wallCache) SetWalls(dx, dy int, walls []linear.Seg2, dist int) {
	if len(walls) == 0 {
		wc.walls = nil
		return
	}
	dxGrid := dx/wallGridSize + 1
	dyGrid := dy/wallGridSize + 1
	rawWalls := make([][]linear.Seg2, dxGrid*dyGrid)
	wc.walls = make([][][]linear.Seg2, dxGrid)
	for i := range wc.walls {
		wc.walls[i] = rawWalls[i*dyGrid : (i+1)*dyGrid]
	}
	buffer := dist/wallGridSize + 1
	for _, wall := range walls {
		x0 := int(wall.P.X / wallGridSize)
		y0 := int(wall.P.Y / wallGridSize)
		x1 := int(wall.Q.X / wallGridSize)
		y1 := int(wall.Q.Y / wallGridSize)
		if x0 > x1 {
			x0, x1 = x1, x0
		}
		if y0 > y1 {
			y0, y1 = y1, y0
		}
		x0 -= buffer
		x1 += buffer
		y0 -= buffer
		y1 += buffer
		if x0 < 0 {
			x0 = 0
		}
		if x1 >= len(wc.walls) {
			x1 = len(wc.walls) - 1
		}
		if y0 < 0 {
			y0 = 0
		}
		if y1 >= len(wc.walls[0]) {
			y1 = len(wc.walls[0]) - 1
		}
		for x := x0; x <= x1; x++ {
			for y := y0; y <= y1; y++ {
				// if wall.Right(linear.Vec2{float64(x * wallGridSize), float64(y * wallGridSize)}) {
				// 	continue
				// }
				wc.walls[x][y] = append(wc.walls[x][y], wall)
			}
		}
	}
}
