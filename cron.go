package cron

import (
	"context"
	"crypto/rand"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/pkg/errors"
)

func uuid() string {
	b := make([]byte, 16)
	rand.Reader.Read(b)
	b[6] = (b[6] & 0x0F) | 0x40
	b[8] = (b[8] &^ 0x40) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func (s byTime) Len() int      { return len(s) }
func (s byTime) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s byTime) Less(i, j int) bool {
	// Two zero times should return false.
	// Otherwise, zero is "greater" than any other time.
	// (To sort it at the end of the list.)
	nextI := s[i].Next()
	nextJ := s[j].Next()
	if nextI.IsZero() {
		return false
	}
	if nextJ.IsZero() {
		return true
	}
	return nextI.Before(nextJ)
}

func (j JobFunc) Run(ctx context.Context) {
	j(ctx)
}

func newEntry(p *Parser, spec string, job Job) (*entry, error) {
	s, err := p.Parse(spec)
	if err != nil {
		return nil, errors.Wrapf(err, `failed to parse cron spec: %s`, spec)
	}

	return &entry{
		id:       uuid(),
		job:      job,
		schedule: s,
	}, nil
}

func (e *entry) ID() string {
	return e.id
}
func (e *entry) ComputeNext(t time.Time) {
	e.next = e.schedule.Next(t)
}
func (e *entry) Next() time.Time {
	return e.next
}
func (e *entry) Run(ctx context.Context) {
	e.job.Run(ctx)
}

func New(options ...Option) *Tab {
	t := &Tab{
		location: time.Local,
		parser:   DefaultParser,
	}
	t.condChanged = sync.NewCond(&t.muChanged)

	for _, o := range options {
		switch o.Name() {
		case locationKey:
			t.location = o.Value().(*time.Location)
		case parserKey:
			t.parser = o.Value().(*Parser)
		}
	}

	return t
}

func (t *Tab) now() time.Time {
	return time.Now().Truncate(time.Second).In(t.location)
}

func (t *Tab) Schedule(spec string, job Job) (string, error) {
	e, err := newEntry(t.parser, spec, job)
	if err != nil {
		return "", errors.Wrap(err, `failed to schedule job`)
	}
	e.id = uuid()
	e.ComputeNext(t.now())

	t.muEntries.Lock()
	defer t.muEntries.Unlock()
	t.entries = append(t.entries, e)

	t.markChanged()
	return e.id, nil
}

func (t *Tab) markChanged() {
	t.muChanged.Lock()
	t.changed = true
	t.condChanged.Broadcast()
	t.muChanged.Unlock()
}

func (t *Tab) Remove(id string) error {
	t.muEntries.Lock()
	defer t.muEntries.Unlock()
	for i, e := range t.entries {
		if e.ID() == id {
			t.entries = append(
				append([]Entry(nil), t.entries[:i]...),
				t.entries[i+1:]...,
			)
			t.markChanged()
			return nil
		}
	}
	return errors.New(`not found`)
}

func (t *Tab) waitChange() <-chan struct{} {
	ch := make(chan struct{})
	go t.notifyChange(ch)
	return ch
}

func (t *Tab) notifyChange(ch chan struct{}) {
	defer close(ch)
	t.muChanged.Lock()
	for !t.changed {
		t.condChanged.Wait()
	}
	t.changed = false
	t.muChanged.Unlock()
	ch <- struct{}{}
}

func (t *Tab) Run(inctx context.Context) {
	ctx, cancel := context.WithCancel(inctx)
	defer cancel()

	t.muEntries.Lock()
	now := t.now()
	for _, e := range t.entries {
		e.ComputeNext(now)
	}
	t.muEntries.Unlock()

	for {
		// Create a copy so we can forget about locking
		t.muEntries.Lock()
		entries := make([]Entry, len(t.entries))
		copy(entries, t.entries)
		t.muEntries.Unlock()

		sort.Sort(byTime(entries))

		if len(entries) == 0 || entries[0].Next().IsZero() {
			select {
			case <-ctx.Done():
				return
			case <-t.waitChange():
			}
			continue
		}

		effective := entries[0].Next()
		select {
		case <-ctx.Done():
			return
		case now = <-time.After(effective.Sub(now)):
			for _, e := range entries {
				if n := e.Next(); n.After(now) {
					break
				}
				go e.Run(ctx)
				e.ComputeNext(now)
			}
			continue
		}
	}
}
