package corev1

import (
	"testing"
	"time"

	"github.com/erikeah/clavel/internal/core"
	"github.com/erikeah/clavel/internal/fieldmaskcommander"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func TestMetadataRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	original := core.Metadata{
		Name:            "server",
		Generation:      3,
		ResourceVersion: "17",
		UID:             "uid-1",
		Labels:          map[string]string{"app": "server"},
		Annotations:     map[string]string{"clavel.io/note": "hello"},
		OwnerReferences: []core.OwnerReference{{
			APIVersion: core.EvaluationAPIVersion,
			Kind:       core.EvaluationKind,
			Name:       "root",
			UID:        "uid-root",
			Controller: true,
		}},
		GenerateName:      "srv-",
		Namespace:         "default",
		Finalizers:        []string{"clavel.io/cache"},
		CreationTimestamp: &now,
	}

	proto := &Metadata{}
	proto.Set(&original)
	back := proto.Convert(nil)

	if back.Name != original.Name || back.UID != original.UID {
		t.Fatalf("identity round trip failed: %+v", back)
	}
	if back.Namespace != original.Namespace || back.GenerateName != original.GenerateName {
		t.Fatalf("identity extras round trip failed: %+v", back)
	}
	if back.Labels["app"] != "server" || back.Annotations["clavel.io/note"] != "hello" {
		t.Fatalf("maps round trip failed: %+v", back)
	}
	if len(back.OwnerReferences) != 1 || back.OwnerReferences[0] != original.OwnerReferences[0] {
		t.Fatalf("ownerReferences round trip failed: %+v", back.OwnerReferences)
	}
	if back.Finalizers == nil || back.Finalizers[0] != "clavel.io/cache" {
		t.Fatalf("finalizers round trip failed: %+v", back.Finalizers)
	}
	if back.CreationTimestamp == nil || !back.CreationTimestamp.Equal(now) {
		t.Fatalf("creationTimestamp = %v, want %v", back.CreationTimestamp, now)
	}
	if back.Generation != 3 || back.ResourceVersion != "17" {
		t.Fatalf("generation/resourceVersion round trip failed: %+v", back)
	}
}

func TestMetadataConvertDropsUnmaskedFields(t *testing.T) {
	original := core.Metadata{
		Name:       "server",
		Labels:     map[string]string{"app": "server"},
		Finalizers: []string{"clavel.io/cache"},
	}
	root := fieldmaskcommander.New(&fieldmaskpb.FieldMask{
		Paths: []string{"data.metadata.finalizers"},
	})
	metadataFmc := root.GoTo("data").GoTo("metadata")

	proto := &Metadata{}
	proto.Set(&original)
	back := proto.Convert(metadataFmc)

	if back.Labels != nil {
		t.Fatalf("Labels = %v, want nil when not masked", back.Labels)
	}
	if back.Finalizers == nil || back.Finalizers[0] != "clavel.io/cache" {
		t.Fatalf("Finalizers = %v, want carried when masked", back.Finalizers)
	}
	if back.Name != "server" {
		t.Fatalf("Name = %q, want always carried", back.Name)
	}
}

func TestEvaluationRoundTripCarriesTypeMeta(t *testing.T) {
	original := &core.Evaluation{
		APIVersion: core.EvaluationAPIVersion,
		Kind:       core.EvaluationKind,
		Metadata:   core.Metadata{Name: "server", UID: "uid-1"},
		Spec:       core.EvaluationSpecification{Reference: "ref#server"},
		Status: core.EvaluationStatus{
			Phase:              core.EvaluationPhaseSucceeded,
			Result:             core.EncodeResult([]byte(`"/nix/store/abc"`)),
			ObservedGeneration: 2,
		},
	}

	proto := &Evaluation{}
	proto.Set(original)
	if proto.GetApiVersion() != core.EvaluationAPIVersion || proto.GetKind() != core.EvaluationKind {
		t.Fatalf("TypeMeta = %s/%s, want %s/%s", proto.GetApiVersion(), proto.GetKind(), core.EvaluationAPIVersion, core.EvaluationKind)
	}
	if proto.GetMetadata().GetName() != "server" {
		t.Fatalf("metadata.name = %q, want server", proto.GetMetadata().GetName())
	}

	back := proto.Convert(nil)
	if back.APIVersion != original.APIVersion || back.Kind != original.Kind {
		t.Fatalf("TypeMeta round trip failed: %s/%s", back.APIVersion, back.Kind)
	}
	if back.Metadata.Name != "server" || back.Metadata.UID != "uid-1" {
		t.Fatalf("metadata round trip failed: %+v", back.Metadata)
	}
	if back.Status != original.Status {
		t.Fatalf("status round trip failed: %+v", back.Status)
	}
	if back.Spec != original.Spec {
		t.Fatalf("spec round trip failed: %+v", back.Spec)
	}
}
