# Gorm-Model

## 约束

软删除通过枚举类型实现；默认字段名为`is_deleted`（`gorm` 是通过 `deleted_at` 实现）

## 使用

```go
var db *gorm.DB
model := NewModel(db)
// 若字段名不一致，可使用此方法声明
// model.SetSoftDeletedKey("is_deleted")
// 若需要查询所有的数据（包含软删除数据），可使用以下方法解除查询条件约束
// model.ClearValidCondition()

// 其他方法见 model_test.go 文件
```

## 测试
```sh
go test .
```