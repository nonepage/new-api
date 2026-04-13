package model

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

var subscriptionConversionTestMigrateOnce sync.Once

func prepareSubscriptionConversionTest(t *testing.T) {
	t.Helper()
	initCol()
	subscriptionConversionTestMigrateOnce.Do(func() {
		migrations := []struct {
			name  string
			model any
		}{
			{name: "SubscriptionPlan", model: &SubscriptionPlan{}},
			{name: "SubscriptionOrder", model: &SubscriptionOrder{}},
			{name: "TopUp", model: &TopUp{}},
			{name: "UserSubscription", model: &UserSubscription{}},
			{name: "SubscriptionPreConsumeRecord", model: &SubscriptionPreConsumeRecord{}},
			{name: "SubscriptionConversionBatch", model: &SubscriptionConversionBatch{}},
			{name: "SubscriptionConversionItem", model: &SubscriptionConversionItem{}},
		}
		for _, migration := range migrations {
			require.NoErrorf(t, DB.AutoMigrate(migration.model), "failed to migrate %s", migration.name)
		}
	})
	t.Cleanup(func() {
		DB.Exec("DELETE FROM subscription_conversion_items")
		DB.Exec("DELETE FROM subscription_conversion_batches")
		DB.Exec("DELETE FROM subscription_pre_consume_records")
		DB.Exec("DELETE FROM user_subscriptions")
		DB.Exec("DELETE FROM subscription_orders")
		DB.Exec("DELETE FROM top_ups")
		DB.Exec("DELETE FROM subscription_plans")
		DB.Exec("DELETE FROM logs")
		DB.Exec("DELETE FROM users")
	})
}

func createSubscriptionConversionUser(t *testing.T, group string, quota int) *User {
	t.Helper()
	user := &User{
		Username:    fmt.Sprintf("sub-conv-%d", time.Now().UnixNano()),
		Password:    "password123",
		DisplayName: "Subscription Conversion",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       group,
		Quota:       quota,
	}
	require.NoError(t, DB.Create(user).Error)
	return user
}

func createSubscriptionConversionPlan(t *testing.T, title string, price float64, upgradeGroup string) *SubscriptionPlan {
	t.Helper()
	plan := &SubscriptionPlan{
		Title:              title,
		PriceAmount:        price,
		Currency:           "USD",
		DurationUnit:       SubscriptionDurationMonth,
		DurationValue:      1,
		Enabled:            true,
		UpgradeGroup:       upgradeGroup,
		QuotaResetPeriod:   SubscriptionResetNever,
		MaxPurchasePerUser: 0,
	}
	require.NoError(t, DB.Create(plan).Error)
	return plan
}

func createUserSubscriptionForConversionTest(t *testing.T, userId int, planId int, startTime int64, endTime int64, status string) *UserSubscription {
	return createUserSubscriptionForConversionTestWithSource(t, userId, planId, startTime, endTime, status, "order", 0, "")
}

func createUserSubscriptionForConversionTestWithSource(t *testing.T, userId int, planId int, startTime int64, endTime int64, status string, source string, purchasePriceAmount float64, purchaseCurrency string) *UserSubscription {
	t.Helper()
	subscription := &UserSubscription{
		UserId:              userId,
		PlanId:              planId,
		AmountTotal:         1000,
		AmountUsed:          100,
		PurchasePriceAmount: purchasePriceAmount,
		PurchaseCurrency:    purchaseCurrency,
		StartTime:           startTime,
		EndTime:             endTime,
		Status:              status,
		Source:              source,
		UpgradeGroup:        "vip",
		PrevUserGroup:       "default",
	}
	require.NoError(t, DB.Create(subscription).Error)
	return subscription
}

func TestPreviewSubscriptionWalletConversion_OnlyActiveSubscriptions(t *testing.T) {
	prepareSubscriptionConversionTest(t)

	user := createSubscriptionConversionUser(t, "vip", 1000)
	now := GetDBTimestamp()
	activePlanA := createSubscriptionConversionPlan(t, "Pro Monthly", 10, "vip")
	activePlanB := createSubscriptionConversionPlan(t, "Team Monthly", 20, "vip")
	expiredPlan := createSubscriptionConversionPlan(t, "Legacy Monthly", 30, "vip")

	createUserSubscriptionForConversionTest(t, user.Id, activePlanA.Id, now-10000, now+10000, SubscriptionStatusActive)
	createUserSubscriptionForConversionTest(t, user.Id, activePlanB.Id, now-8000, now+12000, SubscriptionStatusActive)
	createUserSubscriptionForConversionTest(t, user.Id, expiredPlan.Id, now-20000, now-100, SubscriptionStatusExpired)

	preview, err := PreviewSubscriptionWalletConversion(user.Id)
	require.NoError(t, err)
	require.NotNil(t, preview)
	require.Len(t, preview.Items, 2)

	assert.Equal(t, 2, preview.Summary.SubscriptionCount)
	assert.Equal(t, "vip", preview.Summary.UserGroupBefore)
	assert.Equal(t, "default", preview.Summary.UserGroupAfter)
	assert.True(t, preview.Summary.CanConvert)
	assert.Positive(t, preview.Summary.TotalRefundMoney)
	assert.Positive(t, preview.Summary.TotalRefundQuota)

	totalQuota := int64(0)
	seenPlanIds := make(map[int]struct{}, len(preview.Items))
	for _, item := range preview.Items {
		totalQuota += item.RefundQuota
		seenPlanIds[item.PlanId] = struct{}{}
		assert.Greater(t, item.DurationSeconds, int64(0))
		assert.GreaterOrEqual(t, item.RemainingSeconds, int64(0))
		assert.GreaterOrEqual(t, item.RefundRatio, 0.0)
		assert.LessOrEqual(t, item.RefundRatio, 1.0)
	}

	assert.Equal(t, preview.Summary.TotalRefundQuota, totalQuota)
	assert.Contains(t, seenPlanIds, activePlanA.Id)
	assert.Contains(t, seenPlanIds, activePlanB.Id)
	assert.NotContains(t, seenPlanIds, expiredPlan.Id)
}

func TestExecuteSubscriptionWalletConversion_ConvertsAndIsIdempotent(t *testing.T) {
	prepareSubscriptionConversionTest(t)

	user := createSubscriptionConversionUser(t, "vip", 1000)
	now := GetDBTimestamp()
	planA := createSubscriptionConversionPlan(t, "Starter Monthly", 12, "vip")
	planB := createSubscriptionConversionPlan(t, "Pro Monthly", 18, "vip")

	subA := createUserSubscriptionForConversionTest(t, user.Id, planA.Id, now-10000, now+10000, SubscriptionStatusActive)
	subB := createUserSubscriptionForConversionTest(t, user.Id, planB.Id, now-15000, now+5000, SubscriptionStatusActive)

	requestId := fmt.Sprintf("req-%d", time.Now().UnixNano())
	result, err := ExecuteSubscriptionWalletConversion(user.Id, requestId)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 2, result.SubscriptionCount)
	assert.Equal(t, "vip", result.UserGroupBefore)
	assert.Equal(t, "default", result.UserGroupAfter)
	assert.Positive(t, result.TotalRefundMoney)
	assert.Positive(t, result.TotalRefundQuota)
	assert.Equal(t, user.Quota+int(result.TotalRefundQuota), result.NewUserQuota)

	var reloadedUser User
	require.NoError(t, DB.First(&reloadedUser, user.Id).Error)
	assert.Equal(t, "default", reloadedUser.Group)
	assert.Equal(t, result.NewUserQuota, reloadedUser.Quota)

	var convertedSubs []UserSubscription
	require.NoError(t, DB.Where("id IN ?", []int{subA.Id, subB.Id}).Order("id asc").Find(&convertedSubs).Error)
	require.Len(t, convertedSubs, 2)
	for _, sub := range convertedSubs {
		assert.Equal(t, SubscriptionStatusCancelled, sub.Status)
		assert.Equal(t, SubscriptionCancelReasonWalletConversion, sub.CancelReason)
		assert.Equal(t, result.BatchId, sub.ConversionBatchId)
		assert.Greater(t, sub.CancelledAt, int64(0))
		assert.LessOrEqual(t, sub.EndTime, now+1)
	}

	var batch SubscriptionConversionBatch
	require.NoError(t, DB.First(&batch, result.BatchId).Error)
	assert.Equal(t, "success", batch.Status)
	assert.Equal(t, result.SubscriptionCount, batch.SubscriptionCount)
	assert.Equal(t, result.TotalRefundQuota, batch.TotalRefundQuota)

	var itemCount int64
	require.NoError(t, DB.Model(&SubscriptionConversionItem{}).Where("batch_id = ?", result.BatchId).Count(&itemCount).Error)
	assert.EqualValues(t, 2, itemCount)

	secondResult, err := ExecuteSubscriptionWalletConversion(user.Id, requestId)
	require.NoError(t, err)
	require.NotNil(t, secondResult)
	assert.Equal(t, result.BatchId, secondResult.BatchId)
	assert.Equal(t, result.SubscriptionCount, secondResult.SubscriptionCount)
	assert.Equal(t, result.TotalRefundQuota, secondResult.TotalRefundQuota)
	assert.Equal(t, result.NewUserQuota, secondResult.NewUserQuota)

	var quotaAfterSecondCall int
	require.NoError(t, DB.Model(&User{}).Where("id = ?", user.Id).Select("quota").Find(&quotaAfterSecondCall).Error)
	assert.Equal(t, result.NewUserQuota, quotaAfterSecondCall)
}

func TestExecuteSubscriptionWalletConversion_NoActiveSubscriptionsOnDefaultGroup(t *testing.T) {
	prepareSubscriptionConversionTest(t)

	user := createSubscriptionConversionUser(t, "default", 1000)
	now := GetDBTimestamp()
	plan := createSubscriptionConversionPlan(t, "Expired Monthly", 10, "vip")
	createUserSubscriptionForConversionTest(t, user.Id, plan.Id, now-20000, now-100, SubscriptionStatusExpired)

	result, err := ExecuteSubscriptionWalletConversion(user.Id, fmt.Sprintf("req-%d", time.Now().UnixNano()))
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no active subscriptions")
}

func TestPreviewSubscriptionWalletConversion_NoActiveSubscriptionsFallsBackToDefaultGroup(t *testing.T) {
	prepareSubscriptionConversionTest(t)

	user := createSubscriptionConversionUser(t, "vip", 1000)
	now := GetDBTimestamp()
	plan := createSubscriptionConversionPlan(t, "Expired Monthly", 10, "vip")
	createUserSubscriptionForConversionTest(t, user.Id, plan.Id, now-20000, now-100, SubscriptionStatusExpired)

	preview, err := PreviewSubscriptionWalletConversion(user.Id)
	require.NoError(t, err)
	require.NotNil(t, preview)
	assert.True(t, preview.Summary.CanConvert)
	assert.Equal(t, 0, preview.Summary.SubscriptionCount)
	assert.Equal(t, "vip", preview.Summary.UserGroupBefore)
	assert.Equal(t, "default", preview.Summary.UserGroupAfter)
}

func TestPreviewSubscriptionWalletConversion_NoActiveSubscriptionsIgnoresHistoricalBaseGroup(t *testing.T) {
	prepareSubscriptionConversionTest(t)

	user := createSubscriptionConversionUser(t, "vip", 1000)
	now := GetDBTimestamp()
	plan := createSubscriptionConversionPlan(t, "Expired VIP Monthly", 10, "vip")
	subscription := &UserSubscription{
		UserId:           user.Id,
		PlanId:           plan.Id,
		AmountTotal:      1000,
		AmountUsed:       100,
		StartTime:        now - 20000,
		EndTime:          now - 100,
		Status:           SubscriptionStatusExpired,
		Source:           "order",
		UpgradeGroup:     "vip",
		PrevUserGroup:    "staff",
		CancelReason:     "",
		CancelledAt:      0,
		LastResetTime:    0,
		NextResetTime:    0,
		CreatedAt:        common.GetTimestamp(),
		UpdatedAt:        common.GetTimestamp(),
		PurchaseCurrency: "USD",
	}
	require.NoError(t, DB.Create(subscription).Error)

	preview, err := PreviewSubscriptionWalletConversion(user.Id)
	require.NoError(t, err)
	require.NotNil(t, preview)
	assert.True(t, preview.Summary.CanConvert)
	assert.Equal(t, 0, preview.Summary.SubscriptionCount)
	assert.Equal(t, "vip", preview.Summary.UserGroupBefore)
	assert.Equal(t, "default", preview.Summary.UserGroupAfter)
}

func TestExecuteSubscriptionWalletConversion_NoActiveSubscriptionsResetsToDefaultGroup(t *testing.T) {
	prepareSubscriptionConversionTest(t)

	user := createSubscriptionConversionUser(t, "vip", 1000)
	now := GetDBTimestamp()
	plan := createSubscriptionConversionPlan(t, "Expired Monthly", 10, "vip")
	createUserSubscriptionForConversionTest(t, user.Id, plan.Id, now-20000, now-100, SubscriptionStatusExpired)

	result, err := ExecuteSubscriptionWalletConversion(user.Id, fmt.Sprintf("reset-group-%d", time.Now().UnixNano()))
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 0, result.SubscriptionCount)
	assert.Equal(t, 0.0, result.TotalRefundMoney)
	assert.Equal(t, int64(0), result.TotalRefundQuota)
	assert.Equal(t, "vip", result.UserGroupBefore)
	assert.Equal(t, "default", result.UserGroupAfter)

	var refreshed User
	require.NoError(t, DB.First(&refreshed, user.Id).Error)
	assert.Equal(t, "default", refreshed.Group)
}

func TestExecuteSubscriptionWalletConversion_NoActiveSubscriptionsResetsHistoricalBaseGroupToDefault(t *testing.T) {
	prepareSubscriptionConversionTest(t)

	user := createSubscriptionConversionUser(t, "staff", 1000)
	now := GetDBTimestamp()
	plan := createSubscriptionConversionPlan(t, "Expired VIP Monthly", 10, "vip")
	subscription := &UserSubscription{
		UserId:           user.Id,
		PlanId:           plan.Id,
		AmountTotal:      1000,
		AmountUsed:       100,
		StartTime:        now - 20000,
		EndTime:          now - 100,
		Status:           SubscriptionStatusExpired,
		Source:           "order",
		UpgradeGroup:     "vip",
		PrevUserGroup:    "staff",
		PurchaseCurrency: "USD",
		CreatedAt:        common.GetTimestamp(),
		UpdatedAt:        common.GetTimestamp(),
	}
	require.NoError(t, DB.Create(subscription).Error)

	preview, err := PreviewSubscriptionWalletConversion(user.Id)
	require.NoError(t, err)
	require.NotNil(t, preview)
	assert.True(t, preview.Summary.CanConvert)
	assert.Equal(t, "default", preview.Summary.UserGroupAfter)

	result, err := ExecuteSubscriptionWalletConversion(user.Id, fmt.Sprintf("reset-historical-group-%d", time.Now().UnixNano()))
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 0, result.SubscriptionCount)
	assert.Equal(t, "staff", result.UserGroupBefore)
	assert.Equal(t, "default", result.UserGroupAfter)

	var refreshed User
	require.NoError(t, DB.First(&refreshed, user.Id).Error)
	assert.Equal(t, "default", refreshed.Group)
}

func TestPreviewSubscriptionWalletConversion_UsesPurchaseSnapshotWhenPlanPriceChanges(t *testing.T) {
	prepareSubscriptionConversionTest(t)

	user := createSubscriptionConversionUser(t, "vip", 1000)
	now := GetDBTimestamp()
	plan := createSubscriptionConversionPlan(t, "Repriced Monthly", 10, "vip")
	createUserSubscriptionForConversionTestWithSource(
		t,
		user.Id,
		plan.Id,
		now-1000,
		now+1000,
		SubscriptionStatusActive,
		"order",
		10,
		"USD",
	)
	require.NoError(t, DB.Model(&SubscriptionPlan{}).Where("id = ?", plan.Id).Update("price_amount", 99).Error)

	preview, err := PreviewSubscriptionWalletConversion(user.Id)
	require.NoError(t, err)
	require.NotNil(t, preview)
	require.Len(t, preview.Items, 1)

	item := preview.Items[0]
	assert.Equal(t, 10.0, item.PriceAmount)
	assert.Equal(t, "USD", item.Currency)
	assert.InDelta(t, 5.0, item.RefundMoney, 0.2)
}

func TestPreviewSubscriptionWalletConversion_FallsBackToCurrentPlanPriceWithoutSnapshot(t *testing.T) {
	prepareSubscriptionConversionTest(t)

	user := createSubscriptionConversionUser(t, "vip", 1000)
	now := GetDBTimestamp()
	plan := createSubscriptionConversionPlan(t, "Legacy Monthly", 10, "vip")
	createUserSubscriptionForConversionTestWithSource(
		t,
		user.Id,
		plan.Id,
		now-1000,
		now+1000,
		SubscriptionStatusActive,
		"",
		0,
		"",
	)
	require.NoError(t, DB.Model(&SubscriptionPlan{}).Where("id = ?", plan.Id).Updates(map[string]interface{}{
		"price_amount": 40,
		"currency":     "USD",
	}).Error)

	preview, err := PreviewSubscriptionWalletConversion(user.Id)
	require.NoError(t, err)
	require.NotNil(t, preview)
	require.Len(t, preview.Items, 1)

	item := preview.Items[0]
	assert.Equal(t, 40.0, item.PriceAmount)
	assert.Equal(t, "USD", item.Currency)
	assert.InDelta(t, 20.0, item.RefundMoney, 0.5)
	assert.Positive(t, item.RefundQuota)
}

func TestExecuteSubscriptionWalletConversion_UsesOrderPaidSnapshotWhenPlanPriceIsZero(t *testing.T) {
	prepareSubscriptionConversionTest(t)
	originalRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	t.Cleanup(func() {
		common.RedisEnabled = originalRedisEnabled
	})

	user := createSubscriptionConversionUser(t, "default", 1000)
	plan := createSubscriptionConversionPlan(t, "Zero Price Display Plan", 0, "vip")
	order := &SubscriptionOrder{
		UserId:        user.Id,
		PlanId:        plan.Id,
		Money:         9.9,
		PaidAmount:    9.9,
		PaidCurrency:  "USD",
		TradeNo:       fmt.Sprintf("sub-order-%d", time.Now().UnixNano()),
		PaymentMethod: "stripe",
		Status:        common.TopUpStatusPending,
		CreateTime:    common.GetTimestamp(),
	}
	require.NoError(t, DB.Create(order).Error)
	require.NoError(t, CompleteSubscriptionOrder(order.TradeNo, ""))

	preview, err := PreviewSubscriptionWalletConversion(user.Id)
	require.NoError(t, err)
	require.NotNil(t, preview)
	require.Len(t, preview.Items, 1)

	item := preview.Items[0]
	assert.Equal(t, 9.9, item.PriceAmount)
	assert.Equal(t, "USD", item.Currency)
	assert.Positive(t, item.RefundMoney)
	assert.Positive(t, item.RefundQuota)

	result, err := ExecuteSubscriptionWalletConversion(user.Id, fmt.Sprintf("paid-snapshot-%d", time.Now().UnixNano()))
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Positive(t, result.TotalRefundMoney)
	assert.Positive(t, result.TotalRefundQuota)
	assert.Equal(t, "default", result.UserGroupAfter)
	assert.Equal(t, user.Quota+int(result.TotalRefundQuota), result.NewUserQuota)
}

func TestExecuteSubscriptionWalletConversion_AdminSubscriptionFallsBackToPlanPrice(t *testing.T) {
	prepareSubscriptionConversionTest(t)

	user := createSubscriptionConversionUser(t, "vip", 1000)
	now := GetDBTimestamp()
	plan := createSubscriptionConversionPlan(t, "Admin Granted", 30, "vip")
	createUserSubscriptionForConversionTestWithSource(
		t,
		user.Id,
		plan.Id,
		now-1000,
		now+1000,
		SubscriptionStatusActive,
		"admin",
		0,
		"USD",
	)

	result, err := ExecuteSubscriptionWalletConversion(user.Id, fmt.Sprintf("admin-req-%d", time.Now().UnixNano()))
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 1, result.SubscriptionCount)
	assert.Positive(t, result.TotalRefundQuota)
	assert.Positive(t, result.TotalRefundMoney)
	assert.Equal(t, user.Quota+int(result.TotalRefundQuota), result.NewUserQuota)
}

func TestAdminBindSubscription_ConversionFallsBackToPlanPrice(t *testing.T) {
	prepareSubscriptionConversionTest(t)
	originalRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	t.Cleanup(func() {
		common.RedisEnabled = originalRedisEnabled
	})

	user := createSubscriptionConversionUser(t, "default", 1000)
	plan := createSubscriptionConversionPlan(t, "Admin Bound", 15, "vip")

	_, err := AdminBindSubscription(user.Id, plan.Id, "")
	require.NoError(t, err)

	preview, err := PreviewSubscriptionWalletConversion(user.Id)
	require.NoError(t, err)
	require.NotNil(t, preview)
	require.Len(t, preview.Items, 1)
	assert.Positive(t, preview.Items[0].RefundMoney)
	assert.Positive(t, preview.Items[0].RefundQuota)

	result, err := ExecuteSubscriptionWalletConversion(user.Id, fmt.Sprintf("admin-bind-%d", time.Now().UnixNano()))
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Positive(t, result.TotalRefundMoney)
	assert.Positive(t, result.TotalRefundQuota)
}

func TestCreateUserSubscriptionFromPlanTx_DuplicateUpgradeKeepsFallbackPreviousGroup(t *testing.T) {
	prepareSubscriptionConversionTest(t)

	user := createSubscriptionConversionUser(t, "default", 1000)
	plan := createSubscriptionConversionPlan(t, "VIP Monthly", 10, "vip")

	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		if _, err := CreateUserSubscriptionFromPlanTx(tx, user.Id, plan, "order"); err != nil {
			return err
		}
		_, err := CreateUserSubscriptionFromPlanTx(tx, user.Id, plan, "order")
		return err
	}))

	var subs []UserSubscription
	require.NoError(t, DB.Where("user_id = ?", user.Id).Order("id asc").Find(&subs).Error)
	require.Len(t, subs, 2)
	assert.Equal(t, "default", subs[0].PrevUserGroup)
	assert.Equal(t, "default", subs[1].PrevUserGroup)

	var refreshed User
	require.NoError(t, DB.First(&refreshed, user.Id).Error)
	assert.Equal(t, "vip", refreshed.Group)
}

func TestExpireDueSubscriptions_StackedSameUpgradeFallsBackToOriginalGroup(t *testing.T) {
	prepareSubscriptionConversionTest(t)

	user := createSubscriptionConversionUser(t, "default", 1000)
	plan := createSubscriptionConversionPlan(t, "VIP Monthly", 10, "vip")

	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		if _, err := CreateUserSubscriptionFromPlanTx(tx, user.Id, plan, "order"); err != nil {
			return err
		}
		_, err := CreateUserSubscriptionFromPlanTx(tx, user.Id, plan, "order")
		return err
	}))

	var subs []UserSubscription
	require.NoError(t, DB.Where("user_id = ?", user.Id).Order("id asc").Find(&subs).Error)
	require.Len(t, subs, 2)

	now := common.GetTimestamp()
	require.NoError(t, DB.Model(&UserSubscription{}).Where("id = ?", subs[0].Id).Update("end_time", now-10).Error)
	require.NoError(t, DB.Model(&UserSubscription{}).Where("id = ?", subs[1].Id).Update("end_time", now+3600).Error)

	expiredCount, err := ExpireDueSubscriptions(10)
	require.NoError(t, err)
	assert.Equal(t, 1, expiredCount)

	var midUser User
	require.NoError(t, DB.First(&midUser, user.Id).Error)
	assert.Equal(t, "vip", midUser.Group)

	require.NoError(t, DB.Model(&UserSubscription{}).Where("id = ?", subs[1].Id).Update("end_time", common.GetTimestamp()-1).Error)

	expiredCount, err = ExpireDueSubscriptions(10)
	require.NoError(t, err)
	assert.Equal(t, 1, expiredCount)

	var finalUser User
	require.NoError(t, DB.First(&finalUser, user.Id).Error)
	assert.Equal(t, "default", finalUser.Group)
}

func TestExpireDueSubscriptions_MultiLevelUpgradeFallsBackToLowerActiveGroup(t *testing.T) {
	prepareSubscriptionConversionTest(t)

	user := createSubscriptionConversionUser(t, "default", 1000)
	vipPlan := createSubscriptionConversionPlan(t, "VIP Monthly", 10, "vip")
	proPlan := createSubscriptionConversionPlan(t, "PRO Monthly", 20, "pro")

	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		if _, err := CreateUserSubscriptionFromPlanTx(tx, user.Id, vipPlan, "order"); err != nil {
			return err
		}
		_, err := CreateUserSubscriptionFromPlanTx(tx, user.Id, proPlan, "order")
		return err
	}))

	var subs []UserSubscription
	require.NoError(t, DB.Where("user_id = ?", user.Id).Order("id asc").Find(&subs).Error)
	require.Len(t, subs, 2)

	now := common.GetTimestamp()
	require.NoError(t, DB.Model(&UserSubscription{}).Where("plan_id = ?", vipPlan.Id).Update("end_time", now+3600).Error)
	require.NoError(t, DB.Model(&UserSubscription{}).Where("plan_id = ?", proPlan.Id).Update("end_time", now-1).Error)

	expiredCount, err := ExpireDueSubscriptions(10)
	require.NoError(t, err)
	assert.Equal(t, 1, expiredCount)

	var midUser User
	require.NoError(t, DB.First(&midUser, user.Id).Error)
	assert.Equal(t, "vip", midUser.Group)

	require.NoError(t, DB.Model(&UserSubscription{}).Where("plan_id = ?", vipPlan.Id).Update("end_time", common.GetTimestamp()-1).Error)

	expiredCount, err = ExpireDueSubscriptions(10)
	require.NoError(t, err)
	assert.Equal(t, 1, expiredCount)

	var finalUser User
	require.NoError(t, DB.First(&finalUser, user.Id).Error)
	assert.Equal(t, "default", finalUser.Group)
}

func TestExpireDueSubscriptions_RewiresRemainingFallbackChain(t *testing.T) {
	prepareSubscriptionConversionTest(t)

	user := createSubscriptionConversionUser(t, "default", 1000)
	vipPlan := createSubscriptionConversionPlan(t, "VIP Monthly", 10, "vip")
	proPlan := createSubscriptionConversionPlan(t, "PRO Monthly", 20, "pro")

	var vipSub *UserSubscription
	var proSub *UserSubscription
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		var err error
		vipSub, err = CreateUserSubscriptionFromPlanTx(tx, user.Id, vipPlan, "order")
		if err != nil {
			return err
		}
		proSub, err = CreateUserSubscriptionFromPlanTx(tx, user.Id, proPlan, "order")
		return err
	}))
	require.NotNil(t, vipSub)
	require.NotNil(t, proSub)

	now := common.GetTimestamp()
	require.NoError(t, DB.Model(&UserSubscription{}).Where("id = ?", vipSub.Id).Update("end_time", now-1).Error)
	require.NoError(t, DB.Model(&UserSubscription{}).Where("id = ?", proSub.Id).Update("end_time", now+3600).Error)

	expiredCount, err := ExpireDueSubscriptions(10)
	require.NoError(t, err)
	assert.Equal(t, 1, expiredCount)

	var refreshedPro UserSubscription
	require.NoError(t, DB.Where("id = ?", proSub.Id).First(&refreshedPro).Error)
	assert.Equal(t, "default", refreshedPro.PrevUserGroup)

	require.NoError(t, DB.Where("id = ?", vipSub.Id).Delete(&UserSubscription{}).Error)
	require.NoError(t, DB.Model(&UserSubscription{}).Where("id = ?", proSub.Id).Update("end_time", common.GetTimestamp()-1).Error)

	expiredCount, err = ExpireDueSubscriptions(10)
	require.NoError(t, err)
	assert.Equal(t, 1, expiredCount)

	var finalUser User
	require.NoError(t, DB.First(&finalUser, user.Id).Error)
	assert.Equal(t, "default", finalUser.Group)
}

func TestExpireDueSubscriptions_MultipleDueTiersKeepRemainingFallbackChainConsistent(t *testing.T) {
	prepareSubscriptionConversionTest(t)

	user := createSubscriptionConversionUser(t, "default", 1000)
	vipPlan := createSubscriptionConversionPlan(t, "VIP Monthly", 10, "vip")
	proPlan := createSubscriptionConversionPlan(t, "PRO Monthly", 20, "pro")
	teamPlan := createSubscriptionConversionPlan(t, "TEAM Monthly", 30, "team")

	var teamSub *UserSubscription
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		if _, err := CreateUserSubscriptionFromPlanTx(tx, user.Id, vipPlan, "order"); err != nil {
			return err
		}
		if _, err := CreateUserSubscriptionFromPlanTx(tx, user.Id, proPlan, "order"); err != nil {
			return err
		}
		var err error
		teamSub, err = CreateUserSubscriptionFromPlanTx(tx, user.Id, teamPlan, "order")
		return err
	}))
	require.NotNil(t, teamSub)

	now := common.GetTimestamp()
	require.NoError(t, DB.Model(&UserSubscription{}).Where("plan_id IN ?", []int{vipPlan.Id, proPlan.Id}).Update("end_time", now-1).Error)
	require.NoError(t, DB.Model(&UserSubscription{}).Where("id = ?", teamSub.Id).Update("end_time", now+3600).Error)

	expiredCount, err := ExpireDueSubscriptions(10)
	require.NoError(t, err)
	assert.Equal(t, 2, expiredCount)

	var refreshedTeam UserSubscription
	require.NoError(t, DB.Where("id = ?", teamSub.Id).First(&refreshedTeam).Error)
	assert.Equal(t, "default", refreshedTeam.PrevUserGroup)

	var refreshedUser User
	require.NoError(t, DB.First(&refreshedUser, user.Id).Error)
	assert.Equal(t, "team", refreshedUser.Group)
}

func TestExpireDueSubscriptions_LastExpiredUpgradeFallsBackToDefaultGroup(t *testing.T) {
	prepareSubscriptionConversionTest(t)

	user := createSubscriptionConversionUser(t, "staff", 1000)
	plan := createSubscriptionConversionPlan(t, "VIP Monthly", 10, "vip")

	var sub *UserSubscription
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		var err error
		sub, err = CreateUserSubscriptionFromPlanTx(tx, user.Id, plan, "order")
		return err
	}))
	require.NotNil(t, sub)

	now := common.GetTimestamp()
	require.NoError(t, DB.Model(&UserSubscription{}).Where("id = ?", sub.Id).Update("end_time", now-1).Error)

	expiredCount, err := ExpireDueSubscriptions(10)
	require.NoError(t, err)
	assert.Equal(t, 1, expiredCount)

	var refreshed User
	require.NoError(t, DB.First(&refreshed, user.Id).Error)
	assert.Equal(t, "default", refreshed.Group)
}

func TestAdminInvalidateUserSubscription_LastUpgradeFallsBackToDefaultGroup(t *testing.T) {
	prepareSubscriptionConversionTest(t)

	user := createSubscriptionConversionUser(t, "staff", 1000)
	plan := createSubscriptionConversionPlan(t, "VIP Monthly", 10, "vip")

	var sub *UserSubscription
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		var err error
		sub, err = CreateUserSubscriptionFromPlanTx(tx, user.Id, plan, "order")
		return err
	}))
	require.NotNil(t, sub)

	message, err := AdminInvalidateUserSubscription(sub.Id)
	require.NoError(t, err)
	assert.Contains(t, message, "default")

	var refreshed User
	require.NoError(t, DB.First(&refreshed, user.Id).Error)
	assert.Equal(t, "default", refreshed.Group)
}

func TestExecuteSubscriptionWalletConversion_ResetsToDefaultGroupAfterCancellingActiveSubscriptions(t *testing.T) {
	prepareSubscriptionConversionTest(t)

	user := createSubscriptionConversionUser(t, "staff", 1000)
	plan := createSubscriptionConversionPlan(t, "VIP Monthly", 10, "vip")

	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		if _, err := CreateUserSubscriptionFromPlanTx(tx, user.Id, plan, "order"); err != nil {
			return err
		}
		_, err := CreateUserSubscriptionFromPlanTx(tx, user.Id, plan, "order")
		return err
	}))

	result, err := ExecuteSubscriptionWalletConversion(user.Id, fmt.Sprintf("staff-req-%d", time.Now().UnixNano()))
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "default", result.UserGroupAfter)

	var refreshed User
	require.NoError(t, DB.First(&refreshed, user.Id).Error)
	assert.Equal(t, "default", refreshed.Group)
}

func TestAdminDeleteUserSubscription_FallsBackAfterDeletingLastUpgrade(t *testing.T) {
	prepareSubscriptionConversionTest(t)

	user := createSubscriptionConversionUser(t, "default", 1000)
	plan := createSubscriptionConversionPlan(t, "VIP Monthly", 10, "vip")

	var sub *UserSubscription
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		var err error
		sub, err = CreateUserSubscriptionFromPlanTx(tx, user.Id, plan, "order")
		return err
	}))
	require.NotNil(t, sub)

	var upgraded User
	require.NoError(t, DB.First(&upgraded, user.Id).Error)
	assert.Equal(t, "vip", upgraded.Group)

	message, err := AdminDeleteUserSubscription(sub.Id)
	require.NoError(t, err)
	assert.Contains(t, message, "default")

	var refreshed User
	require.NoError(t, DB.First(&refreshed, user.Id).Error)
	assert.Equal(t, "default", refreshed.Group)

	var count int64
	require.NoError(t, DB.Model(&UserSubscription{}).Where("id = ?", sub.Id).Count(&count).Error)
	assert.Zero(t, count)
}

func TestAdminDeleteUserSubscription_PreservesOriginalFallbackForHigherTier(t *testing.T) {
	prepareSubscriptionConversionTest(t)

	user := createSubscriptionConversionUser(t, "default", 1000)
	vipPlan := createSubscriptionConversionPlan(t, "VIP Monthly", 10, "vip")
	proPlan := createSubscriptionConversionPlan(t, "PRO Monthly", 20, "pro")

	var vipSub *UserSubscription
	var proSub *UserSubscription
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		var err error
		vipSub, err = CreateUserSubscriptionFromPlanTx(tx, user.Id, vipPlan, "order")
		if err != nil {
			return err
		}
		proSub, err = CreateUserSubscriptionFromPlanTx(tx, user.Id, proPlan, "order")
		return err
	}))
	require.NotNil(t, vipSub)
	require.NotNil(t, proSub)

	var upgraded User
	require.NoError(t, DB.First(&upgraded, user.Id).Error)
	assert.Equal(t, "pro", upgraded.Group)

	message, err := AdminDeleteUserSubscription(vipSub.Id)
	require.NoError(t, err)
	assert.Empty(t, message)

	var afterVipDelete User
	require.NoError(t, DB.First(&afterVipDelete, user.Id).Error)
	assert.Equal(t, "pro", afterVipDelete.Group)

	message, err = AdminDeleteUserSubscription(proSub.Id)
	require.NoError(t, err)
	assert.Contains(t, message, "default")

	var finalUser User
	require.NoError(t, DB.First(&finalUser, user.Id).Error)
	assert.Equal(t, "default", finalUser.Group)
}

func TestExecuteSubscriptionWalletConversion_AfterDeletingLowerTierFallsBackToOriginalBaseGroup(t *testing.T) {
	prepareSubscriptionConversionTest(t)

	user := createSubscriptionConversionUser(t, "default", 1000)
	vipPlan := createSubscriptionConversionPlan(t, "VIP Monthly", 10, "vip")
	proPlan := createSubscriptionConversionPlan(t, "PRO Monthly", 20, "pro")

	var vipSub *UserSubscription
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		var err error
		vipSub, err = CreateUserSubscriptionFromPlanTx(tx, user.Id, vipPlan, "order")
		if err != nil {
			return err
		}
		_, err = CreateUserSubscriptionFromPlanTx(tx, user.Id, proPlan, "order")
		return err
	}))
	require.NotNil(t, vipSub)

	_, err := AdminDeleteUserSubscription(vipSub.Id)
	require.NoError(t, err)

	result, err := ExecuteSubscriptionWalletConversion(user.Id, fmt.Sprintf("delete-lower-tier-%d", time.Now().UnixNano()))
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "default", result.UserGroupAfter)

	var refreshed User
	require.NoError(t, DB.First(&refreshed, user.Id).Error)
	assert.Equal(t, "default", refreshed.Group)
}
