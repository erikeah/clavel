package transport

import (
	"context"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
	"github.com/erikeah/clavel/internal/fieldmaskcommander"
	"github.com/erikeah/clavel/internal/interceptors"
	"github.com/erikeah/clavel/internal/project"
	projectv1 "github.com/erikeah/clavel/pkg/api/project/v1"
	"github.com/erikeah/clavel/pkg/api/project/v1/projectv1connect"
)

type projectServiceHandler struct {
	service *project.ProjectService
}

func (handler *projectServiceHandler) Create(
	ctx context.Context,
	request *connect.Request[projectv1.ProjectServiceCreateRequest],
) (*connect.Response[projectv1.ProjectServiceCreateResponse], error) {
	project := request.Msg.GetData().Convert(nil)
	if err := handler.service.Create(ctx, project); err != nil {
		return nil, err
	}
	return connect.NewResponse(&projectv1.ProjectServiceCreateResponse{}), nil
}

func (handler *projectServiceHandler) Delete(
	ctx context.Context,
	request *connect.Request[projectv1.ProjectServiceDeleteRequest],
) (*connect.Response[projectv1.ProjectServiceDeleteResponse], error) {
	if err := handler.service.Delete(ctx, request.Msg.GetName()); err != nil {
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
	for _, project := range projects {
		projectResponse := &projectv1.Project{}
		projectResponse.Set(project)
		response.Data = append(response.Data, projectResponse)
	}
	return connect.NewResponse(response), nil
}

func (handler *projectServiceHandler) Show(
	ctx context.Context,
	request *connect.Request[projectv1.ProjectServiceShowRequest],
) (*connect.Response[projectv1.ProjectServiceShowResponse], error) {
	project, err := handler.service.Show(ctx, request.Msg.GetName())
	if err != nil {
		return nil, err
	}
	response := &projectv1.ProjectServiceShowResponse{
		Data: &projectv1.Project{},
	}
	response.Data.Set(project)
	return connect.NewResponse(response), nil
}

func (handler *projectServiceHandler) Update(ctx context.Context, request *connect.Request[projectv1.ProjectServiceUpdateRequest]) (*connect.Response[projectv1.ProjectServiceUpdateResponse], error) {
	fmc := fieldmaskcommander.New(request.Msg.UpdateMask)
	data := request.Msg.GetData().Convert(fmc.GoTo("data"))
	if err := handler.service.Update(ctx, request.Msg.GetName(), data); err != nil {
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
				response.Data.Set(project)
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
