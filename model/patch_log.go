package model

import "time"

type PatchLog struct {
	PatchID   string    `gorm:"column:patch_id;primaryKey"`
	Status    string    `gorm:"column:status"`
	Level     string    `gorm:"column:level"`
	Message   string    `gorm:"column:message"`
	LogsJSON  string    `gorm:"column:logs_json"`
	AppliedAt time.Time `gorm:"column:applied_at;autoCreateTime"`
}

func (PatchLog) TableName() string {
	return "sys_patch_log"
}
