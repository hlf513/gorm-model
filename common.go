package gorm_model

import "time"

type CommonCols struct {
	ID        int       `gorm:"column:id;primary_key" json:"id"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
	IsDeleted string    `gorm:"column:is_deleted;type:enum('Y','N');default:N" json:"is_deleted"`
}

func (c *CommonCols) SetDefaultValues() {
	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()
	c.IsDeleted = "N"
}
