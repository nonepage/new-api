package model

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

type InvoiceProfile struct {
	Id          int    `json:"id"`
	UserId      int    `json:"user_id" gorm:"index"`
	Type        string `json:"type" gorm:"type:varchar(32);default:'personal'"`
	Title       string `json:"title" gorm:"type:varchar(255)"`
	TaxNo       string `json:"tax_no" gorm:"type:varchar(128);default:''"`
	Email       string `json:"email" gorm:"type:varchar(255);default:''"`
	Phone       string `json:"phone" gorm:"type:varchar(64);default:''"`
	Address     string `json:"address" gorm:"type:varchar(255);default:''"`
	BankName    string `json:"bank_name" gorm:"type:varchar(255);default:''"`
	BankAccount string `json:"bank_account" gorm:"type:varchar(255);default:''"`
	IsDefault   bool   `json:"is_default" gorm:"default:false"`
	CreatedAt   int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt   int64  `json:"updated_at" gorm:"bigint"`
}

type InvoiceProfileSnapshot struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	TaxNo       string `json:"tax_no"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Address     string `json:"address"`
	BankName    string `json:"bank_name"`
	BankAccount string `json:"bank_account"`
}

type InvoiceApplication struct {
	Id              int                      `json:"id"`
	ApplicationNo   string                   `json:"application_no" gorm:"uniqueIndex;type:varchar(64)"`
	UserId          int                      `json:"user_id" gorm:"index"`
	Status          string                   `json:"status" gorm:"type:varchar(32);index"`
	TotalAmount     float64                  `json:"total_amount" gorm:"type:decimal(12,6);default:0"`
	Currency        string                   `json:"currency" gorm:"type:varchar(16);default:''"`
	ProfileSnapshot string                   `json:"profile_snapshot" gorm:"type:text"`
	Remark          string                   `json:"remark" gorm:"type:text"`
	AdminRemark     string                   `json:"admin_remark" gorm:"type:text"`
	ReviewedBy      int                      `json:"reviewed_by" gorm:"index;default:0"`
	ReviewedAt      int64                    `json:"reviewed_at" gorm:"type:bigint;default:0"`
	RejectedReason  string                   `json:"rejected_reason" gorm:"type:text"`
	CreatedAt       int64                    `json:"created_at" gorm:"bigint"`
	UpdatedAt       int64                    `json:"updated_at" gorm:"bigint"`
	Items           []InvoiceApplicationItem `json:"items,omitempty" gorm:"foreignKey:ApplicationId"`
}

type InvoiceApplicationItem struct {
	Id            int     `json:"id"`
	ApplicationId int     `json:"application_id" gorm:"index"`
	TopUpId       int     `json:"topup_id" gorm:"index"`
	OrderTradeNo  string  `json:"order_trade_no" gorm:"type:varchar(255);index"`
	Amount        float64 `json:"amount" gorm:"type:decimal(12,6);default:0"`
	Currency      string  `json:"currency" gorm:"type:varchar(16);default:''"`
	CreatedAt     int64   `json:"created_at" gorm:"bigint"`
}

type InvoiceRecord struct {
	Id           int                        `json:"id"`
	InvoiceNo    string                     `json:"invoice_no" gorm:"uniqueIndex;type:varchar(64)"`
	UserId       int                        `json:"user_id" gorm:"index"`
	Status       string                     `json:"status" gorm:"type:varchar(32);index"`
	TotalAmount  float64                    `json:"total_amount" gorm:"type:decimal(12,6);default:0"`
	Currency     string                     `json:"currency" gorm:"type:varchar(16);default:''"`
	IssuedAt     int64                      `json:"issued_at" gorm:"type:bigint;default:0"`
	IssuerId     int                        `json:"issuer_id" gorm:"index;default:0"`
	FileURL      string                     `json:"file_url" gorm:"type:text"`
	Remark       string                     `json:"remark" gorm:"type:text"`
	CreatedAt    int64                      `json:"created_at" gorm:"bigint"`
	UpdatedAt    int64                      `json:"updated_at" gorm:"bigint"`
	Applications []InvoiceRecordApplication `json:"applications,omitempty" gorm:"foreignKey:InvoiceRecordId"`
}

type InvoiceRecordApplication struct {
	Id              int   `json:"id"`
	InvoiceRecordId int   `json:"invoice_record_id" gorm:"index"`
	ApplicationId   int   `json:"application_id" gorm:"index"`
	CreatedAt       int64 `json:"created_at" gorm:"bigint"`
}

func (p *InvoiceProfile) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	p.CreatedAt = now
	p.UpdatedAt = now
	return nil
}

func (p *InvoiceProfile) BeforeUpdate(tx *gorm.DB) error {
	p.UpdatedAt = common.GetTimestamp()
	return nil
}

func (a *InvoiceApplication) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	a.CreatedAt = now
	a.UpdatedAt = now
	if a.ApplicationNo == "" {
		a.ApplicationNo = buildInvoiceApplicationNo()
	}
	if a.Status == "" {
		a.Status = common.InvoiceApplicationStatusPendingReview
	}
	return nil
}

func (a *InvoiceApplication) BeforeUpdate(tx *gorm.DB) error {
	a.UpdatedAt = common.GetTimestamp()
	return nil
}

func (i *InvoiceApplicationItem) BeforeCreate(tx *gorm.DB) error {
	if i.CreatedAt == 0 {
		i.CreatedAt = common.GetTimestamp()
	}
	return nil
}

func (r *InvoiceRecord) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	r.CreatedAt = now
	r.UpdatedAt = now
	if r.IssuedAt == 0 {
		r.IssuedAt = now
	}
	if r.InvoiceNo == "" {
		r.InvoiceNo = buildInvoiceRecordNo()
	}
	if r.Status == "" {
		r.Status = common.InvoiceRecordStatusIssued
	}
	return nil
}

func (r *InvoiceRecord) BeforeUpdate(tx *gorm.DB) error {
	r.UpdatedAt = common.GetTimestamp()
	return nil
}

func (i *InvoiceRecordApplication) BeforeCreate(tx *gorm.DB) error {
	if i.CreatedAt == 0 {
		i.CreatedAt = common.GetTimestamp()
	}
	return nil
}

func buildInvoiceApplicationNo() string {
	return fmt.Sprintf("IA%s%s", common.GetTimeString(), strings.ToUpper(common.GetRandomString(4)))
}

func buildInvoiceRecordNo() string {
	return fmt.Sprintf("IV%s%s", common.GetTimeString(), strings.ToUpper(common.GetRandomString(4)))
}

func sanitizeInvoiceProfileSnapshot(snapshot InvoiceProfileSnapshot) InvoiceProfileSnapshot {
	snapshot.Type = strings.TrimSpace(snapshot.Type)
	if snapshot.Type == "" {
		snapshot.Type = "personal"
	}
	snapshot.Title = strings.TrimSpace(snapshot.Title)
	snapshot.TaxNo = strings.TrimSpace(snapshot.TaxNo)
	snapshot.Email = strings.TrimSpace(snapshot.Email)
	snapshot.Phone = strings.TrimSpace(snapshot.Phone)
	snapshot.Address = strings.TrimSpace(snapshot.Address)
	snapshot.BankName = strings.TrimSpace(snapshot.BankName)
	snapshot.BankAccount = strings.TrimSpace(snapshot.BankAccount)
	return snapshot
}

func validateInvoiceProfileSnapshot(snapshot InvoiceProfileSnapshot) error {
	if snapshot.Title == "" {
		return errors.New("invoice title is required")
	}
	if snapshot.TaxNo == "" {
		return errors.New("tax number is required")
	}
	if snapshot.Email == "" {
		return errors.New("invoice email is required")
	}
	if _, err := mail.ParseAddress(snapshot.Email); err != nil {
		return errors.New("invoice email is invalid")
	}
	return nil
}

func (p *InvoiceProfile) ToSnapshot() InvoiceProfileSnapshot {
	return InvoiceProfileSnapshot{
		Type:        p.Type,
		Title:       p.Title,
		TaxNo:       p.TaxNo,
		Email:       p.Email,
		Phone:       p.Phone,
		Address:     p.Address,
		BankName:    p.BankName,
		BankAccount: p.BankAccount,
	}
}

func (a *InvoiceApplication) GetProfileSnapshot() InvoiceProfileSnapshot {
	snapshot := InvoiceProfileSnapshot{}
	if a.ProfileSnapshot != "" {
		if err := common.UnmarshalJsonStr(a.ProfileSnapshot, &snapshot); err != nil {
			common.SysError("failed to parse invoice profile snapshot: " + err.Error())
		}
	}
	return snapshot
}

func ListInvoiceProfilesByUser(userId int) ([]*InvoiceProfile, error) {
	var profiles []*InvoiceProfile
	err := DB.Where("user_id = ?", userId).Order("is_default desc, id desc").Find(&profiles).Error
	return profiles, err
}

func GetInvoiceProfileByUser(userId int, profileId int) (*InvoiceProfile, error) {
	var profile InvoiceProfile
	if err := DB.Where("id = ? AND user_id = ?", profileId, userId).First(&profile).Error; err != nil {
		return nil, err
	}
	return &profile, nil
}

func SaveInvoiceProfile(profile *InvoiceProfile) error {
	if profile == nil {
		return errors.New("invoice profile is nil")
	}
	profile.Type = strings.TrimSpace(profile.Type)
	if profile.Type == "" {
		profile.Type = "personal"
	}
	profile.Title = strings.TrimSpace(profile.Title)
	profile.TaxNo = strings.TrimSpace(profile.TaxNo)
	profile.Email = strings.TrimSpace(profile.Email)
	profile.Phone = strings.TrimSpace(profile.Phone)
	profile.Address = strings.TrimSpace(profile.Address)
	profile.BankName = strings.TrimSpace(profile.BankName)
	profile.BankAccount = strings.TrimSpace(profile.BankAccount)
	if err := validateInvoiceProfileSnapshot(profile.ToSnapshot()); err != nil {
		return err
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		if profile.IsDefault {
			if err := tx.Model(&InvoiceProfile{}).Where("user_id = ?", profile.UserId).Update("is_default", false).Error; err != nil {
				return err
			}
		}
		if profile.Id == 0 {
			return tx.Create(profile).Error
		}
		return tx.Model(&InvoiceProfile{}).Where("id = ? AND user_id = ?", profile.Id, profile.UserId).Updates(map[string]interface{}{
			"type":         profile.Type,
			"title":        profile.Title,
			"tax_no":       profile.TaxNo,
			"email":        profile.Email,
			"phone":        profile.Phone,
			"address":      profile.Address,
			"bank_name":    profile.BankName,
			"bank_account": profile.BankAccount,
			"is_default":   profile.IsDefault,
		}).Error
	})
}

func DeleteInvoiceProfile(userId int, profileId int) error {
	return DB.Where("id = ? AND user_id = ?", profileId, userId).Delete(&InvoiceProfile{}).Error
}

func ListInvoiceAvailableTopUps(userId int, keyword string, pageInfo *common.PageInfo) ([]*TopUp, int64, error) {
	var topups []*TopUp
	var total int64
	query := DB.Model(&TopUp{}).Where("user_id = ? AND status = ? AND invoice_status = ?", userId, common.TopUpStatusSuccess, common.InvoiceStatusNone)
	if keyword != "" {
		query = query.Where("trade_no LIKE ?", "%"+keyword+"%")
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&topups).Error; err != nil {
		return nil, 0, err
	}
	return topups, total, nil
}

func ListInvoiceApplicationsByUser(userId int, keyword string, pageInfo *common.PageInfo) ([]*InvoiceApplication, int64, error) {
	return listInvoiceApplications(DB.Model(&InvoiceApplication{}).Where("user_id = ?", userId), keyword, pageInfo)
}

func ListInvoiceApplicationsByAdmin(keyword string, status string, userId int, pageInfo *common.PageInfo) ([]*InvoiceApplication, int64, error) {
	query := DB.Model(&InvoiceApplication{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if userId > 0 {
		query = query.Where("user_id = ?", userId)
	}
	return listInvoiceApplications(query, keyword, pageInfo)
}

func listInvoiceApplications(query *gorm.DB, keyword string, pageInfo *common.PageInfo) ([]*InvoiceApplication, int64, error) {
	var applications []*InvoiceApplication
	var total int64
	if keyword != "" {
		query = query.Where("application_no LIKE ?", "%"+keyword+"%")
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Preload("Items").Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&applications).Error; err != nil {
		return nil, 0, err
	}
	return applications, total, nil
}

func ListInvoiceRecords(keyword string, userId int, pageInfo *common.PageInfo) ([]*InvoiceRecord, int64, error) {
	var records []*InvoiceRecord
	var total int64
	query := DB.Model(&InvoiceRecord{})
	if userId > 0 {
		query = query.Where("user_id = ?", userId)
	}
	if keyword != "" {
		query = query.Where("invoice_no LIKE ?", "%"+keyword+"%")
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Preload("Applications").Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&records).Error; err != nil {
		return nil, 0, err
	}
	return records, total, nil
}

func CreateInvoiceApplication(userId int, topupIDs []int, snapshot InvoiceProfileSnapshot, remark string) (*InvoiceApplication, error) {
	if len(topupIDs) == 0 {
		return nil, errors.New("no orders selected")
	}
	snapshot = sanitizeInvoiceProfileSnapshot(snapshot)
	if err := validateInvoiceProfileSnapshot(snapshot); err != nil {
		return nil, err
	}
	uniqueIDs := make([]int, 0, len(topupIDs))
	seen := make(map[int]struct{}, len(topupIDs))
	for _, id := range topupIDs {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniqueIDs = append(uniqueIDs, id)
	}
	if len(uniqueIDs) == 0 {
		return nil, errors.New("no valid orders selected")
	}

	profileBytes, err := common.Marshal(snapshot)
	if err != nil {
		return nil, err
	}

	var application *InvoiceApplication
	err = DB.Transaction(func(tx *gorm.DB) error {
		var topups []*TopUp
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id IN ? AND user_id = ? AND status = ? AND invoice_status = ?", uniqueIDs, userId, common.TopUpStatusSuccess, common.InvoiceStatusNone).Find(&topups).Error; err != nil {
			return err
		}
		if len(topups) != len(uniqueIDs) {
			return errors.New("some orders are unavailable for invoicing")
		}

		totalAmount := 0.0
		currency := ""
		items := make([]InvoiceApplicationItem, 0, len(topups))
		for _, topup := range topups {
			amount := topup.GetEffectivePaidAmount()
			if amount <= 0 {
				return fmt.Errorf("order %s has invalid paid amount", topup.TradeNo)
			}
			topupCurrency := topup.GetEffectiveCurrency()
			if currency == "" {
				currency = topupCurrency
			} else if currency != topupCurrency {
				return errors.New("selected orders must use the same currency")
			}
			totalAmount += amount
			items = append(items, InvoiceApplicationItem{
				TopUpId:      topup.Id,
				OrderTradeNo: topup.TradeNo,
				Amount:       amount,
				Currency:     topupCurrency,
			})
		}

		application = &InvoiceApplication{
			UserId:          userId,
			Status:          common.InvoiceApplicationStatusPendingReview,
			TotalAmount:     totalAmount,
			Currency:        currency,
			ProfileSnapshot: string(profileBytes),
			Remark:          strings.TrimSpace(remark),
		}
		if err := tx.Create(application).Error; err != nil {
			return err
		}

		for idx := range items {
			items[idx].ApplicationId = application.Id
		}
		if err := tx.Create(&items).Error; err != nil {
			return err
		}
		if err := tx.Model(&TopUp{}).Where("id IN ?", uniqueIDs).Update("invoice_status", common.InvoiceStatusPendingReview).Error; err != nil {
			return err
		}

		application.Items = items
		return nil
	})
	if err != nil {
		return nil, err
	}
	return application, nil
}

func CancelInvoiceApplication(userId int, applicationId int) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var application InvoiceApplication
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ? AND user_id = ?", applicationId, userId).First(&application).Error; err != nil {
			return err
		}
		if application.Status != common.InvoiceApplicationStatusPendingReview {
			return errors.New("only pending applications can be cancelled")
		}
		var items []InvoiceApplicationItem
		if err := tx.Where("application_id = ?", application.Id).Find(&items).Error; err != nil {
			return err
		}
		topupIDs := make([]int, 0, len(items))
		for _, item := range items {
			topupIDs = append(topupIDs, item.TopUpId)
		}
		if err := tx.Model(&InvoiceApplication{}).Where("id = ?", application.Id).Updates(map[string]interface{}{
			"status": common.InvoiceApplicationStatusCancelled,
		}).Error; err != nil {
			return err
		}
		if len(topupIDs) > 0 {
			if err := tx.Model(&TopUp{}).Where("id IN ?", topupIDs).Update("invoice_status", common.InvoiceStatusNone).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func ApproveInvoiceApplication(applicationId int, reviewerId int, adminRemark string) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var application InvoiceApplication
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Preload("Items").Where("id = ?", applicationId).First(&application).Error; err != nil {
			return err
		}
		if application.Status != common.InvoiceApplicationStatusPendingReview {
			return errors.New("application status is invalid")
		}

		record := &InvoiceRecord{
			UserId:      application.UserId,
			Status:      common.InvoiceRecordStatusIssued,
			TotalAmount: application.TotalAmount,
			Currency:    application.Currency,
			IssuerId:    reviewerId,
			Remark:      strings.TrimSpace(adminRemark),
		}
		if err := tx.Create(record).Error; err != nil {
			return err
		}

		if err := tx.Create(&InvoiceRecordApplication{
			InvoiceRecordId: record.Id,
			ApplicationId:   application.Id,
		}).Error; err != nil {
			return err
		}

		if err := tx.Model(&InvoiceApplication{}).Where("id = ?", applicationId).Updates(map[string]interface{}{
			"status":       common.InvoiceApplicationStatusIssued,
			"reviewed_by":  reviewerId,
			"reviewed_at":  common.GetTimestamp(),
			"admin_remark": strings.TrimSpace(adminRemark),
		}).Error; err != nil {
			return err
		}

		return tx.Model(&TopUp{}).
			Where("id IN (?)", tx.Model(&InvoiceApplicationItem{}).Select("top_up_id").Where("application_id = ?", applicationId)).
			Update("invoice_status", common.InvoiceStatusIssued).Error
	})
}

func RejectInvoiceApplication(applicationId int, reviewerId int, rejectedReason string, adminRemark string) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var application InvoiceApplication
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", applicationId).First(&application).Error; err != nil {
			return err
		}
		if application.Status != common.InvoiceApplicationStatusPendingReview {
			return errors.New("application status is invalid")
		}
		var items []InvoiceApplicationItem
		if err := tx.Where("application_id = ?", application.Id).Find(&items).Error; err != nil {
			return err
		}
		topupIDs := make([]int, 0, len(items))
		for _, item := range items {
			topupIDs = append(topupIDs, item.TopUpId)
		}
		if err := tx.Model(&InvoiceApplication{}).Where("id = ?", application.Id).Updates(map[string]interface{}{
			"status":          common.InvoiceApplicationStatusRejected,
			"reviewed_by":     reviewerId,
			"reviewed_at":     common.GetTimestamp(),
			"rejected_reason": strings.TrimSpace(rejectedReason),
			"admin_remark":    strings.TrimSpace(adminRemark),
		}).Error; err != nil {
			return err
		}
		if len(topupIDs) > 0 {
			return tx.Model(&TopUp{}).Where("id IN ?", topupIDs).Update("invoice_status", common.InvoiceStatusNone).Error
		}
		return nil
	})
}

func IssueInvoiceRecord(applicationIDs []int, issuerId int, invoiceNo string, fileURL string, remark string) (*InvoiceRecord, error) {
	if len(applicationIDs) == 0 {
		return nil, errors.New("no applications selected")
	}
	uniqueIDs := make([]int, 0, len(applicationIDs))
	seen := make(map[int]struct{}, len(applicationIDs))
	for _, id := range applicationIDs {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniqueIDs = append(uniqueIDs, id)
	}
	if len(uniqueIDs) == 0 {
		return nil, errors.New("no valid applications selected")
	}

	var record *InvoiceRecord
	err := DB.Transaction(func(tx *gorm.DB) error {
		var applications []InvoiceApplication
		issueableStatuses := []string{
			common.InvoiceApplicationStatusPendingReview,
			common.InvoiceApplicationStatusApproved,
		}
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Preload("Items").Where("id IN ? AND status IN ?", uniqueIDs, issueableStatuses).Find(&applications).Error; err != nil {
			return err
		}
		if len(applications) != len(uniqueIDs) {
			return errors.New("some applications are unavailable for issuing")
		}

		userId := 0
		currency := ""
		totalAmount := 0.0
		topupIDs := make([]int, 0)
		for _, application := range applications {
			if userId == 0 {
				userId = application.UserId
			} else if userId != application.UserId {
				return errors.New("applications must belong to the same user")
			}
			if currency == "" {
				currency = application.Currency
			} else if currency != application.Currency {
				return errors.New("applications must use the same currency")
			}
			totalAmount += application.TotalAmount
			for _, item := range application.Items {
				topupIDs = append(topupIDs, item.TopUpId)
			}
		}

		record = &InvoiceRecord{
			InvoiceNo:   strings.TrimSpace(invoiceNo),
			UserId:      userId,
			Status:      common.InvoiceRecordStatusIssued,
			TotalAmount: totalAmount,
			Currency:    currency,
			IssuerId:    issuerId,
			FileURL:     strings.TrimSpace(fileURL),
			Remark:      strings.TrimSpace(remark),
		}
		if err := tx.Create(record).Error; err != nil {
			return err
		}

		links := make([]InvoiceRecordApplication, 0, len(applications))
		for _, application := range applications {
			links = append(links, InvoiceRecordApplication{
				InvoiceRecordId: record.Id,
				ApplicationId:   application.Id,
			})
		}
		if err := tx.Create(&links).Error; err != nil {
			return err
		}
		if err := tx.Model(&InvoiceApplication{}).Where("id IN ?", uniqueIDs).Updates(map[string]interface{}{
			"status":       common.InvoiceApplicationStatusIssued,
			"reviewed_by":  issuerId,
			"reviewed_at":  common.GetTimestamp(),
			"admin_remark": strings.TrimSpace(remark),
		}).Error; err != nil {
			return err
		}
		if len(topupIDs) > 0 {
			if err := tx.Model(&TopUp{}).Where("id IN ?", topupIDs).Update("invoice_status", common.InvoiceStatusIssued).Error; err != nil {
				return err
			}
		}
		record.Applications = links
		return nil
	})
	if err != nil {
		return nil, err
	}
	return record, nil
}

func VoidInvoiceRecord(recordId int, operatorId int, remark string) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var record InvoiceRecord
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", recordId).First(&record).Error; err != nil {
			return err
		}
		if record.Status != common.InvoiceRecordStatusIssued {
			return errors.New("invoice record status is invalid")
		}
		var links []InvoiceRecordApplication
		if err := tx.Where("invoice_record_id = ?", record.Id).Find(&links).Error; err != nil {
			return err
		}
		applicationIDs := make([]int, 0, len(links))
		for _, link := range links {
			applicationIDs = append(applicationIDs, link.ApplicationId)
		}
		if err := tx.Model(&InvoiceRecord{}).Where("id = ?", record.Id).Updates(map[string]interface{}{
			"status":    common.InvoiceRecordStatusVoided,
			"remark":    strings.TrimSpace(remark),
			"issuer_id": operatorId,
		}).Error; err != nil {
			return err
		}
		if len(applicationIDs) > 0 {
			if err := tx.Model(&InvoiceApplication{}).Where("id IN ?", applicationIDs).Updates(map[string]interface{}{
				"status":          common.InvoiceApplicationStatusPendingReview,
				"reviewed_by":     0,
				"reviewed_at":     0,
				"rejected_reason": "",
			}).Error; err != nil {
				return err
			}
			return tx.Model(&TopUp{}).
				Where("id IN (?)", tx.Model(&InvoiceApplicationItem{}).Select("top_up_id").Where("application_id IN ?", applicationIDs)).
				Update("invoice_status", common.InvoiceStatusPendingReview).Error
		}
		return nil
	})
}
