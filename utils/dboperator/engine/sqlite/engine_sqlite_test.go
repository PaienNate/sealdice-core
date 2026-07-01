package sqlite_test

import (
	"testing"

	"gorm.io/gorm"

	"sealdice-core/utils/constant"
	"sealdice-core/utils/dboperator/engine/sqlite"
)

func TestGetDBByModeAndKeyUsesReadPoolForReadMode(t *testing.T) {
	readDB := &gorm.DB{}
	writeDB := &gorm.DB{}

	engine := sqlite.NewTestSQLiteEngine(
		map[string]*gorm.DB{"data": readDB, "logs": nil, "censor": nil},
		map[string]*gorm.DB{"data": writeDB, "logs": nil, "censor": nil},
	)

	got := engine.GetDataDB(constant.READ)
	if got != readDB {
		t.Fatalf("GetDataDB(READ) = %p, want %p", got, readDB)
	}
}

func TestGetDBByModeAndKeyUsesWritePoolForWriteMode(t *testing.T) {
	readDB := &gorm.DB{}
	writeDB := &gorm.DB{}

	engine := sqlite.NewTestSQLiteEngine(
		map[string]*gorm.DB{"data": readDB, "logs": nil, "censor": nil},
		map[string]*gorm.DB{"data": writeDB, "logs": nil, "censor": nil},
	)

	got := engine.GetDataDB(constant.WRITE)
	if got != writeDB {
		t.Fatalf("GetDataDB(WRITE) = %p, want %p", got, writeDB)
	}
}
