package model

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

func parseUserSetting(raw string) dto.UserSetting {
	setting := dto.UserSetting{
		RecordIpLog: true,
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return setting
	}

	var rawMap map[string]interface{}
	if err := common.UnmarshalJsonStr(raw, &rawMap); err != nil {
		common.SysLog("failed to unmarshal setting: " + err.Error())
		return setting
	}

	if err := common.UnmarshalJsonStr(raw, &setting); err != nil {
		common.SysLog("failed to unmarshal setting: " + err.Error())
		return setting
	}

	// If record_ip_log was not explicitly set in the stored JSON,
	// keep the default (true) regardless of what Unmarshal produced.
	if _, ok := rawMap["record_ip_log"]; !ok {
		setting.RecordIpLog = true
	}
	return setting
}

func publicUserOmitFields() []string {
	return []string{
		"password",
		"register_ip",
		"register_user_agent",
		"last_login_ip",
		"last_login_user_agent",
		"last_login_at",
	}
}
