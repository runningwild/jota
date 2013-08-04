package base

import (
	"bufio"
	"io"
)

type Tailer interface {
	GetLines(dst []string)
}

// tail is a simple struct to keep track of the last N lines written to an
// io.Reader.
type tail struct {
	bufferedSrc *bufio.Reader
	lines       []string
	maxLines    int
}

func newTail(reader io.Reader, n int) *tail {
	return &tail{
		bufferedSrc: bufio.NewReader(reader),
		maxLines:    n,
	}
}

// GetLines copies the last len(dst) lines that have been read into dst.  If
// there are fewer than len(dst) lines available, then the lines are copied into
// the end of dst and all values before that are set to the empty string.
func (t *tail) GetLines(dst []string) {
	t.readAsMuchAsPossible()
	for i := 0; i < len(dst) && i < len(t.lines); i++ {
		if i < len(t.lines) {
			dst[len(dst)-i-1] = t.lines[len(t.lines)-i-1]
		} else {
			dst[len(dst)-i-1] = ""
		}
	}
}

func (t *tail) readAsMuchAsPossible() {
	for {
		line, err := t.bufferedSrc.ReadString('\n')
		if err != nil {
			return
		}
		t.lines = append(t.lines, line)
		if len(t.lines) > t.maxLines {
			t.lines = t.lines[len(t.lines)-t.maxLines:]
		}
	}
}
