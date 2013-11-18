// +build nographics

package game

import (
	"github.com/runningwild/glop/gin"
)

type editorData struct{}

func (editor *editorData) SetSystem(sys interface{}) {}
func (editor *editorData) Active() bool {
	return false
}
func (editor *editorData) Toggle() {}

func (g *Game) HandleEventGroupEditor(group gin.EventGroup) {}
