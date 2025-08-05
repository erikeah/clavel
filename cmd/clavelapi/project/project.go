package transport

import (
	"context"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
	"github.com/erikeah/clavel/internal/exceptions"
	"github.com/erikeah/clavel/internal/project"
	"github.com/erikeah/clavel/internal/transport/interceptors"
	projectv1 "github.com/erikeah/clavel/pkg/api/project/v1"
	"github.com/erikeah/clavel/pkg/api/project/v1/projectv1connect"
	"github.com/jinzhu/copier"
)

type projectServiceHandler struct {
	service project.ProjectService
}

func (handler *projectServiceHandler) Create(
	ctx context.Context,
	request *connect.Request[projectv1.ProjectServiceCreateRequest],
) (*connect.Response[projectv1.ProjectServiceCreateResponse], error) {
	project := &project.Project{}
	err := copier.Copy(project, request.Msg.Data)
	if err != nil {
		return nil, err
	}
	err = handler.service.Create(ctx, project)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&projectv1.ProjectServiceCreateResponse{}), nil
}

func (handler *projectServiceHandler) Delete(
	ctx context.Context,
	request *connect.Request[projectv1.ProjectServiceDeleteRequest],
) (*connect.Response[projectv1.ProjectServiceDeleteResponse], error) {
	if err := handler.service.Delete(ctx, request.Msg.Name); err != nil {
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
	err = copier.Copy(&response.Data, projects)
	if err != nil {
		slog.Error(err.Error())
		return nil, exceptions.InternalFailure
	}
	return connect.NewResponse(response), nil
}

func (handler *projectServiceHandler) Show(
	ctx context.Context,
	request *connect.Request[projectv1.ProjectServiceShowRequest],
) (*connect.Response[projectv1.ProjectServiceShowResponse], error) {
	project, err := handler.service.Show(ctx, request.Msg.Name)
	if err != nil {
		return nil, err
	}
	response := &projectv1.ProjectServiceShowResponse{
		Data: &projectv1.Project{},
	}
	err = copier.Copy(response.Data, project)
	if err != nil {
		slog.Error(err.Error())
		return nil, exceptions.InternalFailure
	}
	return connect.NewResponse(response), nil
}

func NewProjectServiceHandler(service project.ProjectService) (string, http.Handler) {
	serviceHandler := &projectServiceHandler{
		service,
	}
	interceptors := connect.WithInterceptors(interceptors.ErrorInterceptor())
	return projectv1connect.NewProjectServiceHandler(serviceHandler, interceptors)
}
