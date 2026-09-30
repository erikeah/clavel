package clavelgen

import (
	"errors"
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Params configure one integration generation. Field names match the plugin
// parameters accepted from buf.gen.yaml / protoc --clavel-gen_opt=... .
type Params struct {
	Name        string
	Group       string
	Version     string
	Kind        string
	Plural      string
	SpecMessage string
	Out         string
}

// Validate checks for the required parameters.
func (p Params) Validate() error {
	if p.Name == "" || p.Group == "" || p.Version == "" || p.Kind == "" || p.Plural == "" || p.SpecMessage == "" {
		return errors.New("missing required parameters (name, group, version, kind, plural, spec_message)")
	}
	return nil
}

// OutputFile returns where the Nix descriptor should be written, relative to
// the generator's output root.
func (p Params) OutputFile() string {
	if p.Out != "" {
		return p.Out
	}
	return "nix/generated/" + p.Name + ".nix"
}

// Run is the plugin generate function, analogous to protoc-gen-go-grpc's: it
// walks the files in the request, renders the Nix descriptor for the file that
// contains the spec message, and writes it. Requests that do not carry the spec
// message (e.g. other directories under a directory-strategy generation) are
// skipped without error.
func Run(gen *protogen.Plugin, p Params) error {
	if err := p.Validate(); err != nil {
		return err
	}
	full := protoreflect.FullName(p.SpecMessage)
	var specMD protoreflect.MessageDescriptor
	for _, f := range gen.Files {
		if !f.Generate {
			continue
		}
		for i := 0; i < f.Desc.Messages().Len(); i++ {
			if m := FindMessage(f.Desc.Messages().Get(i), full); m != nil {
				specMD = m
			}
		}
	}
	if specMD == nil {
		// Not this plugin's file; generate nothing.
		return nil
	}
	schema, err := EncodeDescriptorSet(gen.Request.GetProtoFile())
	if err != nil {
		return fmt.Errorf("encode descriptor set: %w", err)
	}
	content := Generate(p.Group, p.Version, p.Kind, p.Plural, string(specMD.FullName()), schema, specMD)
	file := gen.NewGeneratedFile(p.OutputFile(), "")
	if _, err := file.Write([]byte(content)); err != nil {
		return fmt.Errorf("write %s: %w", p.OutputFile(), err)
	}
	return nil
}
