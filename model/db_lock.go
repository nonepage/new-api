package model

import (
	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// withRowLock applies FOR UPDATE on databases that support it.
// SQLite serializes writes within a transaction and does not support FOR UPDATE.
func withRowLock(tx *gorm.DB) *gorm.DB {
	if common.UsingSQLite {
		return tx
	}
	return tx.Set("gorm:query_option", "FOR UPDATE")
}
