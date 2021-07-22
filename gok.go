package gok

import (
	"sync"
	"sync/atomic"
	"time"
)

// Gokit the core interface to execute your job
// If a Gokit.Execute() is implemented by a goroutine internally, Gok can't track it.
// Negative e.g.
// type MyGokit struct {}
// func (MyGokit) Identifier() string { return "My Gokit" }
// func (MyGokit) Tag() string { return "My Gokit" }
// func (MyGokit) Execute() error {
//	go func(){
//		fmt.Println("My Gokit")
//	}()
//	return nil
// }
type Gokit interface {
	Execute() error
	Identifier() string // Identifier specific a error's attribution
	Tag() string // Tag specific a group of executor's error
}

type gokit struct {
	identifier string
	tag string
	executor GokitFunc
}

func (gkt *gokit) Execute() error {
	if gkt.executor == nil {
		return nil
	}
	return gkt.executor()
}

func (gkt *gokit) Identifier() string {
	return gkt.identifier
}

func (gkt *gokit) Tag() string {
	return gkt.tag
}

// GokitFunc a lightweight implement of Gokit, generally use when no need to modify executor's attributes.
// if a GokitFunc implemented by a goroutine internally, Gok can't track it.
type GokitFunc func() error

type gokitFuncWrapper struct {}

func (wrapper *gokitFuncWrapper) wrapGokitFunc(f GokitFunc, tag, identifier string) Gokit {
	gkt := &gokit{
		identifier: identifier,
		tag: tag,
		executor: f,
	}
	return gkt
}

type Gok struct {
	// attributes
	capacity uint
	concurrency uint32
	semaphore uint32 // semaphore's max value = concurrency
	runningGokitNum uint32
	waitingGokitNum uint32

	// core
	gokitQueue chan Gokit


	// signals
	pause chan pause
	restart chan restart
	shutdown chan shutdown

	// notify channel
	errorBuses map[string]*ErrorBus // tag => ErrorBus

	// internal
	lock sync.Locker
	wrapper *gokitFuncWrapper
}

func NewGok(capacity uint, concurrency uint32) *Gok {
	return &Gok{
		capacity: capacity,
		concurrency: concurrency,
		gokitQueue: make(chan Gokit,capacity),
		pause: make(chan pause),
		restart: make(chan restart),
		shutdown: make(chan shutdown),
		errorBuses: make(map[string]*ErrorBus),
		lock: &sync.Mutex{},
		wrapper: &gokitFuncWrapper{},
	}
}

func (gok *Gok) increaseNum(addr *uint32) {
	atomic.AddUint32(addr,uint32(1))
}

func (gok *Gok) decreaseNum(addr *uint32) {
	atomic.AddUint32(addr,^uint32(0))
}

func (gok *Gok) increaseRunningGokitNum() {
	gok.increaseNum(&gok.runningGokitNum)
}

func (gok *Gok) decreaseRunningGokitNum() {
	gok.decreaseNum(&gok.runningGokitNum)
}

func (gok *Gok) increaseWaitingGokitNum() {
	gok.increaseNum(&gok.waitingGokitNum)
}

func (gok *Gok) decreaseWaitingGokitNum() {
	gok.decreaseNum(&gok.waitingGokitNum)
}

func (gok *Gok) AddGokit(i Gokit) *ErrorBus {
	gok.lock.Lock()

	errBus, ok := gok.errorBuses[i.Tag()]
	if !ok {
		errBus = NewErrorBus(i.Tag())
		gok.errorBuses[i.Tag()] = errBus
	}

	gok.lock.Unlock()

	gok.gokitQueue <- i

	gok.increaseWaitingGokitNum()
	return errBus
}

func (gok *Gok) AddGokitFunc(f GokitFunc, tag, identifier string) *ErrorBus {
	gkt := gok.wrapper.wrapGokitFunc(f,tag,identifier)

	gok.lock.Lock()

	errBus, ok := gok.errorBuses[tag]
	if !ok {
		errBus = NewErrorBus(tag)
		gok.errorBuses[tag] = errBus
	}

	gok.lock.Unlock()

	gok.gokitQueue <- gkt

	gok.increaseWaitingGokitNum()
	return errBus
}

func (gok *Gok) acquire() {
	semaphore := atomic.LoadUint32(&gok.semaphore)
	atomic.CompareAndSwapUint32(&gok.semaphore,semaphore,semaphore+1)
}

func (gok *Gok) release() {
	atomic.AddUint32(&gok.semaphore,^uint32(0))
}

func (gok *Gok) execute(gkt Gokit) {
	gok.decreaseWaitingGokitNum()
	gok.increaseRunningGokitNum()
	err := gkt.Execute()
	gok.decreaseRunningGokitNum()

	errBus := gok.errorBuses[gkt.Tag()]
	if errBus != nil {
		errBus.Add(&GokError{
			Identifier: gkt.Identifier(),
			Err: err,
		})
	}
	gok.release()
}

func (gok *Gok) Run() {
	go func() {
		START:
			for {
				select {
				case <- gok.pause:
					goto PAUSE
				case <- gok.shutdown:
					return
				default:
					select {
					case gkt := <- gok.gokitQueue:
						gok.acquire()
						go gok.execute(gkt)
					case <- time.After(50 * time.Millisecond):
						break
					}
				}
			}
		PAUSE:
			<- gok.restart
			goto START
	}()
}

func (gok *Gok) Pause() {
	gok.pause <- pause{}
}

func (gok *Gok) Shutdown() {
	gok.shutdown <- shutdown{}
}

func (gok *Gok) Restart() {
	gok.restart <- restart{}
}