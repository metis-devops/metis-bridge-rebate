package policy

import "time"

type RebateType int

const (
	DefaultRebateType RebateType = iota
	GasFeeRebateType
)

type Drip struct {
	Name         string
	MatchAll     bool
	MatchToken   map[string]bool
	CheckIfFirst bool
	CheckIfNoGas bool
	StartTime    time.Time
	EndTime      time.Time
	MinUSDEqual  float64
	RebateType   RebateType
}

func (d *Drip) Match(time time.Time, token string) bool {
	return d.MatchAll || (time.After(d.StartTime) && time.Before(d.EndTime) && d.MatchToken[token])
}
