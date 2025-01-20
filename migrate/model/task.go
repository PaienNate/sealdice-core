package migrate

import (
	"sealdice-core/migrate"
	v120 "sealdice-core/migrate/task/v120"
	"sealdice-core/model"
)

// Task TODO: 应该没有必要给这俩设计ERROR罢……？！
type Task interface {
	Init(manager *migrate.Manager)
	GetUpdateEntry() model.UpgradeEntry
	Run() (error, bool)
}

var (
	_ Task = (*v120.V120Task)(nil)
)
