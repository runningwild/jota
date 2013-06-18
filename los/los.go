package los

import (
	"fmt"
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/linear"
	"math"
)

type Los struct {
	distArray []float64
	rays      []linear.Seg2
	pos       linear.Vec2
	horizon   float64
}

func Make(size int, horizon float64) *Los {
	var l Los
	l.horizon = horizon * horizon
	l.distArray = make([]float64, size)
	for i := range l.distArray {
		l.distArray[i] = l.horizon
	}
	l.rays = make([]linear.Seg2, size)
	for i := range l.rays {
		l.rays[i] = linear.Seg2{
			linear.Vec2{0, 0},
			(linear.Vec2{1, 0}).Rotate(2 * math.Pi * (float64(i)/float64(size) - 0.5)),
		}
	}
	return &l
}
func (l *Los) Reset(pos linear.Vec2) {
	l.pos = pos
	for i := range l.distArray {
		l.distArray[i] = l.horizon
	}
}
func (l *Los) DrawSeg(seg linear.Seg2) {
	seg.P = seg.P.Sub(l.pos)
	seg.Q = seg.Q.Sub(l.pos)
	wrap := len(l.distArray)
	a1 := math.Atan2(seg.P.Y, seg.P.X)
	a2 := math.Atan2(seg.Q.Y, seg.Q.X)
	if a1 > a2 {
		a1, a2 = a2, a1
		seg.P, seg.Q = seg.Q, seg.P
	}
	if a2-a1 > math.Pi {
		a1, a2 = a2, a1
		seg.P, seg.Q = seg.Q, seg.P
	}
	start := int(((a1 / (2 * math.Pi)) + 0.5) * float64(len(l.distArray)))
	end := int(((a2 / (2 * math.Pi)) + 0.5) * float64(len(l.distArray)))

	for i := start % wrap; i != end%wrap; i = (i + 1) % wrap {
		dist2 := l.rays[i].Isect(seg).Mag2()
		// base.Log().Printf("%d: %v\n", i, math.Sqrt(dist2))
		// dist = l.rays[i].Isect(seg).Mag2()

		if dist2 < l.distArray[i] {
			l.distArray[i] = dist2
		}
	}
}

func (l *Los) Render() {
	var v0, v1 linear.Vec2
	gl.Begin(gl.TRIANGLES)
	v1 = (linear.Vec2{-1, 0}).Scale(math.Sqrt(l.distArray[0])).Add(l.pos)
	for i := 1; i <= len(l.distArray); i++ {
		dist := math.Sqrt(l.distArray[i%len(l.distArray)])
		angle := 2 * math.Pi * (float64(i%len(l.distArray))/float64(len(l.distArray)) - 0.5)
		if dist <= 0.0 {
			continue
		}
		v0 = v1
		gl.Color4d(gl.Double(1.0-dist/math.Sqrt(l.horizon)), 1.0, 0.0, 1.0)
		v1 = (linear.Vec2{1, 0}).Rotate(angle).Scale(dist).Add(l.pos)
		gl.Vertex2d(gl.Double(l.pos.X), gl.Double(l.pos.Y))
		gl.Vertex2d(gl.Double(v0.X), gl.Double(v0.Y))
		gl.Vertex2d(gl.Double(v1.X), gl.Double(v1.Y))
	}
	gl.End()
}

// L: Sorted set of line segments on the los ray
// S: Sorted set of event points
// Put all endpoints in S
// Pop from S into C, check for intersections at upper and lower bounds
//   in the event of an intersection, add that event point to S
// Any time the minimum element is removed, note the changes in the los poly

// An event can be one of the following:
// 1. Add a segment (the first endpoint)
// 2. Remove a segment (the second endpoint)
// 3. Reorder segments (an intersection)
//    The reordering is done as follows: Any segment involved in an intersection
//    is removed from the set and stored in a temporary buffer.  Processing
//    continues as normal until the los ray rotates again, at which point all of
//    these segments are readded to the set.  This makes sure that the segments
//    are all in the proper order before anything else has a chance to intersect
//    with them.
type eventType int

func (e eventType) String() string {
	switch e {
	case eventAdd:
		return "Add"
	case eventRemove:
		return "Remove"
	case eventReorder:
		return "Reorder"
	default:
		return "WTF!?!?"
	}
}

const (
	eventAdd eventType = iota
	eventRemove
	eventReorder
)

type event struct {
	eventType eventType
	v         linear.Vec2

	// If this is an add or remove, this is the id of the segment to add or remove
	// Equality is determined by a pointer comparison
	segId segTracker

	// If this is a reorder event, then segId and segId2 are the two segments that
	// need to be reordered at this event.
	segId2 segTracker
}

func (e event) String() string {
	return fmt.Sprintf("%v: %d %d", e.eventType, e.segId, e.segId2)
}

func (e *event) Less(e2 llrb.Item) bool {
	a := math.Atan2(e.v.Y, e.v.X)
	a2 := math.Atan2(e2.(*event).v.Y, e2.(*event).v.X)
	return a < a2
}

func GetLosPoly(pos linear.Vec2, segs []linear.Seg2) linear.Poly {
	// Translate everything so the observer is at the origin
	translated := make([]linear.Seg2, len(segs))
	for i := range translated {
		translated[i].P = segs[i].P.Sub(pos)
		translated[i].Q = segs[i].Q.Sub(pos)
	}

	poly := getLosPolyFromOrigin(translated)

	// Translate everything back to the observer's real position.
	for i := range poly {
		poly[i] = poly[i].Add(pos)
	}
	return poly
}

// Index into the list of segments
type segTracker int

func insertOnIntersection(
	segs []linear.Seg2,
	a, b segTracker,
	eventq *LLRB_event,
	losRay linear.Vec2) {
	if segs[int(a)].DoesIsect(segs[int(b)]) {
		isect := segs[int(a)].Isect(segs[int(b)])
		if math.Atan2(isect.Y, isect.X) > math.Atan2(losRay.Y, losRay.X) {
			fmt.Printf("%d and %d intersect.\n", a, b)
			eventq.InsertNoReplace(event{
				eventType: eventReorder,
				v:         isect,
				segId:     a,
				segId2:    b,
			})
		}
	}
}

func verifyLosOrder(losOrder *LLRB_segTracker, segs []linear.Seg2, losRay linear.Vec2, printIt bool) {
	if printIt {
		fmt.Printf("Los order:\n")
	}
	losOrder.IterateAscending(func(s segTracker) bool {
		seg := segs[s]
		raySeg := linear.Seg2{linear.Vec2{0, 0}, losRay}
		m := raySeg.Isect(seg).Mag()
		if printIt {
			fmt.Printf("%d:%2.2f,  ", s, m)
		}
		return true
	})
	if printIt {
		fmt.Printf("\n")
	}
	var trackers []segTracker
	fmt.Printf("len: %d\n", losOrder.Len())
	for losOrder.Len() > 0 {
		trackers = append(trackers, losOrder.DeleteMin())
		fmt.Printf("removed %d, len: %d\n", trackers[len(trackers)-1], losOrder.Len())
	}
	for _, t := range trackers {
		losOrder.InsertNoReplace(t)
	}
	losOrder.IterateAscending(func(s segTracker) bool {
		seg := segs[s]
		raySeg := linear.Seg2{linear.Vec2{0, 0}, losRay}
		m := raySeg.Isect(seg).Mag()
		if printIt {
			fmt.Printf("%d:%2.2f,  ", s, m)
		}
		return true
	})
	if printIt {
		fmt.Printf("\n\n")
	}
}

func getLosPolyFromOrigin(segs []linear.Seg2) linear.Poly {
	// Make sure the first endpoint in a segment is the one that will be added.
	for i := range segs {
		pa := math.Atan2(segs[i].P.Y, segs[i].P.X)
		qa := math.Atan2(segs[i].Q.Y, segs[i].Q.X)
		if pa > qa {
			segs[i].P, segs[i].Q = segs[i].Q, segs[i].P
		}
	}
	var losPoly linear.Poly

	// Current minimum segment
	var minSeg segTracker = -1

	var losRay linear.Vec2
	lessSegs := func(s1, s2 segTracker) bool {
		if s1 == s2 {
			return false
		}
		seg1 := segs[int(s1)]
		seg2 := segs[int(s2)]
		if seg1.P.X == seg2.P.X && seg1.P.Y == seg2.P.Y {
			target := (linear.Vec2{1, 0}).Rotate(math.Atan2(seg1.P.Y, seg1.P.X) + 1e-5)
			ray := linear.Seg2{linear.Vec2{}, target}
			d1 := ray.Isect(seg1).Mag()
			d2 := ray.Isect(seg2).Mag()
			return d1 < d2
		}
		raySeg := linear.Seg2{linear.Vec2{0, 0}, losRay}
		m1 := raySeg.Isect(seg1).Mag2()
		m2 := raySeg.Isect(seg2).Mag2()
		if m1 == m2 {
			return s1 < s2
		}
		return m1 < m2
	}
	lessEvents := func(a, b event) bool {
		angleA := math.Atan2(a.v.Y, a.v.X)
		angleB := math.Atan2(b.v.Y, b.v.X)
		return angleA < angleB
	}

	losOrder := NewLlrbsegTracker(lessSegs)
	eventq := NewLlrbevent(lessEvents)
	for i := range segs {
		eventq.InsertNoReplace(event{
			eventType: eventAdd,
			v:         segs[i].P,
			segId:     segTracker(i),
		})
		eventq.InsertNoReplace(event{
			eventType: eventRemove,
			v:         segs[i].Q,
			segId:     segTracker(i),
		})
	}

	reorderBuffer := make(map[segTracker]bool)
	var e event = eventq.Min()
	for eventq.Len() > 0 {
		oldRay := e.v
		e = eventq.Min()
		fmt.Printf("e = Min: %v\n", e)

		// Check to see if losRay is actually rotating, if so then we'll re-add any
		// segs that were removed because of an intersection, at this point their
		// orders will have changed so it will be safe to readd them.
		losRay = e.v.Add(oldRay)
		if math.Atan2(losRay.Y, losRay.X) > math.Atan2(oldRay.Y, oldRay.X) {
			if len(reorderBuffer) > 0 {
				fmt.Printf("Reinserting %d things\n", len(reorderBuffer))
				losRay = e.v.Add(oldRay)
				unset := true
				var min, max segTracker
				for tracker := range reorderBuffer {
					if unset {
						min = tracker
						max = tracker
						unset = false
					}
					if lessSegs(tracker, min) {
						min = tracker
					}
					if lessSegs(max, tracker) {
						max = tracker
					}
					losOrder.InsertNoReplace(tracker)
				}
				reorderBuffer = make(map[segTracker]bool)
				verifyLosOrder(losOrder, segs, losRay, true)
				fmt.Printf("Los size: %d\n", losOrder.Len())
				losOrder.IterateAscending(func(s segTracker) bool {
					l, ok := losOrder.LowerBound(s)
					fmt.Printf("Lower(%d): %d - %t\n", s, l, ok)
					return true
				})
				lower, ok := losOrder.LowerBound(min)
				if ok {
					fmt.Printf("Lower Bonus insert on %d and %d!\n", lower, min)
					insertOnIntersection(segs, lower, min, eventq, losRay)
				}
				upper, ok := losOrder.UpperBound(max)
				if ok {
					fmt.Printf("Upper Bonus insert on %d and %d!\n", max, upper)
					insertOnIntersection(segs, max, upper, eventq, losRay)
				}
				minSeg = losOrder.Min()
			}
			// fmt.Printf("Len: %d\n", losOrder.Len())
			// fmt.Printf("Min: %v\n", losOrder.Min())
			verifyLosOrder(losOrder, segs, losRay, true)
		}
		e = eventq.DeleteMin()
		fmt.Printf("e = DeleteMin: %v\n", e)

		// Standard event handling:
		switch e.eventType {
		case eventAdd:
			losRay = e.v
			losOrder.InsertNoReplace(e.segId)
			fmt.Printf("Add %d: %v\n", e.segId, segs[e.segId])
			fmt.Printf("Min: %d\n", losOrder.Min())
			if minSeg != -1 {
				if lessSegs(e.segId, minSeg) {
					losPoly = append(losPoly, (linear.Seg2{linear.Vec2{}, e.v}).Isect(segs[minSeg]))
					losPoly = append(losPoly, e.v)
					minSeg = e.segId
				}
			} else {
				minSeg = losOrder.Min()
				losPoly = append(losPoly, e.v)
			}

			// After adding, check for intersections with the segments that are above
			// and below this one.
			upper, ok := losOrder.UpperBound(e.segId)
			if ok {
				fmt.Printf("Check for upper intersection\n")
				insertOnIntersection(segs, upper, e.segId, eventq, losRay)
			}
			lower, ok := losOrder.LowerBound(e.segId)
			if ok {
				insertOnIntersection(segs, e.segId, lower, eventq, losRay)
			}

		case eventRemove:
			fmt.Printf("Remove %d: %v\n", e.segId, segs[e.segId])
			fmt.Printf("losOrder.Len() = %d\n", losOrder.Len())
			losOrder.Delete(e.segId)
			if minSeg == e.segId {
				losPoly = append(losPoly, e.v)
				if losOrder.Len() > 0 {
					minSeg = losOrder.Min()
					losPoly = append(losPoly, (linear.Seg2{linear.Vec2{}, e.v}).Isect(segs[minSeg]))
				} else {
					minSeg = -1
				}
			}
			fmt.Printf("losOrder.Len() = %d\n", losOrder.Len())
			lower, lOk := losOrder.LowerBound(e.segId)
			upper, uOk := losOrder.UpperBound(e.segId)
			if lOk && uOk {
				insertOnIntersection(segs, lower, upper, eventq, losRay)
			}

		case eventReorder:
			_, ok := losOrder.Delete(e.segId)
			_, ok2 := losOrder.Delete(e.segId2)
			fmt.Printf("Reorder(%d, %d): %t %t\n", e.segId, e.segId2, ok, ok2)
			if minSeg == e.segId || minSeg == e.segId2 {
				if minSeg == e.segId {
					minSeg = e.segId2
				} else {
					minSeg = e.segId
				}
				losPoly = append(losPoly, e.v)
			}
			// fmt.Printf("segs: %d %d %d\n", minSeg, e.segId, e.segId2)
			// if minSeg == e.segId {
			// 	minSeg = e.segId2
			// } else if minSeg == e.segId2 {
			// 	minSeg = e.segId
			// }
			// if ok && ok2 {
			// 	reorderBuffer[e.segId] = true
			// 	reorderBuffer[e.segId2] = true
			// 	fmt.Printf("Added %d and %d to the reorder buffer\n", e.segId, e.segId2)
			// } else {
			// 	fmt.Printf("No adding segments (%d, %d) - (%t, %t) to the reorder buffer.\n", e.segId, e.segId2, ok, ok2)
			// }
			if ok {
				reorderBuffer[e.segId] = true
			}
			if ok2 {
				reorderBuffer[e.segId2] = true
			}
		}

	}
	return losPoly
}

func main() {
	segs := []linear.Seg2{
		linear.Seg2{linear.Vec2{1, -1}, linear.Vec2{2, 1}},
		linear.Seg2{linear.Vec2{2, -1}, linear.Vec2{1, 1}},
		linear.Seg2{linear.Vec2{1.1, -1}, linear.Vec2{1.1, 1}},
		linear.Seg2{linear.Vec2{1.5, -1}, linear.Vec2{1, 0}},
	}
	// segs := []linear.Seg2{
	// 	linear.Seg2{linear.Vec2{1, -1}, linear.Vec2{2, 1}},
	// 	linear.Seg2{linear.Vec2{1.5, -1}, linear.Vec2{1.5, -0.1}},
	// 	linear.Seg2{linear.Vec2{2, -1}, linear.Vec2{1, 1}},
	// 	linear.Seg2{linear.Vec2{10, 0.5}, linear.Vec2{0, 0.5}},
	// }
	// segs := []linear.Seg2{
	// 	linear.Seg2{linear.Vec2{1, -1}, linear.Vec2{1, 1}},
	// 	linear.Seg2{linear.Vec2{0.5, -1}, linear.Vec2{1.5, 1}},
	// 	linear.Seg2{linear.Vec2{1, -1}, linear.Vec2{1, 1}},
	// }
	// segs := []linear.Seg2{
	// 	linear.Seg2{linear.Vec2{1, 2}, linear.Vec2{2, 2}},
	// 	linear.Seg2{linear.Vec2{2, 2}, linear.Vec2{2, 3}},
	// 	linear.Seg2{linear.Vec2{2, 3}, linear.Vec2{1, 3}},
	// 	linear.Seg2{linear.Vec2{1, 3}, linear.Vec2{1, 2}},
	// 	linear.Seg2{linear.Vec2{0, 5}, linear.Vec2{-1, 7}},
	// }

	poly := GetLosPoly(linear.Vec2{0, 0}, segs)
	fmt.Printf("LosPoly:\n")
	for i := range poly {
		fmt.Printf("%v\n", poly[i])
	}
}
func maintest() {
	f := NewLlrbfloat64(func(a, b float64) bool { return a > b })
	f.InsertNoReplace(1.0)
	f.InsertNoReplace(3.0)
	f.InsertNoReplace(5.0)
	for i := 0.0; i < 7.0; i += 0.5 {
		fmt.Printf("%2.2f\t", i)
		b, ok := f.LowerBound(i)
		if ok {
			fmt.Printf("%2.2f\t", b)
		} else {
			fmt.Printf("...\t")
		}
		b, ok = f.UpperBound(i)
		if ok {
			fmt.Printf("%2.2f\t", b)
		} else {
			fmt.Printf("...\t")
		}
		fmt.Printf("\n")
	}
}
