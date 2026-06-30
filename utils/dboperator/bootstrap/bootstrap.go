package bootstrap

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"

	"sealdice-core/logger"
	"sealdice-core/model"
)

const (
	SQLiteLogsTempTable     = "logs__tmp_v150"
	SQLiteAttrsTempTable    = "attrs__tmp_v150"
	SQLiteLogItemsTempTable = "log_items__tmp_v150"
	SQLiteCopyBatchSize     = int64(500)
)

type SQLitePragmaColumn struct {
	Name    string         `gorm:"column:name"`
	Type    string         `gorm:"column:type"`
	NotNull int            `gorm:"column:notnull"`
	Default sql.NullString `gorm:"column:dflt_value"`
	PK      int            `gorm:"column:pk"`
}

type SQLiteExpectedColumn struct {
	Name       string
	Type       string
	PrimaryKey bool
}

var ExpectedSQLiteLogsColumns = []SQLiteExpectedColumn{
	{Name: "id", Type: "INTEGER", PrimaryKey: true},
	{Name: "name", Type: "TEXT"},
	{Name: "group_id", Type: "TEXT"},
	{Name: "created_at", Type: "INTEGER"},
	{Name: "updated_at", Type: "INTEGER"},
	{Name: "size", Type: "INTEGER"},
	{Name: "extra", Type: "TEXT"},
	{Name: "upload_url", Type: "TEXT"},
	{Name: "upload_time", Type: "INTEGER"},
}

var ExpectedSQLiteAttrsColumns = []SQLiteExpectedColumn{
	{Name: "id", Type: "TEXT", PrimaryKey: true},
	{Name: "data", Type: "BLOB"},
	{Name: "attrs_type", Type: "TEXT"},
	{Name: "binding_sheet_id", Type: "TEXT"},
	{Name: "name", Type: "TEXT"},
	{Name: "owner_id", Type: "TEXT"},
	{Name: "sheet_type", Type: "TEXT"},
	{Name: "is_hidden", Type: "NUMERIC"},
	{Name: "created_at", Type: "INTEGER"},
	{Name: "updated_at", Type: "INTEGER"},
}

var ExpectedSQLiteLogItemsColumns = []SQLiteExpectedColumn{
	{Name: "id", Type: "INTEGER", PrimaryKey: true},
	{Name: "log_id", Type: "INTEGER"},
	{Name: "group_id", Type: "TEXT"},
	{Name: "nickname", Type: "TEXT"},
	{Name: "im_userid", Type: "TEXT"},
	{Name: "time", Type: "INTEGER"},
	{Name: "message", Type: "TEXT"},
	{Name: "is_dice", Type: "INTEGER"},
	{Name: "command_id", Type: "INTEGER"},
	{Name: "command_info", Type: "TEXT"},
	{Name: "raw_msg_id", Type: "TEXT"},
	{Name: "user_uniform_id", Type: "TEXT"},
	{Name: "removed", Type: "INTEGER"},
	{Name: "parent_id", Type: "INTEGER"},
}

func DataDB(dialect string, writeDB *gorm.DB) error {
	if dialect == "sqlite" {
		if err := EnsureSQLiteSimpleTable(writeDB, &model.GroupPlayerInfoBase{}); err != nil {
			return err
		}
		if err := EnsureSQLiteSimpleTable(writeDB, &model.GroupInfo{}); err != nil {
			return err
		}
		if err := EnsureSQLiteSimpleTable(writeDB, &model.BanInfo{}); err != nil {
			return err
		}
		if err := EnsureSQLiteSimpleTable(writeDB, &model.EndpointInfo{}); err != nil {
			return err
		}
		return EnsureSQLiteAttrsTable(writeDB)
	}

	return writeDB.AutoMigrate(
		&model.GroupPlayerInfoBase{},
		&model.GroupInfo{},
		&model.BanInfo{},
		&model.EndpointInfo{},
		&model.AttributesItemModel{},
	)
}

func LogDB(dialect string, writeDB *gorm.DB) error {
	switch dialect {
	case "sqlite":
		writeDB.Exec("PRAGMA mmap_size = 16777216")
		writeDB.Exec("PRAGMA cache_size = -32768")
		return EnsureSQLiteLogSchema(writeDB)
	case "mysql":
		if err := writeDB.AutoMigrate(&model.LogInfoHookMySQL{}); err != nil {
			return err
		}
		if err := writeDB.AutoMigrate(&model.LogOneItemHookMySQL{}); err != nil {
			return err
		}
		if err := createMySQLIndexForLogInfo(writeDB); err != nil {
			return err
		}
		if err := createMySQLIndexForLogOneItem(writeDB); err != nil {
			return err
		}
		return calculateLogSize(writeDB)
	default:
		if err := writeDB.AutoMigrate(&model.LogInfo{}); err != nil {
			return err
		}
		return writeDB.AutoMigrate(&model.LogOneItem{})
	}
}

func CensorDB(writeDB *gorm.DB) error {
	return writeDB.AutoMigrate(&model.CensorLog{})
}

func createMySQLIndexForLogInfo(db *gorm.DB) error {
	log := zap.S().Named(logger.LogKeyDatabase)
	if !db.Migrator().HasIndex(&model.LogInfoHookMySQL{}, "idx_log_name") {
		if err := db.Exec("CREATE INDEX idx_log_name ON logs (name(20));").Error; err != nil {
			log.Errorf("创建idx_log_name索引失败,原因为 %v", err)
		}
	}
	if !db.Migrator().HasIndex(&model.LogInfoHookMySQL{}, "idx_logs_group") {
		if err := db.Exec("CREATE INDEX idx_logs_group ON logs (group_id(20));").Error; err != nil {
			log.Errorf("创建idx_logs_group索引失败,原因为 %v", err)
		}
	}
	if !db.Migrator().HasIndex(&model.LogInfoHookMySQL{}, "idx_logs_updated_at") {
		if err := db.Exec("CREATE INDEX idx_logs_updated_at ON logs (updated_at);").Error; err != nil {
			log.Errorf("创建idx_logs_updated_at索引失败,原因为 %v", err)
		}
	}
	return nil
}

func createMySQLIndexForLogOneItem(db *gorm.DB) error {
	log := zap.S().Named(logger.LogKeyDatabase)
	if !db.Migrator().HasIndex(&model.LogOneItemHookMySQL{}, "idx_log_items_group_id") {
		if err := db.Exec("CREATE INDEX idx_log_items_group_id ON log_items(group_id(20))").Error; err != nil {
			log.Errorf("创建idx_logs_group索引失败,原因为 %v", err)
		}
	}
	if !db.Migrator().HasIndex(&model.LogOneItemHookMySQL{}, "idx_raw_msg_id") {
		if err := db.Exec("CREATE INDEX idx_raw_msg_id ON log_items(raw_msg_id(20))").Error; err != nil {
			log.Errorf("创建idx_log_group_id_name索引失败,原因为 %v", err)
		}
	}
	return nil
}

func EnsureSQLiteLogSchema(db *gorm.DB) error {
	if err := EnsureSQLiteLogsTable(db); err != nil {
		return err
	}
	return EnsureSQLiteLogItemsTable(db)
}

func EnsureSQLiteLogsTable(db *gorm.DB) error {
	if !db.Migrator().HasTable("logs") {
		if err := CreateSQLiteLogsTable(db, "logs"); err != nil {
			return err
		}
		return EnsureSQLiteLogsIndexes(db)
	}

	columns, err := loadSQLiteTableColumns(db, "logs")
	if err != nil {
		return err
	}

	if !sqliteColumnsMatch(columns, ExpectedSQLiteLogsColumns) {
		if err := recreateSQLiteLogsTable(db, columns); err != nil {
			return err
		}
	}
	return EnsureSQLiteLogsIndexes(db)
}

func EnsureSQLiteLogsIndexes(db *gorm.DB) error {
	stmts := []string{
		"CREATE INDEX IF NOT EXISTS idx_logs_group ON `logs` (group_id)",
		"CREATE INDEX IF NOT EXISTS idx_logs_updated_at ON `logs` (updated_at)",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_log_group_id_name ON `logs` (group_id, name)",
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}

func recreateSQLiteLogsTable(db *gorm.DB, actual []SQLitePragmaColumn) error {
	if err := dropSQLiteIndexes(db, []string{"idx_logs_group", "idx_logs_updated_at", "idx_log_group_id_name"}); err != nil {
		return err
	}
	actualMap := make(map[string]SQLitePragmaColumn, len(actual))
	for _, col := range actual {
		actualMap[strings.ToLower(col.Name)] = col
	}
	if _, ok := actualMap["id"]; !ok {
		return errors.New("logs 表缺少 id 列，无法迁移")
	}

	if err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", quoteSQLiteIdentifier(SQLiteLogsTempTable))).Error; err != nil {
		return err
	}
	if err := CreateSQLiteLogsTable(db, SQLiteLogsTempTable); err != nil {
		return err
	}

	insertColumns := make([]string, 0, len(ExpectedSQLiteLogsColumns))
	selectColumns := make([]string, 0, len(ExpectedSQLiteLogsColumns))
	for _, exp := range ExpectedSQLiteLogsColumns {
		insertColumns = append(insertColumns, quoteSQLiteIdentifier(exp.Name))
		if _, ok := actualMap[strings.ToLower(exp.Name)]; ok {
			selectColumns = append(selectColumns, quoteSQLiteIdentifier(exp.Name))
		} else {
			selectColumns = append(selectColumns, defaultValueForMissingColumn(exp.Name))
		}
	}

	if err := bulkCopySQLiteTable(db, "logs", SQLiteLogsTempTable, insertColumns, selectColumns); err != nil {
		return err
	}

	if err := db.Exec(fmt.Sprintf("DROP TABLE %s", quoteSQLiteIdentifier("logs"))).Error; err != nil {
		return err
	}
	return db.Exec(fmt.Sprintf("ALTER TABLE %s RENAME TO %s", quoteSQLiteIdentifier(SQLiteLogsTempTable), quoteSQLiteIdentifier("logs"))).Error
}

func EnsureSQLiteLogItemsTable(db *gorm.DB) error {
	if !db.Migrator().HasTable("log_items") {
		if err := CreateSQLiteLogItemsTable(db, "log_items"); err != nil {
			return err
		}
		return EnsureSQLiteLogItemsIndexes(db)
	}

	columns, err := loadSQLiteTableColumns(db, "log_items")
	if err != nil {
		return err
	}

	if !sqliteColumnsMatch(columns, ExpectedSQLiteLogItemsColumns) {
		if err := recreateSQLiteLogItemsTable(db, columns); err != nil {
			return err
		}
	}
	return EnsureSQLiteLogItemsIndexes(db)
}

func EnsureSQLiteAttrsTable(db *gorm.DB) error {
	if !db.Migrator().HasTable("attrs") {
		if err := CreateSQLiteAttrsTable(db, "attrs"); err != nil {
			return err
		}
		return EnsureSQLiteAttrsIndexes(db)
	}

	columns, err := loadSQLiteTableColumns(db, "attrs")
	if err != nil {
		return err
	}

	if !sqliteColumnsMatch(columns, ExpectedSQLiteAttrsColumns) {
		if err := recreateSQLiteAttrsTable(db, columns); err != nil {
			return err
		}
	}
	return EnsureSQLiteAttrsIndexes(db)
}

func EnsureSQLiteAttrsIndexes(db *gorm.DB) error {
	stmts := []string{
		"CREATE INDEX IF NOT EXISTS idx_attrs_attrs_type_id ON `attrs` (attrs_type)",
		"CREATE INDEX IF NOT EXISTS idx_attrs_binding_sheet_id ON `attrs` (binding_sheet_id)",
		"CREATE INDEX IF NOT EXISTS idx_attrs_owner_id_id ON `attrs` (owner_id)",
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}

func EnsureSQLiteSimpleTable(db *gorm.DB, modelValue interface{}) error {
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(modelValue); err != nil {
		return err
	}
	table := stmt.Schema.Table
	if table == "" {
		table = stmt.Table
	}
	if table == "" {
		return fmt.Errorf("unable to determine table name for model %T", modelValue)
	}
	if !db.Migrator().HasTable(table) {
		return db.AutoMigrate(modelValue)
	}
	columns, err := loadSQLiteTableColumns(db, table)
	if err != nil {
		return err
	}
	expected := make([]string, 0, len(stmt.Schema.Fields))
	for _, field := range stmt.Schema.Fields {
		if field.IgnoreMigration {
			continue
		}
		if field.DBName != "" {
			expected = append(expected, strings.ToLower(field.DBName))
		}
	}
	if !sqliteColumnNamesMatch(columns, expected) {
		return db.AutoMigrate(modelValue)
	}
	return nil
}

func loadSQLiteTableColumns(db *gorm.DB, table string) ([]SQLitePragmaColumn, error) {
	query := fmt.Sprintf("PRAGMA table_info(%s)", quoteSQLiteString(table))
	var columns []SQLitePragmaColumn
	if err := db.Raw(query).Scan(&columns).Error; err != nil {
		return nil, err
	}
	return columns, nil
}

func sqliteColumnsMatch(actual []SQLitePragmaColumn, expected []SQLiteExpectedColumn) bool {
	if len(actual) != len(expected) {
		return false
	}
	actualMap := make(map[string]SQLitePragmaColumn, len(actual))
	for _, col := range actual {
		actualMap[strings.ToLower(col.Name)] = col
	}
	for _, exp := range expected {
		col, ok := actualMap[strings.ToLower(exp.Name)]
		if !ok {
			return false
		}
		if normalizeSQLiteType(col.Type) != normalizeSQLiteType(exp.Type) {
			return false
		}
		if exp.PrimaryKey != (col.PK != 0) {
			return false
		}
	}
	return true
}

func sqliteColumnNamesMatch(actual []SQLitePragmaColumn, expected []string) bool {
	if len(actual) != len(expected) {
		return false
	}
	actualSet := make(map[string]struct{}, len(actual))
	for _, col := range actual {
		actualSet[strings.ToLower(col.Name)] = struct{}{}
	}
	for _, name := range expected {
		if _, ok := actualSet[strings.ToLower(name)]; !ok {
			return false
		}
	}
	return true
}

func normalizeSQLiteType(t string) string {
	n := strings.ToUpper(strings.TrimSpace(t))
	n = strings.ReplaceAll(n, " PRIMARY KEY", "")
	n = strings.ReplaceAll(n, " AUTOINCREMENT", "")
	n = strings.ReplaceAll(n, " UNSIGNED", "")
	n = strings.TrimSpace(n)
	switch n {
	case "NUMERIC":
		return "INTEGER"
	case "BOOL", "BOOLEAN":
		return "INTEGER"
	default:
		return n
	}
}

func recreateSQLiteLogItemsTable(db *gorm.DB, actual []SQLitePragmaColumn) error {
	if err := dropSQLiteIndexes(db, []string{"idx_log_items_group_id", "idx_log_items_log_id", "idx_raw_msg_id"}); err != nil {
		return err
	}
	actualMap := make(map[string]SQLitePragmaColumn, len(actual))
	for _, col := range actual {
		actualMap[strings.ToLower(col.Name)] = col
	}
	if _, ok := actualMap["id"]; !ok {
		return errors.New("log_items 表缺少 id 列，无法迁移")
	}

	if err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", quoteSQLiteIdentifier(SQLiteLogItemsTempTable))).Error; err != nil {
		return err
	}
	if err := CreateSQLiteLogItemsTable(db, SQLiteLogItemsTempTable); err != nil {
		return err
	}

	insertColumns := make([]string, 0, len(ExpectedSQLiteLogItemsColumns))
	selectColumns := make([]string, 0, len(ExpectedSQLiteLogItemsColumns))
	for _, exp := range ExpectedSQLiteLogItemsColumns {
		insertColumns = append(insertColumns, quoteSQLiteIdentifier(exp.Name))
		if _, ok := actualMap[strings.ToLower(exp.Name)]; ok {
			selectColumns = append(selectColumns, quoteSQLiteIdentifier(exp.Name))
		} else {
			selectColumns = append(selectColumns, defaultValueForMissingColumn(exp.Name))
		}
	}

	if err := bulkCopySQLiteTable(db, "log_items", SQLiteLogItemsTempTable, insertColumns, selectColumns); err != nil {
		return err
	}

	if err := db.Exec(fmt.Sprintf("DROP TABLE %s", quoteSQLiteIdentifier("log_items"))).Error; err != nil {
		return err
	}
	return db.Exec(fmt.Sprintf("ALTER TABLE %s RENAME TO %s", quoteSQLiteIdentifier(SQLiteLogItemsTempTable), quoteSQLiteIdentifier("log_items"))).Error
}

func recreateSQLiteAttrsTable(db *gorm.DB, actual []SQLitePragmaColumn) error {
	if err := dropSQLiteIndexes(db, []string{"idx_attrs_attrs_type_id", "idx_attrs_binding_sheet_id", "idx_attrs_owner_id_id"}); err != nil {
		return err
	}
	actualMap := make(map[string]SQLitePragmaColumn, len(actual))
	for _, col := range actual {
		actualMap[strings.ToLower(col.Name)] = col
	}
	if _, ok := actualMap["id"]; !ok {
		return errors.New("attrs 表缺少 id 列，无法迁移")
	}

	if err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", quoteSQLiteIdentifier(SQLiteAttrsTempTable))).Error; err != nil {
		return err
	}
	if err := CreateSQLiteAttrsTable(db, SQLiteAttrsTempTable); err != nil {
		return err
	}

	insertColumns := make([]string, 0, len(ExpectedSQLiteAttrsColumns))
	selectColumns := make([]string, 0, len(ExpectedSQLiteAttrsColumns))
	for _, exp := range ExpectedSQLiteAttrsColumns {
		insertColumns = append(insertColumns, quoteSQLiteIdentifier(exp.Name))
		if _, ok := actualMap[strings.ToLower(exp.Name)]; ok {
			selectColumns = append(selectColumns, quoteSQLiteIdentifier(exp.Name))
		} else {
			selectColumns = append(selectColumns, defaultValueForMissingColumn(exp.Name))
		}
	}

	if err := bulkCopySQLiteTable(db, "attrs", SQLiteAttrsTempTable, insertColumns, selectColumns); err != nil {
		return err
	}

	if err := db.Exec(fmt.Sprintf("DROP TABLE %s", quoteSQLiteIdentifier("attrs"))).Error; err != nil {
		return err
	}
	return db.Exec(fmt.Sprintf("ALTER TABLE %s RENAME TO %s", quoteSQLiteIdentifier(SQLiteAttrsTempTable), quoteSQLiteIdentifier("attrs"))).Error
}

func CreateSQLiteAttrsTable(db *gorm.DB, table string) error {
	session := db.Session(&gorm.Session{})
	if table != "" {
		session = session.Table(table)
	}
	err := session.Migrator().CreateTable(&model.AttributesItemModel{})
	if err != nil && strings.Contains(err.Error(), "already exists") {
		return nil
	}
	return err
}

func CreateSQLiteLogsTable(db *gorm.DB, table string) error {
	session := db.Session(&gorm.Session{})
	if table != "" {
		session = session.Table(table)
	}
	err := session.Migrator().CreateTable(&model.LogInfo{})
	if err != nil && strings.Contains(err.Error(), "already exists") {
		return nil
	}
	return err
}

func CreateSQLiteLogItemsTable(db *gorm.DB, table string) error {
	session := db.Session(&gorm.Session{})
	if table != "" {
		session = session.Table(table)
	}
	err := session.Migrator().CreateTable(&model.LogOneItem{})
	if err != nil && strings.Contains(err.Error(), "already exists") {
		return nil
	}
	return err
}

func bulkCopySQLiteTable(db *gorm.DB, src, dst string, insertColumns, selectColumns []string) error {
	if len(insertColumns) == 0 || len(selectColumns) == 0 {
		return nil
	}

	var minRow struct {
		Value sql.NullInt64 `gorm:"column:value"`
	}
	var maxRow struct {
		Value sql.NullInt64 `gorm:"column:value"`
	}

	minQuery := fmt.Sprintf("SELECT rowid AS value FROM %s ORDER BY rowid ASC LIMIT 1", quoteSQLiteIdentifier(src))
	if err := db.Raw(minQuery).Scan(&minRow).Error; err != nil {
		return err
	}
	maxQuery := fmt.Sprintf("SELECT rowid AS value FROM %s ORDER BY rowid DESC LIMIT 1", quoteSQLiteIdentifier(src))
	if err := db.Raw(maxQuery).Scan(&maxRow).Error; err != nil {
		return err
	}
	if !minRow.Value.Valid || !maxRow.Value.Valid {
		return nil
	}

	insertClause := strings.Join(insertColumns, ",")
	selectClause := strings.Join(selectColumns, ",")
	silentButError := gormLogger.New(
		log.New(os.Stderr, "", 0),
		gormLogger.Config{LogLevel: gormLogger.Error},
	)

	idDst := quoteSQLiteIdentifier(dst)
	idSrc := quoteSQLiteIdentifier(src)
	log := zap.S().Named(logger.LogKeyDatabase)
	for startRow := minRow.Value.Int64; startRow <= maxRow.Value.Int64; startRow += SQLiteCopyBatchSize {
		endRow := startRow + SQLiteCopyBatchSize - 1
		copySQL := fmt.Sprintf(
			"INSERT INTO %s (%s) SELECT %s FROM %s WHERE rowid BETWEEN ? AND ?",
			idDst,
			insertClause,
			selectClause,
			idSrc,
		)
		if err := db.Session(&gorm.Session{Logger: silentButError}).
			Exec(copySQL, startRow, endRow).Error; err != nil {
			return err
		}

		val1 := startRow - minRow.Value.Int64
		val2 := maxRow.Value.Int64 - minRow.Value.Int64
		if val2 > 0 {
			log.Infof("已迁移 %d/%d 行到 %s - %.2f%%", val1, val2, idDst, float64(val1)/float64(val2)*100)
		}
	}

	return nil
}

func dropSQLiteIndexes(db *gorm.DB, indexes []string) error {
	for _, name := range indexes {
		stmt := fmt.Sprintf("DROP INDEX IF EXISTS %s", quoteSQLiteIdentifier(name))
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}

func EnsureSQLiteLogItemsIndexes(db *gorm.DB) error {
	stmts := []string{
		"CREATE INDEX IF NOT EXISTS idx_log_items_group_id ON `log_items` (group_id)",
		"CREATE INDEX IF NOT EXISTS idx_log_items_log_id ON `log_items` (log_id)",
		"CREATE INDEX IF NOT EXISTS idx_raw_msg_id ON `log_items` (raw_msg_id)",
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}

func quoteSQLiteIdentifier(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

func quoteSQLiteString(name string) string {
	return "'" + strings.ReplaceAll(name, "'", "''") + "'"
}

func defaultValueForMissingColumn(name string) string {
	switch name {
	case "log_id", "command_id", "time", "is_dice", "removed", "parent_id", "created_at", "updated_at", "size", "upload_time":
		return "0"
	case "name", "group_id", "attrs_type", "binding_sheet_id", "owner_id", "sheet_type", "upload_url":
		return "''"
	case "data", "extra":
		return "NULL"
	default:
		return "NULL"
	}
}

func calculateLogSize(logsDB *gorm.DB) error {
	log := zap.S().Named(logger.LogKeyDatabase)
	var ids []uint64
	var logItemSums []struct {
		LogID uint64
		Count int64
	}
	logsDB.Model(&model.LogInfo{}).Pluck("id", &ids)
	if len(ids) == 0 {
		return nil
	}
	err := logsDB.Model(&model.LogOneItem{}).
		Where("log_id IN ?", ids).
		Group("log_id").
		Select("log_id, COUNT(*) AS count").
		Scan(&logItemSums).Error
	if err != nil {
		log.Infof("Error querying LogOneItem: %v", err)
		return err
	}

	for _, sum := range logItemSums {
		err = logsDB.Model(&model.LogInfo{}).
			Where("id = ?", sum.LogID).
			UpdateColumn("size", sum.Count).Error
		if err != nil {
			log.Errorf("Error updating LogInfo: %v", err)
			return err
		}
	}
	return nil
}
