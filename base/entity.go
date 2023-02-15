package base

import (
	"time"
)

type Entity struct {
	Id             int64 `xorm:"pk autoincr bigint"`
	CreateBy       int64 `xorm:"bigint not null"`
	CreateTime     int64 `xorm:"bigint not null"`
	LastModifyBy   int64 `xorm:"bigint null"`
	LastModifyTime int64 `xorm:"bigint null"`
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
