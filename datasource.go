package datasource

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aluka-7/configuration"
	"github.com/aluka-7/utils"
	"github.com/rs/zerolog/log"
	"xorm.io/xorm"
	xlog "xorm.io/xorm/log"
	"xorm.io/xorm/names"
)

type Config struct {
	Dialect      string                 `json:"dialect"`
	Dsn          string                 `json:"dsn"`
	Debug        bool                   `json:"debug"`
	EnableLog    bool                   `json:"enableLog"`
	Prefix       string                 `json:"prefix"`       // 表名前缀
	MinPoolSize  int                    `json:"minPoolSize"`  // pool最大空闲数
	MaxPoolSize  int                    `json:"maxPoolSize"`  // pool最大连接数
	IdleTimeout  utils.Duration         `json:"idleTimeout"`  // 连接最长存活时间
	QueryTimeout utils.Duration         `json:"queryTimeout"` // 查询超时时间
	ExecTimeout  utils.Duration         `json:"execTimeout"`  // 执行超时时间
	TranTimeout  utils.Duration         `json:"tranTimeout"`  // 事务超时时间
	Expand       map[string]interface{} `json:"expand"`
}
type dataSource struct {
	systemId   string
	cfg        configuration.Configuration
	privileges map[string][]string
}
type DataSource interface {
	Config(dsID string) *Config
	Orm(dsID string) *xorm.Engine
}

/**
 * 获取数据库引擎的唯一实例。
 *
 * @return
 */
func Engine(cfg configuration.Configuration, systemId string) DataSource {
	fmt.Println("Loading Datasource Engine")
	return &dataSource{cfg: cfg, systemId: systemId, privileges: make(map[string][]string, 0)}
}
func (d *dataSource) Config(dsID string) *Config {
	ds, dsID, err := d.getConfiguration(dsID, d.systemId)
	if len(ds.Dsn) == 0 || err != nil {
		panic(fmt.Sprintf("数据源[%s]配置未指定或者读取时发生错误:%+v", dsID, err))
	}
	return ds
}
func (d *dataSource) Orm(dsID string) *xorm.Engine {
	c := d.Config(dsID)
	eng, err := xorm.NewEngine(c.Dialect, c.Dsn)
	if err == nil {
		eng.ShowSQL(c.Debug) // 则会在控制台打印出生成的SQL语句
		if c.EnableLog {
			eng.Logger().SetLevel(xlog.LOG_DEBUG) // 则会在控制台打印调试及以上的信息
		}
		eng.SetTableMapper(names.NewPrefixMapper(names.SnakeMapper{}, c.Prefix))
		eng.SetMaxIdleConns(c.MinPoolSize)                   // 设置连接池的空闲数大小
		eng.SetMaxOpenConns(c.MaxPoolSize)                   // 设置最大打开连接数
		eng.SetConnMaxLifetime(time.Duration(c.IdleTimeout)) // 设置连接的最大生存时间
	} else {
		panic(fmt.Sprintf("初始化datasource引擎出错%+v", err))
	}
	return eng
}

/**
 * 获取指定标示的数据源的配置信息，返回的配置Config对象
 * 需要特别说明：如果给定的数据源标示为Null，则表明是要获取当前业务系统的默认数据源配置信息。
 * </p>
 *
 * @param dsID 数据源的标示，如果为Null则表明是默认数据源
 * @return
 */
func (d *dataSource) getConfiguration(dsID, csID string) (*Config, string, error) {
	config := &Config{}
	// 如果是获取默认的数据源，则使用当前系统的标示，否则鉴权
	if len(dsID) == 0 || dsID == csID {
		dsID = csID
	} else {
		plist := d.systemPrivileges(csID) // 数据库的访问权限鉴权
		if len(plist) == 0 || utils.ContainsString(plist, dsID) == -1 {
			return config, "", fmt.Errorf("系统[%s]无数据源[%s]的访问权限", csID, dsID)
		}
	}
	err := d.readFromConfiguration(dsID, config)
	return config, dsID, err
}

/**
加载数据库的访问权限鉴权
*/
func (d *dataSource) systemPrivileges(csID string) []string {
	d.cfg.Get("base", "datasource", "", []string{"privileges"}, d)
	plist := d.privileges[csID]
	fmt.Printf("系统[%s]的数据源权限:%s", csID, strings.Join(plist, ","))
	return plist
}
func (d *dataSource) Changed(data map[string]string) {
	for _, v := range data {
		var vl map[string][]string
		if err := json.Unmarshal([]byte(v), &vl); err == nil {
			for k, _v := range vl {
				d.privileges[k] = _v
			}
		}
	}
}
func (d *dataSource) readFromConfiguration(dsID string, config *Config) error {
	ex := d.readCommonProperties(config)
	if ex != nil {
		return ex
	}
	fmt.Printf("从配置中心读取数据源配置:/base/datasource/%s\n", dsID)
	ex = d.cfg.Clazz("base", "datasource", "", dsID, config)
	if ex != nil {
		log.Error().Err(ex).Msgf("数据源[%s]的配置获取失败", dsID)
	}
	return ex
}

func (d *dataSource) readCommonProperties(config *Config) error {
	fmt.Println("从配置中心的读取通用数据源配置:/base/datasource/common")
	vl, err := d.cfg.String("base", "datasource", "", "common")
	if err != nil {
		log.Error().Err(err).Msg("配置中心的通用数据源配置获取失败:%v")
	} else {
		if err = json.Unmarshal([]byte(vl), config); err != nil {
			log.Error().Err(err).Msg("解析数据源的通用配置失败")
		}
	}
	return err
}
