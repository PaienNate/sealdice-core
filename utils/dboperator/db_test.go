package dboperator

import (
	"context"
	"testing"

	"gorm.io/gorm"

	"sealdice-core/utils/constant"
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

func TestInitEngineBootstrapsSchema(t *testing.T) {
	prevEngine := engine
	prevErr := errEngineInstance
	prevOnce := once
	t.Cleanup(func() {
		engine = prevEngine
		errEngineInstance = prevErr
		once = prevOnce
	})

	stub := &stubOperator{}
	engine = stub
	errEngineInstance = nil

	if err := stub.Init(context.Background()); err != nil {
		t.Fatalf("initializeEngine() error = %v", err)
	}
	if !stub.initCalled {
		t.Fatal("expected Init to be called")
	}
	if stub.bootstrapCalled {
		t.Fatal("expected BootstrapSchema not to be called during Init")
	}
}

func TestBootstrapDatabaseSchemaCallsBootstrap(t *testing.T) {
	prevEngine := engine
	prevErr := errEngineInstance
	prevOnce := once
	t.Cleanup(func() {
		engine = prevEngine
		errEngineInstance = prevErr
		once = prevOnce
	})

	stub := &stubOperator{}
	engine = stub
	errEngineInstance = nil
	once.Do(func() {})

	if err := BootstrapDatabaseSchema(); err != nil {
		t.Fatalf("BootstrapDatabaseSchema() error = %v", err)
	}
	if !stub.bootstrapCalled {
		t.Fatal("expected BootstrapSchema to be called")
	}
}

func TestBootstrapDatabaseSchemaPropagatesBootstrapError(t *testing.T) {
	prevEngine := engine
	prevErr := errEngineInstance
	prevOnce := once
	t.Cleanup(func() {
		engine = prevEngine
		errEngineInstance = prevErr
		once = prevOnce
	})

	stub := &stubOperator{
		bootstrapErr: context.Canceled,
	}
	engine = stub
	errEngineInstance = nil
	once.Do(func() {})

	err := BootstrapDatabaseSchema()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

var _ operator.DatabaseOperator = (*stubOperator)(nil)
