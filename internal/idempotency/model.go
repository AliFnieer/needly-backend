package idempotency

import "time"

// IdempotencyKey stores a completed response so that retrying the same
// request (identified by the client-supplied Idempotency-Key header) returns
// the original result instead of executing the operation twice.
type IdempotencyKey struct {
	ID           uint      `gorm:"primaryKey" json:"-"`
	UserID       uint      `gorm:"not null;uniqueIndex:uq_idempotency_keys,priority:1" json:"-"`
	Route        string    `gorm:"size:255;not null;uniqueIndex:uq_idempotency_keys,priority:2" json:"-"`
	KeyHash      string    `gorm:"size:64;not null;uniqueIndex:uq_idempotency_keys,priority:3" json:"-"`
	StatusCode   int       `gorm:"not null" json:"-"`
	ContentType  string    `gorm:"size:255;not null;default:'application/json; charset=utf-8'" json:"-"`
	ResponseBody string    `gorm:"type:text;not null" json:"-"`
	CreatedAt    time.Time `json:"-"`
}

func (IdempotencyKey) TableName() string { return "idempotency_keys" }
