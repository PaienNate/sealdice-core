package model

import (
	"sealdice-core/utils/database/dbmodel"
)

// UpgradeEntry 定义升级条目表
type UpgradeEntry struct {
	UUID    string `gorm:"column:id;primaryKey;"`   // 主键，必须定义！
	Version int64  `gorm:"column:version;not null"` // 海豹版本号ID，不可为空

	Priority      int64        `gorm:"column:priority;not null"`                    // 该任务在该版本号中的优先级。
	Description   string       `gorm:"column:description;not null"`                 // 升级内容描述，不可为空
	Breaking      bool         `gorm:"column:breaking;default:false;not null"`      // 是否是破坏性升级
	AppliedAt     dbmodel.Time `gorm:"column:applied_at;default:CURRENT_TIMESTAMP"` // 升级应用时间
	ExitWhenError bool         `gorm:"-"`                                           // 若失败，是否退出程序 这个不用存库
	SealModel                  // 嵌入 SealModel
}

// TableName 显式定义表名
func (UpgradeEntry) TableName() string {
	return "upgrade_entries"
}
