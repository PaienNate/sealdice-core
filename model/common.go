package model

import (
	"gorm.io/plugin/soft_delete"
)

// SealModel 通用GORM MODEL
type SealModel struct {
	CreatedAt int64                 `gorm:"autoCreateTime"`
	UpdatedAt int64                 `gorm:"autoUpdateTime"`
	DeletedAt soft_delete.DeletedAt `gorm:"index"`
}
