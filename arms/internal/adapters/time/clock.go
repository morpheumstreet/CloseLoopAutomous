package timeadapter

import (
	"time"

	"github.com/closeloopautomous/arms/internal/ports"
)

type System struct{}

func (System) Now() time.Time { return time.Now() }

var _ ports.Clock = System{}

type Fixed struct{ T time.Time }

func (f Fixed) Now() time.Time { return f.T }
