package core

import (
	"encoding/json"
	"testing"

	validate "buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func testSchema(t *testing.T) []byte {
	t.Helper()
	fd := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("claveltest/v1/spec.proto"),
		Package: proto.String("claveltest.v1"),
		Syntax:  proto.String("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("TestSpec"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:    proto.String("name"),
						Number:  proto.Int32(1),
						Label:   descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:    descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
						Options: &descriptorpb.FieldOptions{},
					},
				},
			},
		},
	}
	fieldConstraints := &validate.FieldConstraints{
		Type: &validate.FieldConstraints_String_{
			String_: &validate.StringRules{MinLen: proto.Uint64(5)},
		},
	}
	proto.SetExtension(fd.MessageType[0].Field[0].Options, validate.E_Field, fieldConstraints)
	schema, err := proto.Marshal(&descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{fd}})
	if err != nil {
		t.Fatalf("marshal descriptor set: %v", err)
	}
	return schema
}

func TestValidateAndCanonicalizeSpecConstraintFails(t *testing.T) {
	schema := testSchema(t)
	_, err := ValidateAndCanonicalizeSpec(schema, "claveltest.v1.TestSpec", json.RawMessage(`{"name":"abcd"}`))
	if err == nil {
		t.Fatal("expected constraint violation for short name")
	}
}

func TestValidateAndCanonicalizeSpecConstraintPasses(t *testing.T) {
	schema := testSchema(t)
	_, err := ValidateAndCanonicalizeSpec(schema, "claveltest.v1.TestSpec", json.RawMessage(`{"name":"abcde"}`))
	if err != nil {
		t.Fatalf("expected valid spec: %v", err)
	}
}

func TestValidateAndCanonicalizeSpecRejectsUnknownField(t *testing.T) {
	schema := testSchema(t)
	_, err := ValidateAndCanonicalizeSpec(schema, "claveltest.v1.TestSpec", json.RawMessage(`{"nope":true}`))
	if err == nil {
		t.Fatal("expected unknown-field rejection")
	}
}

func TestValidateAndCanonicalizeSpecRejectsUnknownMessage(t *testing.T) {
	schema := testSchema(t)
	_, err := ValidateAndCanonicalizeSpec(schema, "claveltest.v1.DoesNotExist", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected unknown-message rejection")
	}
}
