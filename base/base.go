package base

import (
	"context"

	"github.com/aluka-7/common"
	"github.com/aluka-7/datasource/search"
	"xorm.io/xorm"
)

type IBaseRepository interface {
	Save(bean interface{}) (int64, error)
	Update(id int64, bean interface{}, cols ...string) (int64, error)
	ReadById(ctx context.Context, id int64, bean interface{}, cols ...string) (bool, error)
	Query(ctx context.Context, cq common.Query, list interface{}, count interface{}, cols ...string) (page *common.Pagination, err error)
	Session(ctx context.Context) *xorm.Session
	TxSave(tx *xorm.Session, bean interface{}) (int64, error)
	TxUpdate(tx *xorm.Session, id int64, bean interface{}, cols ...string) (int64, error)
}

func NewBaseRepository(orm *xorm.Engine, column map[string]search.Filter) BaseRepository {
	return BaseRepository{orm: orm, column: column}
}

type BaseRepository struct {
	orm    *xorm.Engine
	column map[string]search.Filter
}

func (b *BaseRepository) Xorm() *xorm.Engine {
	return b.orm
}

func (b *BaseRepository) Save(bean interface{}) (int64, error) {
	return b.orm.Insert(bean)
}
func (b *BaseRepository) Update(id int64, bean interface{}, cols ...string) (int64, error) {
	s := b.orm.ID(id)
	if len(cols) > 0 {
		s.Cols(cols...)
	}
	return s.Update(bean)
}

func (b *BaseRepository) ReadById(ctx context.Context, id int64, bean interface{}, cols ...string) (bool, error) {
	s := b.orm.Context(ctx).ID(id)
	if len(cols) > 0 {
		s.Cols(cols...)
	}
	return s.Get(bean)
}

func (b *BaseRepository) Query(ctx context.Context, cq common.Query, list interface{}, count interface{}, cols ...string) (page *common.Pagination, err error) {
	query := search.NewQuery(cq)
	session := b.orm.Context(ctx)
	query.MarkOrmFiltered(b.column, session)
	order := query.MarkOrder(b.column)
	page = query.MarkPage()
	limit, offset := page.Limit()
	if order != nil {
		session.OrderBy(order.ToString())
	}
	session.Limit(limit, offset)
	if len(cols) > 0 {
		session.Cols(cols...)
	}
	var total int64
	if total, err = session.FindAndCount(list, count); err == nil {
		page.SetTotalRecord(int(total))
	}
	return
}

func (b *BaseRepository) Session(ctx context.Context) *xorm.Session {
	return b.orm.NewSession().Context(ctx)
}

func (b *BaseRepository) TxSave(tx *xorm.Session, bean interface{}) (int64, error) {
	return tx.Insert(bean)
}

func (b *BaseRepository) TxUpdate(tx *xorm.Session, id int64, bean interface{}, cols ...string) (int64, error) {
	tx = tx.ID(id)
	if len(cols) > 0 {
		tx.Cols(cols...)
	}
	return tx.Update(bean)
}
