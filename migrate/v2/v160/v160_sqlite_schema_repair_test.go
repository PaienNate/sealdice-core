package v160

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	gormsqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"sealdice-core/utils/constant"
	operator "sealdice-core/utils/dboperator/engine"
)

type sqliteSchemaRepairTestOperator struct {
	dataDB *gorm.DB
	logDB  *gorm.DB
}

func (o *sqliteSchemaRepairTestOperator) Init(_ context.Context) error           { return nil }
func (o *sqliteSchemaRepairTestOperator) BootstrapSchema() error                 { return nil }
func (o *sqliteSchemaRepairTestOperator) Type() string                           { return constant.SQLITE }
func (o *sqliteSchemaRepairTestOperator) DBCheck()                               {}
func (o *sqliteSchemaRepairTestOperator) GetDataDB(_ constant.DBMode) *gorm.DB   { return o.dataDB }
func (o *sqliteSchemaRepairTestOperator) GetLogDB(_ constant.DBMode) *gorm.DB    { return o.logDB }
func (o *sqliteSchemaRepairTestOperator) GetCensorDB(_ constant.DBMode) *gorm.DB { return o.dataDB }
func (o *sqliteSchemaRepairTestOperator) Close()                                 {}

func openSQLiteSchemaRepairDB(t *testing.T, filename string) *gorm.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), filename)
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

func loadSQLiteTableSQL(t *testing.T, db *gorm.DB, table string) string {
	t.Helper()

	var row struct {
		SQL string `gorm:"column:sql"`
	}
	if err := db.Raw("SELECT sql FROM sqlite_master WHERE type = 'table' AND name = ?", table).Scan(&row).Error; err != nil {
		t.Fatalf("load sqlite_master for %s: %v", table, err)
	}
	return row.SQL
}

func hasSQLiteIndex(t *testing.T, db *gorm.DB, table string, index string) bool {
	t.Helper()

	var count int64
	if err := db.Raw(
		"SELECT COUNT(1) FROM sqlite_master WHERE type = 'index' AND tbl_name = ? AND name = ?",
		table,
		index,
	).Scan(&count).Error; err != nil {
		t.Fatalf("check index %s on %s: %v", index, table, err)
	}
	return count > 0
}

func TestV160SQLiteSchemaRepairMigratesLegacyTables(t *testing.T) {
	dataDB := openSQLiteSchemaRepairDB(t, "data.db")
	logDB := openSQLiteSchemaRepairDB(t, "logs.db")

	if err := dataDB.Exec(`
CREATE TABLE attrs (
	id TEXT PRIMARY KEY,
	updated_at INTEGER,
	data BLOB
)`).Error; err != nil {
		t.Fatalf("create legacy attrs: %v", err)
	}
	if err := dataDB.Exec(`INSERT INTO attrs (id, updated_at, data) VALUES ('user-1', 123, X'7B7D')`).Error; err != nil {
		t.Fatalf("seed legacy attrs: %v", err)
	}

	if err := logDB.Exec(`
CREATE TABLE logs (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT,
	group_id TEXT,
	created_at INTEGER,
	updated_at INTEGER
)`).Error; err != nil {
		t.Fatalf("create legacy logs: %v", err)
	}
	if err := logDB.Exec(`
CREATE TABLE log_items (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	log_id INTEGER,
	group_id TEXT,
	message INTEGER
)`).Error; err != nil {
		t.Fatalf("create legacy log_items: %v", err)
	}
	if err := logDB.Exec(`INSERT INTO logs (id, name, group_id, created_at, updated_at) VALUES (1, 'legacy', 'group', 1, 2)`).Error; err != nil {
		t.Fatalf("seed legacy logs: %v", err)
	}
	if err := logDB.Exec(`INSERT INTO log_items (id, log_id, group_id, message) VALUES (1, 1, 'group', 42)`).Error; err != nil {
		t.Fatalf("seed legacy log_items: %v", err)
	}

	op := &sqliteSchemaRepairTestOperator{dataDB: dataDB, logDB: logDB}
	if err := V160SQLiteSchemaRepairMigrate(op, func(string) {}); err != nil {
		t.Fatalf("V160SQLiteSchemaRepairMigrate() error = %v", err)
	}

	if got := loadSQLiteTableSQL(t, dataDB, "attrs"); got == "" || !containsAll(got, "binding_sheet_id", "owner_id", "created_at") {
		t.Fatalf("attrs schema not repaired: %s", got)
	}
	if got := loadSQLiteTableSQL(t, logDB, "logs"); got == "" || !containsAll(got, "size", "extra", "upload_url", "upload_time") {
		t.Fatalf("logs schema not repaired: %s", got)
	}
	if got := loadSQLiteTableSQL(t, logDB, "log_items"); got == "" || !containsAll(got, "nickname", "raw_msg_id", "parent_id") {
		t.Fatalf("log_items schema not repaired: %s", got)
	}

	if !hasSQLiteIndex(t, logDB, "log_items", "idx_log_delete_by_id") {
		t.Fatal("expected idx_log_delete_by_id after repair")
	}

	var attrsCount, logsCount, itemsCount int64
	if err := dataDB.Table("attrs").Count(&attrsCount).Error; err != nil {
		t.Fatalf("count attrs: %v", err)
	}
	if err := logDB.Table("logs").Count(&logsCount).Error; err != nil {
		t.Fatalf("count logs: %v", err)
	}
	if err := logDB.Table("log_items").Count(&itemsCount).Error; err != nil {
		t.Fatalf("count log_items: %v", err)
	}
	if attrsCount != 1 || logsCount != 1 || itemsCount != 1 {
		t.Fatalf("unexpected row counts attrs=%d logs=%d log_items=%d", attrsCount, logsCount, itemsCount)
	}
}

func containsAll(s string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(strings.ToLower(s), strings.ToLower(part)) {
			return false
		}
	}
	return true
}

var _ operator.DatabaseOperator = (*sqliteSchemaRepairTestOperator)(nil)
