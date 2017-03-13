package cron

import (
	"errors"
	"testing"
	"time"
)

func TestActivation(t *testing.T) {
	tests := []struct {
		time     string
		spec     string
		expected bool
	}{
		// Every fifteen minutes.
		{time: "Mon Jul 9 15:00 2012", spec: "0 0/15 * * *", expected: true},
		{time: "Mon Jul 9 15:45 2012", spec: "0 0/15 * * *", expected: true},
		{time: "Mon Jul 9 15:40 2012", spec: "0 0/15 * * *"},

		// Every fifteen minutes, starting at 5 minutes.
		{time: "Mon Jul 9 15:05 2012", spec: "0 5/15 * * *", expected: true},
		{time: "Mon Jul 9 15:20 2012", spec: "0 5/15 * * *", expected: true},
		{time: "Mon Jul 9 15:50 2012", spec: "0 5/15 * * *", expected: true},

		// Named months
		{time: "Sun Jul 15 15:00 2012", spec: "0 0/15 * * Jul", expected: true},
		{time: "Sun Jul 15 15:00 2012", spec: "0 0/15 * * Jun"},

		// Everything set.
		{time: "Sun Jul 15 08:30 2012", spec: "0 30 08 ? Jul Sun", expected: true},
		{time: "Sun Jul 15 08:30 2012", spec: "0 30 08 15 Jul ?", expected: true},
		{time: "Mon Jul 16 08:30 2012", spec: "0 30 08 ? Jul Sun"},
		{time: "Mon Jul 16 08:30 2012", spec: "0 30 08 15 Jul ?"},

		// Predefined schedules
		{time: "Mon Jul 9 15:00 2012", spec: "@hourly", expected: true},
		{time: "Mon Jul 9 15:04 2012", spec: "@hourly"},
		{time: "Mon Jul 9 15:00 2012", spec: "@daily"},
		{time: "Mon Jul 9 00:00 2012", spec: "@daily", expected: true},
		{time: "Mon Jul 9 00:00 2012", spec: "@weekly"},
		{time: "Sun Jul 8 00:00 2012", spec: "@weekly", expected: true},
		{time: "Sun Jul 8 01:00 2012", spec: "@weekly"},
		{time: "Sun Jul 8 00:00 2012", spec: "@monthly"},
		{time: "Sun Jul 1 00:00 2012", spec: "@monthly", expected: true},

		// Test interaction of DOW and DOM.
		// If both are specified, then only one needs to match.
		{time: "Sun Jul 15 00:00 2012", spec: "0 * * 1,15 * Sun", expected: true},
		{time: "Fri Jun 15 00:00 2012", spec: "0 * * 1,15 * Sun", expected: true},
		{time: "Wed Aug 1 00:00 2012", spec: "0 * * 1,15 * Sun", expected: true},

		// However, if one has a star, then both need to match.
		{time: "Sun Jul 15 00:00 2012", spec: "0 * * * * Mon"},
		{time: "Sun Jul 15 00:00 2012", spec: "0 * * */10 * Sun"},
		{time: "Mon Jul 9 00:00 2012", spec: "0 * * 1,15 * *"},
		{time: "Sun Jul 15 00:00 2012", spec: "0 * * 1,15 * *", expected: true},
		{time: "Sun Jul 15 00:00 2012", spec: "0 * * */2 * Sun", expected: true},
	}

	for _, test := range tests {
		sched, err := Parse(test.spec)
		if err != nil {
			t.Error(err)
			continue
		}
		actual := sched.Next(getTime(test.time).Add(-1 * time.Second))
		expected := getTime(test.time)
		if test.expected && expected != actual || !test.expected && expected == actual {
			t.Errorf("Failed evaluating %s on %s: (expected) %s != %s (actual)",
				test.spec, test.time, expected, actual)
		}
	}
}

func TestNext(t *testing.T) {
	runs := []struct {
		time     string
		spec     string
		expected string
	}{
		// Simple cases
		{time: "Mon Jul 9 14:45 2012", spec: "0 0/15 * * *", expected: "Mon Jul 9 15:00 2012"},
		{time: "Mon Jul 9 14:59 2012", spec: "0 0/15 * * *", expected: "Mon Jul 9 15:00 2012"},
		{time: "Mon Jul 9 14:59:59 2012", spec: "0 0/15 * * *", expected: "Mon Jul 9 15:00 2012"},

		// Wrap around hours
		{time: "Mon Jul 9 15:45 2012", spec: "0 20-35/15 * * *", expected: "Mon Jul 9 16:20 2012"},

		// Wrap around days
		{time: "Mon Jul 9 23:46 2012", spec: "0 */15 * * *", expected: "Tue Jul 10 00:00 2012"},
		{time: "Mon Jul 9 23:45 2012", spec: "0 20-35/15 * * *", expected: "Tue Jul 10 00:20 2012"},
		{time: "Mon Jul 9 23:35:51 2012", spec: "15/35 20-35/15 * * *", expected: "Tue Jul 10 00:20:15 2012"},
		{time: "Mon Jul 9 23:35:51 2012", spec: "15/35 20-35/15 1/2 * *", expected: "Tue Jul 10 01:20:15 2012"},
		{time: "Mon Jul 9 23:35:51 2012", spec: "15/35 20-35/15 10-12 * *", expected: "Tue Jul 10 10:20:15 2012"},

		{time: "Mon Jul 9 23:35:51 2012", spec: "15/35 20-35/15 1/2 */2 * *", expected: "Thu Jul 11 01:20:15 2012"},
		{time: "Mon Jul 9 23:35:51 2012", spec: "15/35 20-35/15 * 9-20 * *", expected: "Wed Jul 10 00:20:15 2012"},
		{time: "Mon Jul 9 23:35:51 2012", spec: "15/35 20-35/15 * 9-20 Jul *", expected: "Wed Jul 10 00:20:15 2012"},

		// Wrap around months
		{time: "Mon Jul 9 23:35 2012", spec: "0 0 0 9 Apr-Oct ?", expected: "Thu Aug 9 00:00 2012"},
		{time: "Mon Jul 9 23:35 2012", spec: "0 0 0 */5 Apr,Aug,Oct Mon", expected: "Mon Aug 6 00:00 2012"},
		{time: "Mon Jul 9 23:35 2012", spec: "0 0 0 */5 Oct Mon", expected: "Mon Oct 1 00:00 2012"},

		// Wrap around years
		{time: "Mon Jul 9 23:35 2012", spec: "0 0 0 * Feb Mon", expected: "Mon Feb 4 00:00 2013"},
		{time: "Mon Jul 9 23:35 2012", spec: "0 0 0 * Feb Mon/2", expected: "Fri Feb 1 00:00 2013"},

		// Wrap around minute, hour, day, month, and year
		{time: "Mon Dec 31 23:59:45 2012", spec: "0 * * * * *", expected: "Tue Jan 1 00:00:00 2013"},

		// Leap year
		{time: "Mon Jul 9 23:35 2012", spec: "0 0 0 29 Feb ?", expected: "Mon Feb 29 00:00 2016"},

		// Daylight savings time 2am EST (-5) -> 3am EDT (-4)
		{time: "2012-03-11T00:00:00-0500", spec: "0 30 2 11 Mar ?", expected: "2013-03-11T02:30:00-0400"},

		// hourly job
		{time: "2012-03-11T00:00:00-0500", spec: "0 0 * * * ?", expected: "2012-03-11T01:00:00-0500"},
		{time: "2012-03-11T01:00:00-0500", spec: "0 0 * * * ?", expected: "2012-03-11T03:00:00-0400"},
		{time: "2012-03-11T03:00:00-0400", spec: "0 0 * * * ?", expected: "2012-03-11T04:00:00-0400"},
		{time: "2012-03-11T04:00:00-0400", spec: "0 0 * * * ?", expected: "2012-03-11T05:00:00-0400"},

		// 1am nightly job
		{time: "2012-03-11T00:00:00-0500", spec: "0 0 1 * * ?", expected: "2012-03-11T01:00:00-0500"},
		{time: "2012-03-11T01:00:00-0500", spec: "0 0 1 * * ?", expected: "2012-03-12T01:00:00-0400"},

		// 2am nightly job (skipped)
		{time: "2012-03-11T00:00:00-0500", spec: "0 0 2 * * ?", expected: "2012-03-12T02:00:00-0400"},

		// Daylight savings time 2am EDT (-4) => 1am EST (-5)
		{time: "2012-11-04T00:00:00-0400", spec: "0 30 2 04 Nov ?", expected: "2012-11-04T02:30:00-0500"},
		{time: "2012-11-04T01:45:00-0400", spec: "0 30 1 04 Nov ?", expected: "2012-11-04T01:30:00-0500"},

		// hourly job
		{time: "2012-11-04T00:00:00-0400", spec: "0 0 * * * ?", expected: "2012-11-04T01:00:00-0400"},
		{time: "2012-11-04T01:00:00-0400", spec: "0 0 * * * ?", expected: "2012-11-04T01:00:00-0500"},
		{time: "2012-11-04T01:00:00-0500", spec: "0 0 * * * ?", expected: "2012-11-04T02:00:00-0500"},

		// 1am nightly job (runs twice)
		{time: "2012-11-04T00:00:00-0400", spec: "0 0 1 * * ?", expected: "2012-11-04T01:00:00-0400"},
		{time: "2012-11-04T01:00:00-0400", spec: "0 0 1 * * ?", expected: "2012-11-04T01:00:00-0500"},
		{time: "2012-11-04T01:00:00-0500", spec: "0 0 1 * * ?", expected: "2012-11-05T01:00:00-0500"},

		// 2am nightly job
		{time: "2012-11-04T00:00:00-0400", spec: "0 0 2 * * ?", expected: "2012-11-04T02:00:00-0500"},
		{time: "2012-11-04T02:00:00-0500", spec: "0 0 2 * * ?", expected: "2012-11-05T02:00:00-0500"},

		// 3am nightly job
		{time: "2012-11-04T00:00:00-0400", spec: "0 0 3 * * ?", expected: "2012-11-04T03:00:00-0500"},
		{time: "2012-11-04T03:00:00-0500", spec: "0 0 3 * * ?", expected: "2012-11-05T03:00:00-0500"},

		// Unsatisfiable
		{time: "Mon Jul 9 23:35 2012", spec: "0 0 0 30 Feb ?", expected: ""},
		{time: "Mon Jul 9 23:35 2012", spec: "0 0 0 31 Apr ?", expected: ""},
	}

	for _, c := range runs {
		sched, err := Parse(c.spec)
		if err != nil {
			t.Error(err)
			continue
		}
		actual := sched.Next(getTime(c.time))
		expected := getTime(c.expected)
		if !actual.Equal(expected) {
			t.Errorf("%s, \"%s\": (expected) %v != %v (actual)", c.time, c.spec, expected, actual)
		}
	}
}

func TestErrors(t *testing.T) {
	invalidSpecs := []string{
		"xyz",
		"60 0 * * *",
		"0 60 * * *",
		"0 0 * * XYZ",
		"TZ=Bogus * * * * *",
		"0-0-0 * * * *",
		"*/5/5 * * * *",
		"* * * 0-5 * *",
		"59-2 * * * *",
	}
	for _, spec := range invalidSpecs {
		_, err := Parse(spec)
		if err == nil {
			t.Error("expected an error parsing: ", spec)
		}
	}
}

func getTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	t, err := time.Parse("Mon Jan 2 15:04 2006", value)
	if err != nil {
		t, err = time.Parse("Mon Jan 2 15:04:05 2006", value)
		if err != nil {
			t, err = time.Parse("2006-01-02T15:04:05-0700", value)
			if err != nil {
				panic(err)
			}
			// Daylight savings time tests require location
			if ny, err := time.LoadLocation("America/New_York"); err == nil {
				t = t.In(ny)
			}
		}
	}

	return t
}

func TestNextWithTz(t *testing.T) {
	runs := []struct {
		time, spec string
		expected   string
	}{
		// Failing tests
		{"2016-01-03T13:09:03+0530", "0 14 14 * * *", "2016-01-03T14:14:00+0530"},
		{"2016-01-03T04:09:03+0530", "0 14 14 * * ?", "2016-01-03T14:14:00+0530"},

		// Passing tests
		{"2016-01-03T14:09:03+0530", "0 14 14 * * *", "2016-01-03T14:14:00+0530"},
		{"2016-01-03T14:00:00+0530", "0 14 14 * * ?", "2016-01-03T14:14:00+0530"},
	}
	for _, c := range runs {
		sched, err := Parse(c.spec)
		if err != nil {
			t.Error(err)
			continue
		}

		effective, err := getTimeTZ(c.time)
		if err != nil {
			t.Errorf("failed to parse spec time: %s", c.time)
			return
		}
		actual := sched.Next(effective)

		expected, err := getTimeTZ(c.expected)
		if err != nil {
			t.Errorf("failed to parse spec expected time: %s", c.expected)
			return
		}

		if !actual.Equal(expected) {
			t.Errorf("%s, \"%s\": (expected) %v != %v (actual)", c.time, c.spec, expected, actual)
		}
	}
}

func getTimeTZ(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse("Mon Jan 2 15:04 2006", value)
	if err == nil {
		return t, nil
	}

	t, err = time.Parse("Mon Jan 2 15:04:05 2006", value)
	if err == nil {
		return t, nil
	}

	t, err = time.Parse("2006-01-02T15:04:05-0700", value)
	if err == nil {
		return t, nil
	}

	return time.Time{}, errors.New("could not parse time")
}
