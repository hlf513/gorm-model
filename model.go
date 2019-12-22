package gorm_model

import (
	"errors"
	"fmt"

	"github.com/jinzhu/gorm"
	"github.com/t-tiger/gorm-bulk-insert"
)

type Model struct {
	db *gorm.DB
	// 有效数据查询条件
	validCondition map[string]interface{}
	deletedKey     string
}

// NewModel 初始化一个 Model 结构体
func NewModel(db *gorm.DB) *Model {
	model := &Model{
		db: db,
	}
	model.SetSoftDeletedKey("is_deleted")
	return model
}

// SetSoftDeletedKey 设置软删除字段
func (m *Model) SetSoftDeletedKey(key string) {
	m.deletedKey = key
	m.validCondition = map[string]interface{}{
		fmt.Sprintf("%s = ?", m.deletedKey): "N",
	}
}

// ClearValidCondition 删除预设的全局查询条件
func (m *Model) ClearValidCondition() {
	m.validCondition = nil
}

// prepare  初始化 select 字段，where 条件；返回 DB
func (m *Model) prepare(fields interface{}, conditions ...map[string]interface{}) *gorm.DB {
	q := m.db
	// 处理 查询字段
	if fields != nil {
		selects := fields.([]interface{})
		if len(selects) != 0 {
			q = q.Select(selects[0])
		}
	}

	// 处理 查询条件
	where := make(map[string]interface{})
	if m.validCondition != nil {
		for k, v := range m.validCondition {
			where[k] = v
		}
	}
	for _, condition := range conditions {
		for k, v := range condition {
			// 会覆盖 m.validCondition
			where[k] = v
		}
	}
	for k, v := range where {
		if v != nil {
			q = q.Where(k, v)
		} else {
			q = q.Where(k)
		}
	}
	return q
}

// IsNil 是否未找到记录
func (m *Model) IsNil(err error) bool {
	return gorm.IsRecordNotFoundError(err)
}

// Create 新增一条数据
func (m *Model) Create(data interface{}) error {
	if m.db.NewRecord(data) {
		if err := m.db.Create(data).Error; err != nil {
			return err
		}
		return nil
	}
	return errors.New("this is not a new record")
}

// BatchInsert 批量插入数据
func (m *Model) BatchInsert(data []interface{}, onceCount int, excludeCols ...string) error {
	return gormbulk.BulkInsert(m.db, data, onceCount, excludeCols...)
}

// FetchOneById 通过 ID 查询一条数据
func (m *Model) FetchOneById(data interface{}, fields ...interface{}) error {
	if m.db.NewRecord(data) {
		return primaryKeyBlankError()
	}
	if err := m.prepare(fields).Limit(1).Find(data).Error; err != nil {
		return err
	}

	return nil
}

// FetchOneByWhere 通过 where 查询一条数据
func (m *Model) FetchOneByWhere(where map[string]interface{}, data interface{}, fields ...interface{}) error {
	if !m.db.NewRecord(data) {
		return primaryKeyNoBlankError()
	}
	if err := m.prepare(fields, where).Limit(1).Find(data).Error; err != nil {
		return err
	}

	return nil
}

// FetchByIds 通过 ids 查询多条记录
func (m *Model) FetchAllByIds(ids, data, order interface{}, fields ...interface{}) error {
	db := m.prepare(fields)
	if order != nil {
		db = db.Order(order)
	}
	if err := db.Find(data, ids).Error; err != nil {
		return err
	}

	return nil
}

// FetchAllByWhere 通过 where 查询多条记录
func (m *Model) FetchAllByWhere(where map[string]interface{}, data, order interface{}, fields ...interface{}) error {
	db := m.prepare(fields, where)
	if order != nil {
		db = db.Order(order)
	}
	if err := db.Find(data).Error; err != nil {
		return err
	}

	return nil
}

// SearchOne 通过复杂搜索条件查询一条数据
func (m *Model) SearchOne(
	tableName, fields string,
	where map[string]interface{},
	data, order interface{},
	groupHaving ...string,
) error {
	db := m.prepare(nil, where).Select(fields).Table(tableName).Limit(1)
	if order != nil {
		db = db.Order(order)
	}
	l := len(groupHaving)
	if l > 0 {
		db = db.Group(groupHaving[0])
		if l > 1 {
			db = db.Having(groupHaving[1])
		}
	}
	if err := db.Scan(data).Error; err != nil {
		return err
	}

	return nil
}

// SearchAll 通过复杂搜索条件批量查询数据
func (m *Model) SearchAll(
	tableName, fields string,
	where map[string]interface{},
	data, order interface{},
	offset, limit int,
	groupHaving ...string,
) error {
	db := m.prepare(nil, where).Select(fields).Table(tableName).Offset(offset).Limit(limit)
	if order != nil {
		db = db.Order(order)
	}
	l := len(groupHaving)
	if l > 0 {
		db = db.Group(groupHaving[0])
		if l > 1 {
			db = db.Having(groupHaving[1])
		}
	}
	if err := db.Scan(data).Error; err != nil {
		return err
	}

	return nil
}

// Count 统计总数
func (m *Model) Count(tableName string, where map[string]interface{}, groupHaving ...string) (int, error) {
	c := 0

	db := m.prepare(nil, where).Table(tableName)
	l := len(groupHaving)
	if l > 0 {
		db = db.Group(groupHaving[0])
		if l > 1 {
			db = db.Having(groupHaving[1])
		}
	}
	if err := db.Count(&c).Error; err != nil {
		return 0, err
	}

	return c, nil
}

// UpdateOneById 根据 ID 更新一条数据
func (m *Model) UpdateOneById(set map[string]interface{}, data interface{}) error {
	if m.db.NewRecord(data) {
		return primaryKeyBlankError()
	}
	// 自动更新 update_at 字段
	if err := m.db.Model(data).Limit(1).Update(set).Error; err != nil {
		return err
	}

	return nil
}

// UpdateAllByWhere 根据 where 批量更新数据
func (m *Model) UpdateAllByWhere(tableName string, where, set map[string]interface{}) error {
	// 自动更新 update_at 字段
	if err := m.prepare(nil, where).Table(tableName).Update(set).Error; err != nil {
		return err
	}

	return nil
}

// DeleteOneById 根据 ID 删除数据；默认是软删除
func (m *Model) DeleteOneById(data interface{}, force ...bool) error {
	if m.db.NewRecord(data) {
		return primaryKeyBlankError()
	}
	if len(force) > 0 && force[0] == true {
		if err := m.db.Limit(1).Delete(data).Error; err != nil {
			return err
		}
	} else {
		return m.UpdateOneById(map[string]interface{}{
			m.deletedKey: "Y",
		}, data)
	}

	return nil
}

// DeleteAllByWhere 根据 where 删除数据，默认是软删除
func (m *Model) DeleteAllByWhere(tableName string, where map[string]interface{}, force ...bool) error {
	if len(force) > 0 && force[0] == true {
		if err := m.prepare(nil, where).Table(tableName).Delete(nil).Error; err != nil {
			return err
		}
	} else {
		return m.UpdateAllByWhere(tableName, where, map[string]interface{}{
			m.deletedKey: "Y",
		})
	}

	return nil
}