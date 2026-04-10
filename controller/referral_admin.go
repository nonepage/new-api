package controller

import (
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func GetReferralAdminSummary(c *gin.Context) {
	summary, err := model.GetReferralAdminSummary()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, summary)
}

func GetReferralAdminRelations(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	items, total, err := model.GetReferralAdminRelations(c.Query("keyword"), pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func GetReferralAdminDetail(c *gin.Context) {
	inviteeId, err := strconv.Atoi(c.Param("invitee_id"))
	if err != nil {
		common.ApiErrorMsg(c, "invalid invitee id")
		return
	}
	detail, err := model.GetReferralAdminDetail(inviteeId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, detail)
}
