package core

import (
	"context"

	"github.com/limes-cloud/kratosx"
	"github.com/limes-cloud/kratosx/library/lock"
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

// Lock 取一把基于 Redis 的分布式锁（kratosx 提供，进程退出时自动释放）。
func (c Context) Lock(key string, opts ...lock.LockOption) lock.Lock {
	return lock.New(c.Ctx(), key, opts...)
}

func MustContext(ctx context.Context, opts ...kratosx.ContextOptionFunc) Context {
	return Context{
		Context: kratosx.MustContext(ctx, opts...),
	}
}
