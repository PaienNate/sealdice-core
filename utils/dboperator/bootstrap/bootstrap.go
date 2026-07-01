package bootstrap

import (
	"strings"

	"gorm.io/gorm"

	"sealdice-core/model"
)

func DataDB(_ string, writeDB *gorm.DB) error { return EnsurePatchLogTable(writeDB) }

func LogDB(_ string, writeDB *gorm.DB) error { return EnsurePatchLogTable(writeDB) }

func CensorDB(writeDB *gorm.DB) error { return EnsurePatchLogTable(writeDB) }

func EnsurePatchLogTable(db *gorm.DB) error {
	session := db.Session(&gorm.Session{})
	err := session.Migrator().CreateTable(&model.PatchLog{})
	if err != nil && strings.Contains(err.Error(), "already exists") {
		return nil
	}
	return err
}
