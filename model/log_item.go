package model

import (
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
)

type LogOneItem struct {
	ID             uint64      `json:"id" gorm:"primaryKey;autoIncrement;column:id"`
	LogID          uint64      `json:"-" gorm:"column:log_id;index:idx_log_items_log_id"`
	GroupID        string      `gorm:"index:idx_log_items_group_id;column:group_id;index:idx_log_delete_by_id"`
	Nickname       string      `json:"nickname" gorm:"column:nickname"`
	IMUserID       string      `json:"IMUserId" gorm:"column:im_userid"`
	Time           int64       `json:"time" gorm:"column:time"`
	Message        string      `json:"message"  gorm:"column:message"`
	IsDice         bool        `json:"isDice"  gorm:"column:is_dice"`
	CommandID      int64       `json:"commandId"  gorm:"column:command_id"`
	CommandInfo    interface{} `json:"commandInfo" gorm:"-"`
	CommandInfoStr string      `json:"-" gorm:"column:command_info"`
	// 这里的RawMsgID 真的什么都有可能
	RawMsgID    interface{} `json:"rawMsgId" gorm:"-"`
	RawMsgIDStr string      `json:"-" gorm:"column:raw_msg_id;index:idx_raw_msg_id;index:idx_log_delete_by_id"`
	UniformID   string      `json:"uniformId" gorm:"column:user_uniform_id"`
	// 数据库里没有的
	Channel string `json:"channel" gorm:"-"`
	// 数据库里有，JSON里没有的
	// 允许default=NULL
	Removed  *int `gorm:"column:removed" json:"-"`
	ParentID *int `gorm:"column:parent_id" json:"-"`
}

// 兼容旧版本的数据库设计
func (*LogOneItem) TableName() string {
	return "log_items"
}

// BeforeSave 钩子函数: 查询前,interface{}转换为json
func (item *LogOneItem) BeforeSave(_ *gorm.DB) (err error) {
	// 将 CommandInfo 转换为 JSON 字符串保存到 CommandInfoStr
	if item.CommandInfo != nil {
		if data, err := json.Marshal(item.CommandInfo); err == nil {
			item.CommandInfoStr = string(data)
		} else {
			return err
		}
	}

	// 将 RawMsgID 转换为 string 字符串，保存到 RawMsgIDStr
	if item.RawMsgID != nil {
		item.RawMsgIDStr = fmt.Sprintf("%v", item.RawMsgID)
	}

	return nil
}

// AfterFind 钩子函数: 查询后,interface{}转换为json
func (item *LogOneItem) AfterFind(_ *gorm.DB) (err error) {
	// 将 CommandInfoStr 从 JSON 字符串反序列化为 CommandInfo
	if item.CommandInfoStr != "" {
		if err := json.Unmarshal([]byte(item.CommandInfoStr), &item.CommandInfo); err != nil {
			return err
		}
	}

	// 将 RawMsgIDStr string 直接赋值给 RawMsgID
	if item.RawMsgIDStr != "" {
		item.RawMsgID = item.RawMsgIDStr
	}

	return nil
}
