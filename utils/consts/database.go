package consts

const (
	SQLITE     = "sqlite"
	MYSQL      = "mysql"
	POSTGRESQL = "postgres"
)

type DBMode string

const (
	READ  DBMode = "read"
	WRITE DBMode = "write"
)
