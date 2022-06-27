// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package context defines the Context type, which carries deadlines,
// cancellation signals, and other request-scoped values across API boundaries
// and between processes.
// context 是一个 request-scoped 的玩意，每一个 request 对应使用一个 context
// 这个 request 完事了之后，这个 context 也应当销毁。
// 正因为会大量销毁、创建，context 的是否足够轻量级，也是关键因素
// 部分关键参数也是可以随着 context 再整个 request-scoped 范围内穿行
//
// Incoming requests to a server should create a Context, and outgoing
// calls to servers should accept a Context. The chain of function
// calls between them must propagate the Context, optionally replacing
// it with a derived Context created using WithCancel, WithDeadline,
// WithTimeout, or WithValue. **When a Context is canceled, all
// Contexts derived from it are also canceled.**
//
// The WithCancel, WithDeadline, and WithTimeout functions take a
// Context (the parent) and return a derived Context (the child) and a
// CancelFunc. Calling the CancelFunc cancels the child and its
// children, removes the parent's reference to the child, and stops
// any associated timers. Failing to call the CancelFunc leaks the
// child and its children until the parent is canceled or the timer
// fires. The go vet tool checks that CancelFuncs are used on all
// control-flow paths.
// 注意要及时调用 CancelFunc，避免资源泄露，用 defer 是非常好的习惯
//
// Programs that use Contexts should follow these rules to keep interfaces
// consistent across packages and enable static analysis tools to check context
// propagation:
//
// Do not store Contexts inside a struct type; instead, pass a Context
// explicitly to each function that needs it. The Context should be the first
// parameter, typically named ctx:
//
// 	func DoSomething(ctx context.Context, arg Arg) error {
// 		// ... use ctx ...
// 	}
//
// Do not pass a nil Context, even if a function permits it. Pass context.TODO
// if you are unsure about which Context to use.
//
// Use context Values only for request-scoped data that transits processes and
// APIs, not for passing optional parameters to functions.
//
// The same Context may be passed to functions running in different goroutines;
// Contexts are safe for simultaneous use by multiple goroutines.
//
// See https://blog.golang.org/context for example code for a server that uses
// Contexts.
package context

import (
	"errors"
	"internal/reflectlite"
	"sync"
	"sync/atomic"
	"time"
)

// A Context carries a deadline, a cancellation signal, and other values across
// API boundaries.
//
// Context's methods may be called by multiple goroutines simultaneously.
type Context interface {
	// Deadline returns the time when work done on behalf of this context
	// should be canceled. Deadline returns ok==false when no deadline is
	// set. Successive calls to Deadline return the same results.
	// 查询设置的 deadline
	Deadline() (deadline time.Time, ok bool)

	// Done returns a channel that's closed when work done on behalf of this
	// context should be canceled. Done may return nil if this context can
	// never be canceled. Successive calls to Done return the same value.
	// **The close of the Done channel may happen asynchronously,
	// after the cancel function returns.**
	// Done 跟 cancel function 的实际执行时间，并没有保证
	//
	// 三种 case：
	// WithCancel arranges for Done to be closed when cancel is called;
	// WithDeadline arranges for Done to be closed when the deadline
	// expires; WithTimeout arranges for Done to be closed when the timeout
	// elapses.
	//
	// Done is provided for use in select statements:
	//
	//  // Stream generates values with DoSomething and sends them to out
	//  // until DoSomething returns an error or ctx.Done is closed.
	//  func Stream(ctx context.Context, out chan<- Value) error {
	//  	for {
	//  		v, err := DoSomething(ctx)
	//  		if err != nil {
	//  			return err
	//  		}
	//  		select {
	//  		case <-ctx.Done():
	//  			return ctx.Err()
	//  		case out <- v:
	//  		}
	//  	}
	//  }
	//
	// See https://blog.golang.org/pipelines for more examples of how to use
	// a Done channel for cancellation.
	// 通过 channel 的形式，让下层接收者能够监听上层控制命令
	Done() <-chan struct{}

	// If Done is not yet closed, Err returns nil.
	// If Done is closed, Err returns a non-nil error explaining why:
	// Canceled if the context was canceled
	// or DeadlineExceeded if the context's deadline passed.
	// After Err returns a non-nil error, successive calls to Err return the same error.
	Err() error

	// Value returns the value associated with this context for key, or nil
	// if no value is associated with key. Successive calls to Value with
	// the same key returns the same result.
	//
	// Use context values only for request-scoped data that transits
	// processes and API boundaries, not for passing optional parameters to
	// functions.
	//
	// A key identifies a specific value in a Context. Functions that wish
	// to store values in Context typically allocate a key in a global
	// variable then use that key as the argument to context.WithValue and
	// Context.Value. A key can be any type that supports equality;
	// packages should define keys as an unexported type to avoid
	// collisions.
	//
	// Packages that define a Context key should provide type-safe accessors
	// for the values stored using that key:
	//
	// 	// Package user defines a User type that's stored in Contexts.
	// 	package user
	//
	// 	import "context"
	//
	// 	// User is the type of value stored in the Contexts.
	// 	type User struct {...}
	//
	// 	// key is an unexported type for keys defined in this package.
	// 	// This prevents collisions with keys defined in other packages.
	// 	type key int
	//
	// 	// userKey is the key for user.User values in Contexts. It is
	// 	// unexported; clients use user.NewContext and user.FromContext
	// 	// instead of using this key directly.
	// 	var userKey key
	//
	// 	// NewContext returns a new Context that carries value u.
	// 	func NewContext(ctx context.Context, u *User) context.Context {
	// 		return context.WithValue(ctx, userKey, u)
	// 	}
	//
	// 	// FromContext returns the User value stored in ctx, if any.
	// 	func FromContext(ctx context.Context) (*User, bool) {
	// 		u, ok := ctx.Value(userKey).(*User)
	// 		return u, ok
	// 	}
	Value(key interface{}) interface{}
}

// Canceled is the error returned by Context.Err when the context is canceled.
var Canceled = errors.New("context canceled")

// DeadlineExceeded is the error returned by Context.Err when the context's
// deadline passes.
var DeadlineExceeded error = deadlineExceededError{}

type deadlineExceededError struct{}

func (deadlineExceededError) Error() string   { return "context deadline exceeded" }
func (deadlineExceededError) Timeout() bool   { return true }
func (deadlineExceededError) Temporary() bool { return true }

// An emptyCtx is never canceled, has no values, and has no deadline. It is not
// struct{}, since vars of this type must have distinct addresses.
type emptyCtx int

func (*emptyCtx) Deadline() (deadline time.Time, ok bool) {
	return
}

func (*emptyCtx) Done() <-chan struct{} {
	return nil
}

func (*emptyCtx) Err() error {
	return nil
}

func (*emptyCtx) Value(key interface{}) interface{} {
	return nil
}

func (e *emptyCtx) String() string {
	switch e {
	case background:
		return "context.Background"
	case todo:
		return "context.TODO"
	}
	return "unknown empty Context"
}

// 大家用的都是同一个，比较好判断是不是走到了 root
// 而且这样能够保证不会对 nil 解引用
var (
	background = new(emptyCtx)
	todo       = new(emptyCtx)
)

// Background returns a non-nil, empty Context. It is never canceled, has no
// values, and has no deadline. It is typically used by the main function,
// initialization, and tests, and as the top-level Context for incoming
// requests.
func Background() Context {
	return background
}

// TODO returns a non-nil, empty Context. Code should use context.TODO when
// it's unclear which Context to use or it is not yet available (because the
// surrounding function has not yet been extended to accept a Context
// parameter).
func TODO() Context {
	return todo
}

// CancelFunc 的语义约定：
// A CancelFunc tells an operation to abandon its work.
// A CancelFunc does not wait for the work to stop.
// A CancelFunc may be called by multiple goroutines simultaneously.
// After the first call, subsequent calls to a CancelFunc do nothing.
// CancelFunc 并不在 Context interface 的约束里面，这也就意味着：
// 并不是每一种 Context 实现都需要具有 CancelFunc 的。
// 这个 CancelFunc 也是由 WithTimeout(), WithDeadline(), WithCancel() 所返回的
// WithValue() 就不返回 CancelFunc
// 所以 CancelFunc 的底层实现 xxx.cancel() 也不需要像 Context.Value() 那样层层递归转发
// 所以每个支持 CancelFunc 的 Context 实现，他们的 canceler interface 都是自己是实现的
type CancelFunc func()

// WithCancel returns a copy of parent with a new Done channel. The returned
// context's Done channel is closed when the returned cancel function is called
// or when the parent context's Done channel is closed, whichever happens first.
// CancelFunc 一旦被调用，或者是 parent context's Done channel is closed，本层 Done channel 也会 close
// 两个情况，要么是 happen-before，不然就是 happen-after
//
// Canceling this context releases resources associated with it, so code should
// call cancel as soon as the operations running in this Context complete.
// 当返回的 Context complete 时，必须 call cancel
func WithCancel(parent Context) (ctx Context, cancel CancelFunc) {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	c := newCancelCtx(parent)   // 生成新一层的 Context
	propagateCancel(parent, &c) // 将这个新生成的子 Context 挂进 parent Context.childern 里面
	// _, 匿名函数把 cancelCtx。cancel 裹起来作为 CancelFunc，让外界有能力调用，但是失去了随意调用能力
	// 只能在产生 WithCancel() 的这一个层次中调用 CancelFunc
	return &c, func() { c.cancel(true, Canceled) }
}

// newCancelCtx returns an initialized cancelCtx.
func newCancelCtx(parent Context) cancelCtx {
	return cancelCtx{Context: parent}
}

// goroutines counts the number of goroutines ever created; for testing.
var goroutines int32

// propagateCancel arranges for child to be canceled when parent is.
// 将子 Context 挂进父 Context 里面，让父 Context 在 cancel 的时候，能够通知子 Context
func propagateCancel(parent Context, child canceler) {
	done := parent.Done()
	if done == nil {
		// 上一层就是 emptyCtx, 没办法向上挂载这个 child Context
		return // parent is never canceled
	}

	select {
	case <-done:
		// 检查 parent.Done() 这个 channel 是不是已经被 close 掉了
		// 是的话，直接把这个 child Context 也 cancel 掉，然后退出就好
		// parent is already canceled
		child.cancel(false, parent.Err())
		return
	default:
		// 没有 close 就直接继续
	}

	if p, ok := parentCancelCtx(parent); ok {
		p.mu.Lock()
		if p.err != nil {
			// parent has already been canceled
			child.cancel(false, p.err)
		} else {
			if p.children == nil {
				p.children = make(map[canceler]struct{})
			}
			// cancelCtx.children[] 的使用通常是两种场景
			// 上游节点 cancel，直接把全部下游 children 都删除掉，所以是遍历
			// 下游节点主动注销，所以下游节点主动把自己删除点，delete(p.children, &self)
			p.children[child] = struct{}{}
		}
		p.mu.Unlock()
	} else {
		atomic.AddInt32(&goroutines, +1)
		go func() {
			// NOTE: 为什么要启动一个独立的 goroutine 进行监听呢？
			// 如果没有找到可取消的父 context。新启动一个协程监控父节点或子节点取消信号
			// 个人感觉是应对开发者可能对 Context.Context 做出的拓展的 case
			// hock 住上游的 cancel 信息，通知下游的 Context
			select {
			case <-parent.Done():
				child.cancel(false, parent.Err())
			case <-child.Done():
			}
		}()
	}
}

// &cancelCtxKey is the key that a cancelCtx returns itself for.
var cancelCtxKey int

// parentCancelCtx returns the underlying *cancelCtx for parent.
// It does this by looking up parent.Value(&cancelCtxKey) to find
// the innermost enclosing *cancelCtx and then checking whether
// parent.Done() matches that *cancelCtx. (If not, the *cancelCtx
// has been wrapped in a custom implementation providing a
// different done channel, in which case we should not bypass it.)
func parentCancelCtx(parent Context) (*cancelCtx, bool) {
	done := parent.Done()
	if done == closedchan || done == nil {
		return nil, false
	}

	/* parent.Value() 有四种走向：
	 * 1. emptyCtx, emptyCtx.Value() 返回 nil，然后对 nil 断言，ok == false
	 * 2. valueCtx, valueCtx.Value() 只能继续转发，直到 emptyCtx 或者是 cancelCtx
	 * 3. timerCtx, 直接转发给上层 cancelCtx.Value()
	 * 4. cancelCtx, cancelCtx.Value() 返回的就是 child 最近的 cancelCtx，然后断言成功
	 * 5.1 cancelCtxKey 不需要任何的 insert 操作，因为压根不存在这么个 key，cancelCtxKey 仅仅是一个标识
	 *     所有 cancelCtx.Value(key) 总是跟内部全局常量 cancelCtxKey 的地址进行比较。
	 *     为的是能够同样利用 Context.Value() 这个 interface method，来寻找最近的 cancelCtx struct
	 */
	p, ok := parent.Value(&cancelCtxKey).(*cancelCtx)
	if !ok {
		// case 1: emptyCtx 找到底了，失败
		// case 2: valueCtx 继续向 valueCtx 的 parent 转发 Value()
		return nil, false
	}
	pdone, _ := p.done.Load().(chan struct{})
	if pdone != done {
		return nil, false
	}
	return p, true
}

// removeChild removes a context from its parent.
func removeChild(parent Context, child canceler) {
	p, ok := parentCancelCtx(parent)
	if !ok {
		return
	}
	p.mu.Lock()
	if p.children != nil {
		delete(p.children, child)
	}
	p.mu.Unlock()
}

// A canceler is a context type that can be canceled directly. The
// implementations are *cancelCtx and *timerCtx.
// 目前标准库里面只有 *cancelCtx and *timerCtx 拥有 cancel 能力，
// 其他都是借用了 *cancelCtx and *timerCtx 的 cancel 能力
// propagateCancel() 使用，避免过多暴露 Context 的其他 method，
// 控制 propagateCancel() 的权限
type canceler interface {
	// removeFromParent case:
	// ture : 是上层 cancel 引发的本层 cancel
	// false: 并不是上层 cancel 引发的本层 cancel
	cancel(removeFromParent bool, err error)
	Done() <-chan struct{}
}

// closedchan is a reusable closed channel.
var closedchan = make(chan struct{})

func init() {
	close(closedchan)
}

// A cancelCtx can be canceled. When canceled, it also cancels any children
// that implement canceler.
// int 占位内存 + 并发保护字段(mu) + 上下游管理字段(Context, done, children)
// cancelCtx 是掌管整个 Context 树的核心；Context.Value() + embedded Context 则是向上搜索 Key-Value 链表的核心
type cancelCtx struct {
	// embedded 一个 Context interface，可以是任意类型的 Context interface 实现
	// 而且它是当前 cancelCtx 的上一层，parent Context
	// 对 WithCancel() 派生出来的 Context，实际上就是使用 cancelCtx 的 method
	/* 为什么有些 Context interface method 可以不实现？
	 * 其实不是「可以不实现」，而是公用同一套默认的实现。
	 * 就比如 cancelCtx, 并没有 Deadline() 这个成员 method，
	 * 但是却被 IDE 认为是 Context.Context 这个 interface。
	 * 这是因为 cancelCtx embedded 了一个 emptyCtx, 直接通过 emptyCtx.Deadline()
	 * 完成了 Context.Context 这个 interface 的要求
	 */
	Context

	mu       sync.Mutex            // protects following fields
	/* 为什么 cancelCtx.done 都用了 atomic 了，还要用 sync.Mutex ?
	 * atomic 是让 cancelCtx.Done() 能够无锁的获取 cancelCtx.done 这个 channel
	 * 但是创建这个 channel、删除这个 channel，并不是一行代码就可以完成的，是几行代码一起完成的（你看 cancelCtx.Done() 的代码）
	 * 所以需要 sync.Mutex 把这一整块的锁住
	 * 但是 cancelCtx.Done() 的第一行，只需要一行代码就可以把整个 channel 拿出来了，就不需要 sync.Mutex 来保护代码块了
	 *
	 * 最终达到这样的效果：
	 * 1. 频繁的查询都是原子操作，可以不加锁
	 * 2. 增删这些涉及到多个字段的操作，必须加锁保护，确保 do or nothing
	 */
	done     atomic.Value          // of chan struct{}, created lazily, closed by first cancel call
	children map[canceler]struct{} // set to nil by the first cancel call
	err      error                 // set to non-nil by the first cancel call
}

// cancelCtx 本身并不携带任何的 key-Value, 所以通常情况下
// cancelCtx.Value() 是应该执行 c.Context.Value(key) 向自己的 parent 转发的
// 但是有一个特殊情况：
// 下层 Context 在向上递归，寻找最近的 cancelCtx，将 Context 挂进最近的 cancelCtx 中
// 之所以不独立一个 API 是为了能够让 emptyCtx/cancelCtx/valyeCtx 都能够无缝递归转发 Value(key) 这个请求
func (c *cancelCtx) Value(key interface{}) interface{} {
	if key == &cancelCtxKey {
		// 特殊 case：用于 parentCancelCtx() 寻找最近的 parent cancelCtx
		// 先遇到的 cancelCtx 就会捕获这个请求，并将自己返回去给 parentCancelCtx()
		return c
	}
	return c.Context.Value(key)
}

// lazy-allocate
func (c *cancelCtx) Done() <-chan struct{} {
	// 无锁的 fast-path
	d := c.done.Load()
	if d != nil {
		return d.(chan struct{})
	}

	// 带锁的 slow-path
	c.mu.Lock()
	defer c.mu.Unlock()
	d = c.done.Load()
	if d == nil {
		d = make(chan struct{})
		c.done.Store(d)
	}
	return d.(chan struct{})
}

func (c *cancelCtx) Err() error {
	c.mu.Lock()
	err := c.err
	c.mu.Unlock()
	return err
}

type stringer interface {
	String() string
}

func contextName(c Context) string {
	if s, ok := c.(stringer); ok {
		return s.String()
	}
	// valueCtx, emptyCtx, cancelCtx, timerCtx 其实都实现了 stringer interface
	// 主要是自己拓展的 Context 实现可能没有实现 stringer 这个 interface 而已
	return reflectlite.TypeOf(c).String()
}

func (c *cancelCtx) String() string {
	return contextName(c.Context) + ".WithCancel"
}

// cancel closes c.done, cancels each of c's children, and, if
// removeFromParent is true, removes c from its parent's children.
// 先执行自己这一层的 cancel(), 然后执行 cancelCtx.chindren[n].cancel()
// 任务：1. cancel 掉本层
//      2. cancel 掉所有 sub-Context
//      3. 去上一层 Context 注销自己的存在
func (c *cancelCtx) cancel(removeFromParent bool, err error) {
	if err == nil {
		panic("context: internal error: missing cancel error")
	}
	c.mu.Lock()
	if c.err != nil {
		c.mu.Unlock()
		return // already canceled
	}
	c.err = err
	d, _ := c.done.Load().(chan struct{})
	if d == nil {
		// 因为 c.done 是 lazy allocate 的
		// 所以要确保后面的 sub-Context 能够看到 channel close 的话，必须设置一个 default 的进去
		c.done.Store(closedchan)
	} else {
		close(d) // channel 的 close 操作，本身就是安全的
	}
	for child := range c.children {
		// NOTE: acquiring the child's lock while holding parent's lock.
		// 将下游 children 的统统 close 掉
		child.cancel(false, err)
	}
	c.children = nil
	c.mu.Unlock() // 一定要自己这一层解锁之后，再去操作其他 Context
	// 不然很可能就会死锁，尤其是 Context 树复杂之后
	if removeFromParent {
		removeChild(c.Context, c)
	}
}

// WithDeadline returns a copy of the parent context with the deadline adjusted
// to be no later than d. If the parent's deadline is already earlier than d,
// WithDeadline(parent, d) is semantically equivalent to parent. The returned
// context's Done channel is closed when the deadline expires, when the returned
// cancel function is called, or when the parent context's Done channel is
// closed, whichever happens first.
// 要是上层的 deadline 比设置的 d 还短，那么 d 就会直接采用 上层的 deadline
// Done() channel 被触发的 case：
// 1. 上游 cancel、timeout 引发的本层 timerCtx 终止
// 2. 本层返回的 CancelFunc 被调用
// 3. 本层 timerCtx timeout 了
//
// Canceling this context releases resources associated with it, so code should
// call cancel as soon as the operations running in this Context complete.
func WithDeadline(parent Context, d time.Time) (Context, CancelFunc) {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	if cur, ok := parent.Deadline(); ok && cur.Before(d) {
		// The current deadline is already sooner than the new one.
		// 节约点资源，没有必要再这一层设置 timer 了
		return WithCancel(parent)
	}
	c := &timerCtx{
		cancelCtx: newCancelCtx(parent),
		deadline:  d,
	}
	propagateCancel(parent, c)
	dur := time.Until(d)
	if dur <= 0 {
		c.cancel(true, DeadlineExceeded) // deadline has already passed
		return c, func() { c.cancel(false, Canceled) }
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err == nil {
		c.timer = time.AfterFunc(dur, func() {
			// timeout 之后，自动 cancel
			c.cancel(true, DeadlineExceeded)
		})
	}
	return c, func() { c.cancel(true, Canceled) }
}

// A timerCtx carries a timer and a deadline. It embeds a cancelCtx to
// implement Done and Err. It implements cancel by stopping its timer then
// delegating to cancelCtx.cancel.
// timerCtx = time.timer + 新的 cancelCtx
// parent Context 将会记录在 cancelCtx.Context 里面
type timerCtx struct {
	// 独立生成一个 cancelCtx 记录 parent Context，并实现 Context interface method
	cancelCtx // timerCtx 的大部分 Context interface 直接转发给 cancelCtx 进行复用
	timer *time.Timer // Under cancelCtx.mu.

	deadline time.Time
}

func (c *timerCtx) Deadline() (deadline time.Time, ok bool) {
	return c.deadline, true
}

func (c *timerCtx) String() string {
	return contextName(c.cancelCtx.Context) + ".WithDeadline(" +
		c.deadline.String() + " [" +
		time.Until(c.deadline).String() + "])"
}

// 任务：1. 停掉 timer;
//      2. cancel 掉下游 Context; (timerCtx.cancelCtx.cancel() 代劳)
//      3. 去上层 Context 注销自己;
func (c *timerCtx) cancel(removeFromParent bool, err error) {
	c.cancelCtx.cancel(false, err) // 一层层转发给真正的 cancelCtx 调用 cancelCtx.cancel()
	if removeFromParent {
		// Remove this timerCtx from its parent cancelCtx's children.
		removeChild(c.cancelCtx.Context, c)
	}
	c.mu.Lock()
	if c.timer != nil {
		c.timer.Stop()
		c.timer = nil
	}
	c.mu.Unlock()
}

// WithTimeout returns WithDeadline(parent, time.Now().Add(timeout)).
//
// Canceling this context releases resources associated with it, so code should
// call cancel as soon as the operations running in this Context complete:
//
// 	func slowOperationWithTimeout(ctx context.Context) (Result, error) {
// 		ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
// 		defer cancel()  // releases resources if slowOperation completes before timeout elapses
// 		return slowOperation(ctx)
// 	}
func WithTimeout(parent Context, timeout time.Duration) (Context, CancelFunc) {
	return WithDeadline(parent, time.Now().Add(timeout))
}

// WithValue returns a copy of parent in which the value associated with key is
// val.
//
// 利用 Context 进行 key-value 暂存的数据，一定要是 request-scoped 的
// Use context Values only for request-scoped data that transits processes and
// APIs, not for passing optional parameters to functions.
//
// key 一定要是能够进行 compare 的类型
// 而且要自己想办法避免跨 package 之间的 key 混用
// 像 &Context.cancelCtxKey 这样利用内部全局变量的地址作为 key 就比较好，不同 package 无法混用
// The provided key must be comparable and should not be of type
// string or any other built-in type to avoid collisions between
// packages using context. Users of WithValue should define their own
// types for keys. To avoid allocating when assigning to an
// interface{}, context keys often have concrete type
// struct{}. Alternatively, exported context key variables' static
// type should be a pointer or interface.
func WithValue(parent Context, key, val interface{}) Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	if key == nil {
		panic("nil key")
	}
	if !reflectlite.TypeOf(key).Comparable() {
		panic("key is not comparable")
	}
	return &valueCtx{parent, key, val}
}

// A valueCtx carries a key-value pair. It implements Value for that key and
// delegates all other calls to the embedded Context.
// 因为每一个 valueCtx 的基础是：Context, 而每一个 Context 实际上是一个指针(*cancelCtx, *timerCtx, *emptyCtx, *valueCtx)
// 这也就意味着，每一层的 Context 实现，里面他们内部的 embedded Context，做出了一个单向链表，只能向着上游查找
type valueCtx struct {
	Context
	key, val interface{}
}

// stringify tries a bit to stringify v, without using fmt, since we don't
// want context depending on the unicode tables. This is only used by
// *valueCtx.String().
func stringify(v interface{}) string {
	switch s := v.(type) {
	case stringer:
		return s.String()
	case string:
		return s
	}
	return "<not Stringer>"
}

func (c *valueCtx) String() string {
	return contextName(c.Context) + ".WithValue(type " +
		reflectlite.TypeOf(c.key).String() +
		", val " + stringify(c.val) + ")"
}

func (c *valueCtx) Value(key interface{}) interface{} {
	if c.key == key {
		// 这一层找到了，开始逐层返回
		return c.val
	}

	// 这一层 Context 找不到的话，那就递归，向 parent 上一层找
	// 不会向下找的
	return c.Context.Value(key)
}
