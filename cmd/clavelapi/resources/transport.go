package transport

import (
	"context"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
	"github.com/erikeah/clavel/internal/core"
	"github.com/erikeah/clavel/internal/fieldmaskcommander"
	"github.com/erikeah/clavel/internal/interceptors"
	corev1 "github.com/erikeah/clavel/pkg/api/core/v1"
	"github.com/erikeah/clavel/pkg/api/core/v1/corev1connect"
)

type resourceServiceHandler struct {
	service *core.CustomResourceService
}

func (handler *resourceServiceHandler) Create(
	ctx context.Context,
	request *connect.Request[corev1.CustomResourceServiceCreateRequest],
) (*connect.Response[corev1.CustomResourceServiceCreateResponse], error) {
	resource := request.Msg.GetData().Convert(nil)
	if err := handler.service.Create(ctx, resource); err != nil {
		return nil, err
	}
	return connect.NewResponse(&corev1.CustomResourceServiceCreateResponse{}), nil
}

func (handler *resourceServiceHandler) Delete(
	ctx context.Context,
	request *connect.Request[corev1.CustomResourceServiceDeleteRequest],
) (*connect.Response[corev1.CustomResourceServiceDeleteResponse], error) {
	if err := handler.service.Delete(ctx, request.Msg.GetGroup(), request.Msg.GetVersion(), request.Msg.GetPlural(), request.Msg.GetName()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&corev1.CustomResourceServiceDeleteResponse{}), nil
}

func (handler *resourceServiceHandler) List(
	ctx context.Context,
	request *connect.Request[corev1.CustomResourceServiceListRequest],
) (*connect.Response[corev1.CustomResourceServiceListResponse], error) {
	resources, err := handler.service.List(ctx, request.Msg.GetGroup(), request.Msg.GetVersion(), request.Msg.GetPlural())
	if err != nil {
		return nil, err
	}
	response := &corev1.CustomResourceServiceListResponse{
		Data: []*corev1.Resource{},
	}
	for _, resource := range resources {
		resourceResponse := &corev1.Resource{}
		resourceResponse.Set(resource)
		response.Data = append(response.Data, resourceResponse)
	}
	return connect.NewResponse(response), nil
}

func (handler *resourceServiceHandler) Show(
	ctx context.Context,
	request *connect.Request[corev1.CustomResourceServiceShowRequest],
) (*connect.Response[corev1.CustomResourceServiceShowResponse], error) {
	resource, err := handler.service.Show(ctx, request.Msg.GetGroup(), request.Msg.GetVersion(), request.Msg.GetPlural(), request.Msg.GetName())
	if err != nil {
		return nil, err
	}
	response := &corev1.CustomResourceServiceShowResponse{
		Data: &corev1.Resource{},
	}
	response.Data.Set(resource)
	return connect.NewResponse(response), nil
}

func (handler *resourceServiceHandler) Update(
	ctx context.Context,
	request *connect.Request[corev1.CustomResourceServiceUpdateRequest],
) (*connect.Response[corev1.CustomResourceServiceUpdateResponse], error) {
	fmc := fieldmaskcommander.New(request.Msg.GetUpdateMask())
	data := request.Msg.GetData().Convert(fmc.GoTo("data"))
	if err := handler.service.Update(ctx, request.Msg.GetGroup(), request.Msg.GetVersion(), request.Msg.GetPlural(), request.Msg.GetName(), data); err != nil {
		return nil, err
	}
	return connect.NewResponse(&corev1.CustomResourceServiceUpdateResponse{}), nil
}

// TODO: Set a query option to list resources first
func (handler *resourceServiceHandler) Watch(
	ctx context.Context,
	request *connect.Request[corev1.CustomResourceServiceWatchRequest],
	stream *connect.ServerStream[corev1.CustomResourceServiceWatchResponse],
) error {
	if request.Msg.GetList() {
		list, err := handler.service.List(ctx, request.Msg.GetGroup(), request.Msg.GetVersion(), request.Msg.GetPlural())
		if err != nil {
			return err
		}
		for _, resource := range list {
			response := &corev1.CustomResourceServiceWatchResponse{
				Data: &corev1.Resource{},
			}
			response.Data.Set(resource)
			if err := stream.Send(response); err != nil {
				return err
			}
		}
	}
	resourceChan, errChan := handler.service.Watch(ctx, request.Msg.GetGroup(), request.Msg.GetVersion(), request.Msg.GetPlural())
	for {
		select {
		case resource, ok := <-resourceChan:
			if ok {
				response := &corev1.CustomResourceServiceWatchResponse{
					Data: &corev1.Resource{},
				}
				response.Data.Set(resource)
				if err := stream.Send(response); err != nil {
					return err
				}
			}
		case err := <-errChan:
			slog.Error(err.Error())
			response := &corev1.CustomResourceServiceWatchResponse{
				Error: &corev1.Error{
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

func NewCustomResourceServiceHandler(service *core.CustomResourceService) (string, http.Handler) {
	serviceHandler := &resourceServiceHandler{
		service,
	}
	interceptors := connect.WithInterceptors(interceptors.ErrorInterceptor())
	return corev1connect.NewCustomResourceServiceHandler(serviceHandler, interceptors)
}
