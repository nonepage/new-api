package controller

import (
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

type invoiceProfileRequest struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	TaxNo       string `json:"tax_no"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Address     string `json:"address"`
	BankName    string `json:"bank_name"`
	BankAccount string `json:"bank_account"`
	IsDefault   bool   `json:"is_default"`
}

type invoiceApplicationRequest struct {
	ProfileId int    `json:"profile_id"`
	TopUpIds  []int  `json:"topup_ids"`
	Remark    string `json:"remark"`

	Type        string `json:"type"`
	Title       string `json:"title"`
	TaxNo       string `json:"tax_no"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Address     string `json:"address"`
	BankName    string `json:"bank_name"`
	BankAccount string `json:"bank_account"`
}

type invoiceReviewRequest struct {
	AdminRemark    string `json:"admin_remark"`
	RejectedReason string `json:"rejected_reason"`
}

type invoiceIssueRequest struct {
	ApplicationIds []int  `json:"application_ids"`
	InvoiceNo      string `json:"invoice_no"`
	FileURL        string `json:"file_url"`
	Remark         string `json:"remark"`
}

type invoiceVoidRequest struct {
	Remark string `json:"remark"`
}

func GetInvoiceProfiles(c *gin.Context) {
	profiles, err := model.ListInvoiceProfilesByUser(c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, profiles)
}

func SaveInvoiceProfile(c *gin.Context) {
	var req invoiceProfileRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, err)
		return
	}
	profile := &model.InvoiceProfile{
		UserId:      c.GetInt("id"),
		Type:        strings.TrimSpace(req.Type),
		Title:       strings.TrimSpace(req.Title),
		TaxNo:       strings.TrimSpace(req.TaxNo),
		Email:       strings.TrimSpace(req.Email),
		Phone:       strings.TrimSpace(req.Phone),
		Address:     strings.TrimSpace(req.Address),
		BankName:    strings.TrimSpace(req.BankName),
		BankAccount: strings.TrimSpace(req.BankAccount),
		IsDefault:   req.IsDefault,
	}
	if idStr := c.Param("id"); idStr != "" {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			common.ApiErrorMsg(c, "invalid invoice profile id")
			return
		}
		profile.Id = id
	}
	if err := model.SaveInvoiceProfile(profile); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, profile)
}

func DeleteInvoiceProfile(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorMsg(c, "invalid invoice profile id")
		return
	}
	if err := model.DeleteInvoiceProfile(c.GetInt("id"), id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, true)
}

func GetInvoiceAvailableOrders(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	items, total, err := model.ListInvoiceAvailableTopUps(c.GetInt("id"), c.Query("keyword"), pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func GetSelfInvoiceApplications(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	items, total, err := model.ListInvoiceApplicationsByUser(c.GetInt("id"), c.Query("keyword"), pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func GetSelfInvoiceRecords(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	items, total, err := model.ListInvoiceRecords(c.Query("keyword"), c.GetInt("id"), pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func CreateInvoiceApplication(c *gin.Context) {
	var req invoiceApplicationRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, err)
		return
	}
	snapshot := model.InvoiceProfileSnapshot{
		Type:        req.Type,
		Title:       req.Title,
		TaxNo:       req.TaxNo,
		Email:       req.Email,
		Phone:       req.Phone,
		Address:     req.Address,
		BankName:    req.BankName,
		BankAccount: req.BankAccount,
	}
	if req.ProfileId > 0 {
		profile, err := model.GetInvoiceProfileByUser(c.GetInt("id"), req.ProfileId)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		snapshot = profile.ToSnapshot()
	}
	application, err := model.CreateInvoiceApplication(c.GetInt("id"), req.TopUpIds, snapshot, req.Remark)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, application)
}

func CancelInvoiceApplication(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorMsg(c, "invalid invoice application id")
		return
	}
	if err := model.CancelInvoiceApplication(c.GetInt("id"), id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, true)
}

func GetAdminInvoiceApplications(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	userId, _ := strconv.Atoi(c.Query("user_id"))
	items, total, err := model.ListInvoiceApplicationsByAdmin(c.Query("keyword"), c.Query("status"), userId, pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func ApproveInvoiceApplication(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorMsg(c, "invalid invoice application id")
		return
	}
	var req invoiceReviewRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.ApproveInvoiceApplication(id, c.GetInt("id"), req.AdminRemark); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, true)
}

func RejectInvoiceApplication(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorMsg(c, "invalid invoice application id")
		return
	}
	var req invoiceReviewRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.RejectInvoiceApplication(id, c.GetInt("id"), req.RejectedReason, req.AdminRemark); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, true)
}

func GetAdminInvoiceRecords(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	userId, _ := strconv.Atoi(c.Query("user_id"))
	items, total, err := model.ListInvoiceRecords(c.Query("keyword"), userId, pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func IssueInvoiceRecord(c *gin.Context) {
	var req invoiceIssueRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, err)
		return
	}
	record, err := model.IssueInvoiceRecord(req.ApplicationIds, c.GetInt("id"), req.InvoiceNo, req.FileURL, req.Remark)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, record)
}

func VoidInvoiceRecord(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorMsg(c, "invalid invoice record id")
		return
	}
	var req invoiceVoidRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.VoidInvoiceRecord(id, c.GetInt("id"), req.Remark); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, true)
}
