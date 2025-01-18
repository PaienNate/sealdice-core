package model

type LogInfo struct {
	ID        uint64 `json:"id" gorm:"primaryKey;autoIncrement;column:id"`
	Name      string `json:"name" gorm:"index:idx_log_group_id_name,unique"`
	GroupID   string `json:"groupId" gorm:"index:idx_logs_group;index:idx_log_group_id_name,unique"`
	CreatedAt int64  `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt int64  `json:"updatedAt" gorm:"column:updated_at;index:idx_logs_update_at"`
	// 允许数据库NULL值
	// 原版代码中，此处标记了db:size，但实际上，该列并不存在。
	// 考虑到该处数据将会为未来log查询提供优化手段，保留该结构体定义，但不使用。
	// 使用GORM:<-:false 无写入权限，这样它就不会建库，但请注意，下面LogGetLogPage处，如果你查询出的名称不是size
	// 不能在这里绑定column，因为column会给你建立那一列。
	// TODO: 将这个字段使用上会不会比后台查询就JOIN更合适？
	Size *int `json:"size" gorm:"column:size"`
	// 数据库里有，json不展示的
	// 允许数据库NULL值（该字段当前不使用）
	Extra *string `json:"-" gorm:"column:extra"`
	// 原本标记为：测试版特供，由于原代码每次都会执行，故直接启用此处column记录。
	UploadURL  string `json:"-" gorm:"column:upload_url"`  // 测试版特供
	UploadTime int    `json:"-" gorm:"column:upload_time"` // 测试版特供
}

func (*LogInfo) TableName() string {
	return "logs"
}
