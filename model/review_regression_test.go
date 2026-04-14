package model

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func prepareReviewRegressionTest(t *testing.T) {
	t.Helper()
	initCol()
	require.NoError(t, DB.Exec(`
		CREATE TABLE IF NOT EXISTS top_ups (
			id INTEGER PRIMARY KEY,
			user_id INTEGER,
			amount BIGINT,
			money DECIMAL(10,6),
			paid_amount DECIMAL(12,6) DEFAULT 0,
			paid_currency VARCHAR(16) DEFAULT '',
			trade_no VARCHAR(255) UNIQUE,
			payment_method VARCHAR(50),
			source_type VARCHAR(32) DEFAULT 'wallet_topup',
			client_ip VARCHAR(64) DEFAULT '',
			user_agent TEXT,
			invoice_status VARCHAR(32) DEFAULT 'none',
			create_time BIGINT,
			complete_time BIGINT,
			status VARCHAR(32)
		)
	`).Error)
	require.NoError(t, DB.Exec(`
		CREATE TABLE IF NOT EXISTS invoice_applications (
			id INTEGER PRIMARY KEY,
			application_no VARCHAR(64) UNIQUE,
			user_id INTEGER,
			status VARCHAR(32),
			total_amount DECIMAL(12,6) DEFAULT 0,
			currency VARCHAR(16) DEFAULT '',
			profile_snapshot TEXT,
			remark TEXT,
			admin_remark TEXT,
			reviewed_by INTEGER DEFAULT 0,
			reviewed_at BIGINT DEFAULT 0,
			rejected_reason TEXT,
			created_at BIGINT,
			updated_at BIGINT
		)
	`).Error)
	require.NoError(t, DB.Exec(`
		CREATE TABLE IF NOT EXISTS invoice_application_items (
			id INTEGER PRIMARY KEY,
			application_id INTEGER,
			top_up_id INTEGER,
			order_trade_no VARCHAR(255),
			amount DECIMAL(12,6) DEFAULT 0,
			currency VARCHAR(16) DEFAULT '',
			created_at BIGINT
		)
	`).Error)
	require.NoError(t, DB.Exec(`
		CREATE TABLE IF NOT EXISTS invoice_records (
			id INTEGER PRIMARY KEY,
			invoice_no VARCHAR(64) UNIQUE,
			user_id INTEGER,
			status VARCHAR(32),
			total_amount DECIMAL(12,6) DEFAULT 0,
			currency VARCHAR(16) DEFAULT '',
			issued_at BIGINT DEFAULT 0,
			issuer_id INTEGER DEFAULT 0,
			file_url TEXT,
			remark TEXT,
			voided_at BIGINT DEFAULT 0,
			voided_by INTEGER DEFAULT 0,
			void_remark TEXT,
			created_at BIGINT,
			updated_at BIGINT
		)
	`).Error)
	require.NoError(t, DB.Exec(`
		CREATE TABLE IF NOT EXISTS invoice_record_applications (
			id INTEGER PRIMARY KEY,
			invoice_record_id INTEGER,
			application_id INTEGER,
			created_at BIGINT
		)
	`).Error)
	require.NoError(t, DB.Exec(`
		CREATE TABLE IF NOT EXISTS logs (
			id INTEGER PRIMARY KEY,
			user_id INTEGER,
			created_at BIGINT,
			type INTEGER,
			content TEXT,
			username VARCHAR(255) DEFAULT '',
			token_name VARCHAR(255) DEFAULT '',
			model_name VARCHAR(255) DEFAULT '',
			quota INTEGER DEFAULT 0,
			prompt_tokens INTEGER DEFAULT 0,
			completion_tokens INTEGER DEFAULT 0,
			use_time INTEGER DEFAULT 0,
			is_stream BOOLEAN DEFAULT 0,
			channel_id INTEGER DEFAULT 0,
			token_id INTEGER DEFAULT 0,
			"group" VARCHAR(64) DEFAULT '',
			ip VARCHAR(64) DEFAULT '',
			request_id VARCHAR(64) DEFAULT '',
			other TEXT
		)
	`).Error)
	t.Cleanup(func() {
		DB.Exec("DELETE FROM logs")
		DB.Exec("DELETE FROM invoice_record_applications")
		DB.Exec("DELETE FROM invoice_records")
		DB.Exec("DELETE FROM invoice_application_items")
		DB.Exec("DELETE FROM invoice_applications")
		DB.Exec("DELETE FROM top_ups")
		DB.Exec("DELETE FROM users")
	})
}

func createReviewTestUser(t *testing.T, username string, inviterId int, registerIP string, loginIP string) *User {
	t.Helper()
	user := &User{
		Username:    username,
		Password:    "password123",
		DisplayName: username,
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		AffCode:     username + "-aff",
		InviterId:   inviterId,
		RegisterIP:  registerIP,
		LastLoginIP: loginIP,
	}
	require.NoError(t, DB.Create(user).Error)
	return user
}

func TestGetReferralAdminSummary_AggregatesWithoutLoadingAllTopUpsIntoGoLogic(t *testing.T) {
	prepareReviewRegressionTest(t)

	inviter := createReviewTestUser(t, "inviter-summary", 0, "1.1.1.1", "2.2.2.2")
	invitee := createReviewTestUser(t, "invitee-summary", inviter.Id, "1.1.1.1", "2.2.2.2")
	invitee.UsedQuota = 128
	require.NoError(t, DB.Model(&User{}).Where("id = ?", invitee.Id).Update("used_quota", invitee.UsedQuota).Error)

	require.NoError(t, DB.Create(&TopUp{
		UserId:        inviter.Id,
		TradeNo:       "inviter-topup-summary",
		Status:        common.TopUpStatusSuccess,
		Money:         20,
		ClientIP:      "3.3.3.3",
		PaymentMethod: "stripe",
		CreateTime:    time.Now().Unix(),
		CompleteTime:  time.Now().Unix(),
	}).Error)
	require.NoError(t, DB.Create(&TopUp{
		UserId:        invitee.Id,
		TradeNo:       "invitee-topup-summary",
		Status:        common.TopUpStatusSuccess,
		PaidAmount:    12.5,
		ClientIP:      "3.3.3.3",
		PaymentMethod: "stripe",
		CreateTime:    time.Now().Unix(),
		CompleteTime:  time.Now().Unix(),
	}).Error)

	summary, err := GetReferralAdminSummary()
	require.NoError(t, err)
	require.NotNil(t, summary)

	assert.EqualValues(t, 1, summary.TotalRelations)
	assert.EqualValues(t, 1, summary.InviterCount)
	assert.EqualValues(t, 1, summary.InviteeWithTopUpCount)
	assert.Equal(t, 12.5, summary.TotalInviteeTopUp)
	assert.EqualValues(t, 128, summary.TotalInviteeUsedQuota)
	assert.EqualValues(t, 1, summary.SameRegisterIPCount)
	assert.EqualValues(t, 1, summary.SameLoginIPCount)
	assert.EqualValues(t, 1, summary.SameTopUpIPCount)
}

func TestBackfillTopUpDefaults_FillsLegacyEmptyFields(t *testing.T) {
	prepareReviewRegressionTest(t)

	user := createReviewTestUser(t, "legacy-topup-user", 0, "", "")
	require.NoError(t, DB.Exec(
		"INSERT INTO top_ups (user_id, amount, money, trade_no, payment_method, source_type, invoice_status, status, create_time, complete_time) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		user.Id,
		100,
		10,
		"legacy-topup",
		"alipay",
		"",
		"",
		common.TopUpStatusSuccess,
		time.Now().Unix(),
		time.Now().Unix(),
	).Error)

	require.NoError(t, backfillTopUpDefaults())

	var topup TopUp
	require.NoError(t, DB.Where("trade_no = ?", "legacy-topup").First(&topup).Error)
	assert.Equal(t, common.TopUpSourceWalletTopUp, topup.SourceType)
	assert.Equal(t, common.InvoiceStatusNone, topup.InvoiceStatus)
}

func TestVoidInvoiceRecord_PreservesIssueAuditAndStoresVoidAudit(t *testing.T) {
	prepareReviewRegressionTest(t)

	user := createReviewTestUser(t, "invoice-user", 0, "", "")
	topup := &TopUp{
		UserId:        user.Id,
		TradeNo:       "invoice-void-topup",
		Status:        common.TopUpStatusSuccess,
		PaidAmount:    18.8,
		PaidCurrency:  "USD",
		InvoiceStatus: common.InvoiceStatusIssued,
		PaymentMethod: "stripe",
		CreateTime:    time.Now().Unix(),
		CompleteTime:  time.Now().Unix(),
	}
	require.NoError(t, DB.Create(topup).Error)

	application := &InvoiceApplication{
		UserId:      user.Id,
		Status:      common.InvoiceApplicationStatusIssued,
		TotalAmount: 18.8,
		Currency:    "USD",
	}
	require.NoError(t, DB.Create(application).Error)
	require.NoError(t, DB.Create(&InvoiceApplicationItem{
		ApplicationId: application.Id,
		TopUpId:       topup.Id,
		OrderTradeNo:  topup.TradeNo,
		Amount:        18.8,
		Currency:      "USD",
	}).Error)

	record := &InvoiceRecord{
		InvoiceNo:   "INV-VOID-TEST",
		UserId:      user.Id,
		Status:      common.InvoiceRecordStatusIssued,
		TotalAmount: 18.8,
		Currency:    "USD",
		IssuedAt:    time.Now().Unix(),
		IssuerId:    1001,
		Remark:      "issued by finance",
	}
	require.NoError(t, DB.Create(record).Error)
	require.NoError(t, DB.Create(&InvoiceRecordApplication{
		InvoiceRecordId: record.Id,
		ApplicationId:   application.Id,
	}).Error)

	require.NoError(t, VoidInvoiceRecord(record.Id, 2002, "voided by finance"))

	var reloadedRecord InvoiceRecord
	require.NoError(t, DB.First(&reloadedRecord, record.Id).Error)
	assert.Equal(t, common.InvoiceRecordStatusVoided, reloadedRecord.Status)
	assert.EqualValues(t, 1001, reloadedRecord.IssuerId)
	assert.Equal(t, "issued by finance", reloadedRecord.Remark)
	assert.EqualValues(t, 2002, reloadedRecord.VoidedBy)
	assert.Equal(t, "voided by finance", reloadedRecord.VoidRemark)
	assert.NotZero(t, reloadedRecord.VoidedAt)
}

func TestCreateInvoiceApplication_UsesRoundedTotalsAndWritesAuditLog(t *testing.T) {
	prepareReviewRegressionTest(t)

	user := createReviewTestUser(t, "invoice-apply-user", 0, "", "")
	firstTopup := &TopUp{
		UserId:        user.Id,
		TradeNo:       "invoice-apply-topup-1",
		Status:        common.TopUpStatusSuccess,
		PaidAmount:    100.1,
		PaidCurrency:  "USD",
		InvoiceStatus: common.InvoiceStatusNone,
		PaymentMethod: "stripe",
		CreateTime:    time.Now().Unix(),
		CompleteTime:  time.Now().Unix(),
	}
	require.NoError(t, DB.Create(firstTopup).Error)
	secondTopup := &TopUp{
		UserId:        user.Id,
		TradeNo:       "invoice-apply-topup-2",
		Status:        common.TopUpStatusSuccess,
		PaidAmount:    100.2,
		PaidCurrency:  "USD",
		InvoiceStatus: common.InvoiceStatusNone,
		PaymentMethod: "stripe",
		CreateTime:    time.Now().Unix(),
		CompleteTime:  time.Now().Unix(),
	}
	require.NoError(t, DB.Create(secondTopup).Error)

	application, err := CreateInvoiceApplication(user.Id, []int{firstTopup.Id, secondTopup.Id}, InvoiceProfileSnapshot{
		Type:  "company",
		Title: "QuantumNous",
		TaxNo: "TAX-001",
		Email: "invoice@example.com",
	}, "precision test")
	require.NoError(t, err)
	require.NotNil(t, application)
	assert.Equal(t, "200.300000", decimal.NewFromFloat(application.TotalAmount).StringFixed(6))

	var logCount int64
	require.NoError(t, DB.Model(&Log{}).
		Where("user_id = ? AND type = ? AND content LIKE ?", user.Id, LogTypeSystem, "%submitted invoice application%").
		Count(&logCount).Error)
	assert.EqualValues(t, 1, logCount)
}

func TestCreateInvoiceApplication_RejectsSubscriptionOrders(t *testing.T) {
	prepareReviewRegressionTest(t)

	user := createReviewTestUser(t, "invoice-subscription-user", 0, "", "")
	topup := &TopUp{
		UserId:        user.Id,
		TradeNo:       "invoice-subscription-topup",
		Status:        common.TopUpStatusSuccess,
		PaidAmount:    288,
		PaidCurrency:  "CNY",
		InvoiceStatus: common.InvoiceStatusNone,
		PaymentMethod: "alipay",
		SourceType:    common.TopUpSourceSubscription,
		CreateTime:    time.Now().Unix(),
		CompleteTime:  time.Now().Unix(),
	}
	require.NoError(t, DB.Create(topup).Error)

	application, err := CreateInvoiceApplication(user.Id, []int{topup.Id}, InvoiceProfileSnapshot{
		Type:  "company",
		Title: "QuantumNous",
		TaxNo: "TAX-SUB-001",
		Email: "invoice@example.com",
	}, "subscription should fail")
	require.Nil(t, application)
	require.EqualError(t, err, "订阅订单暂不支持开具发票")

	var reloaded TopUp
	require.NoError(t, DB.First(&reloaded, topup.Id).Error)
	assert.Equal(t, common.InvoiceStatusNone, reloaded.InvoiceStatus)
}

func TestCreateInvoiceApplication_RejectsAmountBelowMinimum(t *testing.T) {
	prepareReviewRegressionTest(t)

	user := createReviewTestUser(t, "invoice-minimum-user", 0, "", "")
	topup := &TopUp{
		UserId:        user.Id,
		TradeNo:       "invoice-minimum-topup",
		Status:        common.TopUpStatusSuccess,
		PaidAmount:    199.99,
		PaidCurrency:  "CNY",
		InvoiceStatus: common.InvoiceStatusNone,
		PaymentMethod: "alipay",
		SourceType:    common.TopUpSourceWalletTopUp,
		CreateTime:    time.Now().Unix(),
		CompleteTime:  time.Now().Unix(),
	}
	require.NoError(t, DB.Create(topup).Error)

	application, err := CreateInvoiceApplication(user.Id, []int{topup.Id}, InvoiceProfileSnapshot{
		Type:  "company",
		Title: "QuantumNous",
		TaxNo: "TAX-MIN-001",
		Email: "invoice@example.com",
	}, "amount threshold should fail")
	require.Nil(t, application)
	require.EqualError(t, err, "开票申请总金额需大于等于 200 元，请在金额满足条件后再提交申请。")
}

func TestCreateInvoiceApplication_AllowsAmountAtMinimum(t *testing.T) {
	prepareReviewRegressionTest(t)

	user := createReviewTestUser(t, "invoice-minimum-allowed-user", 0, "", "")
	topup := &TopUp{
		UserId:        user.Id,
		TradeNo:       "invoice-minimum-allowed-topup",
		Status:        common.TopUpStatusSuccess,
		PaidAmount:    200,
		PaidCurrency:  "CNY",
		InvoiceStatus: common.InvoiceStatusNone,
		PaymentMethod: "alipay",
		SourceType:    common.TopUpSourceWalletTopUp,
		CreateTime:    time.Now().Unix(),
		CompleteTime:  time.Now().Unix(),
	}
	require.NoError(t, DB.Create(topup).Error)

	application, err := CreateInvoiceApplication(user.Id, []int{topup.Id}, InvoiceProfileSnapshot{
		Type:  "company",
		Title: "QuantumNous",
		TaxNo: "TAX-MIN-ALLOW-001",
		Email: "invoice@example.com",
	}, "amount threshold should pass")
	require.NoError(t, err)
	require.NotNil(t, application)
	assert.Equal(t, "200.000000", decimal.NewFromFloat(application.TotalAmount).StringFixed(6))
}

func TestApproveInvoiceApplication_WritesAdminAndUserAuditLogs(t *testing.T) {
	prepareReviewRegressionTest(t)

	user := createReviewTestUser(t, "invoice-approve-user", 0, "", "")
	topup := &TopUp{
		UserId:        user.Id,
		TradeNo:       "invoice-approve-topup",
		Status:        common.TopUpStatusSuccess,
		PaidAmount:    19.9,
		PaidCurrency:  "USD",
		InvoiceStatus: common.InvoiceStatusPendingReview,
		PaymentMethod: "stripe",
		CreateTime:    time.Now().Unix(),
		CompleteTime:  time.Now().Unix(),
	}
	require.NoError(t, DB.Create(topup).Error)
	application := &InvoiceApplication{
		UserId:      user.Id,
		Status:      common.InvoiceApplicationStatusPendingReview,
		TotalAmount: 19.9,
		Currency:    "USD",
	}
	require.NoError(t, DB.Create(application).Error)
	require.NoError(t, DB.Create(&InvoiceApplicationItem{
		ApplicationId: application.Id,
		TopUpId:       topup.Id,
		OrderTradeNo:  topup.TradeNo,
		Amount:        19.9,
		Currency:      "USD",
	}).Error)

	require.NoError(t, ApproveInvoiceApplication(application.Id, 9001, "approved in test"))

	var adminLogCount int64
	require.NoError(t, DB.Model(&Log{}).
		Where("user_id = ? AND type = ? AND content LIKE ?", 9001, LogTypeManage, "%approved invoice application%").
		Count(&adminLogCount).Error)
	assert.EqualValues(t, 1, adminLogCount)

	var userLogCount int64
	require.NoError(t, DB.Model(&Log{}).
		Where("user_id = ? AND type = ? AND content LIKE ?", user.Id, LogTypeSystem, "%was approved%").
		Count(&userLogCount).Error)
	assert.EqualValues(t, 1, userLogCount)
}

func TestRecordConsumeLog_AlwaysStoresClientIP(t *testing.T) {
	prepareReviewRegressionTest(t)
	gin.SetMode(gin.TestMode)

	user := createReviewTestUser(t, "ip-log-user", 0, "", "")
	settings := dto.UserSetting{RecordIpLog: false}
	user.SetSetting(settings)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", user.Id).Update("setting", user.Setting).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest("POST", "/v1/chat/completions", nil)
	request.RemoteAddr = "203.0.113.8:4567"
	ctx.Request = request
	ctx.Set("username", user.Username)

	RecordConsumeLog(ctx, user.Id, RecordConsumeLogParams{
		ChannelId:        1,
		PromptTokens:     10,
		CompletionTokens: 20,
		ModelName:        "gpt-test",
		TokenName:        "token-test",
		Quota:            100,
		Content:          "consume",
		TokenId:          1,
		UseTimeSeconds:   3,
		Group:            "default",
		Other:            map[string]interface{}{"k": "v"},
	})

	var log Log
	require.NoError(t, DB.Where("user_id = ? AND type = ?", user.Id, LogTypeConsume).First(&log).Error)
	assert.Equal(t, "203.0.113.8", log.Ip)
}
