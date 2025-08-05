package transport

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	"github.com/erikeah/clavel/internal/model"
	"github.com/erikeah/clavel/internal/service"
	"github.com/erikeah/clavel/internal/validation"
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
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	res := connect.NewResponse(&projectv1.ProjectServiceShowResponse{
		Name: project.Name,
		Spec: &projectv1.ProjectSpecification{
			Flakeref: project.Spec.Flakeref,
		},
	})
	return res, nil
}

func (handler *projectServiceHandler) Add(
	ctx context.Context,
	request *connect.Request[projectv1.ProjectServiceAddRequest],
) (*connect.Response[projectv1.ProjectServiceAddResponse], error) {
	project := &model.Project{}
	err := copier.Copy(project, request.Msg)
	if err != nil {
		return nil, err
	}
	err = validation.ValidateProject(project)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	err = handler.service.Add(ctx, project)
	if err != nil {
		return nil, connect.NewError(connect.CodeAlreadyExists, err)
	}
	return &connect.Response[projectv1.ProjectServiceAddResponse]{}, nil
}

func NewProjectServiceHandler(service service.ProjectService) (string, http.Handler) {
	projectServiceHandler := &projectServiceHandler{
		service,
	}
	return projectv1connect.NewProjectServiceHandler(projectServiceHandler)
}
