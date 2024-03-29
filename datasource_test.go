package datasource_test

import (
	"context"
	"fmt"
	"testing"
	"xorm.io/xorm"

	"github.com/aluka-7/common"
	"github.com/aluka-7/configuration"
	"github.com/aluka-7/configuration/backends"
	"github.com/aluka-7/datasource"
	"github.com/aluka-7/datasource/base"
	"github.com/aluka-7/datasource/search"
	_ "github.com/mattn/go-sqlite3"
	. "github.com/smartystreets/goconvey/convey"
)

type Test struct {
	base.Entity   `xorm:"extends"`
	Email         string `xorm:"varchar(100) notnull comment('邮箱')"`
	LoginName     string `xorm:"varchar(25) notnull comment('登陆')"`
	Name          string `xorm:"varchar(25) notnull comment('姓名')"`
	PlainPassword string `xorm:"varchar(16) notnull comment('原始密码')"`
	ShaPassword   string `xorm:"varchar(150) notnull comment('SHA后的密码')"`
}

var conf configuration.Configuration

func initConfig(t *testing.T) {
	conf = configuration.MockEngine(t, backends.StoreConfig{Exp: map[string]string{
		"/system/base/datasource/privileges": "{\"1000\":\"[\"1200\",\"1300\"]\"}",
		"/system/base/datasource/common":     "{\"dialect\":\"sqlite3\",\"debug\":true,\"enableLog\":false,\"minPoolSize\":2,\"maxPoolSize\":10,\"idleTimeout\":\"10s\",\"queryTimeout\":\"2s\",\"execTimeout\":\"2s\",\"tranTimeout\":\"2s\"}",
		"/system/base/datasource/1000":       "{\"dsn\":\"./test.db\",\"prefix\":\"os_1000_\"}",
	}})
}

func TestDataSource(t *testing.T) {
	initConfig(t)
	Convey("test DataSource", t, func() {
		orm := datasource.Engine(conf, "1000").Orm("")
		var slaveOrm []*xorm.Engine
		slaveOrm = append(slaveOrm, datasource.Engine(conf, "1000").Orm(""))
		Convey("Test a sync", func() {
			err := orm.DropTables(new(Test))
			So(err, ShouldBeNil)
			err = orm.Sync2(new(Test))
			So(err, ShouldBeNil)
		})

		Convey("Test a insert", func() {
			actual, err := orm.Insert(&Test{Email: "test@xxxx.cn", LoginName: "test", Name: "单元测试", PlainPassword: "A+1234567890", ShaPassword: "qw42#2(*^%", Entity: base.Entity{CreateBy: 1}})
			So(err, ShouldBeNil)
			expected := int64(1)
			So(actual, ShouldEqual, expected)
		})
		Convey("Test Orm Transaction", func() {
			repo := base.NewBaseRepository(orm, slaveOrm, map[string]search.Filter{
				"email": {FieldName: "email", Operator: search.LIKE},
				"id":    {FieldName: "id", Operator: search.IN},
			})
			se := repo.Session(context.Background())
			err := se.Begin()
			defer se.Close()
			So(err, ShouldBeNil)
			var t = Test{Email: "admin@xxxx.cn", LoginName: "admin", Name: "单元测试", PlainPassword: "A+1234567890", ShaPassword: "qw42#2(*^%", Entity: base.Entity{CreateBy: 1}}
			actual, err := repo.TxSave(se, &t)
			fmt.Printf("%+v\n", t)
			So(err, ShouldBeNil)
			So(actual, ShouldEqual, 1)
			num, err := repo.TxUpdate(se, 2, &Test{Entity: base.Entity{LastModifyBy: 1}})
			So(err, ShouldBeNil)
			So(num, ShouldEqual, 1)
			var ts []Test
			err = se.Find(&ts)
			So(err, ShouldBeNil)
			fmt.Printf("%+v\n", ts)
			err = se.Commit()
			So(err, ShouldBeNil)
		})
		Convey("Test Orm Base Query", func() {
			var val []Test
			repo := base.NewBaseRepository(orm, slaveOrm, map[string]search.Filter{
				"email": {FieldName: "email", Operator: search.LIKE},
				"id":    {FieldName: "id", Operator: search.IN},
			})
			query := common.Query{
				PageSize: 10, Page: 0, Sorted: []struct {
					Id   string `json:"id"`
					Desc bool   `json:"desc"`
				}{{Id: "email", Desc: true}}, Filtered: []struct {
					Id    string      `json:"id"`
					Value interface{} `json:"value"`
				}{{Id: "email", Value: "test@xxxx.cn"}, {Id: "id", Value: "1,2,3"}}}
			page, err := repo.Query(context.Background(), query, &val, &Test{})

			So(err, ShouldBeNil)
			expected := int64(1)
			fmt.Println(page.TotalPages, page.TotalRecords)
			So(page.TotalPages, ShouldEqual, expected)
			So(page.TotalRecords, ShouldEqual, expected)
		})
		Convey("Test Search Query MarkOrmFiltered", func() {
			session := orm.NewSession()
			var val []Test
			query := search.NewQuery(common.Query{
				PageSize: 10, Page: 0, Sorted: []struct {
					Id   string `json:"id"`
					Desc bool   `json:"desc"`
				}{{Id: "email", Desc: true}}, Filtered: []struct {
					Id    string      `json:"id"`
					Value interface{} `json:"value"`
				}{{Id: "email", Value: "test@xxxx.cn"}, {Id: "id", Value: []int{1, 2, 3}}}})
			column := map[string]search.Filter{
				"email": {FieldName: "email", Operator: search.LIKE},
				"id":    {FieldName: "id", Operator: search.IN},
			}
			query.MarkOrmFiltered(column, session)
			order := query.MarkOrder(column)
			limit, offset := query.MarkPage().Limit()
			err := session.OrderBy(order.ToString()).Limit(limit, offset).Find(&val)
			fmt.Println(val)
			So(err, ShouldBeNil)
			actual := len(val)
			expected := int64(1)
			So(actual, ShouldEqual, expected)
		})
		Convey("Test Search Builder Orm Filter EQ", func() {
			filters := []search.Filter{
				{"Email", "test@xxxx.cn", search.EQ},
			}
			session := orm.NewSession()
			var val []Test
			err := search.OrmFilter(filters, session).Find(&val)
			fmt.Println(val)
			So(err, ShouldBeNil)
			actual := len(val)
			expected := int64(1)
			So(actual, ShouldEqual, expected)
		})

		Convey("Test Delete All", func() {
			mn, err := orm.Exec("DELETE FROM `os_1000_test`")
			So(err, ShouldBeNil)
			actual, err := mn.RowsAffected()
			So(err, ShouldBeNil)
			expected := int64(2)
			So(actual, ShouldEqual, expected)
		})
		Reset(func() {
			orm.Close()
		})
	})
}
