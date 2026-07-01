package dboperator_test

import (
	"context"
	"errors"
	"testing"

	"gorm.io/gorm"

	"sealdice-core/utils/constant"
	dboperator "sealdice-core/utils/dboperator"
	operator "sealdice-core/utils/dboperator/engine"
)

type stubOperator struct {
	initCalled      bool
	bootstrapCalled bool
	initErr         error
	bootstrapErr    error
}

func (s *stubOperator) Init(_ context.Context) error {
	s.initCalled = true
	return s.initErr
}

func (s *stubOperator) BootstrapSchema() error {
	s.bootstrapCalled = true
	return s.bootstrapErr
}

func (s *stubOperator) Type() string { return "stub" }

func (s *stubOperator) DBCheck() {}

func (s *stubOperator) GetDataDB(_ constant.DBMode) *gorm.DB { return nil }

func (s *stubOperator) GetLogDB(_ constant.DBMode) *gorm.DB { return nil }

func (s *stubOperator) GetCensorDB(_ constant.DBMode) *gorm.DB { return nil }

func (s *stubOperator) Close() {}

var _ operator.DatabaseOperator = (*stubOperator)(nil)

func TestStubOperatorInitDoesNotBootstrap(t *testing.T) {
	stub := &stubOperator{}

	if err := stub.Init(t.Context()); err != nil {
		t.Fatalf("stub.Init() error = %v", err)
	}
	if !stub.initCalled {
		t.Fatal("expected Init to be called")
	}
	if stub.bootstrapCalled {
		t.Fatal("expected BootstrapSchema not to be called during Init")
	}
}

func TestStubOperatorBootstrapReturnsConfiguredError(t *testing.T) {
	stub := &stubOperator{bootstrapErr: context.Canceled}

	err := stub.BootstrapSchema()
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestBootstrapDatabaseSchemaRemainsExported(t *testing.T) {
	_ = dboperator.BootstrapDatabaseSchema
}
