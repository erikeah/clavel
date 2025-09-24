package fieldmaskcommander

import (
	"strings"

	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type FieldMaskCommander struct {
	*fieldmaskpb.FieldMask
	currentPath string
}

func New(fm *fieldmaskpb.FieldMask) *FieldMaskCommander {
	return &FieldMaskCommander{
		currentPath: "",
		FieldMask:   fm,
	}
}

func (fmc *FieldMaskCommander) GoTo(fieldLevel string) *FieldMaskCommander {
	if fmc == nil {
		return nil
	}
	fieldLevel = strings.Trim(fieldLevel, ".")
	if fmc.currentPath != "" {
		fieldLevel = fmc.currentPath + "." + fieldLevel
	}
	return &FieldMaskCommander{currentPath: fieldLevel, FieldMask: fmc.FieldMask}
}

func (fmc *FieldMaskCommander) IsFieldMasked(field string) bool {
	if fmc == nil {
		return true
	}
	field = strings.Trim(field, ".")
	fieldPath := fmc.currentPath + "." + field
	for _, path := range fmc.GetPaths() {
		if strings.HasPrefix(path, fieldPath) {
			return true
		}
	}
	return false
}
