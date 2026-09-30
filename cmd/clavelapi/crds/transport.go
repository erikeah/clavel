package transport

import (
	"context"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
	"github.com/erikeah/clavel/internal/core"
	"github.com/erikeah/clavel/internal/interceptors"
	corev1 "github.com/erikeah/clavel/pkg/api/core/v1"
	"github.com/erikeah/clavel/pkg/api/core/v1/corev1connect"
)

type crdServiceHandler struct {
	service *core.CustomResourceDefinitionService
}

func (handler *crdServiceHandler) Create(
	ctx context.Context,
	request *connect.Request[corev1.CustomResourceDefinitionServiceCreateRequest],
) (*connect.Response[corev1.CustomResourceDefinitionServiceCreateResponse], error) {
	crd := request.Msg.GetData().Convert()
	if err := handler.service.Create(ctx, crd); err != nil {
		return nil, err
	}
	return connect.NewResponse(&corev1.CustomResourceDefinitionServiceCreateResponse{}), nil
}

func (handler *crdServiceHandler) Delete(
	ctx context.Context,
	request *connect.Request[corev1.CustomResourceDefinitionServiceDeleteRequest],
) (*connect.Response[corev1.CustomResourceDefinitionServiceDeleteResponse], error) {
	if err := handler.service.Delete(ctx, request.Msg.GetGroup(), request.Msg.GetVersion(), request.Msg.GetKind()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&corev1.CustomResourceDefinitionServiceDeleteResponse{}), nil
}

func (handler *crdServiceHandler) List(
	ctx context.Context,
	request *connect.Request[corev1.CustomResourceDefinitionServiceListRequest],
) (*connect.Response[corev1.CustomResourceDefinitionServiceListResponse], error) {
	crds, err := handler.service.List(ctx)
	if err != nil {
		return nil, err
	}
	response := &corev1.CustomResourceDefinitionServiceListResponse{
		Data: []*corev1.CustomResourceDefinition{},
	}
	for _, crd := range crds {
		crdResponse := &corev1.CustomResourceDefinition{}
		crdResponse.Set(crd)
		response.Data = append(response.Data, crdResponse)
	}
	return connect.NewResponse(response), nil
}

func (handler *crdServiceHandler) Show(
	ctx context.Context,
	request *connect.Request[corev1.CustomResourceDefinitionServiceShowRequest],
) (*connect.Response[corev1.CustomResourceDefinitionServiceShowResponse], error) {
	crd, err := handler.service.Show(ctx, request.Msg.GetGroup(), request.Msg.GetVersion(), request.Msg.GetKind())
	if err != nil {
		return nil, err
	}
	response := &corev1.CustomResourceDefinitionServiceShowResponse{
		Data: &corev1.CustomResourceDefinition{},
	}
	response.Data.Set(crd)
	return connect.NewResponse(response), nil
}

// TODO: Set a query option to list resources first
func (handler *crdServiceHandler) Watch(ctx context.Context, request *connect.Request[corev1.CustomResourceDefinitionServiceWatchRequest], stream *connect.ServerStream[corev1.CustomResourceDefinitionServiceWatchResponse]) error {
	if request.Msg.GetList() {
		list, err := handler.service.List(ctx)
		if err != nil {
			return err
		}
		for _, crd := range list {
			response := &corev1.CustomResourceDefinitionServiceWatchResponse{
				Data: &corev1.CustomResourceDefinition{},
			}
			response.Data.Set(crd)
			if err := stream.Send(response); err != nil {
				return err
			}
		}
	}
	crdChan, errChan := handler.service.Watch(ctx)
	for {
		select {
		case crd, ok := <-crdChan:
			if ok {
				response := &corev1.CustomResourceDefinitionServiceWatchResponse{
					Data: &corev1.CustomResourceDefinition{},
				}
				response.Data.Set(crd)
				if err := stream.Send(response); err != nil {
					return err
				}
			}
		case err := <-errChan:
			slog.Error(err.Error())
			response := &corev1.CustomResourceDefinitionServiceWatchResponse{
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

func NewCustomResourceDefinitionServiceHandler(service *core.CustomResourceDefinitionService) (string, http.Handler) {
	serviceHandler := &crdServiceHandler{
		service,
	}
	interceptors := connect.WithInterceptors(interceptors.ErrorInterceptor())
	return corev1connect.NewCustomResourceDefinitionServiceHandler(serviceHandler, interceptors)
}
