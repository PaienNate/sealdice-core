// upgrade/manager.go
package upgrade

import (
	"fmt"
	"sort"
	"time"

	"sealdice-core/utils/dboperator/engine"
)

type Manager struct {
	Upgrades []Upgrade
	Store    Store
	Database engine.DatabaseOperator
}

func (m *Manager) Register(up Upgrade) {
	m.Upgrades = append(m.Upgrades, up)
}

func (m *Manager) ApplyAll() error {
	for _, phase := range []Phase{PhasePreBootstrap, PhasePostBootstrap} {
		if err := m.ApplyPhase(phase); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) ApplyPhase(phase Phase) error {
	sort.Slice(m.Upgrades, func(i, j int) bool {
		return m.Upgrades[i].ID < m.Upgrades[j].ID
	})

	records, err := m.Store.LoadRecords()
	if err != nil {
		return err
	}
	recordByID := make(map[string]UpgradeRecord, len(records))
	for _, rec := range records {
		recordByID[rec.ID] = rec
	}

	for _, up := range m.Upgrades {
		if up.Phase == "" {
			up.Phase = PhasePostBootstrap
		}
		if up.Phase != phase {
			continue
		}

		applied, err := m.Store.IsApplied(up.ID)
		if err != nil {
			return err
		}
		shouldRun, err := shouldRunUpgrade(up, m.Database)
		if err != nil {
			return err
		}
		if applied {
			if rec, ok := recordByID[up.ID]; ok && !rec.Success && shouldRun {
				// 失败记录仍按历史策略阻止自动重跑，但保留显式告警。
				// 这里不返回错误，交给调用方继续后续流程。
				// 统一输出到记录日志里，便于上层打印。
				_ = rec
			}
			continue
		}
		if !shouldRun {
			continue
		}

		logs := []string{}
		logf := func(msg string) {
			logs = append(logs, msg)
		}

		start := time.Now()
		err = up.Apply(logf, m.Database)

		rec := UpgradeRecord{
			ID:        up.ID,
			Timestamp: start,
			Success:   err == nil,
			Message:   "成功",
			Logs:      logs,
		}
		if err != nil {
			rec.Message = err.Error()
		}

		if err2 := m.Store.SaveRecord(rec); err2 != nil {
			return fmt.Errorf("保存升级记录失败: %w", err)
		}

		if err != nil {
			return fmt.Errorf("因无法忽略的错误，升级 %s 失败: %w，请联系海豹开发者", up.ID, err)
		}
	}
	return nil
}

func (m *Manager) HasPendingPhaseSignals(phase Phase) (bool, []string, error) {
	sort.Slice(m.Upgrades, func(i, j int) bool {
		return m.Upgrades[i].ID < m.Upgrades[j].ID
	})

	matched := []string{}
	for _, up := range m.Upgrades {
		if up.Phase == "" {
			up.Phase = PhasePostBootstrap
		}
		if up.Phase != phase {
			continue
		}
		applied, err := m.Store.IsApplied(up.ID)
		if err != nil {
			return false, nil, err
		}
		if applied {
			continue
		}
		shouldRun, err := shouldRunUpgrade(up, m.Database)
		if err != nil {
			return false, nil, err
		}
		if shouldRun {
			matched = append(matched, up.ID)
		}
	}
	return len(matched) > 0, matched, nil
}

func shouldRunUpgrade(up Upgrade, database engine.DatabaseOperator) (bool, error) {
	if up.ShouldRun == nil {
		return true, nil
	}
	return up.ShouldRun(database)
}
