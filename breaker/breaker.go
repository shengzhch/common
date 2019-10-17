//抄的sony的gobreaker
package breaker

import (
	"errors"

	"fmt"
	"sync"
	"time"
)

type State int

const (
	StateClosed State = iota
	StateHalfOpen
	StateOpen
)

var (
	ErrTooManyRequest = errors.New("too many request")
	ErrOpenState      = errors.New("circuit breaker is open")
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateHalfOpen:
		return "half-open"
	case StateOpen:
		return "open"
	default:
		return fmt.Sprintf("unknown state %d", s)
	}
}

type Counts struct {
	Requests             uint32
	TotalSuccesses       uint32
	TotalFailures        uint32
	ConsecutiveSuccesses uint32
	ConsecutiveFailures  uint32
}

//请求
func (c *Counts) onRequest() {
	c.Requests++
}

//成功
func (c *Counts) onSuccess() {
	c.TotalSuccesses++
	c.ConsecutiveSuccesses++
	c.ConsecutiveFailures = 0
}

//失败
func (c *Counts) onFailure() {
	c.TotalFailures++
	c.ConsecutiveFailures++
	c.ConsecutiveSuccesses = 0
}

//清除归零
func (c *Counts) clear() {
	c.Requests = 0
	c.TotalSuccesses = 0
	c.TotalFailures = 0
	c.ConsecutiveSuccesses = 0
	c.ConsecutiveFailures = 0
}

//设置
type Settings struct {
	Name          string
	MaxRequests   uint32
	Interval      time.Duration
	Timeout       time.Duration
	ReadyToTrip   func(counts Counts) bool
	OnStateChange func(name string, from, to State)
}

//熔断器 阻止极有可能失败的请求
// 名称
// 最大请求 是在断路器 半开时 允许通过的最大请求数量。如果MaxRequests是0，断路器只允许一个请求。
// 间隔 间隔是闭合状态下断路器清除内部计数的循环周期。如果间隔为0，断路器在闭合状态下内部计数不清。
// 超时 超时是打开状态的一段时间，打开状态后断路器状态为半开状态。如果超时为0，断路器的超时值设置为60秒。
// 熔断判断 关闭状态下调用失败是是否要打开熔断设置
// 状态处理 状态转移时要做的处理
// 锁 状态 第几代 计数器
// 过期时间
type CircuitBreaker struct {
	name          string
	maxRequests   uint32
	interval      time.Duration
	timeout       time.Duration
	readtToTrip   func(counts Counts) bool
	onStateChange func(name string, from, to State)

	mutex      sync.Mutex
	state      State
	generation uint64
	count      Counts
	expiry     time.Time
}

//
type TwoStepCircuitBreaker struct {
	cb *CircuitBreaker
}

func NewCircuitBreaker(st Settings) *CircuitBreaker {
	cb := new(CircuitBreaker)
	cb.name = st.Name
	cb.interval = st.Interval
	cb.onStateChange = st.OnStateChange
	if st.MaxRequests == 0 {
		cb.maxRequests = 1
	} else {
		cb.maxRequests = st.MaxRequests
	}

	if st.Timeout == 0 {
		cb.timeout = defaultTimeout
	} else {
		cb.timeout = st.Timeout
	}

	if st.ReadyToTrip == nil {
		cb.readtToTrip = defaultReadyToTrip
	} else {
		cb.readtToTrip = st.ReadyToTrip
	}

	cb.toNewGeneration(time.Now())

	return cb
}

func NewTwoStepCircuitBreaker(st Settings) *TwoStepCircuitBreaker {
	return &TwoStepCircuitBreaker{
		cb: NewCircuitBreaker(st),
	}
}

const defaultTimeout = time.Duration(60) * time.Second

func defaultReadyToTrip(counts Counts) bool {
	return counts.ConsecutiveFailures > 5
}

func (cb *CircuitBreaker) Name() string {
	return cb.name
}

func (cb *CircuitBreaker) State() State {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()

	state, _ := cb.currentState(now)

	return state
}

func (cb *CircuitBreaker) Execute(req func() (interface{}, error)) (interface{}, error) {
	//执行req前判断
	generation, err := cb.beforeRequest()
	if err != nil {
		return nil, err
	}

	//若原函数req有panic，接受完做计数后依然把panic返回，中间件不变更逻辑
	defer func() {
		e := recover()
		if e != nil {
			cb.afterRequest(generation, false)
			panic(e)
		}
	}()

	result, err := req()
	cb.afterRequest(generation, err == nil)
	return result, err
}

// Name returns the name of the TwoStepCircuitBreaker.
func (tscb *TwoStepCircuitBreaker) Name() string {
	return tscb.cb.Name()
}

func (tscb *TwoStepCircuitBreaker) State() State {
	return tscb.cb.State()
}

func (tscb *TwoStepCircuitBreaker) Allow() (done func(success bool), err error) {
	generation, err := tscb.cb.beforeRequest()
	if err != nil {
		return nil, err
	}
	return func(success bool) {
		tscb.cb.afterRequest(generation, success)
	}, nil
}

func (cb *CircuitBreaker) beforeRequest() (uint64, error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)

	if state == StateOpen {
		return generation, ErrOpenState
	} else if state == StateHalfOpen && cb.count.Requests >= cb.maxRequests {
		return generation, ErrTooManyRequest
	}
	cb.count.onRequest()
	return generation, nil
}

func (cb *CircuitBreaker) afterRequest(before uint64, success bool) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	now := time.Now()
	state, generation := cb.currentState(now)
	if generation != before {
		return
	}
	if success {
		cb.onSuccess(state, now)
	} else {
		cb.onFailture(state, now)
	}
}

func (cb *CircuitBreaker) onSuccess(state State, now time.Time) {
	switch state {
	case StateClosed:
		cb.count.onSuccess()
	case StateHalfOpen:
		cb.count.onSuccess()
		if cb.count.ConsecutiveSuccesses >= cb.maxRequests {
			cb.setState(StateClosed, now)
		}
	}
}

func (cb *CircuitBreaker) onFailture(state State, now time.Time) {
	switch state {
	case StateClosed:
		cb.count.onFailure()
		if cb.readtToTrip(cb.count) {
			cb.setState(StateOpen, now)
		}
	case StateHalfOpen:
		cb.setState(StateOpen, now)
	}
}

func (cb *CircuitBreaker) currentState(now time.Time) (State, uint64) {
	switch cb.state {
	case StateClosed:
		//过期时间已到且过期时间不为零
		if !cb.expiry.IsZero() && cb.expiry.Before(now) {
			cb.toNewGeneration(now)
		}
	case StateOpen:
		//过期时间已到转移到半开
		if cb.expiry.Before(now) {
			cb.setState(StateHalfOpen, now)
		}
	}
	return cb.state, cb.generation
}

func (cb *CircuitBreaker) setState(state State, now time.Time) {
	if cb.state == state {
		return
	}
	pre := cb.state
	cb.state = state
	cb.toNewGeneration(now)

	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, pre, state)
	}
}

func (cb *CircuitBreaker) toNewGeneration(now time.Time) {
	cb.generation++
	cb.count.clear()
	var zero time.Time
	switch cb.state {
	case StateClosed:
		if cb.interval == 0 {
			cb.expiry = zero
		} else {
			cb.expiry = now.Add(cb.interval)
		}
	case StateOpen:
		cb.expiry = now.Add(cb.timeout)
	default: //半开状态
		cb.expiry = zero
	}
}
