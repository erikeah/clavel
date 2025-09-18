package transport

import (
	"net/http"

	"connectrpc.com/connect"
	"github.com/erikeah/clavel/internal/genericservicehandler"
	"github.com/erikeah/clavel/internal/interceptors"
	"github.com/erikeah/clavel/internal/project"
	projectv1 "github.com/erikeah/clavel/pkg/api/project/v1"
	"github.com/erikeah/clavel/pkg/api/project/v1/projectv1connect"
)

func NewProjectServiceHandler(service project.ProjectService) (string, http.Handler) {
	serviceHandler := genericservicehandler.NewServiceHandler[
		projectv1.ProjectServiceCreateRequest,
		projectv1.ProjectServiceCreateResponse,
		projectv1.ProjectServiceDeleteRequest,
		projectv1.ProjectServiceDeleteResponse,
		projectv1.ProjectServiceListRequest,
		projectv1.ProjectServiceListResponse,
		projectv1.ProjectServiceShowRequest,
		projectv1.ProjectServiceShowResponse,
	](service)
	interceptors := connect.WithInterceptors(interceptors.ErrorInterceptor())
	return projectv1connect.NewProjectServiceHandler(serviceHandler, interceptors)
}
