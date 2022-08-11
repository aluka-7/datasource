# 数据库引擎

获取指定标示的数据源的配置信息，返回的配置Config对象

需要特别说明:如果给定的数据源标示为Null,则表明是要获取当前业务系统的默认数据源配置信息。

## 统一配置中心配置信息

数据源权限地址:`/system/base/datasource/privileges`

```json
{"1000":{"1800","2000","2010"}}
```
数据源配置地址:`/system/base/datasource/{systemId}`
通用数据源配置地址：`/system/base/datasource/common`

```go
type Config struct {
    DbConfig                            // master
    Slaves []DbConfig `json:"slaves"`   // slaves
}

// 单个实例配置
type DbConfig struct {
    Dsn          string                 `json:"dsn"`
    Prefix       string                 `json:"prefix"`    
    Dialect      string                 `json:"dialect"`
    Debug        bool                   `json:"debug"`
    EnableLog    bool                   `json:"enableLog"`
    MinPoolSize  int                    `json:"minPoolSize"`  // pool最大空闲数
    MaxPoolSize  int                    `json:"maxPoolSize"`  // pool最大连接数
    IdleTimeout  utils.Duration         `json:"idleTimeout"`  // 连接最长存活时间
    QueryTimeout utils.Duration         `json:"queryTimeout"` // 查询超时时间
    ExecTimeout  utils.Duration         `json:"execTimeout"`  // 执行超时时间
    TranTimeout  utils.Duration         `json:"tranTimeout"`  // 事务超时时间
    Expand       map[string]interface{} `json:"expand"`
}
```
## 获取数据库引擎的唯一实例
```go
ds:=datasource.Engine()
```
获取数据源

```go
cfg:=ds.Config(dsID string)
```
注意:
1. 当dsID为空时使用当前系统的systemId
2. 当systemId与dsId不相同时会检查是否给授权此实例访问dsID对应的数据源

# SqlTemplate数据查询模板

```go
st:=ds.SqlTemplate(dsID string)
```
### SqlTemplate功能集合

* Open(c *Config) (*SqlTemplate, error)

* SqlTemplate:
    * Begin(c context.Context) (tx *Tx, err error)
    * Exec(c context.Context, query string, args ...interface{}) (res sql.Result, err error)
    * Prepare(query string) (*Stmt, error)
    * Prepared(query string) (stmt *Stmt)
    * Query(c context.Context, query string, args ...interface{}) (rows *Rows, err error)
    * QueryRow(c context.Context, query string, args ...interface{})
    * Close() (err error)
    * Ping(c context.Context) (err error)
* Tx:
    * Exec(query string, args ...interface{}) (res sql.Result, err error)
    * Query(query string, args ...interface{}) (rows *Rows, err error)
    * QueryRow(query string, args ...interface{}) *Row
    * Stmt(stmt *Stmt) *Stmt
    * Prepare(query string) (*Stmt, error)
    * Commit() (err error) 
    * Rollback() (err error)
* Stmt
    * Exec(c context.Context, args ...interface{}) (res sql.Result, err error)
    * Query(c context.Context, args ...interface{}) (rows *Rows, err error)
    * QueryRow(c context.Context, args ...interface{}) (row *Row)
    * Close() (err error)
* Row
    * Scan(dest ...interface{}) (err error)
* Rows
    * Close() (err error)
# 拼接SQL工具
`builder`是一个简单的sql查询字符串生成器

`builder`的递归结构调用,使您可以轻松构建sql字符串

## builder开始
```go
package main

import (
	"fmt"
	"github.com/aluka-7/datasource/builder"
)

func main() {
	var sb = builder.Select("u.id", "u.name", "u.age")
	sb.From("user", "AS u")
	sb.Where("u.id = ?", 1)
	sb.Limit(1)

	sqlStr, args, _ := sb.ToSql()
	fmt.Println("sqlStr:", sqlStr)
	fmt.Println("args:", args)
}
```
上述代码会输出如下内容：

```bash
sql: SELECT u.id, u.name, u.age FROM user AS u WHERE u.id = ? LIMIT ?
args: [10 1]
```

### Select

```go
var sb = builder.&SelectBuilder{}
sb.Selects("u.id", "u.name AS username", "u.age")
sb.Select(builder.Alias("b.amount", "user_amount"))
sb.From("user", "AS u")
sb.LeftJoin("bank", "AS b ON b.user_id = u.id")
sb.Where("u.id = ?", 1)
fmt.Println(sb.ToSql())
```

#### Insert

```go
var ib = builder.&InsertBuilder{}
ib.Table("user")
ib.Columns("name", "age")
ib.Values("用户1", 18)
ib.Values("用户2", 20)
fmt.Println(ib.ToSql())
```

#### Update
```go
var ub = builder.&UpdateBuilder{}
ub.Table("user")
ub.SET("name", "新的名字")
ub.Where("id = ? ", 1)
ub.Limit(1)
fmt.Println(ub.ToSql())
```

#### Delete

```go
var rb = builder.&DeleteBuilder{}
rb.Table("user")
rb.Where("id = ?", 1)
rb.Limit(1)
fmt.Println(rb.ToSql())
```
更多内容请参考 `builder_test.go` 文件。

# row转struct
提供了一些方便的功能,可将struct与Go标准库的database/sql包一起使用.
程序包将结构字段名称与Sql查询列名称匹配。
如果字段与字段名称不同,则字段也可以使用"sql"标签指定匹配的列。
就像'encoding/json'包一样,未导出的字段或标记有`sql:"-"`的字段将被忽略。

For example:

```go
    type T struct {
        F1 string
        F2 string `sql:"field2"`
        F3 string `sql:"-"`
    }
    rows, err := db.Query(fmt.Sprintf("SELECT %s FROM table_name", builder.Columns(T{})))
    ...
    for rows.Next() {
    	var t T
        err = builder.Scan(&t, rows)
        ...
    }
    err = rows.Err() // 获取迭代过程中遇到的任何错误
```

可以使用ColumnsAliased和ScanAliased函数将`sql`语句中的别名表扫描到由相同别名标识的特定结构中:

```go
    type User struct {
        Id int `sql:"id"`
        Username string `sql:"username"`
        Email string `sql:"address"`
        Name string `sql:"name"`
        HomeAddress *Address `sql:"-"`
    }
    type Address struct {
        Id int `sql:"id"`
        City string `sql:"city"`
        Street string `sql:"address"`
    }
    ...
    var user User
    var address Address
    sql := `
SELECT %s, %s FROM users AS u
INNER JOIN address AS a ON a.id = u.address_id
WHERE u.username = ?
`
    sql = fmt.Sprintf(sql, builder.ColumnsAliased(*user, "u"), builder.ColumnsAliased(*address, "a"))
    rows, err := db.Query(sql, "demo")
    if err != nil {
        log.Fatal(err)
    }
    defer rows.Close()
    if rows.Next() {
        err = builder.ScanAliased(&user, rows, "u")
        if err != nil {
            log.Fatal(err)
        }
        err = builder.ScanAliased(&address, rows, "a")
        if err != nil {
            log.Fatal(err)
        }
        user.HomeAddress = address
    }
    fmt.Printf("%+v", *user)
    // output: "{Id:1 Username:demo Email:demo@xxxx.cn Name:demo HomeAddress:0xc21001f570}"
    fmt.Printf("%+v", *user.HomeAddress)
    // output: "{Id:2 City:Vilnius Street:Plento 34}"
```
