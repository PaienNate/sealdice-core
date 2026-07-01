package store

import (
	"encoding/json"

	"gorm.io/gorm"

	"sealdice-core/model"
	upgrade "sealdice-core/utils/upgrader"
)

type DBStore struct {
	db *gorm.DB
}

func NewDBStore(db *gorm.DB, legacyJSONPath string) (*DBStore, error) {
	store := &DBStore{db: db}
	if err := db.AutoMigrate(&model.PatchLog{}); err != nil {
		return nil, err
	}
	if legacyJSONPath != "" {
		if err := store.importLegacyJSON(legacyJSONPath); err != nil {
			return nil, err
		}
	}
	return store, nil
}

func (ds *DBStore) IsApplied(id string) (bool, error) {
	var count int64
	err := ds.db.Model(&model.PatchLog{}).Where("patch_id = ?", id).Count(&count).Error
	return count > 0, err
}

func (ds *DBStore) SaveRecord(rec upgrade.UpgradeRecord) error {
	logsJSON, err := json.Marshal(rec.Logs)
	if err != nil {
		return err
	}
	row := model.PatchLog{
		PatchID:   rec.ID,
		Status:    statusFromRecord(rec),
		Level:     "",
		Message:   rec.Message,
		LogsJSON:  string(logsJSON),
		AppliedAt: rec.Timestamp,
	}
	return ds.db.Where("patch_id = ?", rec.ID).Assign(row).FirstOrCreate(&model.PatchLog{}).Error
}

func (ds *DBStore) LoadRecords() ([]upgrade.UpgradeRecord, error) {
	var rows []model.PatchLog
	if err := ds.db.Order("applied_at ASC").Find(&rows).Error; err != nil {
		return nil, err
	}

	records := make([]upgrade.UpgradeRecord, 0, len(rows))
	for _, row := range rows {
		logs := []string{}
		if row.LogsJSON != "" {
			if err := json.Unmarshal([]byte(row.LogsJSON), &logs); err != nil {
				return nil, err
			}
		}
		records = append(records, upgrade.UpgradeRecord{
			ID:        row.PatchID,
			Timestamp: row.AppliedAt,
			Success:   row.Status == "success",
			Message:   row.Message,
			Logs:      logs,
		})
	}
	return records, nil
}

func (ds *DBStore) importLegacyJSON(path string) error {
	legacy := NewJSONStore(path)
	if !legacy.PathExists() {
		return nil
	}
	records, err := legacy.LoadRecords()
	if err != nil {
		return err
	}
	for _, rec := range records {
		applied, err := ds.IsApplied(rec.ID)
		if err != nil {
			return err
		}
		if applied {
			continue
		}
		if err := ds.SaveRecord(rec); err != nil {
			return err
		}
	}
	return nil
}

func statusFromRecord(rec upgrade.UpgradeRecord) string {
	if rec.Success {
		return "success"
	}
	return "failed"
}
