package idempotency

import (
	"time"

	"gorm.io/gorm"
)

// Retention is how long stored idempotency records are kept. Clients are
// expected to reuse keys only for retries within a bounded window; after
// this the record is purged and the key may be reused safely.
const Retention = 24 * time.Hour

// CleanupOlderThan deletes idempotency records created before the cutoff and
// returns the number of rows removed.
func CleanupOlderThan(db *gorm.DB, cutoff time.Time) (int64, error) {
	res := db.Where("created_at < ?", cutoff).Delete(&IdempotencyKey{})
	return res.RowsAffected, res.Error
}
