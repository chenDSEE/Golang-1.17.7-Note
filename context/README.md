> context 用起来确实可能会很蒙，尤其是要跟 select、channel 一起用
>
> 实际上你抓住一点「context 也是为了能够更好管理 goroutine 的生命周期」，就好了
>
> 所以要不要监听，go 不 go 出去，select、WaitGroup 都是围绕：goroutine 生命周期管理这一点进行决策



> 官方建议：
>
> - Do not store Contexts inside a struct type; instead, pass a Context explicitly to each function that needs it. （不要把 Context 放在一个 request，而是永远在 stack 上，显示地在函数之间进行传递）
> - Do not pass a nil Context, even if a function permits it. 没想好用什么 Context 的时候，那就用 `context.TODO()`, 而不是用 nil
> - Contexts are safe for simultaneous use by multiple goroutines.==Context 是并发安全的==
> - 别忘了调用 `CancelFunc`，否则资源会泄露的

- `context` 是一个 request-scoped 的玩意，每一个 request 对应使用一个 `context`。这个 request 完事了之后，这个 `context` 也应当销毁
- 正因为会大量销毁、创建，`context` 的是否足够轻量级，也是关键因素
- 部分关键参数也是可以随着 `context` 再整个 request-scoped 范围内穿行，而且这些参数也尽可能是 request-scoped 的。
  - 通过 Context 进行参数传递，是可以在不改变其他 package 依赖的前提下，将参数跨越不同 package 进行传递的

- 一个 `Context` 一旦被 canceled 之后，由它派生出来的所有子 `Context` 都被被自动取消。
  - When a Context is canceled, all Contexts derived from it are also canceled.
- 永远不要忘记使用 `CancelFunc` 释放资源。Calling the `CancelFunc` cancels the child and its children, removes the parent's reference to the child, and stops any associated timers.
  - 调用 `CancelFunc` 可以释放这一层 Context 相关的资源
  - 要是忘了调用 `CancelFunc()`, 有可能会发生资源泄露（直到上层的 Context 消亡时候，才释放资源）
  - 用 `defer` 是非常好的习惯





# 为什么 `Context` 能够被这么多地方使用？

好用！大家用起来感觉非常的爽快。这是最为简单直接的答案。

那么爽在哪里呢？

- 这是一个很好的抽象。因为 `Context` 的生命周期贯穿了 server 处理 request，回复 response 的整个过程。这也就意味着，在这个过程中必须的关键信息全部放到这个 `Context` 里面就好了，需要的时候再从 `Context` 中去拿
- 为优雅终止、提前取消提供的 cancel 的能力，有利于 goroutine 的生命周期管理（毕竟没有 cancel 的话，有些并发出去的 goroutine 就不能被取消，会造成资源上的浪费，甚至是 goroutine 的泄露）
  - 还提供的定时 cancel 的能力
- 能够让标准库、自定义的 `Context`、第三方库的 `Context` 无缝的混合使用
  - 标准库的 `context` 包除了提供 `Context` 这个 interface 抽象（接口约定）之外，还提供了程序员操作 `context` 的 public function 作为标准的 API，尽可能让每一个使用 Context 的人都能够采用相同的 API 去操作 `Context`
  - 其实，`context` 的 public function 是作为一套 `Context` 框架存在的！你想想，所有人都只推荐使用 `context` 包提供的 public function 去操作 `Context` interface，而 public function 返回的参数、接收的参数也是 `Context` interface。这造成了什么结果？第三方库自己创建的 `Context`  object 只要满足 `Context` interface 的要求，就可以传出去，跨越不同的 package，无论是第三方 package，还是标准库的
  - 看下面的这个例子：

```go
package p1

type ctx1 struct {
    .....
}

func inputData(ctx1, data) context.Context {
    return context.WithValue(ctx1, key, data)
}

func getData(context.Context) data {
    // 即便跨越了不同的 package，依然能够顺利拿回相应的数据
    return context.Value(ctx1, key)
}
```

```go
package p2

type ctx2 struct {
    .....
}

func inputData(ctx2, data) context.Context {
    return context.WithValue(ctx2, key, data)
}

func getData(context.Context) data {
    return context.Value(ctx2, key)
}
```



```go
package main

import p1
import p2

func main() {
    ctx := p1.inputData(context.background(), data)
    ctx = p2.inputData(ctx, data)
    data = p1.getData(ctx)
}
```







# interface

## Context

```go
// A Context carries a deadline, a cancellation signal, and other values across
// API boundaries.
//
// Context's methods may be called by multiple goroutines simultaneously.
type Context interface {
	Deadline() (deadline time.Time, ok bool)
	// 通过 channel 的形式，让下层接收者能够监听上层控制命令
	// 通常采用 close 来触发 Done() 返回的 channel
	Done() <-chan struct{}
	Err() error
	Value(key interface{}) interface{}
}

```

- Contet 需要做到的事情不多，提供 goroutine 的控制能力 + 在这个 request 范围内，传递关键信息
  - 提供 goroutine 的控制能力：上游 goroutine 能够命令下游 goroutine 提前终止（`CancelFunc()`）；下游 goroutine 能够监听到上游的终止命令（`Context.Done()`）
    - 这其实也就意味着，Context 的存在，是为了沟通不同 goroutine 的。要是只有同一个 goroutine，压根没有 Context 存在的必要
    - 另一方面，下游 goroutine 为什么要监听？监听意味着什么？意味着这个 goroutine 发起了一个异步调用，可能是等待其他 goroutine 完事，也可能是等待其他 server 返回数据。总之，这个 goroutine 目前只能干两件事：等待数据 + 监听上游的请求
  - 传递关键信息：`Context.Value()`
- `valueCtx`, `timerCtx`, `cancelCtx` 没有实现的 Context interface method，实际上最后是由 embedded 的 `Context` 资源来完成的
- Context package 总是使用 pointer 来实现 Context interface



### `CancelFunc` 语义约定

```go
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
```





### `type emptyCtx int`

- Context 最基础的底层，实际上就是这个 `int` 类型
- 因为大部分 method 都是返回 nil，可以作为根节点的判断，递归循环的终止条件

```go
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

// 大家用的都是同一个，比较好判断是不是走到了 root
// 而且这样能够保证不会对 nil 解引用
var (
	background = new(emptyCtx)
	todo       = new(emptyCtx)
)

// 总是一个 non-nil 的 pointer
func TODO() Context {
	return todo
}

func Background() Context {
	return background
}
```



### `type cancelCtx struct`

- 全部类型的 Context 实现，都是通过 `cancleCtx` 来提供 cancel + sub-cancel 能力的
- 所有 Context 树，最后都是由 `cancelCtx.children[]` 来管理的
- `cancelCtx` = int 占位内存 + 并发保护字段(`mu`) + 上下游管理字段(`Context`, `done`, `children`)

**创建**

```go
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
	// 总是 non-nil 的，正常 cancel 的话就是填充 Canceled
	err      error                 // set to non-nil by the first cancel call
}

// CancelFunc 一旦被调用，或者是 parent context's Done channel is closed，本层 Done channel 也会 close
// 两个情况，要么是 happen-before，不然就是 happen-after
//
// 当返回的 Context complete 时，必须 call cancel
func WithCancel(parent Context) (ctx Context, cancel CancelFunc) {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	c := newCancelCtx(parent)   // 生成新一层的 Context
	propagateCancel(parent, &c) // 将这个新生成的子 Context 挂进 parent Context.childern 里面
	// _, 匿名函数把 cancelCtx。cancel 裹起来作为 CancelFunc，
    // 让外界有能力调用，但是失去了随意调用能力
	// 只能在产生 WithCancel() 的这一个层次中调用 CancelFunc
	return &c, func() { c.cancel(true, Canceled) }
}

// newCancelCtx returns an initialized cancelCtx.
func newCancelCtx(parent Context) cancelCtx {
	return cancelCtx{Context: parent}
}
```



**cancel 与 Context 树**

```go
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

// 这是可重入的，注意 channel 是不能多次 close 的
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
		// 这个 cancelCtx 已经调用过 cancel() 了
		c.mu.Unlock()
		return // already canceled
	}
    
	// 下面的操作，只能发生一次
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



```



**`Value()` 转发**

```go
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
```







### `type timerCtx struct`

**创建**

```go
// timerCtx = time.timer + 新的 cancelCtx
// parent Context 将会记录在 cancelCtx.Context 里面
type timerCtx struct {
	// 独立生成一个 cancelCtx 记录 parent Context，并实现 Context interface method
	cancelCtx // timerCtx 的大部分 Context interface 直接转发给 cancelCtx 进行复用
	timer *time.Timer // Under cancelCtx.mu.

	deadline time.Time
}

func WithTimeout(parent Context, timeout time.Duration) (Context, CancelFunc) {
	return WithDeadline(parent, time.Now().Add(timeout))
}

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

```







### `type valueCtx struct`

**创建**

```go
// 因为每一个 valueCtx 的基础是：Context, 而每一个 Context 实际上是一个指针(*cancelCtx, *timerCtx, *emptyCtx, *valueCtx)
// 这也就意味着，每一层的 Context 实现，里面他们内部的 embedded Context，做出了一个单向链表，只能向着上游查找
type valueCtx struct {
	Context
	key, val interface{}
}


// 利用 Context 进行 key-value 暂存的数据，一定要是 request-scoped 的
//
// key 一定要是能够进行 compare 的类型
// 而且要自己想办法避免跨 package 之间的 key 混用
// 像 &Context.cancelCtxKey 这样利用内部全局变量的地址作为 key 就比较好，不同 package 无法混用
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

// 一层层向上递归，c.Context 构成了一个向上的单向链表
func (c *valueCtx) Value(key interface{}) interface{} {
	if c.key == key {
		// 这一层找到了，开始逐层返回
		return c.val
	}

	// 这一层 Context 找不到的话，那就递归，向 parent 上一层找
	// 不会向下找的
	return c.Context.Value(key)
}
```





### valueCtx 中 key 的注意点

- key 需要使用者自己保证不冲突，不被其他 package 读出来

- 像 `&Context.cancelCtxKey` 这样利用内部全局变量的地址作为 key 就比较好，不同 package 无法混用
- 另一个方案则是，在 package 内部重新定义一个内部类型，然后利用这个不公开的内部类型作为  key
  - 因为 key 实际上是一个 `interface{}` 空接口，空接口的比较，是会比较类型是否一致的。就比如下面这个例子

```go
// userIPkey is the context key for the user IP address.  Its value of zero is
// arbitrary.  If this package defined other context keys, they would have
// different integer values.
const userIPKey key = 0

func NewContext(ctx context.Context, userIP net.IP) context.Context {
    return context.WithValue(ctx, userIPKey, userIP)
}

func FromContext(ctx context.Context) (net.IP, bool) {
    // ctx.Value returns nil if ctx has no value for the key;
    // the net.IP type assertion returns ok=false for nil.
    userIP, ok := ctx.Value(userIPKey).(net.IP)
    return userIP, ok
}
```









### 为什么有些 Context interface method 可以不实现？

> ==embedded 默认实现==，达成的效果

其实不是「可以不实现」，而是公用同一套默认的实现。就比如 `cancelCtx`, 并没有 `Deadline()` 这个成员 method，但是却被 IDE 认为是 `Context.Context` 这个 interface。这是因为 `cancelCtx` embedded 了一个 `emptyCtx`, 直接通过 `emptyCtx.Deadline()` 完成了 `Context.Context` 这个 interface 的要求



### 为什么 `cancelCtx.done` 都用了 atomic 了，还要用 sync.Mutex ?

- atomic 是让 cancelCtx.Done() 能够无锁的获取 cancelCtx.done 这个 channel
- 但是创建这个 channel、删除这个 channel，并不是一行代码就可以完成的，是几行代码一起完成的（你看 cancelCtx.Done() 的代码）。所以需要 sync.Mutex 把这一整块的锁住
- 但是 cancelCtx.Done() 的第一行，只需要一行代码就可以把整个 channel 拿出来了，就不需要 sync.Mutex 来保护代码块了



最终达到这样的效果：

1. 频繁的查询都是原子操作，可以不加锁

2. 增删这些涉及到多个字段的操作，必须加锁保护，确保 do or nothing





### 所有不同种类的 Context 实现，最终构成了一个单向链表

- `cancelCtx.Context`, `valueCtx.Context`, `timerCtx.cancelCtx.Context` 这三个字段总是指向上一个 Context 的地址。`emptyCtx` 总是链表头。
- 每个创建 Context 的函数，总是返回实现类型的地址



### 怎么做到 key-value 并发安全的？

既然通过 `Context` 这个字段形成了一个向上的链表，而且每增加一层 key-value 是通过新生成的 `Context` 指向原本的 `Context` ，再加上 key-value 是不可修改的，那也就意味着：

- Write：天然安全。因为永远都是新生成的 `valueCtx` 指向已经存在的 `Context`, 那么我们在将新生成的 `valueCtx` 暴露出去之前，就把 key-value 设置好。所以整个过程是天然并发安全的
- Read：因为 `vlaueCtx` 是不支持修改已有 key-value 的，而且每一个 key-value 删除的时候，意味着一个 `valueCtx` 的删除。所以在 read 的时候，天然不会发生删除、修改的动作。所以 read 的时候，也是不需要做出加锁保护、atomic 保护的

> 注意：
>
> 尽管 `valueCtx` 是并发安全的，但是 value 本身是不是可以并发安全的进行增删查改，这是你作为调用者的职责，Context package 是没有办法帮你解决这个问题的







### Context 实现 struct 的综述

不同的 Context 实现 struct 职责是相互独立的。之所以每个类型的 struct 都能够拥有 Context interface 里面规定的 method，是因为他们内部 embedded 了一个 Context interface。

- `emptyCtx`: 这是所有 Context 的根，基本上都是返回 nil 的，这样我们就可以在递归遍历的时候，利用这一点作为递归遍历的终止条件。
- `cancelCtx`: 全部 Context 都是通过 embedded 一个 `cancelCtx` 来进行 Context 树管理的。
  - cancel: 每当发生 cancel 的时候，最终都是将 cancel 请求，由 `cancelCtx` 完成上游 Context 的注销、向下游传播 cancel。
  - value: `cancelCtx` 针对 `Value()` 仅仅会看看是不是在找最近的 Context 树管理者（也就是 `cancelCtx`, 利用 `cancelCtxKey` 把 `Value()` 递归调用中的请求 hock 住）。其他情况都是直接向上转发 `c.Context.Value(key)`
- `timerCtx`: 在 `cancelCtx` 的基础上，裹上一个 timer（timer + 一个指向 `cancelCtx` 的指针）。
  - cancel: timeout 的时候，主动通过 embedded 的 `cancelCtx` 完成上下游的 cancel 工作。被动 cancel 的时候，也是将 cancel 转交给 embedded 的 `cancelCtx` 完成上下游的 cancel 工作。
  - value: 而 `timerCtx.Value()` 的时候，则是直接让 `timerCtx.cancelCtx.Value()` 完成
- `valueCtx`: 一个指向上层的 embedded `Context` + key + value。只有 `valueCtx` 才会存储键值对，其他种类的 Context 都是直接向上转发 `Value()` 的查找请求
  - cancel: 直接转发给 `valueCtx.Context.cancel()` 处理
  - value: 向上层层递归查找，相当于一个链表的查找。每一个 `valueCtx` 只会保持一对 key-value





## canceler

```go
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

```

- `cancel()` 必须是并发安全的，而且上下游的 cancel 动作，必须只发生一次





# demo

==TODO==: 写一个最佳实践的 demo

> 一次请求要求同时返回连续 10 个用户的信息。每个用户的信息分别包含：用户名查询、年龄查询、info 信息。超过 3 个用户查询失败的话，直接返回 fail，不需要继续进行查询了。
>
> 利用 `UidKey` 作为 `Context` 的 key 存储 UID。

- 其实由两种方案：
  - 一个用户的信息全部查出来之后，才做处理；一个用户的某一个信息查出来了，就立马做处理
  - 等到全部用户的信息查完后之后，才构建 response；一个用户查询完成后，就构建一个 response entry
  - 很显然，单个请求的响应速度上看，是查完一个就处理一个；但是这将会导致 goroutine 的频繁环形跟挂起，并发量会受到影响。等到全部信息查询完成后，才生成 response 在响应速度上确实比较差，但是并发量可以提高，因为浪费在 goroutine 切换、唤醒、休眠的资源少了

```go
package main

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"context"
	"time"
	"math/rand"
	"sync"
)

type User struct {
	id   int
	name string
	info string
}

// 即使被 cancel 之后，query 的结果还是要处理的，只不过可以直接不理会而已
func queryName(nameCH chan string, uid int) {
	time.Sleep(50 * time.Millisecond)
	nameCH <- fmt.Sprintf("name-%d", uid) // never block
}

func queryInfo(infoCH chan string, uid int) {
	time.Sleep(50 * time.Millisecond)
	infoCH <- fmt.Sprintf("info-%d", uid) // never block
}

// 也可以并发去查不同的字段，这时候 checkUserInfo 是负责监听上层 ctx.Done()，以及 merge 其他几个 channel
// 其实监听 close 来退出更好
func checkUserInfo(ctx context.Context, reslutCH chan User, uid int) bool {
	if rand.Int() % 2 == 0 {
		return false
	}
	
	// buffer 1 to avoid goroutine leak
	nameCH := make(chan string, 1)
	infoCH := make(chan string, 1)
	go queryName(nameCH, uid)
	go queryInfo(infoCH, uid)

	// 可以不启用单独的 goroutine 作为 merge + 监听
	// 因为 checkUserInfo() 已经是运行在一个独立的 goroutine 里了，
	// 所以这里就不再单独启动一个 goroutine 进行监听了
	// merge up all result
	var user User
	user.id = uid
	for i := 0; i < 2; i++ {
		select {
		case user.name = <- nameCH:
		case user.info = <- infoCH:
		case <- ctx.Done():
			// all channel will be GC, no goroutine will be hang forever
			fmt.Printf("User[%d] had been cancel\n", uid)
			return true
		}
	}

	if user.id != -1 {
		reslutCH <- user
	}
	return true
}

const queryNum = 10
const errorLimit = 3
// main 作为监控者，设定了 timeout，超时全部 checkUserInfo 都取消；出错次数监控，出错次数太多，全部 checkUserInfo 都取消；
// checkUserInfo 作为被监控者，需要监听 main 发过来的监控信号，
// 同时还要进行实际的查询动作，并发查询结果放到另一个 channel 里，
// 才能跟 ctx.Done() 进行多路复用的监听
func main() {

	fmt.Printf("====== request come for %d users information ======\n", queryNum)

	ctx, cancel := context.WithTimeout(context.Background(), 1 * time.Second)
	defer cancel()

	resultCH := make(chan User, queryNum)

	/* check queryNum users information(fan-in) */
	var wg sync.WaitGroup
	wg.Add(queryNum)
	cnt := int32(0)
	for uid := 0; uid < queryNum; uid++ {
		go func(uid int) {
			if(checkUserInfo(ctx, resultCH, uid) == false) {
				atomic.AddInt32(&cnt, 1)
				fmt.Printf("checkUserInfo() fail for user[%d]\n", uid)
				if atomic.LoadInt32(&cnt) >= errorLimit {
					fmt.Printf("###!!! too many error, cancel all !!!###\n")
					cancel()
				}
			} else {
				fmt.Printf("checkUserInfo() success for user[%d]\n", uid)
			}
			wg.Done()
		}(uid)
	}

	/* print queryNum users information as a response */
	wg.Wait()
	close(resultCH)
	if atomic.LoadInt32(&cnt) < errorLimit {
		for user := range resultCH {
			fmt.Printf("User[id:%d][name:%s][info:%s]\n", user.id, user.name, user.info)
		}
	}

	time.Sleep(5 * time.Second) // wait all sub-goroutine be canceled and exit
	fmt.Printf("====== end with goroutine[%d] ======\n", runtime.NumGoroutine())
}


func init() {
	rand.Seed(time.Now().Unix())
}
```







# Reference

> - https://go.dev/blog/context-and-structs
> - https://go.dev/blog/context
> - 









































