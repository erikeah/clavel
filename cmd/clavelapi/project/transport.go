package transport

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/erikeah/clavel/internal/exceptions"
	"github.com/erikeah/clavel/internal/interceptors"
	"github.com/erikeah/clavel/internal/project"
	projectv1 "github.com/erikeah/clavel/pkg/api/project/v1"
	"github.com/erikeah/clavel/pkg/api/project/v1/projectv1connect"
	"github.com/jinzhu/copier"
)

type projectServiceHandler struct {
	service *project.ProjectService
}

func (handler *projectServiceHandler) Create(
	ctx context.Context,
	request *connect.Request[projectv1.ProjectServiceCreateRequest],
) (*connect.Response[projectv1.ProjectServiceCreateResponse], error) {
	project := &project.Project{}
	if err := copier.CopyWithOption(project, request.Msg.GetData(), copier.Option{IgnoreEmpty: true}); err != nil {
		return nil, err
	}
	if err := handler.service.Create(ctx, project); err != nil {
		return nil, err
	}
	return connect.NewResponse(&projectv1.ProjectServiceCreateResponse{}), nil
}

func (handler *projectServiceHandler) Delete(
	ctx context.Context,
	request *connect.Request[projectv1.ProjectServiceDeleteRequest],
) (*connect.Response[projectv1.ProjectServiceDeleteResponse], error) {
	if err := handler.service.Delete(ctx, request.Msg.GetQuery().GetName()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&projectv1.ProjectServiceDeleteResponse{}), nil
}

func (handler *projectServiceHandler) List(
	ctx context.Context,
	request *connect.Request[projectv1.ProjectServiceListRequest],
) (*connect.Response[projectv1.ProjectServiceListResponse], error) {
	projects, err := handler.service.List(ctx)
	if err != nil {
		return nil, err
	}
	response := &projectv1.ProjectServiceListResponse{
		Data: []*projectv1.Project{},
	}
	if err := copier.Copy(&response.Data, projects); err != nil {
		slog.Error(err.Error())
		return nil, exceptions.InternalFailure
	}
	return connect.NewResponse(response), nil
}

func (handler *projectServiceHandler) Show(
	ctx context.Context,
	request *connect.Request[projectv1.ProjectServiceShowRequest],
) (*connect.Response[projectv1.ProjectServiceShowResponse], error) {
	project, err := handler.service.Show(ctx, request.Msg.GetQuery().GetName())
	if err != nil {
		return nil, err
	}
	response := &projectv1.ProjectServiceShowResponse{
		Data: &projectv1.Project{},
	}
	if err := copier.CopyWithOption(response.Data, project,
		copier.Option{
			Converters: []copier.TypeConverter{
				{
					SrcType: time.Time{},
					DstType: copier.String,
					Fn: func(src interface{}) (interface{}, error) {
						s, ok := src.(time.Time)

						if !ok {
							return nil, exceptions.InternalFailure
						}

						return s.Format(time.RFC3339), nil
					},
				},
			}}); err != nil {
		slog.Error(err.Error())
		return nil, exceptions.InternalFailure
	}
	return connect.NewResponse(response), nil
}

func (handler *projectServiceHandler) Update(ctx context.Context, request *connect.Request[projectv1.ProjectServiceUpdateRequest]) (*connect.Response[projectv1.ProjectServiceUpdateResponse], error) {
	project := &project.Project{}
	if err := copier.Copy(project, request.Msg.GetData()); err != nil {
		return nil, err
	}
	if err := handler.service.Update(ctx, request.Msg.GetQuery().GetName(), project); err != nil {
		return nil, err
	}
	return connect.NewResponse(&projectv1.ProjectServiceUpdateResponse{}), nil
}

// TODO: Set a query option to list resources first
func (handler *projectServiceHandler) Watch(ctx context.Context, _ *connect.Request[projectv1.ProjectServiceWatchRequest], stream *connect.ServerStream[projectv1.ProjectServiceWatchResponse]) error {
	projectChan, errChan := handler.service.Watch(ctx)
	for {
		select {
		case project, ok := <-projectChan:
			if ok {
				response := &projectv1.ProjectServiceWatchResponse{
					Data: &projectv1.Project{},
				}
				if err := copier.Copy(response.Data, project); err != nil {
					slog.Error(err.Error())
					return exceptions.InternalFailure
				}
				if err := stream.Send(response); err != nil {
					return err
				}
			}
		case err := <-errChan:
			slog.Error(err.Error())
			response := &projectv1.ProjectServiceWatchResponse{
				Error: &projectv1.Error{
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

func NewProjectServiceHandler(service *project.ProjectService) (string, http.Handler) {
	serviceHandler := &projectServiceHandler{
		service,
	}
	interceptors := connect.WithInterceptors(interceptors.ErrorInterceptor())
	return projectv1connect.NewProjectServiceHandler(serviceHandler, interceptors)
}
