package v120

import (
	"sealdice-core/migrate"
	"sealdice-core/model"
	"sealdice-core/utils/consts"
)

type V120Task struct {
	Parent *migrate.Manager
}

func (v V120Task) Init(manager *migrate.Manager) {
	v.Parent = manager
}

func (v V120Task) GetUpdateEntry() model.UpgradeEntry {
	return model.UpgradeEntry{
		UUID:        "8c26efd9-d2b3-a130-c2b5-a87e544f148a",
		Version:     0,
		Priority:    0,
		Description: "版本号：1.2.0 升级: Bbolt数据库升级为SQLite数据库，迁移数据",
		// 数据库迁移是破坏性更改
		Breaking: true,
	}
}

func (v V120Task) Run() (error, bool) {
	gormdb := v.Parent.DB.GetDataDB(consts.WRITE)
}

// func TryMigrateToV12() {
//	task := migrate.Task{
//		UpgradeEntry: model.UpgradeEntry{
//			// https://www.bejson.com/encrypt/genuuid/ 生成UUID/
//			UUID:        "8c26efd9-d2b3-a130-c2b5-a87e544f148a",
//			Version:     0,
//			Priority:    0,
//			Description: "版本号：1.2.0 Bbolt数据库升级为SQLite数据库，迁移数据",
//			Breaking:    false,
//		},
//		Parent: nil,
//		Run: func() (error, bool) {
//
//		},
//	}
//
//	_, err := os.Stat("./data/default/data.bdb")
//	if err != nil {
//		return
//	}
//
//	fmt.Fprintln(os.Stdout, "检测到旧数据库存在，试图进行转换")
//	_ = ConvertServe()
//	_ = ConvertLogs()
//	_ = os.Remove("./data/default/data.bdb")
//	fmt.Fprintln(os.Stdout, "V1.2 版本数据库升级完成")
// }
