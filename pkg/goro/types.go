package goro

import "context"

type result[T any] struct {
	value T
	err   error
}

func (r result[T]) IsError() bool {
	return r.err != nil
}

type wrapper[T any] func(context.Context) (T, error)

type errWrapper func(context.Context) error
