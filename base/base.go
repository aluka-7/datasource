package base

import (
	"context"
	"github.com/aluka-7/common"
	"github.com/aluka-7/datasource/search"
	"xorm.io/xorm"
	"math/rand"
	"reflect"
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

func NewBaseRepository(orm *xorm.Engine, column map[string]search.Filter,slaves ...[]*xorm.Engine) BaseRepository {
	var s []*xorm.Engine = nil
	if len(slaves)>0{
		s =slaves[0]
	}
	return BaseRepository{orm: orm, column: column,slaves:s}
}

//type ISession interface {
//	xorm.Session
//}

type BaseRepository struct {
	orm    *xorm.Engine
	column map[string]search.Filter
	slaves []*xorm.Engine

}

func (b *BaseRepository) Ctx(ctx context.Context) BaseRepository {
	session := b.orm.NewSession().Context(ctx).Engine()
	slaCtxs :=[]*xorm.Engine{}
	for _, slave := range b.slaves {
		slaCtxs = append(slaCtxs, slave.NewSession().Context(ctx).Engine())
	}

	return BaseRepository{orm: session, column: b.column ,slaves:slaCtxs}
}

func (b *BaseRepository) Slave() *xorm.Engine {
	return b.slaves[rand.Intn(len(b.slaves))]
}

func (b *BaseRepository) Xorm() *xorm.Engine {
	return b.orm
}

func (b *BaseRepository) Save(bean interface{}) (int64, error) {
	b.orm.Insert(bean)
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

//type BaseRepository struct {
//	orm *xorm.Engine
//}

func (m *BaseRepository) Insert(entity interface{}) error {
	_, err := m.orm.Insert(entity)
	return err
}

func (m *BaseRepository) InsertBatch(entityList interface{}) error {
	session := m.orm.NewSession()
	defer session.Close()

	err := session.Begin()
	if err != nil {
		return err
	}

	_, err = session.InsertMulti(entityList)
	if err != nil {
		session.Rollback()
		return err
	}

	err = session.Commit()
	if err != nil {
		session.Rollback()
		return err
	}

	return nil
}

func (m *BaseRepository) InsertOrUpdate(queryWrapper func() *xorm.Session,entity interface{}) error {

	originalType := reflect.TypeOf(entity)           // 获取原始结构体的类型

	exist, err :=queryWrapper().NoAutoCondition().Get(reflect.New(originalType).Interface())
	if err != nil {
		return err
	}

	if exist {
		_, err = queryWrapper().Update(entity)
	} else {
		_, err = queryWrapper().Insert(entity)
	}

	return err
}

func (m *BaseRepository) InsertOrUpdateBatch(entityList []interface{}) error {
	session := m.orm.NewSession()
	defer session.Close()

	err := session.Begin()
	if err != nil {
		return err
	}

	for _, entity := range entityList {
		exist, err := session.Get(entity)
		if err != nil {
			session.Rollback()
			return err
		}

		if exist {
			_, err = session.Update(entity)
		} else {
			_, err = session.Insert(entity)
		}

		if err != nil {
			session.Rollback()
			return err
		}
	}

	err = session.Commit()
	if err != nil {
		session.Rollback()
		return err
	}

	return nil
}

func (m *BaseRepository) SelectById(id interface{}, entity interface{}) error {
	_, err := m.Slave().ID(id).Get(entity)
	return err
}

func (m *BaseRepository) SelectBatchIds(idList []interface{}, entityList interface{}) error {
	err := m.Slave().In("id", idList...).Find(entityList)
	return err
}

func (m *BaseRepository) SelectList(queryWrapper *xorm.Session, entityList interface{}) error {
	err := queryWrapper.Find(entityList)
	return err
}

func (m *BaseRepository) SelectAll(entityList interface{}) error {
	err := m.Slave().Find(entityList)
	return err
}

func (m *BaseRepository) Count(queryWrapper *xorm.Session, entity interface{}) (int64, error) {
	count, err := queryWrapper.Count()
	return count, err
}

func (m *BaseRepository) SelectOne(queryWrapper *xorm.Session, entity interface{}) error {
	_, err := queryWrapper.Get(entity)
	return err
}


func (m *BaseRepository) SelectPage(page, limit int, queryWrapper *xorm.Session, entityList interface{}) (PageData,error) {
	count, err := queryWrapper.Limit(limit, (page-1)*limit).FindAndCount(entityList)
	//queryWrapper
	//session := *queryWrapper.
	//count, err := m.Count(queryWrapper, nil)
	//err = queryWrapper.Limit(limit, (page-1)*limit).Find(entityList)


	pageData := PageData{}

	if err != nil {
		return pageData, err
	}
	pageData.Total = count
	pageData.List = entityList
	pageData.PageSize = limit
	pageData.PageNum = page
	return pageData, err
}

func (m *BaseRepository) Update2(entity interface{}, updateWrapper *xorm.Session) error {
	_, err := updateWrapper.Update(entity)
	return err
}

func (m *BaseRepository) UpdateById(entity interface{}) error {
	_, err := m.orm.ID(entity).Update(entity)
	return err
}

func (m *BaseRepository) DeleteById(id interface{}, entity interface{}) error {
	_, err := m.orm.ID(id).Delete(entity)
	return err
}

func (m *BaseRepository) DeleteBatchIds(idList []interface{}, entity interface{}) error {
	_, err := m.orm.In("id", idList...).Delete(entity)
	return err
}

func (m *BaseRepository) Delete(queryWrapper *xorm.Session, entity interface{}) error {
	_, err := queryWrapper.Delete(entity)
	return err
}

func (m *BaseRepository) DeleteAll(queryWrapper *xorm.Session, entity interface{}) error {
	_, err := queryWrapper.Delete(entity)
	return err
}

func (m *BaseRepository) DeleteByIdWithFill(entity interface{}) error {
	_, err := m.orm.ID(entity).Update(entity)
	return err
}

func (m *BaseRepository) DeleteByIdsWithFill(entityList interface{}) error {
	session := m.orm.NewSession()
	defer session.Close()

	err := session.Begin()
	if err != nil {
		return err
	}

	_, err = session.ID(entityList).Update(entityList)
	if err != nil {
		session.Rollback()
		return err
	}

	err = session.Commit()
	return err
}

func (m *BaseRepository) DeleteWithFill(entity interface{}) error {
	session := m.orm.NewSession()
	defer session.Close()

	err := session.Begin()
	if err != nil {
		return err
	}

	_, err = session.Delete(entity)
	if err != nil {
		session.Rollback()
		return err
	}

	err = session.Commit()
	return err
}



type BaseService struct {
	mapper BaseRepository
}

func NewBaseService(orm *xorm.Engine, slaves []*xorm.Engine) *BaseService {
	return &BaseService{mapper: BaseRepository{orm: orm, slaves: slaves}}
}



func (s *BaseService) Create(entity interface{}) error {
	return s.mapper.Insert(entity)
}

func (s *BaseService) CreateBatch(entityList interface{}) error {
	return s.mapper.InsertBatch(entityList)
}

func (s *BaseService) CreateOrUpdate(queryWrapper func() *xorm.Session,entity interface{}) error {
	return s.mapper.InsertOrUpdate(queryWrapper,entity)
}

func (s *BaseService) CreateOrUpdateBatch(entityList []interface{}) error {
	return s.mapper.InsertOrUpdateBatch(entityList)
}

func (s *BaseService) GetByID(id interface{}, entity interface{}) error {
	return s.mapper.SelectById(id, entity)
}

func (s *BaseService) GetByIDs(idList []interface{}, entityList interface{}) error {
	return s.mapper.SelectBatchIds(idList, entityList)
}

func (s *BaseService) GetByQuery(queryWrapper *xorm.Session, entityList interface{}) error {
	return s.mapper.SelectList(queryWrapper, entityList)
}

func (s *BaseService) GetAll(entityList interface{}) error {
	return s.mapper.SelectAll(entityList)
}

func (s *BaseService) GetCount(queryWrapper *xorm.Session, entity interface{}) (int64, error) {
	return s.mapper.Count(queryWrapper, entity)
}

func (s *BaseService) GetOne(queryWrapper *xorm.Session, entity interface{}) error {
	return s.mapper.SelectOne(queryWrapper, entity)
}

func (s *BaseService) GetPagedResults(page, limit int, queryWrapper *xorm.Session, entityList interface{}) (PageData,error) {
	return s.mapper.SelectPage(page, limit, queryWrapper, entityList)
}

func (s *BaseService) Update(entity interface{}, updateWrapper *xorm.Session)(error ){
	return s.mapper.Update2(entity, updateWrapper)
}

func (s *BaseService) UpdateByID(entity interface{}) error {
	return s.mapper.UpdateById(entity)
}

func (s *BaseService) DeleteByID(id interface{}, entity interface{}) error {
	return s.mapper.DeleteById(id, entity)
}

func (s *BaseService) DeleteByIDs(idList []interface{}, entity interface{}) error {
	return s.mapper.DeleteBatchIds(idList, entity)
}

func (s *BaseService) DeleteByQuery(queryWrapper *xorm.Session, entity interface{}) error {
	return s.mapper.Delete(queryWrapper, entity)
}

func (s *BaseService) DeleteAll(queryWrapper *xorm.Session, entity interface{}) error {
	return s.mapper.DeleteAll(queryWrapper, entity)
}

func (s *BaseService) DeleteByIDWithFill(entity interface{}) error {
	return s.mapper.DeleteByIdWithFill(entity)
}

func (s *BaseService) DeleteByIDsWithFill(entityList interface{}) error {
	return s.mapper.DeleteByIdsWithFill(entityList)
}

func (s *BaseService) DeleteByQueryWithFill(entity interface{}) error {
	return s.mapper.DeleteWithFill(entity)
}

func (b *BaseService) Ctx(ctx context.Context) BaseService {

	session := b.mapper.orm.NewSession().Context(ctx).Engine()
	return BaseService{mapper: BaseRepository{orm:session , column: b.mapper.column ,slaves:b.mapper.slaves}}
}

func (b *BaseService) Slave() *xorm.Engine {
	return b.mapper.slaves[rand.Intn(len(b.mapper.slaves))]
}
func (b *BaseService) Master() *xorm.Engine {
	return b.mapper.Xorm()
}


func (s *BaseService) SessionSlave() *xorm.Session {
		return s.Slave().NewSession()
}
// Session s:slave m:master数据库
func (s *BaseService) SessionMaster() *xorm.Session {
	return s.mapper.orm.NewSession()
}
//type PageList struct {
//	List []RedeemCodeItem `json:"list"`
//	PageData
//}
type PageData struct {
	Total    int64       `json:"total"`     //总条数
	PageSize int         `json:"page_size"` //第几页
	PageNum  int         `json:"page_num"`  //每页个数
	List     interface{} `json:"list"`     //数据列表
}


