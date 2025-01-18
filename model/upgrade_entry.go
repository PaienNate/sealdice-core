package model

import (
	"sealdice-core/utils/database/dbmodel"
)

// UpgradeEntry 定义升级条目表
type UpgradeEntry struct {
	ID          uint         `gorm:"column:id;primaryKey;autoIncrement"`          // 主键，自增
	Version     int64        `gorm:"column:version;not null"`                     // 版本号，不可为空
	Description string       `gorm:"column:description;not null"`                 // 升级内容描述，不可为空
	Breaking    bool         `gorm:"column:breaking;default:false;not null"`      // 是否是破坏性升级
	AppliedAt   dbmodel.Time `gorm:"column:applied_at;default:CURRENT_TIMESTAMP"` // 升级应用时间
	SealModel                                                                     // 嵌入 SealModel
}

// TableName 显式定义表名
func (UpgradeEntry) TableName() string {
	return "upgrade_entries"
}
