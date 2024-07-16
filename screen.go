package fnf

import (
	"time"
)

type Screen interface {
	Update(time.Duration)
	Draw()
	BeforeScreenTransition()
	BeforeScreenEnd()
	Free()
}
