package dao

import (
	"errors"

	"gorm.io/gorm"

	"sealdice-core/model"
	"sealdice-core/utils/consts"
	"sealdice-core/utils/database"
)

func GetEndpointInfo(operator database.DatabaseOperator, userID string) (*model.EndpointInfo, error) {
	var result *model.EndpointInfo
	db := operator.GetDataDB(consts.READ)
	if len(userID) == 0 {
		return nil, model.ErrEndpointInfoUIDEmpty
	}
	if db == nil {
		return nil, errors.New("db is nil")
	}

	err := db.Model(&model.EndpointInfo{}).
		Where("user_id = ?", userID).
		Select("cmd_num", "cmd_last_time", "online_time", "updated_at").
		Limit(1).
		Find(&result).Error

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	return result, nil
}

func SaveEndpointInfo(operator database.DatabaseOperator, e *model.EndpointInfo) error {
	db := operator.GetDataDB(consts.WRITE)
	// 检查 user_id 是否为空
	if len(e.UserID) == 0 {
		return model.ErrEndpointInfoUIDEmpty
	}
	// 使用 FirstOrCreate 来插入或更新
	if err := db.Where("user_id = ?", e.UserID).Assign(
		"cmd_num", e.CmdNum,
		"cmd_last_time", e.CmdLastTime,
		"online_time", e.OnlineTime,
		"updated_at", e.UpdatedAt,
	).
		// 考虑保持原有逻辑不动
		FirstOrCreate(&e).Error; err != nil {
		// 处理查询或创建时的错误
		return err
	}
	return nil
}
