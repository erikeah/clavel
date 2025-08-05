package project

type ProjectSpecification struct {
	Flakeref string `json:"flakeref"`
}

type Project struct {
	Name string                `json:"name"`
	Spec *ProjectSpecification `json:"spec"`
}
