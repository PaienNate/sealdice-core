package engine

import (
	"context"
	"errors"
	"fmt"
	"os"

	"gorm.io/gorm"

	"sealdice-core/model"
	"sealdice-core/utils/consts"
	"sealdice-core/utils/database/backend"
	"sealdice-core/utils/database/cache"
	log "sealdice-core/utils/kratos"
)

type PGSQLEngine struct {
	DSN      string
	DB       *gorm.DB
	dataDB   *gorm.DB
	logsDB   *gorm.DB
	censorDB *gorm.DB
	ctx      context.Context
	// 其他引擎不需要读写分离
}

func (s *PGSQLEngine) Close() {
	db, err := s.DB.DB()
	if err != nil {
		log.Errorf("failed to close db: %v", err)
		return
	}
	err = db.Close()
	if err != nil {
		return
	}
}

func (s *PGSQLEngine) GetDataDB(_ consts.DBMode) *gorm.DB {
	return s.dataDB
}

func (s *PGSQLEngine) GetLogDB(_ consts.DBMode) *gorm.DB {
	return s.logsDB
}

func (s *PGSQLEngine) GetCensorDB(_ consts.DBMode) *gorm.DB {
	return s.censorDB
}

func (s *PGSQLEngine) Init(ctx context.Context) error {
	if ctx == nil {
		return errors.New("ctx is missing")
	}
	s.ctx = ctx
	s.DSN = os.Getenv("DB_DSN")
	if s.DSN == "" {
		return errors.New("DB_DSN is missing")
	}
	var err error
	s.DB, err = backend.PostgresDBInit(s.DSN)
	if err != nil {
		return err
	}
	// 获取dataDB,logsDB和censorDB并赋值
	s.dataDB, err = s.dataDBInit()
	if err != nil {
		return err
	}
	s.logsDB, err = s.logDBInit()
	if err != nil {
		return err
	}
	s.censorDB, err = s.censorDBInit()
	if err != nil {
		return err
	}
	return nil
}

// DBCheck DB检查
func (s *PGSQLEngine) DBCheck() {
	fmt.Fprintln(os.Stdout, "PostGRESQL 海豹不提供检查，请自行检查数据库！")
}

// DataDBInit 初始化
func (s *PGSQLEngine) dataDBInit() (*gorm.DB, error) {
	// data建表
	dataContext := context.WithValue(s.ctx, cache.CacheKey, cache.DataDBCacheKey)
	dataDB := s.DB.WithContext(dataContext)
	err := dataDB.AutoMigrate(
		&model.GroupPlayerInfoBase{},
		&model.GroupInfo{},
		&model.BanInfo{},
		&model.EndpointInfo{},
		&model.AttributesItemModel{},
		// 150 新增：更新版本记录功能
		&model.UpgradeEntry{},
	)
	if err != nil {
		return nil, err
	}
	return dataDB, nil
}

func (s *PGSQLEngine) logDBInit() (*gorm.DB, error) {
	// logs建表
	logsContext := context.WithValue(s.ctx, cache.CacheKey, cache.LogsDBCacheKey)
	logDB := s.DB.WithContext(logsContext)
	if err := logDB.AutoMigrate(&model.LogInfo{}, &model.LogOneItem{}); err != nil {
		return nil, err
	}
	return logDB, nil
}

func (s *PGSQLEngine) censorDBInit() (*gorm.DB, error) {
	censorContext := context.WithValue(s.ctx, cache.CacheKey, cache.CensorsDBCacheKey)
	censorDB := s.DB.WithContext(censorContext)
	// 创建基本的表结构，并通过标签定义索引
	if err := censorDB.AutoMigrate(&model.CensorLog{}); err != nil {
		return nil, err
	}
	return censorDB, nil
}
