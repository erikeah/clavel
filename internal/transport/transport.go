package transport

import (
	"context"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
	"github.com/erikeah/clavel/internal/core"
	"github.com/erikeah/clavel/internal/fieldmaskcommander"
	"github.com/erikeah/clavel/internal/genericstore"
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

// Watch streams the resource stream to the client. With List the server first
// replays the current state terminated by a sync marker, then forwards changes
// until ctx is cancelled. A closed store channel is disabled rather than
// drained so the remaining one can still be read.
func (handler *evaluationServiceHandler) Watch(ctx context.Context, request *connect.Request[corev1.EvaluationServiceWatchRequest], stream *connect.ServerStream[corev1.EvaluationServiceWatchResponse]) error {
	eventChan, errChan := handler.service.Watch(ctx, request.Msg.GetList())
	for eventChan != nil || errChan != nil {
		select {
		case event, ok := <-eventChan:
			if !ok {
				eventChan = nil
				continue
			}
			if err := stream.Send(watchResponse(event)); err != nil {
				return err
			}
		case err, ok := <-errChan:
			if !ok {
				errChan = nil
				continue
			}
			slog.Error(err.Error())
			if err := stream.Send(&corev1.EvaluationServiceWatchResponse{
				Error: &corev1.Error{Message: err.Error()},
			}); err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

// watchResponse maps a store event onto the wire: puts carry the object,
// deletes carry only the identity they refer to.
func watchResponse(event core.EvaluationWatchEvent) *corev1.EvaluationServiceWatchResponse {
	response := &corev1.EvaluationServiceWatchResponse{Name: event.Name}
	switch event.Type {
	case genericstore.EventTypePut:
		response.Type = corev1.EventType_EVENT_TYPE_PUT
		response.Data = &corev1.Evaluation{}
		response.Data.Set(event.Object)
	case genericstore.EventTypeDelete:
		response.Type = corev1.EventType_EVENT_TYPE_DELETE
	case genericstore.EventTypeSync:
		response.Type = corev1.EventType_EVENT_TYPE_SYNC
	}
	return response
}

func NewEvaluationServiceHandler(service *core.EvaluationService) (string, http.Handler) {
	serviceHandler := &evaluationServiceHandler{
		service,
	}
	interceptors := connect.WithInterceptors(interceptors.ErrorInterceptor())
	return corev1connect.NewEvaluationServiceHandler(serviceHandler, interceptors)
}
