package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"sealdice-core/utils/constant"
)

func TestBootstrapSchemaCreatesPatchLogOnly(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("DATADIR", dataDir)

	engine := &SQLiteEngine{}
	if err := engine.Init(context.Background()); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	t.Cleanup(engine.Close)

	if err := engine.BootstrapSchema(); err != nil {
		t.Fatalf("BootstrapSchema() error = %v", err)
	}

	assertTable := func(dbPath string) {
		if _, err := os.Stat(dbPath); err != nil {
			t.Fatalf("expected db file %s: %v", dbPath, err)
		}
	}

	assertTable(filepath.Join(dataDir, "data.db"))
	assertTable(filepath.Join(dataDir, "data-logs.db"))
	assertTable(filepath.Join(dataDir, "data-censor.db"))

	checkHasTable := func(queryDB func() bool, table string) {
		if !queryDB() {
			t.Fatalf("expected table %s to exist", table)
		}
	}

	checkHasTable(func() bool { return engine.GetDataDB(constant.WRITE).Migrator().HasTable("sys_patch_log") }, "sys_patch_log")
	checkHasTable(func() bool { return engine.GetLogDB(constant.WRITE).Migrator().HasTable("sys_patch_log") }, "sys_patch_log")
	checkHasTable(func() bool { return engine.GetCensorDB(constant.WRITE).Migrator().HasTable("sys_patch_log") }, "sys_patch_log")

	for _, table := range []string{"attrs", "group_info", "logs", "log_items", "censor_log"} {
		if engine.GetDataDB(constant.WRITE).Migrator().HasTable(table) ||
			engine.GetLogDB(constant.WRITE).Migrator().HasTable(table) ||
			engine.GetCensorDB(constant.WRITE).Migrator().HasTable(table) {
			t.Fatalf("did not expect business table %s to exist", table)
		}
	}
}
