package game

import (
	"github.com/runningwild/jota/base"
	"github.com/runningwild/linear"
	"sync"
)

const pathingDataGrid = 128

type PathingData struct {
	sync.RWMutex

	// direction [srcX][srcY][dstX][dstY]
	dirs [][][][]pathingDataCell

	// List of directly connected cells for every position
	conns [][][]pathingConnection

	// Fine grained locking so that we can ask to compute the paths from a single
	// source and retrieve the results later.
	srcData [][]pathingSrcData
}

type pathingSrcData struct {
	sync.RWMutex
	once     sync.Once
	complete bool
}

type pathingDataCell struct {
	angle  float64
	direct bool
	filled bool
	dist   float64
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

func makePathingData(room *Room) *PathingData {
	var pd PathingData
	dx := (room.Dx + pathingDataGrid - 1) / pathingDataGrid
	dy := (room.Dy + pathingDataGrid - 1) / pathingDataGrid
	pd.dirs = make([][][][]pathingDataCell, dx)
	pd.conns = make([][][]pathingConnection, dx)
	pd.srcData = make([][]pathingSrcData, dx)
	for i := range pd.dirs {
		pd.dirs[i] = make([][][]pathingDataCell, dy)
		pd.conns[i] = make([][]pathingConnection, dy)
		pd.srcData[i] = make([]pathingSrcData, dy)
		for j := range pd.dirs[i] {
			pd.dirs[i][j] = make([][]pathingDataCell, dx)
			for k := range pd.dirs[i][j] {
				pd.dirs[i][j][k] = make([]pathingDataCell, dy)
			}
			pd.findAllDirectPaths(i, j, room)
		}
	}

	return &pd
}

func (pd *PathingData) findAllDirectPaths(srcx, srcy int, room *Room) {
	src := linear.Vec2{(float64(srcx) + 0.5) * pathingDataGrid, (float64(srcy) + 0.5) * pathingDataGrid}
	for x := range pd.dirs[srcx][srcy] {
		for y := range pd.dirs[srcx][srcy][x] {
			dst := linear.Vec2{(float64(x) + 0.5) * pathingDataGrid, (float64(y) + 0.5) * pathingDataGrid}
			if srcx == 1 && srcy == 4 {
			}
			if room.ExistsLos(src, dst) {
				if srcx == 1 && srcy == 4 {
				}
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
	for _, conn := range pd.conns[srcx][srcy] {
		if conn.x == srcx && conn.y == srcy {
			continue
		}
		next.Push(pathingNode{
			originx: conn.x,
			originy: conn.y,
			dstx:    conn.x,
			dsty:    conn.y,
			dist:    conn.dist,
		})
		paths[conn.x][conn.y].dist = conn.dist
	}
	debug := srcx == 1 && srcy == 4 && false
	if debug {
	}
	for len(next) > 0 {
		if debug {
		}
		node := next.Pop()
		if debug {
		}
		cell := &paths[node.dstx][node.dsty]
		if cell.filled {
			if debug {
			}
			continue
		}
		cell.filled = true
		if !cell.direct {
			if debug {
			}
			cell.angle = (linear.Vec2{float64(node.originx - srcx), float64(node.originy - srcy)}).Angle()
		}
		for _, conn := range pd.conns[node.dstx][node.dsty] {
			if paths[conn.x][conn.y].filled {
				continue
			}
			next.Push(pathingNode{
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
	x := int(src.X / pathingDataGrid)
	y := int(src.Y / pathingDataGrid)
	x2 := int(dst.X / pathingDataGrid)
	y2 := int(dst.Y / pathingDataGrid)
	if x < 0 || y < 0 || x >= len(pd.srcData) || y >= len(pd.srcData[x]) {
		return linear.Vec2{0, 0}
	}
	srcData := &pd.srcData[x][y]
	srcData.RLock()
	defer srcData.RUnlock()
	if !srcData.complete {
		srcData.once.Do(func() {
			go func() {
				srcData.Lock()
				defer srcData.Unlock()
				pd.findAllPaths(x, y)
				base.Log().Printf("Computed %d %d", x, y)
				srcData.complete = true
			}()
		})
		return dst.Sub(src).Norm()
	}
	cell := pd.dirs[x][y][x2][y2]
	if !cell.direct {
		return (linear.Vec2{1, 0}).Rotate(cell.angle)
	}
	return dst.Sub(src).Norm()
}

// An pathHeap is a min-heap of pathingNode.
type pathHeap []pathingNode

// Push pushes the element x onto the heap. The complexity is
// O(log(n)) where n = h.Len().
//
func (h *pathHeap) Push(node pathingNode) {
	*h = append(*h, node)
	h.up(len(*h) - 1)
}

// Pop removes the minimum element (according to Less) from the heap
// and returns it. The complexity is O(log(n)) where n = h.Len().
// Same as Remove(h, 0).
//
func (h *pathHeap) Pop() pathingNode {
	n := len(*h) - 1
	(*h)[0], (*h)[n] = (*h)[n], (*h)[0]
	h.down(0, n)
	r := (*h)[n]
	*h = (*h)[0:n]
	return r
}

func (h *pathHeap) up(j int) {
	for {
		i := (j - 1) / 2 // parent
		if i == j || !((*h)[j].dist < (*h)[i].dist) {
			break
		}
		(*h)[i], (*h)[j] = (*h)[j], (*h)[i]
		j = i
	}
}

func (h *pathHeap) down(i, n int) {
	for {
		j1 := 2*i + 1
		if j1 >= n || j1 < 0 { // j1 < 0 after int overflow
			break
		}
		j := j1 // left child
		if j2 := j1 + 1; j2 < n && !((*h)[j1].dist < (*h)[j2].dist) {
			j = j2 // = 2*i + 2  // right child
		}
		if !((*h)[j].dist < (*h)[i].dist) {
			break
		}
		(*h)[i], (*h)[j] = (*h)[j], (*h)[i]
		i = j
	}
}
