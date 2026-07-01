package upgrade

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"gorm.io/gorm"

	"sealdice-core/utils/constant"
	operator "sealdice-core/utils/dboperator/engine"
)

type managerTestOperator struct{}

func (o *managerTestOperator) Init(_ context.Context) error           { return nil }
func (o *managerTestOperator) BootstrapSchema() error                 { return nil }
func (o *managerTestOperator) Type() string                           { return "test" }
func (o *managerTestOperator) DBCheck()                               {}
func (o *managerTestOperator) GetDataDB(_ constant.DBMode) *gorm.DB   { return nil }
func (o *managerTestOperator) GetLogDB(_ constant.DBMode) *gorm.DB    { return nil }
func (o *managerTestOperator) GetCensorDB(_ constant.DBMode) *gorm.DB { return nil }
func (o *managerTestOperator) Close()                                 {}

type managerTestStore struct {
	applied map[string]bool
	records []UpgradeRecord
}

func (s *managerTestStore) IsApplied(id string) (bool, error) {
	return s.applied[id], nil
}

func (s *managerTestStore) SaveRecord(record UpgradeRecord) error {
	s.records = append(s.records, record)
	return nil
}

func (s *managerTestStore) LoadRecords() ([]UpgradeRecord, error) {
	return append([]UpgradeRecord(nil), s.records...), nil
}

func TestManagerApplyPhaseRunsOnlyMatchingPhaseWithSignals(t *testing.T) {
	store := &managerTestStore{applied: map[string]bool{}}
	op := &managerTestOperator{}
	var executed []string

	mgr := &Manager{Store: store, Database: op}
	mgr.Register(Upgrade{
		ID:    "002_post",
		Phase: PhasePostBootstrap,
		ShouldRun: func(_ operator.DatabaseOperator) (bool, error) {
			return true, nil
		},
		Apply: func(logf func(string), _ operator.DatabaseOperator) error {
			executed = append(executed, "002_post")
			logf("post")
			return nil
		},
	})
	mgr.Register(Upgrade{
		ID:    "001_pre",
		Phase: PhasePreBootstrap,
		ShouldRun: func(_ operator.DatabaseOperator) (bool, error) {
			return true, nil
		},
		Apply: func(logf func(string), _ operator.DatabaseOperator) error {
			executed = append(executed, "001_pre")
			logf("pre")
			return nil
		},
	})
	mgr.Register(Upgrade{
		ID:    "003_pre_skip",
		Phase: PhasePreBootstrap,
		ShouldRun: func(_ operator.DatabaseOperator) (bool, error) {
			return false, nil
		},
		Apply: func(logf func(string), _ operator.DatabaseOperator) error {
			t.Fatal("skip upgrade should not execute")
			return nil
		},
	})

	if err := mgr.ApplyPhase(PhasePreBootstrap); err != nil {
		t.Fatalf("ApplyPhase(PreBootstrap) error = %v", err)
	}

	want := []string{"001_pre"}
	if !reflect.DeepEqual(executed, want) {
		t.Fatalf("executed = %v, want %v", executed, want)
	}
	if len(store.records) != 1 || store.records[0].ID != "001_pre" || !store.records[0].Success {
		t.Fatalf("unexpected saved records: %+v", store.records)
	}
}

func TestManagerApplyPhaseSkipsMetadataAppliedEvenIfSignalStillExists(t *testing.T) {
	store := &managerTestStore{applied: map[string]bool{"001_pre": true}}
	op := &managerTestOperator{}
	mgr := &Manager{Store: store, Database: op}

	mgr.Register(Upgrade{
		ID:    "001_pre",
		Phase: PhasePreBootstrap,
		ShouldRun: func(_ operator.DatabaseOperator) (bool, error) {
			return true, nil
		},
		Apply: func(logf func(string), _ operator.DatabaseOperator) error {
			t.Fatal("metadata-applied upgrade should not execute")
			return nil
		},
	})

	if err := mgr.ApplyPhase(PhasePreBootstrap); err != nil {
		t.Fatalf("ApplyPhase(PreBootstrap) error = %v", err)
	}
	if len(store.records) != 0 {
		t.Fatalf("unexpected saved records: %+v", store.records)
	}
}

func TestManagerApplyPhaseSavesFailureRecordAndStops(t *testing.T) {
	store := &managerTestStore{applied: map[string]bool{}}
	op := &managerTestOperator{}
	mgr := &Manager{Store: store, Database: op}

	boom := errors.New("boom")
	mgr.Register(Upgrade{
		ID:    "001_pre",
		Phase: PhasePreBootstrap,
		ShouldRun: func(_ operator.DatabaseOperator) (bool, error) {
			return true, nil
		},
		Apply: func(logf func(string), _ operator.DatabaseOperator) error {
			logf("before boom")
			return boom
		},
	})

	err := mgr.ApplyPhase(PhasePreBootstrap)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if len(store.records) != 1 {
		t.Fatalf("records len = %d, want 1", len(store.records))
	}
	if store.records[0].ID != "001_pre" || store.records[0].Success {
		t.Fatalf("unexpected failure record: %+v", store.records[0])
	}
}

func TestManagerDetectsPhaseSignals(t *testing.T) {
	store := &managerTestStore{applied: map[string]bool{"002_pre_done": true}}
	op := &managerTestOperator{}
	mgr := &Manager{Store: store, Database: op}

	mgr.Register(Upgrade{
		ID:    "001_pre",
		Phase: PhasePreBootstrap,
		ShouldRun: func(_ operator.DatabaseOperator) (bool, error) {
			return false, nil
		},
	})
	mgr.Register(Upgrade{
		ID:    "002_pre_done",
		Phase: PhasePreBootstrap,
		ShouldRun: func(_ operator.DatabaseOperator) (bool, error) {
			return true, nil
		},
	})
	mgr.Register(Upgrade{
		ID:    "003_pre_need",
		Phase: PhasePreBootstrap,
		ShouldRun: func(_ operator.DatabaseOperator) (bool, error) {
			return true, nil
		},
	})
	mgr.Register(Upgrade{
		ID:    "004_post_need",
		Phase: PhasePostBootstrap,
		ShouldRun: func(_ operator.DatabaseOperator) (bool, error) {
			return true, nil
		},
	})

	hasSignals, matched, err := mgr.HasPendingPhaseSignals(PhasePreBootstrap)
	if err != nil {
		t.Fatalf("HasPendingPhaseSignals() error = %v", err)
	}
	if !hasSignals {
		t.Fatal("expected pending pre-bootstrap signals")
	}
	want := []string{"003_pre_need"}
	if !reflect.DeepEqual(matched, want) {
		t.Fatalf("matched = %v, want %v", matched, want)
	}
}

var _ operator.DatabaseOperator = (*managerTestOperator)(nil)
