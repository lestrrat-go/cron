package cron

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// Configuration options for creating a parser. Most options specify which
// fields should be included, while others enable features. If a field is not
// included the parser will assume a default value. These options do not change
// the order fields are parse in.
type ParseOption int

const (
	ParseSecond      ParseOption = 1 << iota // Seconds field, default 0
	ParseMinute                              // Minutes field, default 0
	ParseHour                                // Hours field, default 0
	ParseDom                                 // Day of month field, default *
	ParseMonth                               // Month field, default *
	ParseDow                                 // Day of week field, default *
	ParseDowOptional                         // Optional day of week field, default *
	ParseDescriptor                          // Allow descriptors such as @monthly, @weekly, etc.
)

var places = []ParseOption{
	ParseSecond,
	ParseMinute,
	ParseHour,
	ParseDom,
	ParseMonth,
	ParseDow,
}

var defaults = []string{
	"0",
	"0",
	"0",
	"*",
	"*",
	"*",
}

// Creates a custom Parser with custom options.
//
//  // Standard parser without descriptors
//  specParser := NewParser(Minute | Hour | Dom | Month | Dow)
//  sched, err := specParser.Parse("0 0 15 */3 *")
//
//  // Same as above, just excludes time fields
//  subsParser := NewParser(Dom | Month | Dow)
//  sched, err := specParser.Parse("15 */3 *")
//
//  // Same as above, just makes Dow optional
//  subsParser := NewParser(Dom | Month | DowOptional)
//  sched, err := specParser.Parse("15 */3")
//
func NewParser(options ParseOption) *Parser {
	optionals := 0
	if options&ParseDowOptional > 0 {
		options |= ParseDow
		optionals++
	}
	return &Parser{options, optionals}
}

// Parse returns a new crontab schedule representing the given spec.
// It returns a descriptive error if the spec is not valid.
// It accepts crontab specs and features configured by NewParser.
func (p Parser) Parse(spec string) (Schedule, error) {
	if len(spec) == 0 {
		return nil, fmt.Errorf("Empty spec string")
	}
	if spec[0] == '@' && p.options&ParseDescriptor > 0 {
		return parseDescriptor(spec)
	}

	// Figure out how many fields we need
	max := 0
	for _, place := range places {
		if p.options&place > 0 {
			max++
		}
	}
	min := max - p.optionals

	// Split fields on whitespace
	fields := strings.Fields(spec)

	// Validate number of fields
	if count := len(fields); count < min || count > max {
		if min == max {
			return nil, fmt.Errorf("Expected exactly %d fields, found %d: %s", min, count, spec)
		}
		return nil, fmt.Errorf("Expected %d to %d fields, found %d: %s", min, max, count, spec)
	}

	// Fill in missing fields
	fields = expandFields(fields, p.options)

	var err error
	field := func(field string, r bounds) uint64 {
		if err != nil {
			return 0
		}
		var bits uint64
		bits, err = getField(field, r)
		return bits
	}

	var (
		second     = field(fields[0], seconds)
		minute     = field(fields[1], minutes)
		hour       = field(fields[2], hours)
		dayofmonth = field(fields[3], dom)
		month      = field(fields[4], months)
		dayofweek  = field(fields[5], dow)
	)
	if err != nil {
		return nil, err
	}

	return &schedule{
		Second: second,
		Minute: minute,
		Hour:   hour,
		Dom:    dayofmonth,
		Month:  month,
		Dow:    dayofweek,
	}, nil
}

func expandFields(fields []string, options ParseOption) []string {
	n := 0
	count := len(fields)
	expFields := make([]string, len(places))
	copy(expFields, defaults)
	for i, place := range places {
		if options&place > 0 {
			expFields[i] = fields[n]
			n++
		}
		if n == count {
			break
		}
	}
	return expFields
}

var StandardParser = NewParser(
	ParseMinute | ParseHour | ParseDom | ParseMonth | ParseDow | ParseDescriptor,
)

// ParseStandard returns a new crontab schedule representing the given standardSpec
// (https://en.wikipedia.org/wiki/Cron). It differs from Parse requiring to always
// pass 5 entries representing: minute, hour, day of month, month and day of week,
// in that order. It returns a descriptive error if the spec is not valid.
//
// It accepts
//   - Standard crontab specs, e.g. "* * * * ?"
//   - Descriptors, e.g. "@midnight", "@every 1h30m"
func ParseStandard(standardSpec string) (Schedule, error) {
	return StandardParser.Parse(standardSpec)
}

var DefaultParser = NewParser(
	ParseSecond | ParseMinute | ParseHour | ParseDom | ParseMonth | ParseDowOptional | ParseDescriptor,
)

// Parse returns a new crontab schedule representing the given spec.
// It returns a descriptive error if the spec is not valid.
//
// It accepts
//   - Full crontab specs, e.g. "* * * * * ?"
//   - Descriptors, e.g. "@midnight", "@every 1h30m"
func Parse(spec string) (Schedule, error) {
	return DefaultParser.Parse(spec)
}

// getField returns an Int with the bits set representing all of the times that
// the field represents or error parsing field value.  A "field" is a comma-separated
// list of "ranges".
func getField(field string, r bounds) (uint64, error) {
	var bits uint64
	ranges := strings.FieldsFunc(field, func(r rune) bool { return r == ',' })
	for _, expr := range ranges {
		bit, err := getRange(expr, r)
		if err != nil {
			return bits, err
		}
		bits |= bit
	}
	return bits, nil
}

// getRange returns the bits indicated by the given expression:
//   number | number "-" number [ "/" number ]
// or error parsing range.
func getRange(expr string, r bounds) (uint64, error) {
	var (
		start, end, step uint
		rangeAndStep     = strings.Split(expr, "/")
		lowAndHigh       = strings.Split(rangeAndStep[0], "-")
		singleDigit      = len(lowAndHigh) == 1
		err              error
	)

	var extra uint64
	if lowAndHigh[0] == "*" || lowAndHigh[0] == "?" {
		start = r.min
		end = r.max
		extra = starBit
	} else {
		start, err = parseIntOrName(lowAndHigh[0], r.names)
		if err != nil {
			return 0, err
		}
		switch len(lowAndHigh) {
		case 1:
			end = start
		case 2:
			end, err = parseIntOrName(lowAndHigh[1], r.names)
			if err != nil {
				return 0, err
			}
		default:
			return 0, fmt.Errorf("Too many hyphens: %s", expr)
		}
	}

	switch len(rangeAndStep) {
	case 1:
		step = 1
	case 2:
		step, err = mustParseInt(rangeAndStep[1])
		if err != nil {
			return 0, err
		}

		// Special handling: "N/step" means "N-max/step".
		if singleDigit {
			end = r.max
		}
	default:
		return 0, fmt.Errorf("Too many slashes: %s", expr)
	}

	if start < r.min {
		return 0, fmt.Errorf("Beginning of range (%d) below minimum (%d): %s", start, r.min, expr)
	}
	if end > r.max {
		return 0, fmt.Errorf("End of range (%d) above maximum (%d): %s", end, r.max, expr)
	}
	if start > end {
		return 0, fmt.Errorf("Beginning of range (%d) beyond end of range (%d): %s", start, end, expr)
	}
	if step == 0 {
		return 0, fmt.Errorf("Step of range should be a positive number: %s", expr)
	}

	return getBits(start, end, step) | extra, nil
}

// parseIntOrName returns the (possibly-named) integer contained in expr.
func parseIntOrName(expr string, names map[string]uint) (uint, error) {
	if names != nil {
		if namedInt, ok := names[strings.ToLower(expr)]; ok {
			return namedInt, nil
		}
	}
	return mustParseInt(expr)
}

// mustParseInt parses the given expression as an int or returns an error.
func mustParseInt(expr string) (uint, error) {
	num, err := strconv.Atoi(expr)
	if err != nil {
		return 0, fmt.Errorf("Failed to parse int from %s: %s", expr, err)
	}
	if num < 0 {
		return 0, fmt.Errorf("Negative number (%d) not allowed: %s", num, expr)
	}

	return uint(num), nil
}

// getBits sets all bits in the range [min, max], modulo the given step size.
func getBits(min, max, step uint) uint64 {
	var bits uint64

	// If step is 1, use shifts.
	if step == 1 {
		return ^(math.MaxUint64 << (max + 1)) & (math.MaxUint64 << min)
	}

	// Else, use a simple loop.
	for i := min; i <= max; i += step {
		bits |= 1 << i
	}
	return bits
}

// all returns all bits within the given bounds.  (plus the star bit)
func all(r bounds) uint64 {
	return getBits(r.min, r.max, 1) | starBit
}

// parseDescriptor returns a predefined schedule for the expression, or error if none matches.
func parseDescriptor(descriptor string) (Schedule, error) {
	const every = "@every "
	if strings.HasPrefix(descriptor, every) {
		duration, err := time.ParseDuration(descriptor[len(every):])
		if err != nil {
			return nil, fmt.Errorf("Failed to parse duration %s: %s", descriptor, err)
		}
		return Every(duration), nil
	}

	var s schedule
	s.Second = 1 << seconds.min
	s.Minute = 1 << minutes.min
	s.Hour = 1 << hours.min
	s.Dom = 1 << dom.min
	s.Month = 1 << months.min
	s.Dow = 1 << dow.min

	switch descriptor {
	case "@yearly", "@annually":
		s.Dow = all(dow)
	case "@monthly":
		s.Month = all(months)
		s.Dow = all(dow)
	case "@weekly":
		s.Dom = all(dom)
		s.Month = all(months)
	case "@daily", "@midnight":
		s.Dom = all(dom)
		s.Month = all(months)
		s.Dow = all(dow)
	case "@hourly":
		s.Hour = all(hours)
		s.Dom = all(dom)
		s.Month = all(months)
		s.Dow = all(dow)
	default:
		return nil, fmt.Errorf("Unrecognized descriptor: %s", descriptor)
	}
	return &s, nil
}
