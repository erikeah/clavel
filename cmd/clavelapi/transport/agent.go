package transport

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	"github.com/erikeah/clavel/internal/service"
	agentv1 "github.com/erikeah/clavel/pkg/pb/agent/v1"
	"github.com/erikeah/clavel/pkg/pb/agent/v1/agentv1connect"
)

type AgentServiceHandler struct {
	service *service.AgentService
}

func (handler *AgentServiceHandler) Show(
	ctx context.Context,
	request *connect.Request[agentv1.AgentServiceShowRequest],
) (*connect.Response[agentv1.AgentServiceShowResponse], error) {
	agent := handler.service.Show(request.Msg.Name)
	if agent == nil {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	res := connect.NewResponse(&agentv1.AgentServiceShowResponse{
		Name: agent.Name,
		Spec: &agentv1.AgentSpecification{
			Deployments: agent.Spec.Deployments,
		},
	})
	return res, nil
}

func NewAgentServiceHandler() (string, http.Handler) {
	agentServiceHandler := &AgentServiceHandler{
		service: service.GetAgentServiceInstance(),
	}
	return agentv1connect.NewAgentServiceHandler(agentServiceHandler)
}
