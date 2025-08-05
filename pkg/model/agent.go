package model

type AgentSpecification struct {
	Deployments []string
}

func NewAgentSpecification(opts *AgentSpecification) *AgentSpecification {
	spec := &AgentSpecification{}
	spec.Deployments = opts.Deployments
	return spec
}

type Agent struct {
	Name string
	Spec *AgentSpecification
}

func NewAgent(opts *Agent) *Agent {
	agent := &Agent{}
	agent.Name = opts.Name
	agent.Spec = NewAgentSpecification(opts.Spec)
	return agent
}
