package generator

import (
	"fmt"
	"github.com/runningwild/cmwc"
	"github.com/runningwild/linear"
	"github.com/runningwild/magnus/base"
	"math"
	"math/rand"
)

type Room struct {
	Walls  map[string]linear.Poly
	Lava   map[string]linear.Poly
	Starts []linear.Vec2
	Dx, Dy int
	NextId int

	// Only filled for moba rooms
	Moba struct {
		SideData []mobaRoomSideData
	}
}

type mobaRoomSideData struct {
	Base   linear.Vec2   // Position of the base for this side
	Towers []linear.Vec2 // Positions of the towers for this side
	// Will also need waypoints for units, production and whatnot.
}

var nextIdInt int

func nextId() string {
	nextIdInt++
	return fmt.Sprintf("%d", nextIdInt)
}

func distFromPointToSeg(p linear.Vec2, s linear.Seg2) float64 {
	s.P = s.P.Sub(p)
	s.Q = s.Q.Sub(p)
	cross := s.Ray().Cross()
	crossSeg := linear.Seg2{Q: cross}
	if crossSeg.Left(s.P) != crossSeg.Left(s.Q) {
		return s.DistFromOrigin()
	}
	da := s.P.Mag()
	db := s.Q.Mag()
	if da < db {
		return da
	}
	return db
}

func gridify(f float64, grid int) float64 {
	f += float64(grid) / 2
	return f - math.Mod(f, float64(grid))
}

func GenerateRoom(dx, dy, radius float64, grid int, seed int64) Room {
	var room Room
	room.Lava = make(map[string]linear.Poly)
	room.Walls = make(map[string]linear.Poly)
	nextIdInt = 0
	room.Dx = int(dx)
	room.Dy = int(dy)
	c := cmwc.MakeGoodCmwc()
	if seed == 0 {
		c.SeedWithDevRand()
		n := c.Int63()
		c = cmwc.MakeGoodCmwc()
		base.Log().Printf("SEED: %v", n)
		c.Seed(n)
	} else {
		c.Seed(seed)
	}
	sanity := int(math.Sqrt(dx * dy))
	r := rand.New(c)
	var poss []linear.Vec2
	for sanity > 0 {
		pos := linear.Vec2{r.Float64() * (dx - radius + 1), r.Float64() * (dy - radius + 1)}
		good := true
		for _, p := range poss {
			if p.Sub(pos).Mag() < 2*(2*radius) {
				good = false
				break
			}
		}
		if !good {
			sanity--
			continue
		}
		poss = append(poss, pos)
	}

	// Find the pair of points that maximizes distance
	maxDist := 0.0
	for i := range poss {
		for j := range poss {
			dist := poss[i].Sub(poss[j]).Mag()
			if dist > maxDist {
				maxDist = dist
			}
		}
	}
	var a, b int
	minDist := maxDist * 3 / 4
	hits := 1.0
	for i := range poss {
		for j := range poss {
			if i == j {
				continue
			}
			dist := poss[i].Sub(poss[j]).Mag()
			if dist > minDist && r.Float64() < 1.0/hits {
				a, b = i, j
				hits = hits + 1
			}
		}
	}

	room.Starts = []linear.Vec2{poss[a], poss[b]}
	for _, start := range room.Starts {
		var data mobaRoomSideData
		data.Base = start
		data.Towers = append(data.Towers, start.Add(linear.Vec2{10, 0}))
		room.Moba.SideData = append(room.Moba.SideData, data)
	}

	for i, p := range poss {
		if i == a || i == b {
			continue
		}
		p = linear.Vec2{gridify(p.X, grid), gridify(p.Y, grid)}
		g := float64(grid)
		room.Lava[nextId()] = linear.Poly{
			linear.Vec2{p.X, p.Y},
			linear.Vec2{p.X, p.Y + g},
			linear.Vec2{p.X + g, p.Y + g},
			linear.Vec2{p.X + g, p.Y},
		}
	}

	sanity = int(math.Pow(dx*dy, 0.20))
	var segs []linear.Seg2
	for sanity > 0 {
		a := linear.Vec2{r.Float64() * (dx), r.Float64() * (dy)}
		length := r.Float64()*radius + (radius)
		angle := float64(r.Intn(4)) * 3.1415926535 / 2
		ray := (linear.Vec2{1, 0}).Rotate(angle)
		seg := linear.Seg2{a, a.Add(ray.Scale(length))}
		seg.P.X = gridify(seg.P.X, grid)
		seg.P.Y = gridify(seg.P.Y, grid)
		seg.Q.X = gridify(seg.Q.X, grid)
		seg.Q.Y = gridify(seg.Q.Y, grid)
		good := true
		if seg.P.X <= 0 || seg.P.X >= dx || seg.P.Y <= 0 || seg.P.Y >= dy {
			good = false
		}
		if seg.Q.X <= 0 || seg.Q.X >= dx || seg.Q.Y <= 0 || seg.Q.Y >= dy {
			good = false
		}
		if seg.P.X == seg.Q.X && seg.P.Y == seg.Q.Y {
			good = false
		}
		// Can't get too close to a circle
		for _, p := range poss {
			if distFromPointToSeg(p, seg) < radius/2 {
				good = false
				break
			}
		}

		// Check to make sure this segment isn't coincident with any othe segment.
		// To avoid annoying degeneracies we'll rotate the segment slightly.
		rot := linear.Seg2{seg.P, seg.Ray().Rotate(0.01).Add(seg.P)}
		for _, cur := range segs {
			if rot.DoesIsect(cur) {
				good = false
				break
			}
		}

		if !good {
			sanity--
			continue
		}
		segs = append(segs, seg)
	}

	for _, s := range segs {
		right := s.Ray().Cross().Norm().Scale(-float64(grid))
		s2 := linear.Seg2{s.Q.Add(right), s.P.Add(right)}
		room.Walls[nextId()] = linear.Poly{s.P, s.Q, s2.P, s2.Q}
	}
	room.NextId = nextIdInt
	return room
}
