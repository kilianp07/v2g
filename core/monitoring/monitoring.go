package monitoring

import "time"

// Monitor defines methods used for error reporting.
type Monitor interface {
	CaptureException(err error, tags map[string]string)
	Recover()
	Flush(timeout time.Duration)
}

type NopMonitor struct{}

func (NopMonitor) CaptureException(error, map[string]string) {}
func (NopMonitor) Recover()                                  {}
func (NopMonitor) Flush(time.Duration)                       {}

var current Monitor = NopMonitor{}

// Init sets the global monitor implementation.
func Init(m Monitor) {
	if m != nil {
		current = m
	}
}

// CaptureException records the error with optional tags.
func CaptureException(err error, tags map[string]string) {
	if current != nil {
		current.CaptureException(err, tags)
	}
}

// Recover captures panics in goroutines.
func Recover() {
	if current != nil {
		current.Recover()
	}
}

// Flush flushes buffered events.
func Flush(d time.Duration) {
	if current != nil {
		current.Flush(d)
	}
}
