package core

import (
	"context"
	"errors"
	"time"

	"github.com/erikeah/clavel/internal/exceptions"
)

type EvaluationService struct {
	store       EvaluationStore
	setDefaults func(*Evaluation) error
	merge       func(over *Evaluation, from *Evaluation) (bool, error)
	validate    func(Evaluation) error
}

func (s *EvaluationService) Create(ctx context.Context, data *Evaluation) error {
	resource := &Evaluation{}
	if err := s.setDefaults(resource); err != nil {
		return err
	}
	if _, err := s.merge(resource, data); err != nil {
		return err
	}
	resource.Status = EvaluationStatus{}
	if err := s.validate(*resource); err != nil {
		return errors.Join(exceptions.InvalidArguments, err)
	}
	if err := s.store.Create(ctx, resource.Name, resource); err != nil {
		return err
	}
	return nil
}

func (s *EvaluationService) Delete(ctx context.Context, name string) error {
	target, err := s.Show(ctx, name)
	if err != nil {
		return err
	}
	if len(target.Metadata.Finalizers) > 0 {
		nowUTC := time.Now().UTC()
		target.Metadata.DeletionTimestamp = &nowUTC
		return s.Update(ctx, name, target)
	} else {
		return s.store.Delete(ctx, name)
	}
}

func (s *EvaluationService) List(ctx context.Context) ([]*Evaluation, error) {
	return s.store.List(ctx)
}

func (s *EvaluationService) Show(ctx context.Context, name string) (*Evaluation, error) {
	return s.store.FindOne(ctx, name)
}

func (s *EvaluationService) Update(ctx context.Context, name string, data *Evaluation) error {
	if data == nil {
		return exceptions.InvalidArguments
	}
	target, err := s.Show(ctx, name)
	if err != nil {
		return err
	}
	if hasChanged, err := s.merge(target, data); err != nil {
		return errors.Join(exceptions.InternalFailure, err)
	} else if !hasChanged {
		return nil
	}
	if err := s.validate(*target); err != nil {
		return errors.Join(exceptions.InvalidArguments, err)
	}
	return s.store.Update(ctx, name, target)
}

func (s *EvaluationService) Watch(ctx context.Context) (<-chan *Evaluation, <-chan error) {
	return s.store.Watch(ctx)
}

func NewEvaluationService(store EvaluationStore) *EvaluationService {
	return &EvaluationService{
		store:       store,
		validate:    ValidateEvaluation,
		merge:       MergeEvaluation,
		setDefaults: SetDefaults_Evaluation,
	}
}
