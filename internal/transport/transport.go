package transport

import (
	"context"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
	"github.com/erikeah/clavel/internal/core"
	"github.com/erikeah/clavel/internal/fieldmaskcommander"
	"github.com/erikeah/clavel/internal/transport/interceptors"
	corev1 "github.com/erikeah/clavel/pkg/api/core/v1"
	"github.com/erikeah/clavel/pkg/api/core/v1/corev1connect"
)

type evaluationServiceHandler struct {
	service *core.EvaluationService
}

func (handler *evaluationServiceHandler) Create(
	ctx context.Context,
	request *connect.Request[corev1.EvaluationServiceCreateRequest],
) (*connect.Response[corev1.EvaluationServiceCreateResponse], error) {
	evaluation := request.Msg.GetData().Convert(nil)
	if err := handler.service.Create(ctx, evaluation); err != nil {
		return nil, err
	}
	return connect.NewResponse(&corev1.EvaluationServiceCreateResponse{}), nil
}

func (handler *evaluationServiceHandler) Delete(
	ctx context.Context,
	request *connect.Request[corev1.EvaluationServiceDeleteRequest],
) (*connect.Response[corev1.EvaluationServiceDeleteResponse], error) {
	if err := handler.service.Delete(ctx, request.Msg.GetName()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&corev1.EvaluationServiceDeleteResponse{}), nil
}

func (handler *evaluationServiceHandler) List(
	ctx context.Context,
	request *connect.Request[corev1.EvaluationServiceListRequest],
) (*connect.Response[corev1.EvaluationServiceListResponse], error) {
	evaluations, err := handler.service.List(ctx)
	if err != nil {
		return nil, err
	}
	response := &corev1.EvaluationServiceListResponse{
		Data: []*corev1.Evaluation{},
	}
	for _, evaluation := range evaluations {
		evaluationResponse := &corev1.Evaluation{}
		evaluationResponse.Set(evaluation)
		response.Data = append(response.Data, evaluationResponse)
	}
	return connect.NewResponse(response), nil
}

func (handler *evaluationServiceHandler) Show(
	ctx context.Context,
	request *connect.Request[corev1.EvaluationServiceShowRequest],
) (*connect.Response[corev1.EvaluationServiceShowResponse], error) {
	evaluation, err := handler.service.Show(ctx, request.Msg.GetName())
	if err != nil {
		return nil, err
	}
	response := &corev1.EvaluationServiceShowResponse{
		Data: &corev1.Evaluation{},
	}
	response.Data.Set(evaluation)
	return connect.NewResponse(response), nil
}

func (handler *evaluationServiceHandler) Update(ctx context.Context, request *connect.Request[corev1.EvaluationServiceUpdateRequest]) (*connect.Response[corev1.EvaluationServiceUpdateResponse], error) {
	fmc := fieldmaskcommander.New(request.Msg.UpdateMask)
	data := request.Msg.GetData().Convert(fmc.GoTo("data"))
	if err := handler.service.Update(ctx, request.Msg.GetName(), data); err != nil {
		return nil, err
	}
	return connect.NewResponse(&corev1.EvaluationServiceUpdateResponse{}), nil
}

// TODO: Set a query option to list resources first
func (handler *evaluationServiceHandler) Watch(ctx context.Context, request *connect.Request[corev1.EvaluationServiceWatchRequest], stream *connect.ServerStream[corev1.EvaluationServiceWatchResponse]) error {
	if request.Msg.List {
		list, err := handler.service.List(ctx)
		if err != nil {
			return err
		}
		for _, evaluation := range list {
			response := &corev1.EvaluationServiceWatchResponse{
				Data: &corev1.Evaluation{},
			}
			response.Data.Set(evaluation)
			if err := stream.Send(response); err != nil {
				return err
			}
		}
	}
	evaluationChan, errChan := handler.service.Watch(ctx)
	for {
		select {
		case evaluation, ok := <-evaluationChan:
			if ok {
				response := &corev1.EvaluationServiceWatchResponse{
					Data: &corev1.Evaluation{},
				}
				response.Data.Set(evaluation)
				if err := stream.Send(response); err != nil {
					return err
				}
			}
		case err := <-errChan:
			slog.Error(err.Error())
			response := &corev1.EvaluationServiceWatchResponse{
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

func NewEvaluationServiceHandler(service *core.EvaluationService) (string, http.Handler) {
	serviceHandler := &evaluationServiceHandler{
		service,
	}
	interceptors := connect.WithInterceptors(interceptors.ErrorInterceptor())
	return corev1connect.NewEvaluationServiceHandler(serviceHandler, interceptors)
}
