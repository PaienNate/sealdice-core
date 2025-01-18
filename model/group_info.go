package model

// GroupInfo 模型
type GroupInfo struct {
	ID        string `gorm:"column:id;primaryKey"` // 主键，字符串类型
	CreatedAt int    `gorm:"column:created_at"`    // 创建时间
	UpdatedAt *int64 `gorm:"column:updated_at"`    // 更新时间，int64类型
	Data      []byte `gorm:"column:data"`          // BLOB 类型字段，使用 []byte 表示
}

func (*GroupInfo) TableName() string {
	return "group_info"
}
