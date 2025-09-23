package utils

import (
	"slices"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func RelocateFieldMask(message proto.Message, fm *fieldmaskpb.FieldMask, location string) (*fieldmaskpb.FieldMask, error) {
	if fm == nil || message == nil {
		return nil, nil
	}
	location = strings.Trim(location, ".")
	if location == "" {
		return fieldmaskpb.New(message, fm.Paths...)
	}
	paths := []string{}
	for _, path := range fm.Paths {
		newPath, ok := strings.CutPrefix(path, location + ".")
		if ok {
			paths = append(paths, newPath)
		}
	}
	return fieldmaskpb.New(message, paths...)
}

func IsFieldMasked(mask *fieldmaskpb.FieldMask, fieldPath string) bool {
	if mask == nil {
		return false
	}
	return slices.Contains(mask.Paths, fieldPath)
}
