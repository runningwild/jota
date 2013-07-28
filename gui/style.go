package gui

type StyleStack interface {
	PushStyle(style map[string]interface{})
	Pop()
	Get(name string) interface{}
}

// Very much like CSS
type styleStack struct {
	styles []map[string]interface{}
}

func (s *styleStack) PushStyle(style map[string]interface{}) {
	s.styles = append(s.styles, style)
}

func (s *styleStack) Pop() {
	if len(s.styles) == 0 {
		panic("Can't pop a styleStack with no style.")
	}
	s.styles = s.styles[0 : len(s.styles)-1]
}

func (s *styleStack) Get(name string) interface{} {
	for i := len(s.styles) - 1; i >= 0; i-- {
		if val, ok := s.styles[i][name]; ok {
			return val
		}
	}
	return nil
}
