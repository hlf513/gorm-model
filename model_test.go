package gorm_model

import (
	"fmt"
	"log"
	"testing"

	gormbulk "github.com/hlf513/gorm-bulk-insert"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/stretchr/testify/assert"
)

var db *gorm.DB

func init() {
	var err error
	db, err = gorm.Open(
		"mysql",
		fmt.Sprintf(
			"%s:%s@tcp(%s)/%s?charset=utf8mb4,utf8&parseTime=True&loc=Local",
			"root",
			"123456",
			"nas.local",
			"gorm_model",
		))
	if err != nil {
		log.Fatal("mysql.connect:", err.Error())
		return
	}

	// 可使用单数表格
	db.SingularTable(true)
	// 打印出 SQL
	db.LogMode(true)
	// 创建测试表结构
	db.AutoMigrate(&User{})
	
	db.Exec("Truncate table user")
}

type User struct {
	CommonCols
	UserID   int    `gorm:"column:user_id" json:"user_id"`
	UserName string `gorm:"column:user_name" json:"user_name"`
}

func (u User) TableName() string {
	return "user"
}

func getModel() *Model {
	model := NewModel(db)
	return model
}

func TestModel_Create(t *testing.T) {
	var user = User{UserID: 99}
	user.SetDefaultValues()
	err := getModel().Create(&user)

	assert.NoError(t, err)
	assert.NotEmpty(t, user)
	assert.Equal(t, 99, user.UserID)
}

func TestModel_BatchInsert(t *testing.T) {
	var users []interface{}
	for i := 1; i <= 10; i++ {
		users = append(users, User{UserID: i})
	}
	err := getModel().BatchInsert(users, 3, gormbulk.Insert)

	assert.NoError(t, err)
}

func TestModel_FetchOneById(t *testing.T) {
	var users = []struct {
		user     User
		excepted int
	}{
		// 结构体只有 ID 字段会生成 where 条件，其他字段会忽略
		{User{
			CommonCols: CommonCols{ID: 1},
			UserID:     12,
			UserName:   "",
		}, 99},
		{User{
			CommonCols: CommonCols{ID: 2},
			UserID:     13,
			UserName:   "",
		}, 1},
	}

	model := getModel()
	for _, v := range users {
		var u User
		err := model.FetchOneById(v.user.TableName(), v.user.ID, &u, "user_id")
		assert.NoError(t, err)
		assert.Equal(t, v.excepted, u.UserID)
	}
}

func TestModel_FetchOneByWhere(t *testing.T) {
	var users = []struct {
		where    map[string]interface{}
		user     User
		excepted int
	}{
		// 结构体只有 ID 字段会生成 where 条件，其他字段会忽略
		{map[string]interface{}{
			"id = ?": 1,
		}, User{UserID: 12}, 99},
		{map[string]interface{}{
			"user_id = ?": 1,
			"id = ?":      2,
		}, User{UserID: 13}, 1},
	}

	model := getModel()
	for _, v := range users {
		var u User
		err := model.FetchOneByWhere(v.user.TableName(), v.where, &u, "user_id")
		assert.NoError(t, err)
		assert.Equal(t, v.excepted, u.UserID)
	}
}

func TestModel_FetchAllByIds(t *testing.T) {
	// 这里的 ID 不起作用；但是查询后会覆盖这里的初始化数据
	var users = []User{
		{
			CommonCols: CommonCols{ID: 3},
			UserID:     10,
		},
		{
			CommonCols: CommonCols{ID: 4},
			UserID:     20,
		},
	}
	model := getModel()
	err := model.FetchAllByIds(User{}.TableName(), []int{-1, -2}, &users, "user_id")
	assert.NoError(t, err)
	assert.Equal(t, []User{}, users) // 被覆盖了

	var users2 = []struct {
		ids      interface{}
		users    []User
		excepted int
	}{
		{[]int{1, 2}, []User{}, 2},
		{[]int{-1, -2}, []User{}, 0},
	}
	for _, v := range users2 {
		err := model.FetchAllByIds(User{}.TableName(), v.ids, &v.users, "id desc")
		assert.NoError(t, err)
		assert.Equal(t, v.excepted, len(v.users))
	}
}

func TestModel_FetchAllByWhere(t *testing.T) {
	// 这里的 ID 不起作用；但是查询后会覆盖这里的初始化数据
	var users = []User{
		{
			CommonCols: CommonCols{ID: 3},
			UserID:     0,
		},
		{
			CommonCols: CommonCols{ID: 4},
			UserID:     0,
		},
	}
	model := getModel()
	err := model.FetchAllByWhere(User{}.TableName(), map[string]interface{}{
		"id < ?": 0,
	}, &users, "id desc", "user_id")
	assert.NoError(t, err)
	assert.Equal(t, []User{}, users) // 被覆盖了

	var users2 = []struct {
		where    map[string]interface{}
		users    []User
		excepted int
	}{
		{map[string]interface{}{
			"id >= ?": 10,
		}, []User{}, 2},
		{map[string]interface{}{
			"id = ?": 0,
		}, []User{}, 0},
	}
	for _, v := range users2 {
		err := model.FetchAllByWhere(User{}.TableName(), v.where, &v.users, "id desc", "user_id")
		assert.NoError(t, err)
		assert.Equal(t, v.excepted, len(v.users))
		if v.excepted > 0 {
			// 没有 select id，所以 ID = 0
			assert.Equal(t, 0, v.users[0].ID)
		}
	}
}

func TestModel_SearchOne(t *testing.T) {
	type user struct {
		Id     int
		UserId int
	}

	var u user
	model := getModel()
	err := model.SearchOne("user", "id,user_id,count(*) t", map[string]interface{}{
		"user_id = ?": 99,
	}, &u, "id desc", "user_id,id", "t > 0")
	assert.NoError(t, err)
	assert.Equal(t, 1, u.Id)
}

func TestModel_SearchAll(t *testing.T) {
	type user struct {
		Id     int
		UserId int
	}
	var u []user
	var total int
	model := getModel()
	err := model.SearchAll("user", "id,user_id,count(*) t", map[string]interface{}{
		"id > ?": 0,
	}, &u, "id desc", &total, 0, 2, "user_id,id", "t > 0")
	assert.NoError(t, err)
	assert.True(t, true, len(u) == 2)
}

func TestModel_Count(t *testing.T) {
	model := getModel()
	count, err := model.Count("user", map[string]interface{}{
		"id > ?": 10,
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	count, err = model.Count("user", map[string]interface{}{
		"id > ?":         10,
		"is_deleted = ?": "Y", // 注意，这里会替换全局 is_deleted = N 的搜索条件
	})
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestModel_UpdateAllByWhere(t *testing.T) {
	model := getModel()
	var u User
	err := model.UpdateAllByWhere(u, map[string]interface{}{
		"id = ?": 1,
	}, map[string]interface{}{
		"user_name": "user1",
	})
	assert.NoError(t, err)
	fmt.Println(u)
}

func TestModel_DeleteAllByWhere(t *testing.T) {
	model := getModel()
	var u User
	err := model.DeleteAllByWhere(u, map[string]interface{}{
		"id <= ?": 3,
	})
	assert.NoError(t, err)

	err = model.DeleteAllByWhere(u, map[string]interface{}{
		"id <= ?": 3,
	}, true)
	assert.NoError(t, err)
}
