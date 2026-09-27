// Package progress provides a live in-place ANSI progress indicator (spec §F-12).
// Automatically disabled when stdout is not a real console.
package progress

import (
	"fmt"
	"io"
	"sync/atomic"
	"time"
)

// Indicator writes a live progress line to w using ANSI escape codes.
// Call Start() to begin, update Lines/ActiveConsumer atomically, Stop() to finish.
type Indicator struct {
	w             io.Writer
	enabled       bool
	lines         atomic.Int64
	activeLabel   atomic.Value // string
	start         time.Time
	done          chan struct{}
}

// New creates an Indicator. enabled should be windows.IsConsole(os.Stdout) && !noProgress.
func New(w io.Writer, enabled bool) *Indicator {
	p := &Indicator{w: w, enabled: enabled, done: make(chan struct{})}
	p.activeLabel.Store("")
	return p
}

// Start begins the background refresh goroutine (every 250ms).
func (p *Indicator) Start() {
	if !p.enabled {
		return
	}
	p.start = time.Now()
	go func() {
		t := time.NewTicker(250 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				p.render()
			case <-p.done:
				p.clear()
				return
			}
		}
	}()
}

// SetLines updates the processed line/byte count.
func (p *Indicator) SetLines(n int64) { p.lines.Store(n) }

// SetActive sets the label of the currently running consumer (e.g. "Consumer [2]").
func (p *Indicator) SetActive(label string) { p.activeLabel.Store(label) }

// Stop halts the indicator and clears the line.
func (p *Indicator) Stop() {
	if !p.enabled {
		return
	}
	close(p.done)
}

func (p *Indicator) render() {
	elapsed := time.Since(p.start).Round(time.Second)
	label, _ := p.activeLabel.Load().(string)
	active := ""
	if label != "" {
		active = fmt.Sprintf(" | %s running", label)
	}
	line := fmt.Sprintf("\r\033[K[gopipe] ▶ %s lines processed%s | %s elapsed",
		fmtInt(p.lines.Load()), active, elapsed)
	_, _ = fmt.Fprint(p.w, line)
}

func (p *Indicator) clear() {
	_, _ = fmt.Fprint(p.w, "\r\033[K")
}

func fmtInt(n int64) string {
	s := fmt.Sprintf("%d", n)
	out := make([]byte, 0, len(s)+len(s)/3)
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, byte(c))
	}
	return string(out)
}
