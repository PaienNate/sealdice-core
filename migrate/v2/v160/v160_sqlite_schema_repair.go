package v160

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
	"sealdice-core/utils/constant"
	operator "sealdice-core/utils/dboperator/engine"
	upgrade "sealdice-core/utils/upgrader"
)

const (
	sqliteLogsTempTable     = "logs__tmp_v160"
	sqliteAttrsTempTable    = "attrs__tmp_v160"
	sqliteLogItemsTempTable = "log_items__tmp_v160"
	sqliteCopyBatchSize     = int64(500)
)

type sqlitePragmaColumn struct {
	Name    string         `gorm:"column:name"`
	Type    string         `gorm:"column:type"`
	NotNull int            `gorm:"column:notnull"`
	Default sql.NullString `gorm:"column:dflt_value"`
	PK      int            `gorm:"column:pk"`
}

type sqliteExpectedColumn struct {
	Name       string
	Type       string
	PrimaryKey bool
}

var expectedSQLiteLogsColumns = []sqliteExpectedColumn{
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

var expectedSQLiteAttrsColumns = []sqliteExpectedColumn{
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

var expectedSQLiteLogItemsColumns = []sqliteExpectedColumn{
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

func V160SQLiteSchemaRepairMigrate(dboperator operator.DatabaseOperator, logf func(string)) error {
	if dboperator.Type() != constant.SQLITE {
		return nil
	}

	dataDB := dboperator.GetDataDB(constant.WRITE)
	logDB := dboperator.GetLogDB(constant.WRITE)

	if err := ensureSQLiteTableShape(
		logDB,
		"logs",
		sqliteLogsTempTable,
		expectedSQLiteLogsColumns,
		createSQLiteLogsTable,
		[]string{"idx_logs_group", "idx_logs_updated_at", "idx_log_group_id_name"},
	); err != nil {
		return err
	}
	if err := ensureSQLiteTableShape(
		logDB,
		"log_items",
		sqliteLogItemsTempTable,
		expectedSQLiteLogItemsColumns,
		createSQLiteLogItemsTable,
		[]string{"idx_log_items_group_id", "idx_log_items_log_id", "idx_raw_msg_id", "idx_log_delete_by_id"},
	); err != nil {
		return err
	}
	if err := ensureSQLiteTableShape(
		dataDB,
		"attrs",
		sqliteAttrsTempTable,
		expectedSQLiteAttrsColumns,
		createSQLiteAttrsTable,
		[]string{"idx_attrs_attrs_type_id", "idx_attrs_binding_sheet_id", "idx_attrs_owner_id_id"},
	); err != nil {
		return err
	}
	if err := ensureSQLiteLogIndexes(logDB); err != nil {
		return err
	}
	if dataDB.Migrator().HasTable("attrs") {
		if err := ensureSQLiteAttrsIndexes(dataDB); err != nil {
			return err
		}
	}

	logf("SQLite 历史表结构兼容修复完成")
	return nil
}

func ensureSQLiteAttrsIndexes(db *gorm.DB) error {
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

func ensureSQLiteLogIndexes(db *gorm.DB) error {
	stmts := []string{
		"CREATE INDEX IF NOT EXISTS idx_logs_group ON `logs` (group_id)",
		"CREATE INDEX IF NOT EXISTS idx_logs_updated_at ON `logs` (updated_at)",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_log_group_id_name ON `logs` (group_id, name)",
		"CREATE INDEX IF NOT EXISTS idx_log_items_group_id ON `log_items` (group_id)",
		"CREATE INDEX IF NOT EXISTS idx_log_items_log_id ON `log_items` (log_id)",
		"CREATE INDEX IF NOT EXISTS idx_raw_msg_id ON `log_items` (raw_msg_id)",
		"CREATE INDEX IF NOT EXISTS idx_log_delete_by_id ON `log_items` (group_id, raw_msg_id, id)",
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}

var V160SQLiteSchemaRepairMigration = upgrade.Upgrade{
	ID:    "008b_V160SQLiteSchemaRepairMigration",
	Phase: upgrade.PhasePostBootstrap,
	Description: `
# 升级说明
将旧版 SQLite 表结构修复迁移为显式升级步骤，不再由 bootstrap 隐式改写已有表。
`,
	ShouldRun: func(operator operator.DatabaseOperator) (bool, error) {
		if operator.Type() != constant.SQLITE {
			return false, nil
		}
		dataDB := operator.GetDataDB(constant.READ)
		logDB := operator.GetLogDB(constant.READ)

		needLogs, err := needsSQLiteTableShapeRepair(logDB, "logs", expectedSQLiteLogsColumns)
		if err != nil {
			return false, err
		}
		if needLogs {
			return true, nil
		}
		needLogItems, err := needsSQLiteTableShapeRepair(logDB, "log_items", expectedSQLiteLogItemsColumns)
		if err != nil {
			return false, err
		}
		if needLogItems {
			return true, nil
		}
		return needsSQLiteTableShapeRepair(dataDB, "attrs", expectedSQLiteAttrsColumns)
	},
	Apply: func(logf func(string), operator operator.DatabaseOperator) error {
		logf("[INFO] V160 SQLite 历史表结构修复开始")
		if err := V160SQLiteSchemaRepairMigrate(operator, logf); err != nil {
			return err
		}
		logf("[INFO] V160 SQLite 历史表结构修复完毕")
		return nil
	},
}

func needsSQLiteTableShapeRepair(db *gorm.DB, table string, expected []sqliteExpectedColumn) (bool, error) {
	if !db.Migrator().HasTable(table) {
		return false, nil
	}
	columns, err := loadSQLiteTableColumns(db, table)
	if err != nil {
		return false, err
	}
	return !sqliteColumnsMatch(columns, expected), nil
}

func init() {
	if sqliteCopyBatchSize <= 0 {
		panic(errors.New("sqliteCopyBatchSize must be positive"))
	}
}

func ensureSQLiteTableShape(
	db *gorm.DB,
	table string,
	tempTable string,
	expected []sqliteExpectedColumn,
	createTable func(*gorm.DB, string) error,
	indexesToDrop []string,
) error {
	if !db.Migrator().HasTable(table) {
		return nil
	}

	columns, err := loadSQLiteTableColumns(db, table)
	if err != nil {
		return err
	}
	if sqliteColumnsMatch(columns, expected) {
		return nil
	}

	return recreateSQLiteTable(db, table, tempTable, expected, createTable, indexesToDrop, columns)
}

func recreateSQLiteTable(
	db *gorm.DB,
	table string,
	tempTable string,
	expected []sqliteExpectedColumn,
	createTable func(*gorm.DB, string) error,
	indexesToDrop []string,
	actual []sqlitePragmaColumn,
) error {
	if err := dropSQLiteIndexes(db, indexesToDrop); err != nil {
		return err
	}

	actualMap := make(map[string]sqlitePragmaColumn, len(actual))
	for _, col := range actual {
		actualMap[strings.ToLower(col.Name)] = col
	}
	if _, ok := actualMap["id"]; !ok {
		return fmt.Errorf("%s 表缺少 id 列，无法迁移", table)
	}

	if err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", quoteSQLiteIdentifier(tempTable))).Error; err != nil {
		return err
	}
	if err := createTable(db, tempTable); err != nil {
		return err
	}

	insertColumns := make([]string, 0, len(expected))
	selectColumns := make([]string, 0, len(expected))
	for _, exp := range expected {
		insertColumns = append(insertColumns, quoteSQLiteIdentifier(exp.Name))
		if _, ok := actualMap[strings.ToLower(exp.Name)]; ok {
			selectColumns = append(selectColumns, quoteSQLiteIdentifier(exp.Name))
		} else {
			selectColumns = append(selectColumns, defaultValueForMissingColumn(exp.Name))
		}
	}

	if err := bulkCopySQLiteTable(db, table, tempTable, insertColumns, selectColumns); err != nil {
		return err
	}
	if err := db.Exec(fmt.Sprintf("DROP TABLE %s", quoteSQLiteIdentifier(table))).Error; err != nil {
		return err
	}
	return db.Exec(
		fmt.Sprintf("ALTER TABLE %s RENAME TO %s", quoteSQLiteIdentifier(tempTable), quoteSQLiteIdentifier(table)),
	).Error
}

func loadSQLiteTableColumns(db *gorm.DB, table string) ([]sqlitePragmaColumn, error) {
	query := fmt.Sprintf("PRAGMA table_info(%s)", quoteSQLiteString(table))
	var columns []sqlitePragmaColumn
	if err := db.Raw(query).Scan(&columns).Error; err != nil {
		return nil, err
	}
	return columns, nil
}

func sqliteColumnsMatch(actual []sqlitePragmaColumn, expected []sqliteExpectedColumn) bool {
	if len(actual) != len(expected) {
		return false
	}
	actualMap := make(map[string]sqlitePragmaColumn, len(actual))
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

func createSQLiteAttrsTable(db *gorm.DB, table string) error {
	if table == "" {
		table = "attrs"
	}
	err := db.Exec(fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
	id TEXT PRIMARY KEY,
	data BLOB,
	attrs_type TEXT DEFAULT NULL,
	binding_sheet_id TEXT DEFAULT '',
	name TEXT DEFAULT '',
	owner_id TEXT DEFAULT '',
	sheet_type TEXT DEFAULT '',
	is_hidden BOOLEAN DEFAULT FALSE,
	created_at INTEGER DEFAULT 0,
	updated_at INTEGER DEFAULT 0
)`, quoteSQLiteIdentifier(table))).Error
	if err != nil && strings.Contains(err.Error(), "already exists") {
		return nil
	}
	return err
}

func createSQLiteLogsTable(db *gorm.DB, table string) error {
	stmt := `
CREATE TABLE IF NOT EXISTS %s (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT,
	group_id TEXT,
	created_at INTEGER,
	updated_at INTEGER,
	size INTEGER,
	extra TEXT,
	upload_url TEXT,
	upload_time INTEGER
)`
	err := db.Exec(fmt.Sprintf(stmt, quoteSQLiteIdentifier(table))).Error
	if err != nil && strings.Contains(err.Error(), "already exists") {
		return nil
	}
	return err
}

func createSQLiteLogItemsTable(db *gorm.DB, table string) error {
	stmt := `
CREATE TABLE IF NOT EXISTS %s (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	log_id INTEGER,
	group_id TEXT,
	nickname TEXT,
	im_userid TEXT,
	time INTEGER,
	message TEXT,
	is_dice INTEGER,
	command_id INTEGER,
	command_info TEXT,
	raw_msg_id TEXT,
	user_uniform_id TEXT,
	removed INTEGER,
	parent_id INTEGER
)`
	err := db.Exec(fmt.Sprintf(stmt, quoteSQLiteIdentifier(table))).Error
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
	for startRow := minRow.Value.Int64; startRow <= maxRow.Value.Int64; startRow += sqliteCopyBatchSize {
		endRow := startRow + sqliteCopyBatchSize - 1
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

		copied := startRow - minRow.Value.Int64
		total := maxRow.Value.Int64 - minRow.Value.Int64
		if total > 0 {
			log.Infof("已迁移 %d/%d 行到 %s - %.2f%%", copied, total, idDst, float64(copied)/float64(total)*100)
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
