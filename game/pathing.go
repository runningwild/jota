package game

import (
	"container/heap"
	"github.com/runningwild/jota/base"
	"github.com/runningwild/linear"
)

const pathingDataGrid = 64

type PathingData struct {
	// direction [srcX][srcY][dstX][dstY]
	dirs [][][][]pathingDataCell

	// List of directly connected cells for every position
	conns [][][]pathingConnection
}

type pathingDataCell struct {
	angle  float64
	direct bool
	filled bool
}

type pathingConnection struct {
	x, y int
	dist float64
}

type pathingNode struct {
	// every non-direct path will be src->intermediate1->intermediaten...->dst
	// originx,originy is the first intermediate cell
	originx, originy int

	dstx, dsty int
	dist       float64
}

// An pathHeap is a min-heap of pathingNode.
type pathHeap []pathingNode

func (h pathHeap) Len() int           { return len(h) }
func (h pathHeap) Less(i, j int) bool { return h[i].dist < h[j].dist }
func (h pathHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *pathHeap) Push(x interface{}) {
	// Push and Pop use pointer receivers because they modify the slice's length,
	// not just its contents.
	*h = append(*h, x.(pathingNode))
}

func (h *pathHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

func makePathingData(room *Room) *PathingData {
	var pd PathingData
	dx := room.Dx/pathingDataGrid + 1
	dy := room.Dy/pathingDataGrid + 1
	pd.dirs = make([][][][]pathingDataCell, dx)
	pd.conns = make([][][]pathingConnection, dx)
	for i := range pd.dirs {
		pd.dirs[i] = make([][][]pathingDataCell, dy)
		pd.conns[i] = make([][]pathingConnection, dy)
		for j := range pd.dirs[i] {
			pd.dirs[i][j] = make([][]pathingDataCell, dx)
			for k := range pd.dirs[i][j] {
				pd.dirs[i][j][k] = make([]pathingDataCell, dy)
			}
			pd.findAllDirectPaths(i, j, room)
		}
	}

	for i := range pd.dirs {
		for j := range pd.dirs[i] {
			pd.findAllPaths(i, j)
		}
	}
	// pd.findAllPaths(5, 12)
	base.Log().Printf("3,8->30,8: %v", pd.dirs[3][8][30][8])

	return &pd
}

func (pd *PathingData) findAllDirectPaths(srcx, srcy int, room *Room) {
	src := linear.Vec2{(float64(srcx) + 0.5) * pathingDataGrid, (float64(srcy) + 0.5) * pathingDataGrid}
	for x := range pd.dirs[srcx][srcy] {
		for y := range pd.dirs[srcx][srcy][x] {
			dst := linear.Vec2{(float64(x) + 0.5) * pathingDataGrid, (float64(y) + 0.5) * pathingDataGrid}
			if room.ExistsLos(src, dst) {
				pd.conns[srcx][srcy] = append(pd.conns[srcx][srcy], pathingConnection{
					x:    x,
					y:    y,
					dist: dst.Sub(src).Mag(),
				})
				data := &pd.dirs[srcx][srcy][x][y]
				data.angle = dst.Sub(src).Angle()
				data.direct = true
				// data.filled = true
			}
		}
	}
}

func (pd *PathingData) findAllPaths(srcx, srcy int) {
	paths := pd.dirs[srcx][srcy]
	var next pathHeap
	heap.Init(&next)
	for _, conn := range pd.conns[srcx][srcy] {
		if conn.x == srcx && conn.y == srcy {
			continue
		}
		heap.Push(&next, pathingNode{
			originx: conn.x,
			originy: conn.y,
			dstx:    conn.x,
			dsty:    conn.y,
			dist:    conn.dist,
		})
	}
	debug := srcx == 5 && srcy == 12
	debug = false
	if debug {
		base.Log().Printf("Starting pq len: %d", next.Len())
	}
	for next.Len() > 0 {
		node := heap.Pop(&next).(pathingNode)
		cell := &paths[node.dstx][node.dsty]
		if debug && node.dstx > 20 && node.dsty == srcy+1 {
			base.Log().Printf("Popping %v", node)
		}
		if cell.filled {
			continue
		}
		cell.filled = true
		if !cell.direct {
			if debug {
				base.Log().Printf("Setting angle: %v %v", float64(node.originx-srcx), float64(node.originy-srcy))
			}
			cell.angle = (linear.Vec2{float64(node.originx - srcx), float64(node.originy - srcy)}).Angle()
		}
		for _, conn := range pd.conns[node.dstx][node.dsty] {
			if paths[conn.x][conn.y].filled {
				continue
			}
			heap.Push(&next, pathingNode{
				originx: node.originx,
				originy: node.originy,
				dstx:    conn.x,
				dsty:    conn.y,
				dist:    node.dist + conn.dist,
			})
		}
	}
}

func (pd *PathingData) Dir(src, dst linear.Vec2) linear.Vec2 {
	x := int(src.X/pathingDataGrid + 0.5)
	y := int(src.Y/pathingDataGrid + 0.5)
	x2 := int(dst.X/pathingDataGrid + 0.5)
	y2 := int(dst.Y/pathingDataGrid + 0.5)
	cell := pd.dirs[x][y][x2][y2]
	if !cell.direct {
		return (linear.Vec2{1, 0}).Rotate(cell.angle)
	}
	return dst.Sub(src).Norm()
}
