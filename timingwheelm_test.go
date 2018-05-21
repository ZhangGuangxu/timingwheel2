package timingwheel

import (
	"sync"
	"testing"
	"time"
)

var count = 1000
var maxTime = time.Duration(int64(count) * int64(time.Millisecond))

type Item struct {
	createTime time.Time
}

func newItem() *Item {
	return &Item{
		createTime: time.Now(),
	}
}

func (i *Item) ShouldRelease() bool {
	return time.Now().Sub(i.createTime) > maxTime
}

func (i *Item) Release() {

}

func TestWheel(test *testing.T) {
	slotCnt := 100
	w, err := NewTimingWheel(maxTime, slotCnt)
	if err != nil {
		test.Errorf("NewTimingWheel return error[%v]", err)
	}

	quit := make(chan bool, 1)
	shouldQuit := func() bool {
		select {
		case <-quit:
			return true
		default:
			return false
		}
	}

	var wg sync.WaitGroup
	deferFunc := func() {
		wg.Done()
	}

	wg.Add(1)
	go w.Run(shouldQuit, deferFunc)

	wg.Add(1)
	go func() {
		defer deferFunc()

		ticker := time.NewTicker(1 * time.Millisecond)
		endT := time.Now().Add(1 * time.Second)

		for time.Now().Before(endT) {
			select {
			case <-ticker.C:
				w.AddItem(newItem())
			}
		}
	}()

	now := time.Now()
	t1 := now.Add(1 * time.Second)
	tm := now.Add(1100 * time.Millisecond)
	t2 := now.Add(2 * time.Second)
	var itemCount int
	ticker := time.NewTicker(100 * time.Millisecond)

	for t := now; t.Before(t2); {
		select {
		case <-ticker.C:
			t = time.Now()
			c := w.itemCount()
			if t.Before(t1) {
				if !(c > itemCount) {
					test.Error("not more than before")
				}
			} else if t.After(tm) {
				if !(c < itemCount) {
					test.Error("not less than before")
				}
			}
			itemCount = c
		}
	}

	time.Sleep(1500 * time.Millisecond)
	quit <- true
	wg.Wait()
}

func TestTimingwheel2(test *testing.T) {
	if _, err := NewTimingWheel(-1, 2); err == nil {
		test.Error("should return error")
	}
	if _, err := NewTimingWheel(0, 2); err == nil {
		test.Error("should return error")
	}
	if _, err := NewTimingWheel(2, -1); err == nil {
		test.Error("should return error")
	}
	if _, err := NewTimingWheel(2, 0); err == nil {
		test.Error("should return error")
	}

	if _, err := NewTimingWheel(1, 2); err == nil {
		test.Error("should return error")
	}

	w, _ := NewTimingWheel(3*time.Nanosecond, 2)
	if w.stepTime != 2 {
		test.Error("w.stepTime should be 2")
	}

	slotCnt := 5
	w, err := NewTimingWheel(maxTime, slotCnt)
	if err != nil {
		test.Errorf("NewTimingWheel return error[%v]", err)
	}

	quit := make(chan bool, 1)
	shouldQuit := func() bool {
		select {
		case <-quit:
			return true
		default:
			return false
		}
	}

	var wg sync.WaitGroup
	deferFunc := func() {
		wg.Done()
	}

	wg.Add(1)
	go w.Run(shouldQuit, deferFunc)

	wg.Add(1)
	go func() {
		defer deferFunc()

		ticker := time.NewTicker(1 * time.Millisecond)
		endT := time.Now().Add(2 * time.Second)

		for time.Now().Before(endT) {
			select {
			case <-ticker.C:
				w.AddItem(newItem())
			}
		}
	}()

	time.Sleep(2100 * time.Millisecond)
	quit <- true
	wg.Wait()
}
