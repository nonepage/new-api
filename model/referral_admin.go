package model

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

type ReferralAdminSummary struct {
	TotalRelations        int64   `json:"total_relations"`
	InviterCount          int64   `json:"inviter_count"`
	InviteeWithTopUpCount int64   `json:"invitee_with_topup_count"`
	TotalInviteeTopUp     float64 `json:"total_invitee_topup"`
	TotalInviteeUsedQuota int64   `json:"total_invitee_used_quota"`
	SameRegisterIPCount   int64   `json:"same_register_ip_count"`
	SameLoginIPCount      int64   `json:"same_login_ip_count"`
	SameTopUpIPCount      int64   `json:"same_topup_ip_count"`
}

type ReferralAdminRelation struct {
	InviterId          int      `json:"inviter_id"`
	InviterUsername    string   `json:"inviter_username"`
	InviterDisplayName string   `json:"inviter_display_name"`
	InviterStatus      int      `json:"inviter_status"`
	InviterAffCount    int      `json:"inviter_aff_count"`
	InviterRewardQuota int      `json:"inviter_reward_quota"`
	InviterRegisterIP  string   `json:"inviter_register_ip"`
	InviterLastLoginIP string   `json:"inviter_last_login_ip"`
	InviterLastTopUpIP string   `json:"inviter_last_topup_ip"`
	InviteeId          int      `json:"invitee_id"`
	InviteeUsername    string   `json:"invitee_username"`
	InviteeDisplayName string   `json:"invitee_display_name"`
	InviteeStatus      int      `json:"invitee_status"`
	InviteeRegisterIP  string   `json:"invitee_register_ip"`
	InviteeLastLoginIP string   `json:"invitee_last_login_ip"`
	InviteeLastTopUpIP string   `json:"invitee_last_topup_ip"`
	InviteeTopUpAmount float64  `json:"invitee_topup_amount"`
	InviteeUsedQuota   int      `json:"invitee_used_quota"`
	RiskTags           []string `json:"risk_tags"`
}

type ReferralAdminDetail struct {
	Relation ReferralAdminRelation `json:"relation"`
	TopUps   []*TopUp              `json:"topups"`
}

type referralTopUpAggregate struct {
	TotalAmount float64
	LastIP      string
}

func GetReferralAdminSummary() (*ReferralAdminSummary, error) {
	summary := &ReferralAdminSummary{}
	relationQuery := DB.Model(&User{}).Where("inviter_id > 0")

	if err := relationQuery.Count(&summary.TotalRelations).Error; err != nil {
		return nil, err
	}
	if summary.TotalRelations == 0 {
		return summary, nil
	}

	if err := DB.Model(&User{}).Where("inviter_id > 0").Distinct("inviter_id").Count(&summary.InviterCount).Error; err != nil {
		return nil, err
	}

	type usedQuotaSummary struct {
		TotalUsedQuota int64 `gorm:"column:total_used_quota"`
	}
	var usedQuota usedQuotaSummary
	if err := DB.Model(&User{}).
		Select("COALESCE(SUM(used_quota), 0) AS total_used_quota").
		Where("inviter_id > 0").
		Scan(&usedQuota).Error; err != nil {
		return nil, err
	}
	summary.TotalInviteeUsedQuota = usedQuota.TotalUsedQuota

	type topUpSummary struct {
		InviteeWithTopUpCount int64   `gorm:"column:invitee_with_topup_count"`
		TotalInviteeTopUp     float64 `gorm:"column:total_invitee_topup"`
	}
	var rechargeSummary topUpSummary
	effectiveAmountExpr := "CASE WHEN top_ups.paid_amount > 0 THEN top_ups.paid_amount WHEN top_ups.money > 0 THEN top_ups.money ELSE top_ups.amount END"
	if err := DB.Table("top_ups").
		Joins("JOIN users invitees ON invitees.id = top_ups.user_id").
		Select("COUNT(DISTINCT top_ups.user_id) AS invitee_with_topup_count, COALESCE(SUM("+effectiveAmountExpr+"), 0) AS total_invitee_topup").
		Where("invitees.inviter_id > 0 AND top_ups.status = ?", common.TopUpStatusSuccess).
		Scan(&rechargeSummary).Error; err != nil {
		return nil, err
	}
	summary.InviteeWithTopUpCount = rechargeSummary.InviteeWithTopUpCount
	summary.TotalInviteeTopUp = rechargeSummary.TotalInviteeTopUp

	if err := DB.Table("users AS invitees").
		Joins("JOIN users AS inviters ON inviters.id = invitees.inviter_id").
		Where("invitees.inviter_id > 0").
		Where("invitees.register_ip <> '' AND inviters.register_ip <> ''").
		Where("invitees.register_ip = inviters.register_ip").
		Count(&summary.SameRegisterIPCount).Error; err != nil {
		return nil, err
	}

	if err := DB.Table("users AS invitees").
		Joins("JOIN users AS inviters ON inviters.id = invitees.inviter_id").
		Where("invitees.inviter_id > 0").
		Where("invitees.last_login_ip <> '' AND inviters.last_login_ip <> ''").
		Where("invitees.last_login_ip = inviters.last_login_ip").
		Count(&summary.SameLoginIPCount).Error; err != nil {
		return nil, err
	}

	latestTopUpSubQuery := DB.Model(&TopUp{}).
		Select("user_id, MAX(id) AS latest_id").
		Where("status = ?", common.TopUpStatusSuccess).
		Group("user_id")

	if err := DB.Table("users AS invitees").
		Joins("JOIN users AS inviters ON inviters.id = invitees.inviter_id").
		Joins("JOIN (?) AS invitee_latest ON invitee_latest.user_id = invitees.id", latestTopUpSubQuery).
		Joins("JOIN top_ups AS invitee_topups ON invitee_topups.id = invitee_latest.latest_id").
		Joins("JOIN (?) AS inviter_latest ON inviter_latest.user_id = inviters.id", latestTopUpSubQuery).
		Joins("JOIN top_ups AS inviter_topups ON inviter_topups.id = inviter_latest.latest_id").
		Where("invitees.inviter_id > 0").
		Where("invitee_topups.client_ip <> '' AND inviter_topups.client_ip <> ''").
		Where("invitee_topups.client_ip = inviter_topups.client_ip").
		Count(&summary.SameTopUpIPCount).Error; err != nil {
		return nil, err
	}

	return summary, nil
}

func GetReferralAdminRelations(keyword string, pageInfo *common.PageInfo) ([]*ReferralAdminRelation, int64, error) {
	baseQuery := DB.Model(&User{})
	invitees, total, err := listReferralInvitees(baseQuery, keyword, pageInfo)
	if err != nil {
		return nil, 0, err
	}
	if len(invitees) == 0 {
		return []*ReferralAdminRelation{}, total, nil
	}

	inviteeIDs := make([]int, 0, len(invitees))
	inviterIDs := make([]int, 0, len(invitees))
	inviterIDSet := make(map[int]struct{})
	for _, invitee := range invitees {
		inviteeIDs = append(inviteeIDs, invitee.Id)
		if invitee.InviterId > 0 {
			if _, ok := inviterIDSet[invitee.InviterId]; !ok {
				inviterIDSet[invitee.InviterId] = struct{}{}
				inviterIDs = append(inviterIDs, invitee.InviterId)
			}
		}
	}

	userMap, err := getUsersByIDs(append(append([]int{}, inviteeIDs...), inviterIDs...))
	if err != nil {
		return nil, 0, err
	}
	aggregates, err := getReferralTopUpAggregates(inviteeIDs, inviterIDs)
	if err != nil {
		return nil, 0, err
	}

	relations := make([]*ReferralAdminRelation, 0, len(invitees))
	for _, invitee := range invitees {
		inviter := userMap[invitee.InviterId]
		if inviter == nil {
			continue
		}
		relation := &ReferralAdminRelation{
			InviterId:          inviter.Id,
			InviterUsername:    inviter.Username,
			InviterDisplayName: inviter.DisplayName,
			InviterStatus:      inviter.Status,
			InviterAffCount:    inviter.AffCount,
			InviterRewardQuota: inviter.AffHistoryQuota,
			InviterRegisterIP:  inviter.RegisterIP,
			InviterLastLoginIP: inviter.LastLoginIP,
			InviterLastTopUpIP: aggregates[inviter.Id].LastIP,
			InviteeId:          invitee.Id,
			InviteeUsername:    invitee.Username,
			InviteeDisplayName: invitee.DisplayName,
			InviteeStatus:      invitee.Status,
			InviteeRegisterIP:  invitee.RegisterIP,
			InviteeLastLoginIP: invitee.LastLoginIP,
			InviteeLastTopUpIP: aggregates[invitee.Id].LastIP,
			InviteeTopUpAmount: aggregates[invitee.Id].TotalAmount,
			InviteeUsedQuota:   invitee.UsedQuota,
			RiskTags:           buildReferralRiskTags(inviter, invitee, aggregates[inviter.Id], aggregates[invitee.Id]),
		}
		relations = append(relations, relation)
	}

	return relations, total, nil
}

func GetReferralAdminDetail(inviteeId int) (*ReferralAdminDetail, error) {
	var invitee User
	if err := DB.Where("id = ? AND inviter_id > 0", inviteeId).First(&invitee).Error; err != nil {
		return nil, err
	}
	userMap, err := getUsersByIDs([]int{invitee.InviterId, invitee.Id})
	if err != nil {
		return nil, err
	}
	inviter := userMap[invitee.InviterId]
	if inviter == nil {
		return nil, errors.New("inviter not found")
	}
	aggregates, err := getReferralTopUpAggregates([]int{invitee.Id}, []int{inviter.Id})
	if err != nil {
		return nil, err
	}
	var topups []*TopUp
	if err := DB.Where("user_id = ? AND status = ?", invitee.Id, common.TopUpStatusSuccess).Order("id desc").Find(&topups).Error; err != nil {
		return nil, err
	}
	return &ReferralAdminDetail{
		Relation: ReferralAdminRelation{
			InviterId:          inviter.Id,
			InviterUsername:    inviter.Username,
			InviterDisplayName: inviter.DisplayName,
			InviterStatus:      inviter.Status,
			InviterAffCount:    inviter.AffCount,
			InviterRewardQuota: inviter.AffHistoryQuota,
			InviterRegisterIP:  inviter.RegisterIP,
			InviterLastLoginIP: inviter.LastLoginIP,
			InviterLastTopUpIP: aggregates[inviter.Id].LastIP,
			InviteeId:          invitee.Id,
			InviteeUsername:    invitee.Username,
			InviteeDisplayName: invitee.DisplayName,
			InviteeStatus:      invitee.Status,
			InviteeRegisterIP:  invitee.RegisterIP,
			InviteeLastLoginIP: invitee.LastLoginIP,
			InviteeLastTopUpIP: aggregates[invitee.Id].LastIP,
			InviteeTopUpAmount: aggregates[invitee.Id].TotalAmount,
			InviteeUsedQuota:   invitee.UsedQuota,
			RiskTags:           buildReferralRiskTags(inviter, &invitee, aggregates[inviter.Id], aggregates[invitee.Id]),
		},
		TopUps: topups,
	}, nil
}

func buildReferralRiskTags(inviter *User, invitee *User, inviterTopup referralTopUpAggregate, inviteeTopup referralTopUpAggregate) []string {
	tags := make([]string, 0, 4)
	if inviter == nil || invitee == nil {
		return tags
	}
	if invitee.RegisterIP != "" && inviter.RegisterIP != "" && invitee.RegisterIP == inviter.RegisterIP {
		tags = append(tags, "注册 IP 一致")
	}
	if invitee.LastLoginIP != "" && inviter.LastLoginIP != "" && invitee.LastLoginIP == inviter.LastLoginIP {
		tags = append(tags, "最近登录 IP 一致")
	}
	if inviteeTopup.LastIP != "" && inviterTopup.LastIP != "" && inviteeTopup.LastIP == inviterTopup.LastIP {
		tags = append(tags, "最近支付 IP 一致")
	}
	if inviteeTopup.TotalAmount > 0 && invitee.UsedQuota == 0 {
		tags = append(tags, "有充值但无消费")
	}
	return tags
}

func listReferralInvitees(baseQuery *gorm.DB, keyword string, pageInfo *common.PageInfo) ([]*User, int64, error) {
	query := baseQuery.Where("inviter_id > 0")
	if keyword != "" {
		like := "%" + strings.TrimSpace(keyword) + "%"
		var inviterIDs []int
		if err := DB.Model(&User{}).
			Where("username LIKE ? OR display_name LIKE ? OR email LIKE ?", like, like, like).
			Pluck("id", &inviterIDs).Error; err != nil {
			return nil, 0, err
		}
		searchQuery := query.Where(
			DB.Where("username LIKE ? OR display_name LIKE ? OR email LIKE ?", like, like, like),
		)
		if len(inviterIDs) > 0 {
			searchQuery = searchQuery.Or("inviter_id IN ?", inviterIDs)
		}
		query = searchQuery
	}

	var total int64
	countQuery := query.Session(&gorm.Session{})
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query = query.Order("id desc")
	if pageInfo != nil {
		query = query.Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx())
	}

	var invitees []*User
	if err := query.Find(&invitees).Error; err != nil {
		return nil, 0, err
	}
	return invitees, total, nil
}

func getUsersByIDs(userIDs []int) (map[int]*User, error) {
	userMap := make(map[int]*User)
	if len(userIDs) == 0 {
		return userMap, nil
	}
	var users []*User
	if err := DB.Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		return nil, err
	}
	for _, user := range users {
		userMap[user.Id] = user
	}
	return userMap, nil
}

func getReferralTopUpAggregates(inviteeIDs []int, inviterIDs []int) (map[int]referralTopUpAggregate, error) {
	userIDs := append(append([]int{}, inviteeIDs...), inviterIDs...)
	aggregates := make(map[int]referralTopUpAggregate)
	if len(userIDs) == 0 {
		return aggregates, nil
	}

	type totalRow struct {
		UserId      int     `gorm:"column:user_id"`
		TotalAmount float64 `gorm:"column:total_amount"`
	}
	var totalRows []totalRow
	effectiveAmountExpr := "CASE WHEN paid_amount > 0 THEN paid_amount WHEN money > 0 THEN money ELSE amount END"
	if err := DB.Model(&TopUp{}).
		Select("user_id, COALESCE(SUM("+effectiveAmountExpr+"), 0) AS total_amount").
		Where("user_id IN ? AND status = ?", userIDs, common.TopUpStatusSuccess).
		Group("user_id").
		Scan(&totalRows).Error; err != nil {
		return nil, err
	}
	for _, row := range totalRows {
		aggregate := aggregates[row.UserId]
		aggregate.TotalAmount = row.TotalAmount
		aggregates[row.UserId] = aggregate
	}

	latestTopUpSubQuery := DB.Model(&TopUp{}).
		Select("user_id, MAX(id) AS latest_id").
		Where("user_id IN ? AND status = ?", userIDs, common.TopUpStatusSuccess).
		Group("user_id")

	type latestRow struct {
		UserId int    `gorm:"column:user_id"`
		LastIP string `gorm:"column:last_ip"`
	}
	var latestRows []latestRow
	if err := DB.Table("top_ups AS topup").
		Select("topup.user_id, topup.client_ip AS last_ip").
		Joins("JOIN (?) AS latest ON latest.latest_id = topup.id", latestTopUpSubQuery).
		Scan(&latestRows).Error; err != nil {
		return nil, err
	}
	for _, row := range latestRows {
		aggregate := aggregates[row.UserId]
		aggregate.LastIP = row.LastIP
		aggregates[row.UserId] = aggregate
	}

	return aggregates, nil
}
