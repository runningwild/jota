package base

import (
  "fmt"
  "strings"
  "github.com/runningwild/glop/gin"
)

type KeyBinds map[string]string
type KeyMap map[string]gin.Key

var (
  default_map KeyMap
)

func SetDefaultKeyMap(km KeyMap) {
  default_map = km
}
func GetDefaultKeyMap() KeyMap {
  return default_map
}

func getKeysFromString(str string) []gin.KeyId {
  parts := strings.Split(str, "+")
  var kids []gin.KeyId
  for _, part := range parts {
    part = osSpecifyKey(part)
    var kid gin.KeyId
    switch {
    case len(part) == 1: // Single character - should be ascii
      kid = gin.KeyId(part[0])

    case part == "ctrl":
      kid = gin.EitherControl

    case part == "shift":
      kid = gin.EitherShift

    case part == "alt":
      kid = gin.EitherAlt

    case part == "gui":
      kid = gin.EitherGui

    case part == "space":
      kid = gin.Space

    case part == "rmouse":
      kid = gin.MouseRButton

    case part == "lmouse":
      kid = gin.MouseLButton

    case part == "up":
      kid = gin.Up

    case part == "down":
      kid = gin.Down

    default:
      key := gin.In().GetKeyByName(part)
      if key == nil {
        panic(fmt.Sprintf("Unknown key '%s'", part))
      }
      kid = key.Id()
    }
    kids = append(kids, kid)
  }
  return kids
}

func (kb KeyBinds) MakeKeyMap() KeyMap {
  key_map := make(KeyMap)
  for key, val := range kb {
    parts := strings.Split(val, ",")
    var binds []gin.Key
    for i, part := range parts {
      kids := getKeysFromString(part)

      if len(kids) == 1 {
        binds = append(binds, gin.In().GetKey(kids[0]))
      } else {
        // The last kid is the main kid and the rest are modifiers
        main := kids[len(kids)-1]
        kids = kids[0 : len(kids)-1]
        var down []bool
        for _ = range kids {
          down = append(down, true)
        }
        binds = append(binds, gin.In().BindDerivedKey(fmt.Sprintf("%s:%d", key, i), gin.In().MakeBinding(main, kids, down)))
      }
    }
    if len(binds) == 1 {
      key_map[key] = binds[0]
    } else {
      var actual_binds []gin.Binding
      for i := range binds {
        actual_binds = append(actual_binds, gin.In().MakeBinding(binds[i].Id(), nil, nil))
      }
      key_map[key] = gin.In().BindDerivedKey("name", actual_binds...)
    }
  }
  return key_map
}
