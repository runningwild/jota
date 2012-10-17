package base

import (
  "bufio"
  gl "github.com/chsc/gogl/gl21"
  "github.com/runningwild/glop/gin"
  "github.com/runningwild/glop/gui"
  "strings"
  "unicode"
)

const maxLines = 25
const maxLineLength = 150

// A simple gui element that will display the last several lines of text from
// a log file (TODO: and also allow you to enter some basic commands).
type Console struct {
  gui.BasicZone
  lines      [maxLines]string
  start, end int
  xscroll    float64

  input *bufio.Reader
  cmd   []byte
  dict  *gui.Dictionary
}

func MakeConsole() *Console {
  if log_console == nil {
    panic("Cannot make a console until the logging system has been set up.")
  }
  var c Console
  c.BasicZone.Ex = true
  c.BasicZone.Ey = true
  c.BasicZone.Request_dims = gui.Dims{1000, 1000}
  c.input = bufio.NewReader(log_console)
  c.dict = GetDictionary(12)
  return &c
}

func (c *Console) String() string {
  return "console"
}

func (c *Console) Think(ui *gui.Gui, dt int64) {
  for line, _, err := c.input.ReadLine(); err == nil; line, _, err = c.input.ReadLine() {
    c.lines[c.end] = string(line)
    c.end = (c.end + 1) % len(c.lines)
    if c.start == c.end {
      c.start = (c.start + 1) % len(c.lines)
    }
  }
}

func (c *Console) Respond(ui *gui.Gui, group gui.EventGroup) bool {
  if found, event := group.FindEvent(GetDefaultKeyMap()["console"].Id()); found && event.Type == gin.Press {
    if group.Focus {
      ui.DropFocus()
    } else {
      ui.TakeFocus(c)
    }
    return true
  }
  if group.Focus {
    if found, event := group.FindEvent(gin.Left); found && event.Type == gin.Press {
      c.xscroll += 250
    }
    if found, event := group.FindEvent(gin.Right); found && event.Type == gin.Press {
      c.xscroll -= 250
    }
  }
  if c.xscroll > 0 {
    c.xscroll = 0
  }
  if found, event := group.FindEvent(gin.Space); found && event.Type == gin.Press {
    c.xscroll = 0
  }

  if group.Events[0].Type == gin.Press {
    r := rune(group.Events[0].Key.Id())
    if r < 256 {
      if gin.In().GetKey(gin.EitherShift).IsDown() {
        r = unicode.ToUpper(r)
      }
      c.cmd = append(c.cmd, byte(r))
    }
  }

  return group.Focus
}

func (c *Console) Draw(region gui.Region) {
}

func (c *Console) DrawFocused(region gui.Region) {
  gl.Color4d(0.2, 0, 0.3, 0.8)
  gl.Disable(gl.TEXTURE_2D)
  gl.Begin(gl.QUADS)
  {
    x := int32(region.X)
    y := int32(region.Y)
    x2 := int32(region.X + region.Dx)
    y2 := int32(region.Y + region.Dy)
    gl.Vertex2i(x, y)
    gl.Vertex2i(x, y2)
    gl.Vertex2i(x2, y2)
    gl.Vertex2i(x2, y)
  }
  gl.End()
  gl.Color4d(1, 1, 1, 1)
  y := float64(region.Y) + float64(len(c.lines))*c.dict.MaxHeight()
  do_color := func(line string) {
    if strings.HasPrefix(line, "LOG") {
      gl.Color4d(1, 1, 1, 1)
    }
    if strings.HasPrefix(line, "WARN") {
      gl.Color4d(1, 1, 0, 1)
    }
    if strings.HasPrefix(line, "ERROR") {
      gl.Color4d(1, 0, 0, 1)
    }
  }
  if c.start > c.end {
    for i := c.start; i < len(c.lines); i++ {
      do_color(c.lines[i])
      c.dict.RenderString(c.lines[i], c.xscroll, y, 0, c.dict.MaxHeight(), gui.Left)
      y -= c.dict.MaxHeight()
    }
    for i := 0; i < c.end; i++ {
      do_color(c.lines[i])
      c.dict.RenderString(c.lines[i], c.xscroll, y, 0, c.dict.MaxHeight(), gui.Left)
      y -= c.dict.MaxHeight()
    }
  } else {
    for i := c.start; i < c.end && i < len(c.lines); i++ {
      do_color(c.lines[i])
      c.dict.RenderString(c.lines[i], c.xscroll, y, 0, c.dict.MaxHeight(), gui.Left)
      y -= c.dict.MaxHeight()
    }
  }
  c.dict.RenderString(string(c.cmd), c.xscroll, y, 0, c.dict.MaxHeight(), gui.Left)
}
