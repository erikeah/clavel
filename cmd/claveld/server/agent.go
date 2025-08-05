package server

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	"github.com/erikeah/clavel/cmd/claveld/service"
	agent_v1 "github.com/erikeah/clavel/pkg/pb/agent/v1"
	"github.com/erikeah/clavel/pkg/pb/agent/v1/agent_v1connect"
)

type AgentServiceHandler struct {
	service *service.AgentService
}

func (handler *AgentServiceHandler) Show(
	ctx context.Context,
	request *connect.Request[agent_v1.AgentServiceShowRequest],
) (*connect.Response[agent_v1.AgentServiceShowResponse], error) {
	agent := handler.service.Show(request.Msg.Name)
	if agent == nil {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	res := connect.NewResponse(&agent_v1.AgentServiceShowResponse{
		Name: agent.Name,
		Spec: &agent_v1.AgentSpecification{
			Deployments: agent.Spec.Deployments,
		},
	})
	return res, nil
}

func NewAgentServiceHandler() (string, http.Handler) {
	agentServiceHandler := &AgentServiceHandler{
		service: service.GetAgentServiceInstance(),
	}
	return agent_v1connect.NewAgentServiceHandler(agentServiceHandler)
}
