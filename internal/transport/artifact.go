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

type artifactServiceHandler struct {
	service *core.ArtifactService
}

func (handler *artifactServiceHandler) Create(
	ctx context.Context,
	request *connect.Request[corev1.ArtifactServiceCreateRequest],
) (*connect.Response[corev1.ArtifactServiceCreateResponse], error) {
	artifact := request.Msg.GetData().Convert(nil)
	if err := handler.service.Create(ctx, artifact); err != nil {
		return nil, err
	}
	return connect.NewResponse(&corev1.ArtifactServiceCreateResponse{}), nil
}

func (handler *artifactServiceHandler) Delete(
	ctx context.Context,
	request *connect.Request[corev1.ArtifactServiceDeleteRequest],
) (*connect.Response[corev1.ArtifactServiceDeleteResponse], error) {
	if err := handler.service.Delete(ctx, request.Msg.GetName()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&corev1.ArtifactServiceDeleteResponse{}), nil
}

func (handler *artifactServiceHandler) List(
	ctx context.Context,
	request *connect.Request[corev1.ArtifactServiceListRequest],
) (*connect.Response[corev1.ArtifactServiceListResponse], error) {
	artifacts, err := handler.service.List(ctx)
	if err != nil {
		return nil, err
	}
	response := &corev1.ArtifactServiceListResponse{
		Data: []*corev1.Artifact{},
	}
	for _, artifact := range artifacts {
		artifactResponse := &corev1.Artifact{}
		artifactResponse.Set(artifact)
		response.Data = append(response.Data, artifactResponse)
	}
	return connect.NewResponse(response), nil
}

func (handler *artifactServiceHandler) Show(
	ctx context.Context,
	request *connect.Request[corev1.ArtifactServiceShowRequest],
) (*connect.Response[corev1.ArtifactServiceShowResponse], error) {
	artifact, err := handler.service.Show(ctx, request.Msg.GetName())
	if err != nil {
		return nil, err
	}
	response := &corev1.ArtifactServiceShowResponse{
		Data: &corev1.Artifact{},
	}
	response.Data.Set(artifact)
	return connect.NewResponse(response), nil
}

func (handler *artifactServiceHandler) Update(ctx context.Context, request *connect.Request[corev1.ArtifactServiceUpdateRequest]) (*connect.Response[corev1.ArtifactServiceUpdateResponse], error) {
	fmc := fieldmaskcommander.New(request.Msg.UpdateMask)
	data := request.Msg.GetData().Convert(fmc.GoTo("data"))
	if err := handler.service.Update(ctx, request.Msg.GetName(), data); err != nil {
		return nil, err
	}
	return connect.NewResponse(&corev1.ArtifactServiceUpdateResponse{}), nil
}

// Watch streams the resource stream to the client. With List the server first
// replays the current state terminated by a sync marker, then forwards changes
// until ctx is cancelled. A closed store channel is disabled rather than
// drained so the remaining one can still be read.
func (handler *artifactServiceHandler) Watch(ctx context.Context, request *connect.Request[corev1.ArtifactServiceWatchRequest], stream *connect.ServerStream[corev1.ArtifactServiceWatchResponse]) error {
	eventChan, errChan := handler.service.Watch(ctx, request.Msg.GetList())
	for eventChan != nil || errChan != nil {
		select {
		case event, ok := <-eventChan:
			if !ok {
				eventChan = nil
				continue
			}
			if err := stream.Send(artifactWatchResponse(event)); err != nil {
				return err
			}
		case err, ok := <-errChan:
			if !ok {
				errChan = nil
				continue
			}
			slog.Error(err.Error())
			if err := stream.Send(&corev1.ArtifactServiceWatchResponse{
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

// artifactWatchResponse maps a store event onto the wire: puts carry the
// object, deletes carry only the identity they refer to.
func artifactWatchResponse(event core.ArtifactWatchEvent) *corev1.ArtifactServiceWatchResponse {
	response := &corev1.ArtifactServiceWatchResponse{Name: event.Name}
	switch event.Type {
	case genericstore.EventTypePut:
		response.Type = corev1.EventType_EVENT_TYPE_PUT
		response.Data = &corev1.Artifact{}
		response.Data.Set(event.Object)
	case genericstore.EventTypeDelete:
		response.Type = corev1.EventType_EVENT_TYPE_DELETE
	case genericstore.EventTypeSync:
		response.Type = corev1.EventType_EVENT_TYPE_SYNC
	}
	return response
}

func NewArtifactServiceHandler(service *core.ArtifactService) (string, http.Handler) {
	serviceHandler := &artifactServiceHandler{
		service,
	}
	interceptors := connect.WithInterceptors(interceptors.ErrorInterceptor())
	return corev1connect.NewArtifactServiceHandler(serviceHandler, interceptors)
}
