package gorm_model

import (
	"errors"
	"fmt"

	"github.com/hlf513/gorm-bulk-insert"
	"github.com/jinzhu/gorm"
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
func (m *Model) BatchInsert(data []interface{}, onceCount int, action string, excludeCols ...string) error {
	return gormbulk.BulkInsert(m.db, data, onceCount, action, excludeCols...)
}

// FetchOneById 通过 ID 查询一条数据
func (m *Model) FetchOneById(table string, id int, data interface{}, fields ...interface{}) error {
	if !m.db.NewRecord(data) {
		return primaryKeyNoBlankError()
	}
	err := m.prepare(fields).Table(table).Limit(1).Find(data, id).Error
	if gorm.IsRecordNotFoundError(err) {
		return nil
	}
	if err != nil {
		return err
	}

	return nil
}

// FetchOneByWhere 通过 where 查询一条数据
func (m *Model) FetchOneByWhere(table string, where map[string]interface{}, data interface{}, fields ...interface{}) error {
	if !m.db.NewRecord(data) {
		return primaryKeyNoBlankError()
	}
	err := m.prepare(fields, where).Table(table).Limit(1).Find(data).Error
	if gorm.IsRecordNotFoundError(err) {
		return nil
	}
	if err != nil {
		return err
	}

	return nil
}

// FetchByIds 通过 ids 查询多条记录
func (m *Model) FetchAllByIds(table string, ids, data, order interface{}, fields ...interface{}) error {
	db := m.prepare(fields)
	if order != nil {
		db = db.Order(order)
	}
	if err := db.Table(table).Find(data, ids).Error; err != nil {
		return err
	}

	return nil
}

// FetchAllByWhere 通过 where 查询多条记录
func (m *Model) FetchAllByWhere(table string, where map[string]interface{}, data, order interface{}, fields ...interface{}) error {
	db := m.prepare(fields, where)
	if order != nil {
		db = db.Order(order)
	}
	if err := db.Table(table).Find(data).Error; err != nil {
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
	err := db.Scan(data).Error
	if gorm.IsRecordNotFoundError(err) {
		return nil
	}
	if err != nil {
		return err
	}

	return nil
}

// SearchAll 通过复杂搜索条件批量查询数据
func (m *Model) SearchAll(
	tableName, fields string,
	where map[string]interface{},
	data, order interface{},
	total *int,
	offset, limit int,
	groupHaving ...string,
) error {
	db := m.prepare(nil, where).Select(fields).Table(tableName)
	l := len(groupHaving)
	if l > 0 {
		db = db.Group(groupHaving[0])
		if l > 1 {
			db = db.Having(groupHaving[1])
		}
	}
	if total != nil {
		db.Count(total)
	}
	if order != nil {
		db = db.Order(order)
	}
	if limit > 0 {
		db = db.Offset(offset).Limit(limit)
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

// UpdateOneByWhere 根据 where 更新一条数据
func (m *Model) UpdateOneByWhere(model interface{}, where, set map[string]interface{}) error {
	if !m.db.NewRecord(model) {
		return primaryKeyNoBlankError()
	}
	db := m.prepare(nil, where).Model(model)
	// 自动更新 update_at 字段
	if err := db.Limit(1).Update(set).Error; err != nil {
		return err
	}

	return nil
}

// UpdateAllByWhere 根据 where 批量更新数据
func (m *Model) UpdateAllByWhere(model interface{}, where, set map[string]interface{}) error {
	if !m.db.NewRecord(model) {
		return primaryKeyNoBlankError()
	}
	// 自动更新 update_at 字段
	if err := m.prepare(nil, where).Model(model).Update(set).Error; err != nil {
		return err
	}

	return nil
}

// DeleteOneByWhere 根据 Where 删除数据；默认是软删除
func (m *Model) DeleteOneByWhere(model interface{}, where map[string]interface{}, force ...bool) error {
	if !m.db.NewRecord(model) {
		return primaryKeyNoBlankError()
	}

	db := m.prepare(nil, where).Model(model)

	if len(force) > 0 && force[0] == true {
		deleteKey := m.deletedKey
		m.ClearValidCondition()
		defer m.SetSoftDeletedKey(deleteKey)
		if err := db.Limit(1).Delete(model).Error; err != nil {
			return err
		}
	} else {
		return m.UpdateOneByWhere(model, where, map[string]interface{}{
			m.deletedKey: "Y",
		})
	}

	return nil
}

// DeleteAllByWhere 根据 where 删除数据，默认是软删除
func (m *Model) DeleteAllByWhere(model interface{}, where map[string]interface{}, force ...bool) error {
	if !m.db.NewRecord(model) {
		return primaryKeyNoBlankError()
	}
	if len(force) > 0 && force[0] == true {
		deleteKey := m.deletedKey
		m.ClearValidCondition()
		defer m.SetSoftDeletedKey(deleteKey)
		if err := m.prepare(nil, where).Model(model).Delete(model).Error; err != nil {
			return err
		}
	} else {
		return m.UpdateAllByWhere(model, where, map[string]interface{}{
			m.deletedKey: "Y",
		})
	}

	return nil
}
