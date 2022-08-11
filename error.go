package datasource

import (
	"github.com/go-sql-driver/mysql"
)

// 详细错误信息:https://dev.mysql.com/doc/mysql-errors/5.7/en/server-error-reference.html

type DbError uint16

const (
	DBDuplicateEntryKey = 1062
)

func CheckDBError(err error) (e DbError) {
	if v, ok := err.(*mysql.MySQLError); ok {
		e = DbError(v.Number)
	}
	return
}
