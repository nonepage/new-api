package model

import (
	"sort"

	"github.com/QuantumNous/new-api/setting"
)

func getDefaultUserGroupName() string {
	groups := setting.GetUserUsableGroupsCopy()
	if len(groups) == 0 {
		return "default"
	}
	if _, ok := groups["default"]; ok {
		return "default"
	}

	names := make([]string, 0, len(groups))
	for groupName := range groups {
		if groupName != "" {
			names = append(names, groupName)
		}
	}
	if len(names) == 0 {
		return "default"
	}
	sort.Strings(names)
	return names[0]
}
