// clavel-gen is a protoc plugin (usable with buf) that derives a Nix descriptor
// from a resource spec proto message. Install it by building this program and
// making it accessible within your PATH with the name clavel-gen, then register
// it in buf.gen.yaml:
//
//	plugins:
//	  - local: clavel-gen
//	    out: .
//	    opt:
//	    - name=nixos
//	    - group=clavel.io
//	    - version=v1
//	    - kind=NixosUnit
//	    - plural=nixosunits
//	    - spec_message=clavel.units.nixos.v1.NixosUnitSpec
//
// For each file in the request that contains the configured spec message it
// writes nix/generated/<name>.nix with the base64 FileDescriptorSet schema and
// a Nix submodule mirroring the spec fields.
package main

import (
	"flag"
	"fmt"

	"github.com/erikeah/clavel/internal/clavelgen"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"
)

const version = "0.1.0"

func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Printf("clavel-gen %v\n", version)
		return
	}

	var paramsFlag flag.FlagSet
	params := clavelgen.Params{}
	flags := map[string]*string{
		"name":         &params.Name,
		"group":        &params.Group,
		"version":      &params.Version,
		"kind":         &params.Kind,
		"plural":       &params.Plural,
		"spec_message": &params.SpecMessage,
		"out":          &params.Out,
	}
	for name, dst := range flags {
		paramsFlag.StringVar(dst, name, "", "")
	}

	protogen.Options{
		ParamFunc: paramsFlag.Set,
	}.Run(func(gen *protogen.Plugin) error {
		gen.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)
		return clavelgen.Run(gen, params)
	})
}
