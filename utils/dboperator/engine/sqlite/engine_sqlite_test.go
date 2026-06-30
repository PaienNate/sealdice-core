package sqlite

import (
	"testing"

	"gorm.io/gorm"

	"sealdice-core/utils/constant"
)

func TestGetDBByModeAndKeyUsesReadPoolForReadMode(t *testing.T) {
	readDB := &gorm.DB{}
	writeDB := &gorm.DB{}

	engine := &SQLiteEngine{
		readList: map[dbName]*gorm.DB{
			DataDBKey: readDB,
		},
		writeList: map[dbName]*gorm.DB{
			DataDBKey: writeDB,
		},
	}

	got := engine.GetDataDB(constant.READ)
	if got != readDB {
		t.Fatalf("GetDataDB(READ) = %p, want %p", got, readDB)
	}
}

func TestGetDBByModeAndKeyUsesWritePoolForWriteMode(t *testing.T) {
	readDB := &gorm.DB{}
	writeDB := &gorm.DB{}

	engine := &SQLiteEngine{
		readList: map[dbName]*gorm.DB{
			DataDBKey: readDB,
		},
		writeList: map[dbName]*gorm.DB{
			DataDBKey: writeDB,
		},
	}

	got := engine.GetDataDB(constant.WRITE)
	if got != writeDB {
		t.Fatalf("GetDataDB(WRITE) = %p, want %p", got, writeDB)
	}
}
