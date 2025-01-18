package dao

import (
	"gorm.io/gorm/clause"

	"sealdice-core/model"
	"sealdice-core/utils/consts"
	"sealdice-core/utils/database"
)

// GroupInfoListGet 使用 GORM 实现，遍历 group_info 表中的数据并调用回调函数
func GroupInfoListGet(operator database.DatabaseOperator, callback func(id string, updatedAt int64, data []byte)) error {
	db := operator.GetDataDB(consts.READ)
	// 创建一个保存查询结果的结构体
	var results []struct {
		ID        string `gorm:"column:id"`         // 字段 id
		UpdatedAt *int64 `gorm:"column:updated_at"` // 由于可能存在 NULL，定义为指针类型
		Data      []byte `gorm:"column:data"`       // 字段 data
	}

	// 使用 GORM 查询 group_info 表中的 id, updated_at, data 列
	err := db.Model(&model.GroupInfo{}).Select("id, updated_at, data").Find(&results).Error
	if err != nil {
		// 如果查询发生错误，返回错误信息
		return err
	}

	// 遍历查询结果
	for _, result := range results {
		var updatedAt int64

		// 如果 updatedAt 是 NULL，需要跳过该字段
		if result.UpdatedAt != nil {
			updatedAt = *result.UpdatedAt
		}

		// 调用回调函数，传递 id, updatedAt, data
		callback(result.ID, updatedAt, result.Data)
	}

	// 返回 nil 表示操作成功
	return nil
}

// GroupInfoSave 保存群组信息
func GroupInfoSave(operator database.DatabaseOperator, groupID string, updatedAt int64, data []byte) error {
	// 使用写数据库
	db := operator.GetDataDB(consts.WRITE)
	// 使用 gorm 的 Upsert 功能实现插入或更新
	groupInfo := model.GroupInfo{
		ID:        groupID,
		UpdatedAt: &updatedAt,
		Data:      data,
	}
	result := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"updated_at", "data"}),
	}).Create(&groupInfo)
	return result.Error
}
