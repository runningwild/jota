package ability

import (
	"encoding/gob"
	gl "github.com/chsc/gogl/gl21"
	"github.com/runningwild/cgf"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/linear"
	"github.com/runningwild/magnus/base"
	"github.com/runningwild/magnus/game"
)

func makeRemovePoly(params map[string]int) game.Ability {
	var r removePoly
	return &r
}

func init() {
	game.RegisterAbility("removePoly", makeRemovePoly)
}

type removePoly struct {
	Done   bool
	Target string
}

func (r *removePoly) Activate(int) ([]cgf.Event, bool) {
	r.Done = false
	r.Target = ""
	return nil, true
}
func (r *removePoly) Deactivate(player_id int) []cgf.Event {
	return nil
}
func (r *removePoly) Respond(player_id int, group gin.EventGroup) bool {
	if r.Target == "" {
		return false
	}
	if found, event := group.FindEvent(gin.AnyMouseLButton); found && event.Type == gin.Press {
		r.Done = true
		return true
	}
	return false
}
func (r *removePoly) Think(player_id int, game *game.Game, mouse linear.Vec2) ([]cgf.Event, bool) {
	if r.Done && r.Target != "" {
		if _, ok := game.Room.Walls[r.Target]; ok && !game.IsExistingPolyVisible(r.Target) {
			event := removePolyEvent{r.Target}
			return []cgf.Event{&event}, true
		}
	}
	r.Done = false
	r.Target = ""
	for i, poly := range game.Room.Walls {
		if linear.VecInsideConvexPoly(mouse, poly) {
			r.Target = i
			break
		}
	}
	return nil, false
}
func (r *removePoly) Draw(player_id int, game *game.Game) {
	gl.Disable(gl.TEXTURE_2D)

	visible := game.IsExistingPolyVisible(r.Target)
	if visible {
		gl.Color4ub(255, 0, 0, 255)
	} else {
		gl.Color4ub(0, 255, 0, 255)
	}
	poly := game.Room.Walls[r.Target]
	gl.Begin(gl.LINES)
	for i := range poly {
		seg := poly.Seg(i)
		gl.Vertex2i(gl.Int(seg.P.X), gl.Int(seg.P.Y))
		gl.Vertex2i(gl.Int(seg.Q.X), gl.Int(seg.Q.Y))
	}
	gl.End()
}

type removePolyEvent struct {
	Target string
}

func init() {
	gob.Register(removePolyEvent{})
}

func (r removePolyEvent) Apply(_g interface{}) {
	g := _g.(*game.Game)
	if g.IsExistingPolyVisible(r.Target) {
		base.Warn().Printf("Tried to do a removePolyEvent with a visible target: %d", r.Target)
		return
	}
	// NEXT: We're storing the index, but what if the index changes between
	// when the event is enqueued and the event is applied?  We might try to
	// remove a different polygon.  So what are the options to avoid this:
	// 1. Store the poly itself and check that - maybe
	// 2. Best idea I think is to store g.Room.Walls as a map[int]linear.Poly
	//    and that way there is never any confusion.  This would require a
	//    method on game that inserts polys with unique ids, no big deal.
	delete(g.Room.Walls, r.Target)
}
