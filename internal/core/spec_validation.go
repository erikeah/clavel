package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/bufbuild/protovalidate-go"
	"github.com/erikeah/clavel/internal/exceptions"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

var (
	pvOnce      sync.Once
	pvValidator *protovalidate.Validator
	pvErr       error
)

func protoValidator() (*protovalidate.Validator, error) {
	pvOnce.Do(func() {
		pvValidator, pvErr = protovalidate.New()
	})
	return pvValidator, pvErr
}

// specResolver resolves dependencies of a consumer's serialized descriptor
// set, falling back to the process-wide registry for well-known types.
type specResolver struct {
	local map[string]protoreflect.FileDescriptor
}

func (r *specResolver) FindFileByPath(path string) (protoreflect.FileDescriptor, error) {
	if f, ok := r.local[path]; ok {
		return f, nil
	}
	return protoregistry.GlobalFiles.FindFileByPath(path)
}

func (r *specResolver) FindDescriptorByName(name protoreflect.FullName) (protoreflect.Descriptor, error) {
	return protoregistry.GlobalFiles.FindDescriptorByName(name)
}

func buildSpecRegistry(schema []byte) (*protoregistry.Files, error) {
	fdset := &descriptorpb.FileDescriptorSet{}
	if err := proto.Unmarshal(schema, fdset); err != nil {
		return nil, errors.Join(errors.New("failed to parse descriptor set"), err)
	}
	resolver := &specResolver{local: map[string]protoreflect.FileDescriptor{}}
	files := &protoregistry.Files{}
	for _, fd := range fdset.GetFile() {
		f, err := protodesc.NewFile(fd, resolver)
		if err != nil {
			return nil, errors.Join(errors.New("failed to build file descriptor"), err)
		}
		resolver.local[fd.GetName()] = f
		if err := files.RegisterFile(f); err != nil {
			return nil, errors.Join(errors.New("failed to register file descriptor"), err)
		}
	}
	return files, nil
}

// ValidateAndCanonicalizeSpec validates a spec against the referenced message
// in a serialized FileDescriptorSet and returns its canonical protojson form.
func ValidateAndCanonicalizeSpec(schema []byte, specMessage string, spec json.RawMessage) (json.RawMessage, error) {
	if len(schema) == 0 {
		return nil, errors.Join(exceptions.InvalidArguments, errors.New("schema is empty"))
	}
	if specMessage == "" {
		return nil, errors.Join(exceptions.InvalidArguments, errors.New("spec_message is empty"))
	}
	if len(spec) == 0 {
		return nil, errors.Join(exceptions.InvalidArguments, errors.New("spec is empty"))
	}
	files, err := buildSpecRegistry(schema)
	if err != nil {
		return nil, errors.Join(exceptions.InternalFailure, err)
	}
	desc, err := files.FindDescriptorByName(protoreflect.FullName(specMessage))
	if err != nil {
		return nil, errors.Join(exceptions.InternalFailure, errors.New("spec message not found"))
	}
	md, ok := desc.(protoreflect.MessageDescriptor)
	if !ok {
		return nil, errors.Join(exceptions.InternalFailure, errors.New("spec_message does not reference a message"))
	}
	msg := dynamicpb.NewMessage(md)
	if err := protojson.Unmarshal(spec, msg); err != nil {
		return nil, errors.Join(exceptions.InvalidArguments, err)
	}
	validator, err := protoValidator()
	if err != nil {
		return nil, errors.Join(exceptions.InternalFailure, errors.New("failed to initialize spec validator"))
	}
	if err := validator.Validate(msg); err != nil {
		return nil, errors.Join(exceptions.InvalidArguments, err)
	}
	canonical, err := protojson.Marshal(msg)
	if err != nil {
		return nil, errors.Join(exceptions.InternalFailure, err)
	}
	return canonical, nil
}

// SpecStringField decodes a spec against the referenced message in a
// serialized FileDescriptorSet and returns the string value of field.
func SpecStringField(schema []byte, specMessage string, spec json.RawMessage, field string) (string, error) {
	if len(schema) == 0 {
		return "", errors.Join(exceptions.InternalFailure, errors.New("schema is empty"))
	}
	if specMessage == "" {
		return "", errors.Join(exceptions.InternalFailure, errors.New("spec_message is empty"))
	}
	if len(spec) == 0 {
		return "", errors.Join(exceptions.InvalidArguments, errors.New("spec is empty"))
	}
	files, err := buildSpecRegistry(schema)
	if err != nil {
		return "", errors.Join(exceptions.InternalFailure, err)
	}
	desc, err := files.FindDescriptorByName(protoreflect.FullName(specMessage))
	if err != nil {
		return "", errors.Join(exceptions.InternalFailure, errors.New("spec message not found"))
	}
	md, ok := desc.(protoreflect.MessageDescriptor)
	if !ok {
		return "", errors.Join(exceptions.InternalFailure, errors.New("spec_message does not reference a message"))
	}
	msg := dynamicpb.NewMessage(md)
	if err := protojson.Unmarshal(spec, msg); err != nil {
		return "", errors.Join(exceptions.InvalidArguments, err)
	}
	fd := md.Fields().ByName(protoreflect.Name(field))
	if fd == nil {
		return "", errors.Join(exceptions.InvalidArguments, fmt.Errorf("field %q not found", field))
	}
	if fd.Kind() != protoreflect.StringKind {
		return "", errors.Join(exceptions.InvalidArguments, fmt.Errorf("field %q is not a string", field))
	}
	return msg.Get(fd).String(), nil
}
