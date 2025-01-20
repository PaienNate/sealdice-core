package migrate

import "sealdice-core/utils/database"

type Manager struct {
	// 数据库操作对象
	DB database.DatabaseOperator
}

// 初始化函数
func NewManager(db database.DatabaseOperator) *Manager {
	return &Manager{
		DB: db,
	}
}
