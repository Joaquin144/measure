package cmd

import "time"

// Metrics stores certain metrics used by the program
// to keep track of progress of ingestion operations.
type Metrics struct {
	AppCount       int
	EventFileCount int
	EventCount     int
	BuildCount     int
	ingestDuration time.Duration
}

// bumpBuild bumps the build count of
// Metrics.
func (m *Metrics) bumpBuild() {
	m.BuildCount = m.BuildCount + 1
}

// bumpEventFile bumps the event file count
// of Metrics.
func (m *Metrics) bumpEventFile() {
	m.EventFileCount = m.EventFileCount + 1
}

// bumpEvent bumps the event count by
// n.
func (m *Metrics) bumpEvent(n int) {
	m.EventCount = m.EventCount + n
}

// bumpApp bumps the app count
// Metrics.
func (m *Metrics) bumpApp() {
	m.AppCount = m.AppCount + 1
}

// setIngestDuration sets the total duration of
// events ingestion.
func (m *Metrics) setIngestDuration(d time.Duration) {
	m.ingestDuration = d
}
