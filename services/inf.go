package services

import (
	"context"

	"github.com/qtraffics/qtfra/ex"
)

type LifeCycle interface {
	Start(ctx context.Context) error
	Close() error
}
type PreStarter interface {
	PreStart(ctx context.Context) error
}

type PostStarter interface {
	PostStart(ctx context.Context) error
}

type PreCloser interface {
	PreClose() error
}

type PostCloser interface {
	PostClose() error
}

type FullLifeCycle interface {
	LifeCycle

	PreStarter
	PostStarter

	PreCloser
	PostCloser
}

type Service interface {
	LifeCycle
	Type() string
}

type NoopService struct{}

func (n *NoopService) Start(ctx context.Context) error { return nil }
func (n *NoopService) Close() error                    { return nil }
func (n *NoopService) Type() string                    { return "" }

// Start executes the start sequence for a LifeCycle implementer.
// It calls PreStart (if implemented), Start, and PostStart (if implemented).
func Start(ctx context.Context, lf LifeCycle) error {
	if ctx == nil {
		ctx = context.Background()
	}

	// Execute PreStart if the interface is implemented
	if pre, ok := lf.(PreStarter); ok {
		if err := pre.PreStart(ctx); err != nil {
			return ex.Cause(err, "PreStart")
		}
	}

	// Execute the mandatory Start method
	if err := lf.Start(ctx); err != nil {
		return ex.Cause(err, "Start")
	}

	// Execute PostStart if the interface is implemented
	if post, ok := lf.(PostStarter); ok {
		if err := post.PostStart(ctx); err != nil {
			return ex.Cause(err, "PostStart")
		}
	}

	return nil
}

// Close executes the close sequence for a LifeCycle implementer.
// It calls PreClose (if implemented), Close, and PostClose (if implemented).
func Close(lf LifeCycle) error {
	// Execute PreClose if the interface is implemented
	if pre, ok := lf.(PreCloser); ok {
		if err := pre.PreClose(); err != nil {
			return ex.Cause(err, "PreClose")
		}
	}

	// Execute the mandatory Close method
	if err := lf.Close(); err != nil {
		return ex.Cause(err, "Close")
	}

	// Execute PostClose if the interface is implemented
	if post, ok := lf.(PostCloser); ok {
		if err := post.PostClose(); err != nil {
			return ex.Cause(err, "PostClose")
		}
	}

	return nil
}
