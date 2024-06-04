package clock

import (
	"time"
)

// Clock Interface to make testing easier.
type Clock interface {
	Now() time.Time
}

type Application struct {
	Location *time.Location
}

func (a *Application) Now() time.Time {
	return time.Now().In(a.Location)
}
