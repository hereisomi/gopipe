// Package summary collects per-consumer run statistics and prints the
// execution summary at pipeline completion (spec §F-11).
package summary

import (
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"time"
)

// ConsumerStats tracks outcomes for one consumer across all invocations.
type ConsumerStats struct {
	Label     string
	Runs      atomic.Int64
	Failed    atomic.Int64
	Retried   atomic.Int64
	Recovered atomic.Int64 // failed then eventually succeeded after retry
}

// RecordRun records one consumer invocation outcome.
// failed: exit code != 0 on final attempt. retried: at least one retry was attempted.
// recovered: retried and ultimately succeeded.
func (s *ConsumerStats) RecordRun(failed, retried, recovered bool) {
	s.Runs.Add(1)
	if failed {
		s.Failed.Add(1)
	}
	if retried {
		s.Retried.Add(1)
	}
	if recovered {
		s.Recovered.Add(1)
	}
}

// Report holds the full pipeline execution summary.
type Report struct {
	Producer  string
	Mode      string
	StartTime time.Time
	EndTime   time.Time
	Lines     int64
	Bytes     int64
	Binary    bool
	ExitCode  int
	Consumers []*ConsumerStats
}

// Print writes the formatted summary block to w (spec §F-11).
func (r *Report) Print(w io.Writer) {
	dur := r.EndTime.Sub(r.StartTime)
	sep := strings.Repeat("─", 45)
	fmt.Fprintln(w, sep)
	fmt.Fprintln(w, "gopipe Execution Summary")
	fmt.Fprintln(w, sep)
	fmt.Fprintf(w, "Producer      : %s\n", r.Producer)
	fmt.Fprintf(w, "Consumers     : %d\n", len(r.Consumers))
	fmt.Fprintf(w, "Mode          : %s\n", r.Mode)
	fmt.Fprintf(w, "Duration      : %s\n", dur.Round(time.Millisecond))
	if r.Binary {
		fmt.Fprintf(w, "Bytes         : %s\n", fmtBytes(r.Bytes))
	} else {
		fmt.Fprintf(w, "Lines         : %s\n", fmtInt(r.Lines))
	}
	for i, c := range r.Consumers {
		status := "OK"
		if c.Failed.Load() > 0 {
			status = "FAIL"
		}
		fmt.Fprintf(w, "Consumer [%d]  : %-38s — %s  (%d runs, %d failed, %d retried, %d recovered)\n",
			i+1, c.Label, status,
			c.Runs.Load(), c.Failed.Load(), c.Retried.Load(), c.Recovered.Load())
	}
	fmt.Fprintf(w, "Exit Code     : %d\n", r.ExitCode)
	fmt.Fprintln(w, sep)
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

func fmtBytes(b int64) string {
	switch {
	case b >= 1<<20:
		return fmt.Sprintf("%.2f MB", float64(b)/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.2f KB", float64(b)/(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
