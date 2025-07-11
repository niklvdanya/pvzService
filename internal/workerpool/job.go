package workerpool

import "context"

type Response struct {
	Value any
	Err   error
}

type Job struct {
	Ctx  context.Context
	Run  func(context.Context) (any, error)
	Resp chan Response
}
