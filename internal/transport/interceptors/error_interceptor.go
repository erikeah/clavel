package interceptors

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/erikeah/clavel/internal/exceptions"
)

func ErrorInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			response, err := next(ctx, req)
			if err != nil {
				if errors.Is(err, exceptions.DoesNotExist) {
					return nil, connect.NewError(connect.CodeNotFound, err)
				}
				if errors.Is(err, exceptions.InvalidArguments) {
					return nil, connect.NewError(connect.CodeInvalidArgument, err)
				}
				if errors.Is(err, exceptions.AlreadyExist) {
					return nil, connect.NewError(connect.CodeAlreadyExists, err)
				}
				if errors.Is(err, exceptions.ExternalFailure) {
					return nil, connect.NewError(connect.CodeAborted, err)
				}
				if errors.Is(err, exceptions.InternalFailure) {
					return nil, connect.NewError(connect.CodeInternal, err)
				}
				if connectErr, ok := err.(*connect.Error); ok {
					return nil, connectErr
				}
				return nil, connect.NewError(connect.CodeUnknown, err)
			}
			return response, nil
		}
	}
}
