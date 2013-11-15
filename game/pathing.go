package game

import (
	"github.com/runningwild/linear"
	"sync"
)

const pathingDataGrid = 64

type PathingData struct {
	sync.RWMutex

	// direction [dstX][dstY][srcX][srcY]
	dirs [][][][]pathingDataCell

	// List of directly connected cells for every position
	conns [][][]pathingConnection

	// Fine grained locking so that we can ask to compute the paths from a single
	// destination and retrieve the results later.
	dstData [][]pathingDstData

	// Makes sure that we can't do any complete paths until we've finished all
	// direct paths.  This way we don't block the initial Think() on doing all
	// direct paths, which, until the ExistsLos() is faster, will be kinda slow.
	finishDirectPaths sync.WaitGroup
}

type pathingDstData struct {
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
	srcx, srcy int
	dstx, dsty int
	dist       float64
}

func makePathingData(room *Room) *PathingData {
	var pd PathingData
	dx := (room.Dx + pathingDataGrid - 1) / pathingDataGrid
	dy := (room.Dy + pathingDataGrid - 1) / pathingDataGrid
	pd.dirs = make([][][][]pathingDataCell, dx)
	pd.conns = make([][][]pathingConnection, dx)
	pd.dstData = make([][]pathingDstData, dx)
	pd.finishDirectPaths.Add(dx * dy)
	for i := range pd.dirs {
		pd.dirs[i] = make([][][]pathingDataCell, dy)
		pd.conns[i] = make([][]pathingConnection, dy)
		pd.dstData[i] = make([]pathingDstData, dy)
		for j := range pd.dirs[i] {
			pd.dirs[i][j] = make([][]pathingDataCell, dx)
			for k := range pd.dirs[i][j] {
				pd.dirs[i][j][k] = make([]pathingDataCell, dy)
			}
			go pd.findAllDirectPaths(i, j, room)
		}
	}
	return &pd
}

func (pd *PathingData) findAllDirectPaths(dstx, dsty int, room *Room) {
	defer pd.finishDirectPaths.Done()
	dst := linear.Vec2{(float64(dstx) + 0.5) * pathingDataGrid, (float64(dsty) + 0.5) * pathingDataGrid}
	for x := range pd.dirs[dstx][dsty] {
		for y := range pd.dirs[dstx][dsty][x] {
			src := linear.Vec2{(float64(x) + 0.5) * pathingDataGrid, (float64(y) + 0.5) * pathingDataGrid}
			if dstx == 1 && dsty == 4 {
			}
			if room.ExistsLos(src, dst) {
				if dstx == 1 && dsty == 4 {
				}
				pd.conns[dstx][dsty] = append(pd.conns[dstx][dsty], pathingConnection{
					x:    x,
					y:    y,
					dist: dst.Sub(src).Mag(),
				})
				data := &pd.dirs[dstx][dsty][x][y]
				data.angle = src.Sub(dst).Angle()
				data.direct = true
				// data.filled = true
			}
		}
	}
}

func (pd *PathingData) findAllPaths(dstx, dsty int) {
	paths := pd.dirs[dstx][dsty]
	var next pathHeap
	for _, conn := range pd.conns[dstx][dsty] {
		if conn.x == dstx && conn.y == dsty {
			continue
		}
		next.Push(pathingNode{
			srcx: conn.x,
			srcy: conn.y,
			dstx: dstx,
			dsty: dsty,
			dist: conn.dist,
		})
		paths[conn.x][conn.y].dist = conn.dist
	}
	for len(next) > 0 {
		node := next.Pop()
		cell := &paths[node.srcx][node.srcy]
		if cell.filled {
			continue
		}
		cell.filled = true
		if !cell.direct {
			cell.angle = (linear.Vec2{float64(node.dstx - node.srcx), float64(node.dsty - node.srcy)}).Angle()
		}
		for _, conn := range pd.conns[node.srcx][node.srcy] {
			if paths[conn.x][conn.y].filled {
				continue
			}
			next.Push(pathingNode{
				srcx: conn.x,
				srcy: conn.y,
				dstx: node.srcx,
				dsty: node.srcy,
				dist: node.dist + conn.dist,
			})
		}
	}
}

func (pd *PathingData) Dir(src, dst linear.Vec2) linear.Vec2 {
	x := int(src.X / pathingDataGrid)
	y := int(src.Y / pathingDataGrid)
	x2 := int(dst.X / pathingDataGrid)
	y2 := int(dst.Y / pathingDataGrid)
	if x < 0 || y < 0 || x >= len(pd.dstData) || y >= len(pd.dstData[x]) {
		return linear.Vec2{0, 0}
	}
	if x2 < 0 || y2 < 0 || x2 >= len(pd.dstData) || y2 >= len(pd.dstData[x2]) {
		return linear.Vec2{0, 0}
	}
	dstData := &pd.dstData[x2][y2]
	dstData.RLock()
	defer dstData.RUnlock()
	if !dstData.complete {
		dstData.once.Do(func() {
			go func() {
				pd.finishDirectPaths.Wait()
				dstData.Lock()
				defer dstData.Unlock()
				pd.findAllPaths(x2, y2)
				dstData.complete = true
			}()
		})
		return dst.Sub(src).Norm()
	}
	cell := pd.dirs[x2][y2][x][y]
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
