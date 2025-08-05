package errors

import (
	"fmt"
)

type Error interface {
	Code() ErrorCode
	Error() string
	Unwrap() []Error
	SetChild(Error)
}

type error struct {
	code  ErrorCode
	msg   string
	child Error
}

func (e *error) SetChild(err Error) {
	e.child = err
}

func (e *error) Error() string {
	if e.child != nil {
		return fmt.Sprintf("%s\n%s\n", e.msg, e.child.Error())
	}
	return e.msg
}

func (e *error) Code() ErrorCode {
	return e.code
}

func (e *error) Unwrap() []Error {
	if e.child != nil {
		var errs []Error
		errs = append(errs, e.child.Unwrap()...)
		return errs
	}
	return []Error{e}
}

func New(code ErrorCode, msg string) Error {
	return &error{
		msg:  msg,
		code: code,
	}
}

func Join(errs ...Error) Error {
	if len(errs) == 1 {
		return errs[0]
	}
	var grandparent Error
	var prev Error
	for _, err := range errs {
		if err != nil {
			if prev != nil {
				prev.SetChild(err)
			}
			if grandparent == nil {
				grandparent = err
			}
			prev = err
		}
	}
	return grandparent
}
