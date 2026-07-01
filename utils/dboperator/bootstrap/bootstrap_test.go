package bootstrap

import (
	"path/filepath"
	"testing"

	gormsqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func openBootstrapSQLiteTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "bootstrap.db")
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

func loadTableSQL(t *testing.T, db *gorm.DB, table string) string {
	t.Helper()

	var row struct {
		SQL string `gorm:"column:sql"`
	}
	if err := db.Raw(
		"SELECT sql FROM sqlite_master WHERE type = 'table' AND name = ?",
		table,
	).Scan(&row).Error; err != nil {
		t.Fatalf("load sqlite_master for %s: %v", table, err)
	}
	return row.SQL
}

func countIndexes(t *testing.T, db *gorm.DB, table string) int64 {
	t.Helper()

	var count int64
	if err := db.Raw(
		"SELECT COUNT(1) AS count FROM sqlite_master WHERE type = 'index' AND tbl_name = ?",
		table,
	).Scan(&count).Error; err != nil {
		t.Fatalf("count indexes for %s: %v", table, err)
	}
	return count
}

func TestPackageBuildsWithoutHookModels(t *testing.T) {
	// The package compiles only if bootstrap no longer depends on the removed
	// HookMySQL models.
}

func TestSQLiteDataDBCreatesOnlyPatchLogTable(t *testing.T) {
	db := openBootstrapSQLiteTestDB(t)

	if err := DataDB("sqlite", db); err != nil {
		t.Fatalf("DataDB(sqlite) error = %v", err)
	}

	if !db.Migrator().HasTable("sys_patch_log") {
		t.Fatal("expected sys_patch_log to exist")
	}

	for _, table := range []string{
		"group_player_info",
		"group_info",
		"ban_info",
		"endpoint_info",
		"attrs",
		"logs",
		"log_items",
		"censor_log",
	} {
		if db.Migrator().HasTable(table) {
			t.Fatalf("did not expect business table %s to exist", table)
		}
	}
}

func TestSQLiteLogDBCreatesOnlyPatchLogTable(t *testing.T) {
	db := openBootstrapSQLiteTestDB(t)

	if err := LogDB("sqlite", db); err != nil {
		t.Fatalf("LogDB(sqlite) error = %v", err)
	}

	if !db.Migrator().HasTable("sys_patch_log") {
		t.Fatal("expected sys_patch_log to exist")
	}
	for _, table := range []string{"logs", "log_items", "attrs"} {
		if db.Migrator().HasTable(table) {
			t.Fatalf("did not expect business table %s to exist", table)
		}
	}
}
