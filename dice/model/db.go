package model

import (
	"fmt"
	"path/filepath"

	"gorm.io/gorm"

	log "sealdice-core/utils/kratos"
)

func DBCheck(dataDir string) {
	checkDB := func(db *gorm.DB) bool {
		rows, err := db.Exec("PRAGMA integrity_check").Rows() //nolint:execinquery
		if err != nil {
			return false
		}
		defer rows.Close()
		var ok bool
		for rows.Next() {
			var s string
			if errR := rows.Scan(&s); errR != nil {
				ok = false
				break
			}
			fmt.Println(s)
			if s == "ok" {
				ok = true
			}
		}

		if errR := rows.Err(); errR != nil {
			ok = false
		}
		return ok
	}

	var ok1, ok2, ok3 bool
	var dataDB *gorm.DB
	var logsDB *gorm.DB
	var censorDB *gorm.DB
	var err error

	dbDataPath, _ := filepath.Abs(filepath.Join(dataDir, "data.db"))
	dataDB, err = _SQLiteDBInit(dbDataPath, false)
	if err != nil {
		fmt.Println("数据库 data.db 无法打开")
	} else {
		ok1 = checkDB(dataDB)
		db, _ := dataDB.DB()
		// 关闭
		db.Close()
	}

	dbDataLogsPath, _ := filepath.Abs(filepath.Join(dataDir, "data-logs.db"))
	logsDB, err = _SQLiteDBInit(dbDataLogsPath, false)
	if err != nil {
		fmt.Println("数据库 data-logs.db 无法打开")
	} else {
		ok2 = checkDB(logsDB)
		db, _ := logsDB.DB()
		// 关闭db
		db.Close()
	}

	dbDataCensorPath, _ := filepath.Abs(filepath.Join(dataDir, "data-censor.db"))
	censorDB, err = _SQLiteDBInit(dbDataCensorPath, false)
	if err != nil {
		fmt.Println("数据库 data-censor.db 无法打开")
	} else {
		ok3 = checkDB(censorDB)
		db, _ := censorDB.DB()
		// 关闭db
		db.Close()
	}

	fmt.Println("数据库检查结果：")
	fmt.Println("data.db:", ok1)
	fmt.Println("data-logs.db:", ok2)
	fmt.Println("data-censor.db:", ok3)
}

func SQLiteDBInit(dataDir string) (dataDB *gorm.DB, logsDB *gorm.DB, err error) {
	dbDataPath, _ := filepath.Abs(filepath.Join(dataDir, "data.db"))
	dataDB, err = _SQLiteDBInit(dbDataPath, true)
	if err != nil {
		return nil, nil, err
	}
	// data建表
	err = dataDB.AutoMigrate(
		&GroupPlayerInfoBase{},
		&GroupInfo{},
		&BanInfo{},
		&EndpointInfo{},
		// ATTRS_NEW
		&AttributesItemModel{},
	)
	if err != nil {
		return nil, nil, err
	}
	log.Info("数据库初始化中，请稍候...")
	// data vacuum
	err = InitVacuum(dataDB)
	if err != nil {
		return nil, nil, err
	}
	logsDB, err = LogDBInit(dataDir)
	log.Info("数据库初始化完毕")
	return
}

// LogDBInit SQLITE初始化
func LogDBInit(dataDir string) (logsDB *gorm.DB, err error) {
	dbDataLogsPath, _ := filepath.Abs(filepath.Join(dataDir, "data-logs.db"))
	logsDB, err = _SQLiteDBInit(dbDataLogsPath, true)
	if err != nil {
		return nil, err
	}
	// logs建表
	if err := logsDB.AutoMigrate(&LogInfo{}, &LogOneItem{}); err != nil {
		return nil, err
	}
	err = InitVacuum(logsDB)
	if err != nil {
		return nil, err
	}
	return logsDB, nil
}

func SQLiteCensorDBInit(dataDir string) (censorDB *gorm.DB, err error) {
	path, err := filepath.Abs(filepath.Join(dataDir, "data-censor.db"))
	if err != nil {
		return
	}
	censorDB, err = _SQLiteDBInit(path, true)
	if err != nil {
		return
	}
	// 创建基本的表结构，并通过标签定义索引
	if err = censorDB.AutoMigrate(&CensorLog{}); err != nil {
		return nil, err
	}
	err = InitVacuum(censorDB)
	if err != nil {
		return nil, err
	}
	return censorDB, nil
}

func InitVacuum(db *gorm.DB) error {
	// 检查数据库驱动是否为 SQLite
	if db.Dialector.Name() != "sqlite" {
		log.Debug("非SQLITE，跳过运行VACUUM")
		return nil
	}

	// 使用 GORM 执行 vacuum 操作，并将数据库保存到指定路径
	err := db.Exec("VACUUM").Error
	return err // 返回错误
}
