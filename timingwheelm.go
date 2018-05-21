package timingwheel

import (
	"errors"
	"sync"
	"time"
)

// Releaser is a interface decribes objects which can release.
type Releaser interface {
	ShouldRelease() bool
	Release()
}

const (
	initSize = 32
)

var errInvalidDuration = errors.New("invalid duration")

type itemCon map[interface{}]bool
type slotCon []itemCon

// TimingWheel is a data structure that manages some items which may be out of time soon.
type TimingWheel struct {
	stepTime time.Duration // Every step time, the wheel rolls one index forward.

	mux      sync.Mutex // for slots and curIndex
	slots    slotCon
	curIndex int
}

// NewTimingWheel returns a TimingWheel instance.
// Parameter max is the cycle of every item get checked.
// Parameter slotCnt is the slot count of the wheel. Every slot can contain a few items.
// max and slotCnt MUST be positive.
// max MUST >= slotCnt. It is the best way that max%slotCnt == 0.
func NewTimingWheel(max time.Duration, slotCnt int) (*TimingWheel, error) {
	if max <= 0 || slotCnt <= 0 {
		return nil, errInvalidDuration
	}

	n := max.Nanoseconds()
	c := int64(slotCnt)
	if n < c {
		return nil, errInvalidDuration
	}
	s := n / c
	if n%c > 0 {
		s++
	}

	slots := make(slotCon, slotCnt)
	for i := range slots {
		slots[i] = make(itemCon, initSize)
	}

	tw := &TimingWheel{
		stepTime: time.Duration(s),
		slots:    slots,
	}
	return tw, nil
}

// AddItem adds an item to this wheel.
func (tw *TimingWheel) AddItem(item interface{}) {
	tw.mux.Lock()
	defer tw.mux.Unlock()
	tw.slots[tw.curIndex][item] = true
}

// Run rolls this wheel.
// Run has a dead loop.
func (tw *TimingWheel) Run(shouldQuit func() bool, deferFunc func()) {
	defer deferFunc()

	ticker := time.NewTicker(tw.stepTime)

	for {
		if shouldQuit() {
			break
		}

		select {
		case <-ticker.C:
			tw.stepForward()
		}
	}
}

type timingwheelObserver interface {
	beforeStep()
	afterStep()
	afterRelease()
	afterMove()
}

// runWithStepObserver needs two step observers.
// This function is for ease of unit test.
func (tw *TimingWheel) runWithStepObserver(shouldQuit func() bool, deferFunc func(), ob timingwheelObserver) {
	defer deferFunc()

	ticker := time.NewTicker(tw.stepTime)

	for {
		if shouldQuit() {
			break
		}

		select {
		case <-ticker.C:
			ob.beforeStep()
			tw.stepForwardWithObserver(ob)
			ob.afterStep()
		}
	}
}

// move one index forward and check items in new index.
func (tw *TimingWheel) stepForward() {
	tw.mux.Lock()
	defer tw.mux.Unlock()

	idx := tw.curIndex + 1
	if idx >= len(tw.slots) {
		idx = 0
	}
	tw.curIndex = idx

	curSlot := tw.slots[idx]

	for k := range curSlot {
		if r, ok := k.(Releaser); ok {
			if r.ShouldRelease() {
				delete(curSlot, k)
				r.Release()
			}
		}
	}
}

func (tw *TimingWheel) stepForwardWithObserver(ob timingwheelObserver) {
	tw.mux.Lock()
	defer tw.mux.Unlock()

	idx := tw.curIndex + 1
	if idx >= len(tw.slots) {
		idx = 0
	}
	tw.curIndex = idx

	curSlot := tw.slots[idx]

	for k := range curSlot {
		if r, ok := k.(Releaser); ok {
			if r.ShouldRelease() {
				delete(curSlot, k)
				r.Release()
				ob.afterRelease()
			}
		}
	}
}

// itemCount returns the item count in this wheel.
// This function is just for unit test.
func (tw *TimingWheel) itemCount() int {
	tw.mux.Lock()
	defer tw.mux.Unlock()

	var total int
	for _, slot := range tw.slots {
		total += len(slot)
	}
	return total
}
