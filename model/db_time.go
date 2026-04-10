package model

import (
	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// GetDBTimestamp returns a UNIX timestamp from database time.
// Falls back to application time on error.
func GetDBTimestamp() int64 {
	return getDBTimestampByHandle(DB)
}

func getDBTimestampTx(tx *gorm.DB) int64 {
	return getDBTimestampByHandle(tx)
}

func getDBTimestampByHandle(handle *gorm.DB) int64 {
	if handle == nil {
		handle = DB
	}
	var ts int64
	var err error
	switch {
	case common.UsingPostgreSQL:
		err = handle.Raw("SELECT EXTRACT(EPOCH FROM NOW())::bigint").Scan(&ts).Error
	case common.UsingSQLite:
		err = handle.Raw("SELECT strftime('%s','now')").Scan(&ts).Error
	default:
		err = handle.Raw("SELECT UNIX_TIMESTAMP()").Scan(&ts).Error
	}
	if err != nil || ts <= 0 {
		return common.GetTimestamp()
	}
	return ts
}
