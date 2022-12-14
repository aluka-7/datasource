package base

import (
	"time"
)

type Entity struct {
	Id             int64 `xorm:"pk autoincr bigint comment('主键/周进/2020-02-02')" json:"id"`
	CreateBy       int64 `xorm:"bigint not null comment('创建者/周进/2020-02-02')" json:"createBy"`
	CreateTime     int64 `xorm:"bigint not null comment('创建时间/周进/2020-02-02')" json:"createTime"`
	LastModifyBy   int64 `xorm:"bigint null COMMENT('记录最后修改者/黄旭龙/2020-02-02')" json:"lastModifyBy"`
	LastModifyTime int64 `xorm:"bigint null COMMENT('记录最后修改时间/黄旭龙/2020-02-02')" json:"lastModifyTime"`
}

func (t *Entity) BeforeInsert() {
	if t.CreateTime == 0 {
		t.CreateTime = time.Now().Unix()
	}
}

func (t *Entity) BeforeUpdate() {
	if t.LastModifyTime == 0 {
		t.LastModifyTime = time.Now().Unix()
	}
}
