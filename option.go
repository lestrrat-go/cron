package cron

import "time"

func (o *option) Name() string {
	return o.name
}

func (o *option) Value() interface{} {
	return o.value
}

func WithParser(p *Parser) Option {
	return &option{
		name:  parserKey,
		value: p,
	}
}

func WithLocation(loc *time.Location) Option {
	return &option{
		name:  locationKey,
		value: loc,
	}
}
