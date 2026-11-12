// Package builtin 链接全部内置数据库驱动：空导入它，四个驱动就各自注册进 dbdriver。
package builtin

import (
	_ "github.com/octays/opskat/internal/dbdriver/mssql"
	_ "github.com/octays/opskat/internal/dbdriver/mysql"
	_ "github.com/octays/opskat/internal/dbdriver/postgresql"
	_ "github.com/octays/opskat/internal/dbdriver/sqlite"
)
