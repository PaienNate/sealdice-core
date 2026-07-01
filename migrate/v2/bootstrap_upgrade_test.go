package v2_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"go.etcd.io/bbolt"
	gormsqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	v2 "sealdice-core/migrate/v2"
	v120 "sealdice-core/migrate/v2/v120"
	"sealdice-core/model"
	"sealdice-core/utils/constant"
	"sealdice-core/utils/dboperator/bootstrap"
	operator "sealdice-core/utils/dboperator/engine"
)

type bootstrapUpgradeTestOperator struct {
	dataDB *gorm.DB
	logDB  *gorm.DB
}

func (o *bootstrapUpgradeTestOperator) Init(_ context.Context) error           { return nil }
func (o *bootstrapUpgradeTestOperator) BootstrapSchema() error                 { return nil }
func (o *bootstrapUpgradeTestOperator) Type() string                           { return constant.SQLITE }
func (o *bootstrapUpgradeTestOperator) DBCheck()                               {}
func (o *bootstrapUpgradeTestOperator) GetDataDB(_ constant.DBMode) *gorm.DB   { return o.dataDB }
func (o *bootstrapUpgradeTestOperator) GetLogDB(_ constant.DBMode) *gorm.DB    { return o.logDB }
func (o *bootstrapUpgradeTestOperator) GetCensorDB(_ constant.DBMode) *gorm.DB { return o.dataDB }
func (o *bootstrapUpgradeTestOperator) Close()                                 {}

func openBootstrapUpgradeSQLiteDB(t *testing.T, path string) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(gormsqlite.Open(path), &gorm.Config{
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

func writeLegacyV120Files(t *testing.T, root string) {
	t.Helper()

	dataDir := filepath.Join(root, "data", "default")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("mkdir data/default: %v", err)
	}

	serveYAML := `imSession:
  servicesAt:
    QQ-Group:1001:
      groupId: QQ-Group:1001
      groupName: Test Group
      players:
        QQ:2001:
          name: Tester
          userId: QQ:2001
          lastCommandTime: 123
          autoSetNameTemplate: default
          diceSideNum: 100
      diceIds:
        QQ:sealdice: true
`
	if err := os.WriteFile(filepath.Join(dataDir, "serve.yaml"), []byte(serveYAML), 0o644); err != nil {
		t.Fatalf("write serve.yaml: %v", err)
	}

	boltPath := filepath.Join(dataDir, "data.bdb")
	db, err := bbolt.Open(boltPath, 0o644, nil)
	if err != nil {
		t.Fatalf("open bbolt: %v", err)
	}
	defer func() {
		_ = db.Close()
	}()

	if err := db.Update(func(tx *bbolt.Tx) error {
		logs, err := tx.CreateBucketIfNotExists([]byte("logs"))
		if err != nil {
			return err
		}
		group, err := logs.CreateBucketIfNotExists([]byte("QQ-Group:1001"))
		if err != nil {
			return err
		}
		logBucket, err := group.CreateBucketIfNotExists([]byte("session-log"))
		if err != nil {
			return err
		}

		item := v120.LogOneItem{
			Nickname:  "Tester",
			IMUserID:  "QQ:2001",
			Time:      12345,
			Message:   "legacy message",
			IsDice:    false,
			CommandID: 1,
			RawMsgID:  "raw-1",
			UniformID: "QQ:2001",
		}
		rawItem, err := json.Marshal(item)
		if err != nil {
			return err
		}
		if putErr := logBucket.Put([]byte{0, 0, 0, 0, 0, 0, 0, 1}, rawItem); putErr != nil {
			return putErr
		}

		attrsUser, err := tx.CreateBucketIfNotExists([]byte("attrs_user"))
		if err != nil {
			return err
		}
		if err := attrsUser.Put([]byte("QQ:2001"), []byte(`{"$mstr":{"typeId":1,"value":"legacy","expiredTime":0}}`)); err != nil {
			return err
		}
		return nil
	}); err != nil {
		t.Fatalf("seed bbolt: %v", err)
	}
}

func TestV120MigrationStillRunsAfterBootstrap(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	writeLegacyV120Files(t, root)

	dataDB := openBootstrapUpgradeSQLiteDB(t, filepath.Join(root, "data", "default", "data.db"))
	logDB := openBootstrapUpgradeSQLiteDB(t, filepath.Join(root, "data", "default", "data-logs.db"))
	op := &bootstrapUpgradeTestOperator{dataDB: dataDB, logDB: logDB}

	if err := bootstrap.DataDB(constant.SQLITE, dataDB); err != nil {
		t.Fatalf("bootstrap.DataDB(): %v", err)
	}
	if err := bootstrap.LogDB(constant.SQLITE, logDB); err != nil {
		t.Fatalf("bootstrap.LogDB(): %v", err)
	}

	if err := v120.V120Migration.Apply(func(string) {}, op); err != nil {
		t.Fatalf("V120Migration.Apply(): %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, "data", "default", "serve.yaml.old")); err != nil {
		t.Fatalf("expected serve.yaml.old: %v", err)
	}

	var groupPlayerCount, groupInfoCount, attrsUserCount, logCount, itemCount int64
	if err := dataDB.Table("group_player_info").Count(&groupPlayerCount).Error; err != nil {
		t.Fatalf("count group_player_info: %v", err)
	}
	if err := dataDB.Table("group_info").Count(&groupInfoCount).Error; err != nil {
		t.Fatalf("count group_info: %v", err)
	}
	if err := dataDB.Table("attrs_user").Count(&attrsUserCount).Error; err != nil {
		t.Fatalf("count attrs_user: %v", err)
	}
	if err := logDB.Table("logs").Count(&logCount).Error; err != nil {
		t.Fatalf("count logs: %v", err)
	}
	if err := logDB.Table("log_items").Count(&itemCount).Error; err != nil {
		t.Fatalf("count log_items: %v", err)
	}

	if groupPlayerCount != 1 {
		t.Fatalf("group_player_info count = %d, want 1", groupPlayerCount)
	}
	if groupInfoCount != 1 {
		t.Fatalf("group_info count = %d, want 1", groupInfoCount)
	}
	if attrsUserCount != 1 {
		t.Fatalf("attrs_user count = %d, want 1", attrsUserCount)
	}
	if logCount != 1 {
		t.Fatalf("logs count = %d, want 1", logCount)
	}
	if itemCount != 1 {
		t.Fatalf("log_items count = %d, want 1", itemCount)
	}

	var logInfo model.LogInfo
	if err := logDB.First(&logInfo).Error; err != nil {
		t.Fatalf("load migrated log info: %v", err)
	}
	if logInfo.GroupID != "QQ-Group:1001" || logInfo.Name != "session-log" {
		t.Fatalf("unexpected migrated log info: %+v", logInfo)
	}
}

func TestHasPreBootstrapPendingSignalsDetectsV120LegacyArtifacts(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	writeLegacyV120Files(t, root)

	dataDB := openBootstrapUpgradeSQLiteDB(t, filepath.Join(root, "data", "default", "data.db"))
	logDB := openBootstrapUpgradeSQLiteDB(t, filepath.Join(root, "data", "default", "data-logs.db"))
	op := &bootstrapUpgradeTestOperator{dataDB: dataDB, logDB: logDB}

	hasPending, matched, err := v2.HasPreBootstrapPendingSignals(op)
	if err != nil {
		t.Fatalf("HasPreBootstrapPendingSignals() error = %v", err)
	}
	if !hasPending {
		t.Fatal("expected pending pre-bootstrap signals")
	}
	if len(matched) == 0 || matched[0] != v120.V120Migration.ID {
		t.Fatalf("unexpected matched upgrades: %v", matched)
	}
}

func TestInitUpgraderBuildsBusinessSchemaForFreshDatabase(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	dataDir := filepath.Join(root, "data", "default")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("mkdir data/default: %v", err)
	}

	dataDB := openBootstrapUpgradeSQLiteDB(t, filepath.Join(dataDir, "data.db"))
	logDB := openBootstrapUpgradeSQLiteDB(t, filepath.Join(dataDir, "data-logs.db"))
	op := &bootstrapUpgradeTestOperator{dataDB: dataDB, logDB: logDB}

	if err := bootstrap.DataDB(constant.SQLITE, dataDB); err != nil {
		t.Fatalf("bootstrap.DataDB(): %v", err)
	}
	if err := bootstrap.LogDB(constant.SQLITE, logDB); err != nil {
		t.Fatalf("bootstrap.LogDB(): %v", err)
	}

	if err := v2.InitUpgrader(op); err != nil {
		t.Fatalf("InitUpgrader() error = %v", err)
	}

	for _, table := range []string{
		"group_player_info",
		"group_info",
		"ban_info",
		"attrs",
		"logs",
		"log_items",
	} {
		hasTable := dataDB.Migrator().HasTable(table) || logDB.Migrator().HasTable(table)
		if !hasTable {
			t.Fatalf("expected table %s after fresh upgrade pipeline", table)
		}
	}
}

var _ operator.DatabaseOperator = (*bootstrapUpgradeTestOperator)(nil)
