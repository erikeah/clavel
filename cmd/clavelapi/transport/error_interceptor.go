package transport

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/erikeah/clavel/internal/exception"
)

func ErrorInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			response, err := next(ctx, req)
			if err != nil {
				if errors.Is(err, exception.DoesNotExist) {
					return nil, connect.NewError(connect.CodeNotFound, err)
				}
				if errors.Is(err, exception.InvalidArguments) {
					return nil, connect.NewError(connect.CodeInvalidArgument, err)
				}
				if errors.Is(err, exception.AlreadyExist) {
					return nil, connect.NewError(connect.CodeAlreadyExists, err)
				}
				if errors.Is(err, exception.ExternalFailure) {
					return nil, connect.NewError(connect.CodeAborted, err)
				}
				if errors.Is(err, exception.InternalFailure) {
					return nil, connect.NewError(connect.CodeInternal, err)
				}
				return nil, connect.NewError(connect.CodeUnknown, err)
			}
			return response, nil
		}
	}
}
