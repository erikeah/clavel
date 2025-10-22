package transport

import (
	"context"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
	"github.com/erikeah/clavel/internal/fieldmaskcommander"
	"github.com/erikeah/clavel/internal/interceptors"
	sourceapi "github.com/erikeah/clavel/internal/source/api"
	sourcev1 "github.com/erikeah/clavel/pkg/api/source/v1"
	"github.com/erikeah/clavel/pkg/api/source/v1/sourcev1connect"
)

type sourceServiceHandler struct {
	service *sourceapi.SourceService
}

func (handler *sourceServiceHandler) Create(
	ctx context.Context,
	request *connect.Request[sourcev1.SourceServiceCreateRequest],
) (*connect.Response[sourcev1.SourceServiceCreateResponse], error) {
	source := request.Msg.GetData().Convert(nil)
	if err := handler.service.Create(ctx, source); err != nil {
		return nil, err
	}
	return connect.NewResponse(&sourcev1.SourceServiceCreateResponse{}), nil
}

func (handler *sourceServiceHandler) Delete(
	ctx context.Context,
	request *connect.Request[sourcev1.SourceServiceDeleteRequest],
) (*connect.Response[sourcev1.SourceServiceDeleteResponse], error) {
	if err := handler.service.Delete(ctx, request.Msg.GetName()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&sourcev1.SourceServiceDeleteResponse{}), nil
}

func (handler *sourceServiceHandler) List(
	ctx context.Context,
	request *connect.Request[sourcev1.SourceServiceListRequest],
) (*connect.Response[sourcev1.SourceServiceListResponse], error) {
	sources, err := handler.service.List(ctx)
	if err != nil {
		return nil, err
	}
	response := &sourcev1.SourceServiceListResponse{
		Data: []*sourcev1.Source{},
	}
	for _, source := range sources {
		sourceResponse := &sourcev1.Source{}
		sourceResponse.Set(source)
		response.Data = append(response.Data, sourceResponse)
	}
	return connect.NewResponse(response), nil
}

func (handler *sourceServiceHandler) Show(
	ctx context.Context,
	request *connect.Request[sourcev1.SourceServiceShowRequest],
) (*connect.Response[sourcev1.SourceServiceShowResponse], error) {
	source, err := handler.service.Show(ctx, request.Msg.GetName())
	if err != nil {
		return nil, err
	}
	response := &sourcev1.SourceServiceShowResponse{
		Data: &sourcev1.Source{},
	}
	response.Data.Set(source)
	return connect.NewResponse(response), nil
}

func (handler *sourceServiceHandler) Update(ctx context.Context, request *connect.Request[sourcev1.SourceServiceUpdateRequest]) (*connect.Response[sourcev1.SourceServiceUpdateResponse], error) {
	fmc := fieldmaskcommander.New(request.Msg.UpdateMask)
	data := request.Msg.GetData().Convert(fmc.GoTo("data"))
	if err := handler.service.Update(ctx, request.Msg.GetName(), data); err != nil {
		return nil, err
	}
	return connect.NewResponse(&sourcev1.SourceServiceUpdateResponse{}), nil
}

// TODO: Set a query option to list resources first
func (handler *sourceServiceHandler) Watch(ctx context.Context, request *connect.Request[sourcev1.SourceServiceWatchRequest], stream *connect.ServerStream[sourcev1.SourceServiceWatchResponse]) error {
	if request.Msg.List {
		list, err := handler.service.List(ctx)
		if err != nil {
			return err
		}
		for _, source := range list {
			response := &sourcev1.SourceServiceWatchResponse{
				Data: &sourcev1.Source{},
			}
			response.Data.Set(source)
			if err := stream.Send(response); err != nil {
				return err
			}
		}
	}
	sourceChan, errChan := handler.service.Watch(ctx)
	for {
		select {
		case source, ok := <-sourceChan:
			if ok {
				response := &sourcev1.SourceServiceWatchResponse{
					Data: &sourcev1.Source{},
				}
				response.Data.Set(source)
				if err := stream.Send(response); err != nil {
					return err
				}
			}
		case err := <-errChan:
			slog.Error(err.Error())
			response := &sourcev1.SourceServiceWatchResponse{
				Error: &sourcev1.Error{
					Message: err.Error(),
				},
			}
			if err := stream.Send(response); err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func NewSourceServiceHandler(service *sourceapi.SourceService) (string, http.Handler) {
	serviceHandler := &sourceServiceHandler{
		service,
	}
	interceptors := connect.WithInterceptors(interceptors.ErrorInterceptor())
	return sourcev1connect.NewSourceServiceHandler(serviceHandler, interceptors)
}
