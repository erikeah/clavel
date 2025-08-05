package service

import (
	"sync"

	"github.com/erikeah/clavel/pkg/model"
)

type AgentService struct{}

var agentServiceInstance *AgentService

func (*AgentService) Show(name string) *model.Agent {
	for _, agent := range agents {
		if agent.Name == name {
			return model.NewAgent(agent)
		}
	}
	return nil
}

func GetAgentServiceInstance() *AgentService {
	sync.OnceFunc(func() {
		agentServiceInstance = &AgentService{}
	})
	return agentServiceInstance
}

var agents []*model.Agent = []*model.Agent{{
	Name: "Foo",
	Spec: &model.AgentSpecification{
		Deployments: []string{"vm1"}}}, {
	Name: "Bar",
	Spec: &model.AgentSpecification{
		Deployments: []string{"vm2"}}},
}
