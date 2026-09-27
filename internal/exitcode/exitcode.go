// Package exitcode defines the structured exit code scheme for gopipe (spec §F-09).
package exitcode

const (
	OK              = 0  // producer + all consumers succeeded
	ConsumerFailed  = 1  // producer OK, at least one consumer failed
	Timeout         = 2  // pipeline killed by timeout
	Cancelled       = 3  // pipeline cancelled by user (Ctrl+C / SIGTERM)
	ProducerFailed  = 10 // producer failed, all consumers succeeded
	BothFailed      = 11 // producer failed + at least one consumer failed
)

// Combine returns the correct exit code given the producer and consumer outcome.
func Combine(producerFailed, consumerFailed bool) int {
	switch {
	case producerFailed && consumerFailed:
		return BothFailed
	case producerFailed:
		return ProducerFailed
	case consumerFailed:
		return ConsumerFailed
	default:
		return OK
	}
}
