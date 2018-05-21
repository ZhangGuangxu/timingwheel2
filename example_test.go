package timingwheel

import (
	"fmt"
	"log"
	"sync"
	"time"
)

type wheelObserver struct {
	step          int
	releaseCnt    int
	moveCnt       int
	afterStepFunc func(w *wheelObserver)
}

func (w *wheelObserver) beforeStep() {
	w.step++
	w.releaseCnt = 0
	w.moveCnt = 0
}

func (w *wheelObserver) afterStep() {
	w.afterStepFunc(w)
}

func afterStepOutput(w *wheelObserver) {
	log.Printf("step[%d], releaseCnt[%d], moveCnt[%d]\n", w.step, w.releaseCnt, w.moveCnt)
}

func afterStepNooutput(w *wheelObserver) {}

func (w *wheelObserver) afterRelease() {
	w.releaseCnt++
}

func (w *wheelObserver) afterMove() {
	//log.Println(w.moveCnt)
	w.moveCnt++
	//log.Println(w.moveCnt)
}

func ExampleTimingWheel() {
	slotCnt := 200
	w, err := NewTimingWheel(maxTime, slotCnt)
	if err != nil {
		log.Println(err)
		return
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

	ob := &wheelObserver{
		afterStepFunc: afterStepOutput,
	}

	wg.Add(1)
	go w.runWithStepObserver(shouldQuit, deferFunc, ob)

	quitAdder := make(chan bool, 1)
	wg.Add(1)
	go func() {
		defer deferFunc()

		ticker := time.NewTicker(10 * time.Millisecond)

		for {
			select {
			case <-quitAdder:
				return
			case <-ticker.C:
				w.AddItem(newItem())
			}
		}
	}()

	time.Sleep(1000 * time.Millisecond)
	quitAdder <- true
	time.Sleep(2000 * time.Millisecond)
	quit <- true
	wg.Wait()
}

func ExampleTimingWheel2() {
	// Output: go
	slotCnt := 200
	w, err := NewTimingWheel(maxTime, slotCnt)
	if err != nil {
		log.Println(err)
		return
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

	ob := &wheelObserver{
		afterStepFunc: afterStepNooutput,
	}

	wg.Add(1)
	go w.runWithStepObserver(shouldQuit, deferFunc, ob)

	quitAdder := make(chan bool, 1)
	wg.Add(1)
	go func() {
		defer deferFunc()

		ticker := time.NewTicker(10 * time.Millisecond)

		for {
			select {
			case <-quitAdder:
				return
			case <-ticker.C:
				w.AddItem(newItem())
			}
		}
	}()

	time.Sleep(1000 * time.Millisecond)
	quitAdder <- true
	time.Sleep(2000 * time.Millisecond)
	quit <- true
	wg.Wait()

	fmt.Printf("%d\n", w.itemCount())
	// Output: 0
}
