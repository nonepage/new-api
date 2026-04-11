package model

import (
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func prepareUserEditTest(t *testing.T) {
	t.Helper()
	initCol()
	t.Cleanup(func() {
		DB.Exec("DELETE FROM users")
	})
}

func createUserForEditTest(t *testing.T, referralRate *float64) *User {
	t.Helper()
	user := &User{
		Username:                  fmt.Sprintf("user-edit-%d", time.Now().UnixNano()),
		Password:                  "password123",
		DisplayName:               "User Edit",
		Role:                      common.RoleCommonUser,
		Status:                    common.UserStatusEnabled,
		Group:                     "default",
		Quota:                     100,
		ReferralCommissionPercent: referralRate,
	}
	require.NoError(t, DB.Create(user).Error)
	return user
}

func TestUserEdit_UpdatesReferralCommissionOverride(t *testing.T) {
	prepareUserEditTest(t)

	user := createUserForEditTest(t, nil)
	rate := 18.5

	updated := &User{
		Id:                        user.Id,
		Username:                  user.Username,
		DisplayName:               user.DisplayName,
		Group:                     user.Group,
		Quota:                     user.Quota,
		Remark:                    user.Remark,
		ReferralCommissionPercent: &rate,
	}
	require.NoError(t, updated.Edit(false))

	var reloaded User
	require.NoError(t, DB.First(&reloaded, user.Id).Error)
	require.NotNil(t, reloaded.ReferralCommissionPercent)
	assert.Equal(t, rate, *reloaded.ReferralCommissionPercent)
}

func TestUserEdit_ClearsReferralCommissionOverride(t *testing.T) {
	prepareUserEditTest(t)

	rate := 12.0
	user := createUserForEditTest(t, &rate)

	updated := &User{
		Id:          user.Id,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Group:       user.Group,
		Quota:       user.Quota,
		Remark:      user.Remark,
	}
	require.NoError(t, updated.Edit(false))

	var reloaded User
	require.NoError(t, DB.First(&reloaded, user.Id).Error)
	assert.Nil(t, reloaded.ReferralCommissionPercent)
}
