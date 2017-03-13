package cron

import (
	"context"
	"sync"
	"time"
)

const (
	locationKey = "location"
	parserKey = "parser"
)

type JobFunc func(context.Context)
type Job interface {
	Run(context.Context)
}

// The Schedule describes a job's duty cycle.
type Schedule interface {
	// Return the next activation time, later than the given time.
	// Next is invoked initially, and then each time the job is run.
	Next(time.Time) time.Time
}

// constantDelay represents a simple recurring duty cycle, 
// e.g. "Every 5 minutes". It does not support jobs more frequent
// than once a second.
type constantDelay struct {
  delay time.Duration
}

// schedule specifies a duty cycle (to the second granularity),
// based on a traditional crontab specification. It is computed
// initially and stored as bit sets.
type schedule struct {
	Second, Minute, Hour, Dom, Month, Dow uint64
}

// Tab represents a crontab object that manages entries, and
// dispatches jobs when appropriate
type Tab struct {
	changed     bool
	condChanged *sync.Cond
	entries     []Entry
	location    *time.Location
	muEntries   sync.RWMutex
	muChanged   sync.Mutex
	parser      *Parser
}

type Option interface{
	Name() string
	Value() interface{}
}
type option struct {
	name  string
	value interface{}
}

// A custom Parser that can be configured.
type Parser struct {
	options   ParseOption
	optionals int
}

type Entry interface {
	ID() string
	Next() time.Time
	ComputeNext(time.Time)
	Run(context.Context)
}
type entry struct {
	id       string
	job      Job
	next     time.Time
	schedule Schedule
}

type byTime []Entry
