package coroutine

import "fmt"

// Channel must be used instead of native go channel by coroutine code.
// Use Context.NewChannel method to create an instance.
type Channel interface {
	Recv(ctx Context) (v interface{}, more bool)    // more is false when channel is closed
	RecvAsync() (v interface{}, ok bool, more bool) // ok is true when value was returned, more is false when channel is closed

	Send(ctx Context, v interface{})
	SendAsync(v interface{}) (ok bool) // ok when value was sent
	Close()                            // prohibit sends
}

// RecvCaseFunc is executed when a value is received from the corresponding channel
type RecvCaseFunc func(v interface{}, more bool)

// SendCaseFunc is executed when value was sent to a correspondent channel
type SendCaseFunc func()

// DefaultCaseFunc is executed when none of the channel cases executed
type DefaultCaseFunc func()

// Selector must be used instead of native go select by coroutine code
// Use Context.NewSelector method to create an instance.
type Selector interface {
	AddRecv(c Channel, f RecvCaseFunc) Selector
	AddSend(c Channel, v interface{}, f SendCaseFunc) Selector
	AddDefault(f DefaultCaseFunc)
	Select(ctx Context)
}

// Func is a body of a coroutine
type Func func(ctx Context)

// PanicError contains information about panicked coroutine
type PanicError interface {
	error
	Value() interface{} // Value passed to panic call
	StackTrace() string // Stack trace of a panicked coroutine
}

// Dispatcher is a container of a set of coroutines.
type Dispatcher interface {
	// ExecuteUntilAllBlocked executes coroutines one by one in deterministic order
	// until all of them are completed or blocked on Channel or Selector
	ExecuteUntilAllBlocked() (err PanicError)
	// IsDone returns true when all of coroutines are completed
	IsDone() bool
	Close()             // Destroys all coroutines without waiting for their completion
	StackTrace() string // Stack trace of all coroutines owned by the Dispatcher instance
}

// NewDispatcher creates a new Dispatcher instance with a root coroutine function.
func NewDispatcher(root Func) Dispatcher {
	result := &dispatcherImpl{}
	rootCtx := new(emptyCtx)
	result.newCoroutine(rootCtx, root)
	return result
}

const contextKey = "coroutines"

// NewChannel create new Channel instance
func NewChannel(ctx Context) Channel {
	ctxImpl := ctx.Value(contextKey).(*coroutineState)
	ctxImpl.dispatcher.channelSequence++
	return NewNamedChannel(ctx, fmt.Sprintf("chan-%v", ctxImpl.dispatcher.channelSequence))
}

// NewNamedChannel create new Channel instance with a given human readable name.
// Name appears in stack traces that are blocked on this channel.
func NewNamedChannel(ctx Context, name string) Channel {
	return &channelImpl{name: name}
}

// NewBufferedChannel create new buffered Channel instance
func NewBufferedChannel(ctx Context, size int) Channel {
	return &channelImpl{size: size}
}

// NewNamedBufferedChannel create new BufferedChannel instance with a given human readable name.
// Name appears in stack traces that are blocked on this Channel.
func NewNamedBufferedChannel(ctx Context, name string, size int) Channel {
	return &channelImpl{name: name, size: size}
}

// NewSelector creates a new Selector instance.
func NewSelector(ctx Context) Selector {
	ctxImpl := ctx.Value(contextKey).(*coroutineState)
	ctxImpl.dispatcher.selectorSequence++
	return NewNamedSelector(ctx, fmt.Sprintf("selector-%v", ctxImpl.dispatcher.selectorSequence))
}

// NewNamedSelector creates a new Selector instance with a given human readable name.
// Name appears in stack traces that are blocked on this Selector.
func NewNamedSelector(ctx Context, name string) Selector {
	return &selectorImpl{name: name}
}

// NewCoroutine creates a new coroutine. It has similar semantic to goroutine in a context of the workflow.
func NewCoroutine(ctx Context, f Func) {
	ctxImpl := ctx.Value(contextKey).(*coroutineState)
	ctxImpl.dispatcher.newCoroutine(ctx, f)
}

// NewNamedCoroutine creates a new coroutine with a given human readable name.
// It has similar semantic to goroutine in a context of the workflow.
// Name appears in stack traces that are blocked on this Channel.
func NewNamedCoroutine(ctx Context, name string, f Func) {
	ctxImpl := ctx.Value(contextKey).(*coroutineState)
	ctxImpl.dispatcher.newNamedCoroutine(ctx, name, f)

}