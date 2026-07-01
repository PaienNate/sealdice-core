package store

import (
	"path/filepath"
	"testing"
	"time"

	gormsqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	upgrade "sealdice-core/utils/upgrader"
)

func openStoreTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "store.db")
	db, err := gorm.Open(gormsqlite.Open(dbPath), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB(): %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})
	return db
}

func TestDBStoreTreatsFailureRecordAsApplied(t *testing.T) {
	db := openStoreTestDB(t)
	store, err := NewDBStore(db, "")
	if err != nil {
		t.Fatalf("NewDBStore() error = %v", err)
	}

	rec := upgrade.UpgradeRecord{
		ID:        "001_failure",
		Timestamp: time.Now(),
		Success:   false,
		Message:   "failed before",
	}
	if err := store.SaveRecord(rec); err != nil {
		t.Fatalf("SaveRecord() error = %v", err)
	}

	applied, err := store.IsApplied("001_failure")
	if err != nil {
		t.Fatalf("IsApplied() error = %v", err)
	}
	if !applied {
		t.Fatal("expected failure record to count as applied")
	}
}

func TestDBStoreLoadRecordsPreservesSuccessFlag(t *testing.T) {
	db := openStoreTestDB(t)
	store, err := NewDBStore(db, "")
	if err != nil {
		t.Fatalf("NewDBStore() error = %v", err)
	}

	failed := upgrade.UpgradeRecord{
		ID:        "001_failure",
		Timestamp: time.Now(),
		Success:   false,
		Message:   "failed before",
	}
	succeeded := upgrade.UpgradeRecord{
		ID:        "002_success",
		Timestamp: time.Now(),
		Success:   true,
		Message:   "ok",
	}
	if err := store.SaveRecord(failed); err != nil {
		t.Fatalf("SaveRecord(failed) error = %v", err)
	}
	if err := store.SaveRecord(succeeded); err != nil {
		t.Fatalf("SaveRecord(succeeded) error = %v", err)
	}

	records, err := store.LoadRecords()
	if err != nil {
		t.Fatalf("LoadRecords() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("len(records) = %d, want 2", len(records))
	}
	if records[0].ID != failed.ID || records[0].Success {
		t.Fatalf("unexpected failed record: %+v", records[0])
	}
	if records[1].ID != succeeded.ID || !records[1].Success {
		t.Fatalf("unexpected success record: %+v", records[1])
	}
}

func TestDBStoreImportsLegacyJSONRecords(t *testing.T) {
	db := openStoreTestDB(t)
	jsonPath := filepath.Join(t.TempDir(), "upgrade_metadata.json")

	legacy := NewJSONStore(jsonPath)
	rec := upgrade.UpgradeRecord{
		ID:        "001_legacy",
		Timestamp: time.Now(),
		Success:   true,
		Message:   "legacy ok",
	}
	if err := legacy.SaveRecord(rec); err != nil {
		t.Fatalf("legacy.SaveRecord() error = %v", err)
	}

	store, err := NewDBStore(db, jsonPath)
	if err != nil {
		t.Fatalf("NewDBStore() error = %v", err)
	}

	applied, err := store.IsApplied("001_legacy")
	if err != nil {
		t.Fatalf("IsApplied() error = %v", err)
	}
	if !applied {
		t.Fatal("expected imported legacy record to count as applied")
	}

	records, err := store.LoadRecords()
	if err != nil {
		t.Fatalf("LoadRecords() error = %v", err)
	}
	if len(records) != 1 || records[0].ID != "001_legacy" {
		t.Fatalf("unexpected imported records: %+v", records)
	}
}
