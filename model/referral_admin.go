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
	invitees, _, err := listReferralInvitees(DB.Model(&User{}), "", nil)
	if err != nil {
		return nil, err
	}
	summary := &ReferralAdminSummary{
		TotalRelations: int64(len(invitees)),
	}
	if len(invitees) == 0 {
		return summary, nil
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
		summary.TotalInviteeUsedQuota += int64(invitee.UsedQuota)
	}
	summary.InviterCount = int64(len(inviterIDs))

	userMap, err := getUsersByIDs(append(append([]int{}, inviteeIDs...), inviterIDs...))
	if err != nil {
		return nil, err
	}
	aggregates, err := getReferralTopUpAggregates(inviteeIDs, inviterIDs)
	if err != nil {
		return nil, err
	}

	for _, invitee := range invitees {
		inviter := userMap[invitee.InviterId]
		inviteeTopup := aggregates[invitee.Id]
		inviterTopup := aggregates[invitee.InviterId]
		if inviteeTopup.TotalAmount > 0 {
			summary.InviteeWithTopUpCount++
			summary.TotalInviteeTopUp += inviteeTopup.TotalAmount
		}
		if inviter != nil {
			if invitee.RegisterIP != "" && inviter.RegisterIP != "" && invitee.RegisterIP == inviter.RegisterIP {
				summary.SameRegisterIPCount++
			}
			if invitee.LastLoginIP != "" && inviter.LastLoginIP != "" && invitee.LastLoginIP == inviter.LastLoginIP {
				summary.SameLoginIPCount++
			}
			if inviteeTopup.LastIP != "" && inviterTopup.LastIP != "" && inviteeTopup.LastIP == inviterTopup.LastIP {
				summary.SameTopUpIPCount++
			}
		}
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
		tags = append(tags, "注册IP一致")
	}
	if invitee.LastLoginIP != "" && inviter.LastLoginIP != "" && invitee.LastLoginIP == inviter.LastLoginIP {
		tags = append(tags, "最近登录IP一致")
	}
	if inviteeTopup.LastIP != "" && inviterTopup.LastIP != "" && inviteeTopup.LastIP == inviterTopup.LastIP {
		tags = append(tags, "最近支付IP一致")
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
		searchQuery := query.Where("username LIKE ? OR display_name LIKE ? OR email LIKE ?", like, like, like)
		if len(inviterIDs) > 0 {
			searchQuery = searchQuery.Or("inviter_id IN ?", inviterIDs)
		}
		query = searchQuery
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
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
	var topups []*TopUp
	if err := DB.Where("user_id IN ? AND status = ?", userIDs, common.TopUpStatusSuccess).
		Order("user_id asc").
		Order("id desc").
		Find(&topups).Error; err != nil {
		return nil, err
	}
	for _, topup := range topups {
		aggregate := aggregates[topup.UserId]
		aggregate.TotalAmount += topup.GetEffectivePaidAmount()
		if aggregate.LastIP == "" && topup.ClientIP != "" {
			aggregate.LastIP = topup.ClientIP
		}
		aggregates[topup.UserId] = aggregate
	}
	return aggregates, nil
}
