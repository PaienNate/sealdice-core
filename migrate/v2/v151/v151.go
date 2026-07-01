package v151

import (
	"sealdice-core/utils/constant"
	"sealdice-core/utils/dboperator/engine"
	upgrade "sealdice-core/utils/upgrader"
)

var V151GORMCleanMigration = upgrade.Upgrade{
	ID:    "007_V151GORMCleanMigration", // TODO：需要合理的生成逻辑，这个等提交了PR再后续讨论
	Phase: upgrade.PhasePostBootstrap,
	Description: `
# 升级说明
删除了因为GORM更新导致逻辑失效后，错误插入的部分数据
`,
	ShouldRun: func(operator engine.DatabaseOperator) (bool, error) {
		db := operator.GetDataDB(constant.READ)
		var count int64
		if db.Migrator().HasTable("ban_info") {
			if err := db.Table("ban_info").Where("data IS NULL OR data = '' OR length(data) = 0").Count(&count).Error; err == nil && count > 0 {
				return true, nil
			}
		}
		if db.Migrator().HasTable("attrs") {
			count = 0
			if err := db.Table("attrs").Where("data IS NULL OR data = '' OR length(data) = 0").Count(&count).Error; err == nil && count > 0 {
				return true, nil
			}
		}
		return false, nil
	},
	Apply: func(logf func(string), operator engine.DatabaseOperator) error {
		logf("[INFO] V151数据库清理错误插入数据开始")
		err := V151GORMCleanMigrate(operator, logf)
		if err != nil {
			return err
		}
		logf("[INFO] V151数据库清理错误插入数据处置完毕")
		return nil
	},
}
