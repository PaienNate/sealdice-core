package v150_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	gormsqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"sealdice-core/dice"
	"sealdice-core/dice/service"
	v150 "sealdice-core/migrate/v2/v150"
	"sealdice-core/model"
	"sealdice-core/utils/constant"
	"sealdice-core/utils/dboperator/bootstrap"
	operator "sealdice-core/utils/dboperator/engine"
	upgrade "sealdice-core/utils/upgrader"
)

type v150BootstrapTestOperator struct {
	dataDB *gorm.DB
	logDB  *gorm.DB
}

func (o *v150BootstrapTestOperator) Init(_ context.Context) error           { return nil }
func (o *v150BootstrapTestOperator) BootstrapSchema() error                 { return nil }
func (o *v150BootstrapTestOperator) Type() string                           { return constant.SQLITE }
func (o *v150BootstrapTestOperator) DBCheck()                               {}
func (o *v150BootstrapTestOperator) GetDataDB(_ constant.DBMode) *gorm.DB   { return o.dataDB }
func (o *v150BootstrapTestOperator) GetLogDB(_ constant.DBMode) *gorm.DB    { return o.logDB }
func (o *v150BootstrapTestOperator) GetCensorDB(_ constant.DBMode) *gorm.DB { return o.dataDB }
func (o *v150BootstrapTestOperator) Close()                                 {}

func openV150SQLiteTestDB(t *testing.T, filename string) *gorm.DB {
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

func mustJSONValueMap(t *testing.T, values map[string]*dice.VMValue) []byte {
	t.Helper()

	raw, err := json.Marshal(values)
	if err != nil {
		t.Fatalf("marshal legacy attrs: %v", err)
	}
	return raw
}

func mustJSONStringValueMap(t *testing.T, values map[string]*dice.VMValue) string {
	t.Helper()
	return string(mustJSONValueMap(t, values))
}

func TestV150AttrsMigrateAfterBootstrapKeepsLegacyUpgradePath(t *testing.T) {
	dataDB := openV150SQLiteTestDB(t, "data.db")
	logDB := openV150SQLiteTestDB(t, "logs.db")

	if err := dataDB.Exec(`
CREATE TABLE attrs_user (
	id TEXT PRIMARY KEY,
	updated_at INTEGER,
	data BLOB
)`).Error; err != nil {
		t.Fatalf("create attrs_user: %v", err)
	}
	if err := dataDB.Exec(`
CREATE TABLE attrs_group_user (
	id TEXT PRIMARY KEY,
	updated_at INTEGER,
	data BLOB
)`).Error; err != nil {
		t.Fatalf("create attrs_group_user: %v", err)
	}
	if err := dataDB.Exec(`
CREATE TABLE attrs_group (
	id TEXT PRIMARY KEY,
	updated_at INTEGER,
	data BLOB
)`).Error; err != nil {
		t.Fatalf("create attrs_group: %v", err)
	}

	userData := mustJSONValueMap(t, map[string]*dice.VMValue{
		"$:group-bind:QQ-Group:1001": dice.VMValueNew(dice.VMTypeString, "Hero"),
		"$ch:Hero": dice.VMValueNew(dice.VMTypeString, mustJSONStringValueMap(t, map[string]*dice.VMValue{
			"hp": dice.VMValueNew(dice.VMTypeInt64, int64(12)),
		})),
		"$mstr": dice.VMValueNew(dice.VMTypeString, "value"),
	})
	groupUserData := mustJSONValueMap(t, map[string]*dice.VMValue{
		"$:cardBindMark": dice.VMValueNew(dice.VMTypeString, "Hero"),
		"hp":             dice.VMValueNew(dice.VMTypeInt64, int64(10)),
	})
	groupData := mustJSONValueMap(t, map[string]*dice.VMValue{
		"flag": dice.VMValueNew(dice.VMTypeString, "group"),
	})

	if err := dataDB.Exec(`INSERT INTO attrs_user (id, updated_at, data) VALUES (?, ?, ?)`, "QQ:2001", 101, userData).Error; err != nil {
		t.Fatalf("seed attrs_user: %v", err)
	}
	if err := dataDB.Exec(`INSERT INTO attrs_group_user (id, updated_at, data) VALUES (?, ?, ?)`, "QQ-Group:1001-QQ:2001", 102, groupUserData).Error; err != nil {
		t.Fatalf("seed attrs_group_user: %v", err)
	}
	if err := dataDB.Exec(`INSERT INTO attrs_group (id, updated_at, data) VALUES (?, ?, ?)`, "QQ-Group:1001", 103, groupData).Error; err != nil {
		t.Fatalf("seed attrs_group: %v", err)
	}

	if err := bootstrap.DataDB(constant.SQLITE, dataDB); err != nil {
		t.Fatalf("bootstrap.DataDB() error = %v", err)
	}
	if !dataDB.Migrator().HasTable("sys_patch_log") {
		t.Fatal("expected sys_patch_log after bootstrap")
	}
	if dataDB.Migrator().HasTable("attrs") {
		t.Fatal("did not expect attrs table after bootstrap")
	}

	op := &v150BootstrapTestOperator{dataDB: dataDB, logDB: logDB}
	if err := v150.V150AttrsMigrate(op, func(string) {}); err != nil {
		t.Fatalf("V150AttrsMigrate() error = %v", err)
	}

	for _, table := range []string{"attrs_user", "attrs_group_user", "attrs_group"} {
		if dataDB.Migrator().HasTable(table) {
			t.Fatalf("expected legacy table %s to be dropped", table)
		}
	}

	var items []model.AttributesItemModel
	if err := dataDB.Order("id ASC").Find(&items).Error; err != nil {
		t.Fatalf("load attrs: %v", err)
	}
	if len(items) < 3 {
		t.Fatalf("expected migrated attrs rows, got %d", len(items))
	}

	var countsByType = map[string]int{}
	for _, item := range items {
		countsByType[item.AttrsType]++
	}
	if countsByType[service.AttrsTypeUser] != 1 {
		t.Fatalf("expected 1 user attrs row, got %d", countsByType[service.AttrsTypeUser])
	}
	if countsByType[service.AttrsTypeGroupUser] != 1 {
		t.Fatalf("expected 1 group_user attrs row, got %d", countsByType[service.AttrsTypeGroupUser])
	}
	if countsByType[service.AttrsTypeGroup] != 1 {
		t.Fatalf("expected 1 group attrs row, got %d", countsByType[service.AttrsTypeGroup])
	}
	if countsByType[service.AttrsTypeCharacter] != 1 {
		t.Fatalf("expected 1 character attrs row, got %d", countsByType[service.AttrsTypeCharacter])
	}
}

func TestV150UpgradeAttrsShouldRunDetectsLegacyTables(t *testing.T) {
	dataDB := openV150SQLiteTestDB(t, "data.db")
	logDB := openV150SQLiteTestDB(t, "logs.db")

	if err := dataDB.Exec(`
CREATE TABLE attrs_user (
	id TEXT PRIMARY KEY,
	updated_at INTEGER,
	data BLOB
)`).Error; err != nil {
		t.Fatalf("create attrs_user: %v", err)
	}

	op := &v150BootstrapTestOperator{dataDB: dataDB, logDB: logDB}
	shouldRun, err := v150.V150UpgradeAttrsMigration.ShouldRun(op)
	if err != nil {
		t.Fatalf("ShouldRun() error = %v", err)
	}
	if !shouldRun {
		t.Fatal("expected legacy attrs tables to trigger pre-bootstrap migration")
	}
	if v150.V150UpgradeAttrsMigration.Phase != upgrade.PhasePreBootstrap {
		t.Fatalf("phase = %q, want %q", v150.V150UpgradeAttrsMigration.Phase, upgrade.PhasePreBootstrap)
	}
}

func TestV150AttrsMigrateCreatesAttrsTableForFreshSchemaPath(t *testing.T) {
	dataDB := openV150SQLiteTestDB(t, "data.db")
	logDB := openV150SQLiteTestDB(t, "logs.db")

	op := &v150BootstrapTestOperator{dataDB: dataDB, logDB: logDB}
	if err := v150.V150AttrsMigrate(op, func(string) {}); err != nil {
		t.Fatalf("V150AttrsMigrate() error = %v", err)
	}

	if !dataDB.Migrator().HasTable("attrs") {
		t.Fatal("expected attrs table after V150 migration on fresh schema path")
	}
}

var _ operator.DatabaseOperator = (*v150BootstrapTestOperator)(nil)
