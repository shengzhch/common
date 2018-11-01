//分发器多goruntinues处理job函数
package dispatcher

import (
	"sync"
)

type Job func()

type Worker struct {
	JobQueue chan Job
	wg       *sync.WaitGroup
}

func NewWorker(jq chan Job, wg *sync.WaitGroup) Worker {
	return Worker{
		wg:       wg,
		JobQueue: jq,
	}
}

func (w Worker) Start() {
	go func() {
		for c := range w.JobQueue {
			c()
			w.wg.Done()
		}
	}()
}

type Dispatcher struct {
	worknum  int
	maxjob   int
	wg       *sync.WaitGroup
	JobQueue chan Job
}

func NewDispatcher(w, m int) *Dispatcher {
	return &Dispatcher{
		worknum:  w,
		wg:       new(sync.WaitGroup),
		JobQueue: make(chan Job, m),
	}
}

func (d *Dispatcher) Run() {
	for i := 0; i < d.worknum; i++ {
		worker := NewWorker(d.JobQueue, d.wg)
		worker.Start()
	}
}

func (d *Dispatcher) Receive(f func()) {
	d.wg.Add(1)
	d.JobQueue <- f
}

func (d *Dispatcher) WaitForEnd() {
	d.wg.Wait()
	close(d.JobQueue)
}
