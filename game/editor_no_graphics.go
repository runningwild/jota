// +build nographics

package game

type editorData struct{}

func (editor *editorData) SetSystem(sys interface{}) {}
func (editor *editorData) Active() bool {
	return false
}
func (editor *editorData) Toggle() {}
