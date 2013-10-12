package ability

// import (
// 	"encoding/gob"
// 	gl "github.com/chsc/gogl/gl21"
// 	"github.com/runningwild/cgf"
// 	"github.com/runningwild/glop/gin"
// 	"github.com/runningwild/linear"
// 	"github.com/runningwild/jota/base"
// 	"github.com/runningwild/jota/game"
// )

// func makePlacePoly(params map[string]int) game.Ability {
// 	if len(params) != 1 {
// 		base.Error().Fatalf("makePlacePoly expects a single element in params.")
// 	}
// 	var p placePoly
// 	if _, ok := params["wall"]; ok {
// 		p.Payload = placeWall
// 	}
// 	if _, ok := params["pests"]; ok {
// 		p.Payload = placePests
// 	}
// 	return &p
// }

// func init() {
// 	game.RegisterAbility("placePoly", makePlacePoly)
// }

// type placementPayload int

// const (
// 	placeWall placementPayload = iota
// 	placePests
// )

// type placePoly struct {
// 	Placeable bool
// 	Done      bool
// 	Payload   placementPayload
// 	Poly      linear.Poly
// 	Target    linear.Poly
// }

// func (p *placePoly) Activate(game.Gid, bool) ([]cgf.Event, bool) {
// 	p.Done = false
// 	p.Poly = linear.Poly{
// 		linear.Vec2{0, 0},
// 		linear.Vec2{0, 50},
// 		linear.Vec2{50, 50},
// 		linear.Vec2{50, 0},
// 	}
// 	p.Target = nil
// 	return nil, true
// }
// func (p *placePoly) Deactivate(gid game.Gid) []cgf.Event {
// 	return nil
// }
// func (p *placePoly) Respond(gid game.Gid, group gin.EventGroup) bool {
// 	if !p.Placeable {
// 		return false
// 	}
// 	if found, event := group.FindEvent(gin.AnyMouseLButton); found && event.Type == gin.Press {
// 		p.Done = true
// 		return true
// 	}
// 	return false
// }
// func (p *placePoly) Think(gid game.Gid, game *game.Game, mouse linear.Vec2) ([]cgf.Event, bool) {
// 	if p.Done {
// 		event := placePolyEvent{p.Target, p.Payload}
// 		return []cgf.Event{&event}, true
// 	}
// 	mouse.X -= float64(int(mouse.X) % 25)
// 	mouse.Y -= float64(int(mouse.Y) % 25)
// 	if p.Target == nil {
// 		p.Target = make(linear.Poly, len(p.Poly))
// 	}
// 	for i := range p.Poly {
// 		p.Target[i] = p.Poly[i].Add(mouse)
// 	}
// 	p.Placeable = game.IsPolyPlaceable(p.Target)
// 	return nil, false
// }
// func (p *placePoly) Draw(gid game.Gid, game *game.Game) {
// 	gl.Disable(gl.TEXTURE_2D)
// 	placeable := game.IsPolyPlaceable(p.Target)
// 	if placeable {
// 		gl.Color4ub(255, 255, 255, 255)
// 	} else {
// 		gl.Color4ub(255, 0, 0, 255)
// 	}
// 	gl.Begin(gl.LINES)
// 	for i := range p.Target {
// 		seg := p.Target.Seg(i)
// 		gl.Vertex2i(gl.Int(seg.P.X), gl.Int(seg.P.Y))
// 		gl.Vertex2i(gl.Int(seg.Q.X), gl.Int(seg.Q.Y))
// 	}
// 	gl.End()
// }

// type placePolyEvent struct {
// 	Poly    linear.Poly
// 	Payload placementPayload
// }

// func init() {
// 	gob.Register(placePolyEvent{})
// }

// func (p placePolyEvent) Apply(_g interface{}) {
// 	g := _g.(*game.Game)
// 	if !g.IsPolyPlaceable(p.Poly) {
// 		return
// 	}
// 	switch p.Payload {
// 	case placeWall:
// 		g.Room.AddWall(p.Poly)
// 	case placePests:
// 		numPests := 5
// 		mid := p.Poly[0].Add(p.Poly[2]).Scale(0.5)
// 		for i := 0; i < numPests; i++ {
// 			rot := (linear.Vec2{15, 0}).Rotate(float64(i) * 2 * 3.1415926535 / float64(numPests))
// 			g.AddPest(mid.Add(rot))
// 		}
// 	}
// }
