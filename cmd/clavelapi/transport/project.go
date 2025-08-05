package transport

import (
	"context"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
	"github.com/erikeah/clavel/internal/exception"
	"github.com/erikeah/clavel/internal/model"
	"github.com/erikeah/clavel/internal/service"
	projectv1 "github.com/erikeah/clavel/pkg/pb/project/v1"
	"github.com/erikeah/clavel/pkg/pb/project/v1/projectv1connect"
	"github.com/jinzhu/copier"
)

type projectServiceHandler struct {
	service service.ProjectService
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
		return nil, exception.InternalFailure
	}
	return connect.NewResponse(response), nil
}

func (handler *projectServiceHandler) Add(
	ctx context.Context,
	request *connect.Request[projectv1.ProjectServiceAddRequest],
) (*connect.Response[projectv1.ProjectServiceAddResponse], error) {
	project := &model.Project{}
	err := copier.Copy(project, request.Msg.Data)
	if err != nil {
		return nil, err
	}
	err = handler.service.Add(ctx, project)
	if err != nil {
		return nil, err
	}
	return &connect.Response[projectv1.ProjectServiceAddResponse]{}, nil
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
		return nil, exception.InternalFailure
	}
	return connect.NewResponse(response), nil
}

func NewProjectServiceHandler(service service.ProjectService) (string, http.Handler) {
	serviceHandler := &projectServiceHandler{
		service,
	}
	interceptors := connect.WithInterceptors(ErrorInterceptor())
	return projectv1connect.NewProjectServiceHandler(serviceHandler, interceptors)
}
