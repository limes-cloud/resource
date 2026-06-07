package core

import (
	"context"

	"github.com/limes-cloud/kratosx"
	"github.com/limes-cloud/manager/api/authorize"
	"github.com/limes-cloud/manager/api/errors"
)

type Context struct {
	kratosx.Context
}

func (c Context) Transaction(fn func(ctx Context) error, name ...string) error {
	return c.Context.Transaction(func(ctx kratosx.Context) error {
		return fn(Context{Context: ctx})
	}, name...)
}

func (Context) Config() *Conf {
	return conf
}

func (c Context) Auth() *authorize.CheckAuthReply {
	info := authorize.Get(c.Context)
	if info == nil {
		c.Exit(errors.NotLoginError())
	}
	return info
}

func (c Context) Clone() Context {
	return MustContext(context.WithoutCancel(c.Context))
}

func MustContext(ctx context.Context, opts ...kratosx.ContextOptionFunc) Context {
	return Context{
		Context: kratosx.MustContext(ctx, opts...),
	}
}
