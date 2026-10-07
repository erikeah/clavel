package interceptors

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/erikeah/clavel/internal/exceptions"
)

func TestErrorInterceptorMapsSentinels(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want connect.Code
	}{
		{name: "conflict", err: exceptions.Conflict, want: connect.CodeFailedPrecondition},
		{name: "conflict joined", err: errors.Join(exceptions.Conflict, errors.New("stale")), want: connect.CodeFailedPrecondition},
		{name: "does not exist", err: exceptions.DoesNotExist, want: connect.CodeNotFound},
		{name: "invalid arguments", err: exceptions.InvalidArguments, want: connect.CodeInvalidArgument},
		{name: "already exists", err: exceptions.AlreadyExist, want: connect.CodeAlreadyExists},
	}
	interceptor := ErrorInterceptor()
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			handler := interceptor(func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
				return nil, testCase.err
			})
			_, err := handler(context.Background(), nil)
			if err == nil {
				t.Fatal("error = nil, want a mapped Connect error")
			}
			connectErr, ok := err.(*connect.Error)
			if !ok {
				t.Fatalf("error = %T, want *connect.Error", err)
			}
			if connectErr.Code() != testCase.want {
				t.Fatalf("code = %v, want %v", connectErr.Code(), testCase.want)
			}
		})
	}
}

func TestErrorInterceptorPassesSuccessThrough(t *testing.T) {
	interceptor := ErrorInterceptor()
	handler := interceptor(func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, nil
	})
	response, err := handler(context.Background(), nil)
	if err != nil {
		t.Fatalf("error = %v, want nil", err)
	}
	if response != nil {
		t.Fatalf("response = %v, want nil", response)
	}
}
