package dao

import (
	"strings"
	"sync"

	"sealdice-core/utils/database/backend"
	log "sealdice-core/utils/kratos"
	"sealdice-core/utils/spinner"
)

// DBVacuum 整理数据库
func DBVacuum() {
	done := make(chan interface{}, 1)
	log.Info("开始进行数据库整理")

	go spinner.WithLines(done, 3, 10)
	defer func() {
		done <- struct{}{}
	}()

	wg := sync.WaitGroup{}
	wg.Add(3)

	vacuum := func(path string, wg *sync.WaitGroup) {
		defer wg.Done()
		// 使用 GORM 初始化数据库
		vacuumDB, err := backend.SQLiteDBInit(path, true)
		// 数据库类型不是 SQLite 直接返回
		if !strings.Contains(vacuumDB.Dialector.Name(), "sqlite") {
			return
		}
		defer func() {
			rawdb, err2 := vacuumDB.DB()
			if err2 != nil {
				return
			}
			err = rawdb.Close()
			if err != nil {
				return
			}
		}()
		if err != nil {
			log.Errorf("清理 %q 时出现错误：%v", path, err)
			return
		}
		err = vacuumDB.Exec("VACUUM;").Error
		if err != nil {
			log.Errorf("清理 %q 时出现错误：%v", path, err)
		}
	}

	go vacuum("./data/default/data.db", &wg)
	go vacuum("./data/default/data-logs.db", &wg)
	go vacuum("./data/default/data-censor.db", &wg)

	wg.Wait()

	log.Info("数据库整理完成")
}
