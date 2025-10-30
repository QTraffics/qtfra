package registry

import (
	"context"
	"sync"

	"github.com/qtraffics/qtfra/values"
)

func NewServiceContext(ctx context.Context) (context.Context, Registry) {
	var r Registry
	if ctx == nil {
		ctx = context.Background()
	} else {
		r = FromContext(ctx)
	}

	if r == nil {
		st := NewRegistry()
		ctx = context.WithValue(ctx, values.Zero[*Registry](), st)
		return ctx, st
	}
	return ctx, r
}

func FromContext(ctx context.Context) Registry {
	v := ctx.Value(values.Zero[*Registry]())
	if v == nil {
		return nil
	}
	return v.(Registry)
}

func ContextFrom[T any](ctx context.Context) (T, bool) {
	r := FromContext(ctx)
	if r == nil {
		return values.Zero[T](), false
	}
	v := r.Get(values.Zero[*T]())
	if v == nil {
		return values.Zero[T](), false
	}
	return v.(T), true
}

func ContextFromPtr[T any](ctx context.Context) *T {
	r := FromContext(ctx)
	if r == nil {
		return nil
	}
	v := r.Get(values.Zero[*T]())
	if v == nil {
		return nil
	}
	return v.(*T)
}

func ContextWith[T any](ctx context.Context, v T) context.Context {
	r := FromContext(ctx)
	if r == nil {
		ctx, r = NewServiceContext(ctx)
	}
	r.Register(values.Zero[*T](), v)
	return ctx
}

func MustContextWith[T any](ctx context.Context, v T) {
	r := FromContext(ctx)
	if r == nil {
		panic("services : missing service registry in context")
	}
	r.Register(values.Zero[*T](), v)
}

type Registry interface {
	Register(serviceType any, service any) any
	Get(serviceType any) any
}

func NewRegistry() Registry {
	return &defaultRegistry{
		serviceTypes: make(map[any]any),
	}
}

type defaultRegistry struct {
	serviceTypes map[any]any
	access       sync.RWMutex
}

func (r *defaultRegistry) Register(serviceType any, service any) any {
	r.access.Lock()
	defer r.access.Unlock()
	oldService := r.serviceTypes[serviceType]
	r.serviceTypes[serviceType] = service
	return oldService
}

func (r *defaultRegistry) Get(serviceType any) any {
	r.access.RLock()
	defer r.access.RUnlock()
	return r.serviceTypes[serviceType]
}
