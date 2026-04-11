package controller

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thanhpk/randstr"
)

const (
	PaymentMethodCreem   = "creem"
	CreemSignatureHeader = "creem-signature"
)

var creemAdaptor = &CreemAdaptor{}

// 鐢熸垚HMAC-SHA256绛惧悕
func generateCreemSignature(payload string, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(payload))
	return hex.EncodeToString(h.Sum(nil))
}

// 楠岃瘉Creem webhook绛惧悕
func verifyCreemSignature(payload string, signature string, secret string) bool {
	if secret == "" {
		log.Printf("Creem Webhook missing signature")
		if setting.CreemTestMode {
			log.Printf("Skip Creem webhook sign verify in test mode")
			return true
		}
		return false
	}

	expectedSignature := generateCreemSignature(payload, secret)
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

type CreemPayRequest struct {
	ProductId     string `json:"product_id"`
	PaymentMethod string `json:"payment_method"`
}

type CreemProduct struct {
	ProductId string  `json:"productId"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	Currency  string  `json:"currency"`
	Quota     int64   `json:"quota"`
}

type CreemAdaptor struct {
}

func (*CreemAdaptor) RequestPay(c *gin.Context, req *CreemPayRequest) {
	if req.PaymentMethod != PaymentMethodCreem {
		c.JSON(200, gin.H{"message": "error", "data": "unsupported payment method"})
		return
	}

	if req.ProductId == "" {
		c.JSON(200, gin.H{"message": "error", "data": "please select a product"})
		return
	}

	var products []CreemProduct
	err := json.Unmarshal([]byte(setting.CreemProducts), &products)
	if err != nil {
		log.Println("failed to parse Creem products", err)
		c.JSON(200, gin.H{"message": "error", "data": "product config error"})
		return
	}

	var selectedProduct *CreemProduct
	for _, product := range products {
		if product.ProductId == req.ProductId {
			selectedProduct = &product
			break
		}
	}

	if selectedProduct == nil {
		c.JSON(200, gin.H{"message": "error", "data": "product not found"})
		return
	}

	id := c.GetInt("id")
	user, _ := model.GetUserById(id, false)
	reference := fmt.Sprintf("creem-api-ref-%d-%d-%s", user.Id, time.Now().UnixMilli(), randstr.String(4))
	referenceId := "ref_" + common.Sha1([]byte(reference))

	topUp := &model.TopUp{
		UserId:        id,
		Amount:        selectedProduct.Quota,
		Money:         selectedProduct.Price,
		PaidAmount:    selectedProduct.Price,
		PaidCurrency:  "USD",
		TradeNo:       referenceId,
		PaymentMethod: PaymentMethodCreem,
		SourceType:    common.TopUpSourceWalletTopUp,
		ClientIP:      c.ClientIP(),
		UserAgent:     c.Request.UserAgent(),
		InvoiceStatus: common.InvoiceStatusNone,
		CreateTime:    time.Now().Unix(),
		Status:        common.TopUpStatusPending,
	}
	if err = topUp.Insert(); err != nil {
		log.Printf("failed to create Creem topup: %v", err)
		c.JSON(200, gin.H{"message": "error", "data": "create order failed"})
		return
	}

	checkoutURL, err := genCreemLink(referenceId, selectedProduct, user.Email, user.Username)
	if err != nil {
		log.Printf("failed to create Creem checkout url: %v", err)
		c.JSON(200, gin.H{"message": "error", "data": "create checkout url failed"})
		return
	}

	log.Printf("Creem order created, user=%d trade_no=%s product=%s quota=%d price=%.2f", id, referenceId, selectedProduct.Name, selectedProduct.Quota, selectedProduct.Price)
	c.JSON(200, gin.H{
		"message": "success",
		"data": gin.H{
			"checkout_url": checkoutURL,
			"order_id":     referenceId,
		},
	})
}

func RequestCreemPay(c *gin.Context) {
	var req CreemPayRequest

	// 璇诲彇body鍐呭鐢ㄤ簬鎵撳嵃锛屽悓鏃朵繚鐣欏師濮嬫暟鎹緵鍚庣画浣跨敤
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("read creem pay req body err: %v", err)
		c.JSON(200, gin.H{"message": "error", "data": "read query error"})
		return
	}

	// 鎵撳嵃body鍐呭
	log.Printf("creem pay request body: %s", string(bodyBytes))

	// 閲嶆柊璁剧疆body渚涘悗缁殑ShouldBindJSON浣跨敤
	c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	err = c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	creemAdaptor.RequestPay(c, &req)
}

// 鏂扮殑Creem Webhook缁撴瀯浣擄紝鍖归厤瀹為檯鐨剋ebhook鏁版嵁鏍煎紡
type CreemWebhookEvent struct {
	Id        string `json:"id"`
	EventType string `json:"eventType"`
	CreatedAt int64  `json:"created_at"`
	Object    struct {
		Id        string `json:"id"`
		Object    string `json:"object"`
		RequestId string `json:"request_id"`
		Order     struct {
			Object      string `json:"object"`
			Id          string `json:"id"`
			Customer    string `json:"customer"`
			Product     string `json:"product"`
			Amount      int    `json:"amount"`
			Currency    string `json:"currency"`
			SubTotal    int    `json:"sub_total"`
			TaxAmount   int    `json:"tax_amount"`
			AmountDue   int    `json:"amount_due"`
			AmountPaid  int    `json:"amount_paid"`
			Status      string `json:"status"`
			Type        string `json:"type"`
			Transaction string `json:"transaction"`
			CreatedAt   string `json:"created_at"`
			UpdatedAt   string `json:"updated_at"`
			Mode        string `json:"mode"`
		} `json:"order"`
		Product struct {
			Id                string  `json:"id"`
			Object            string  `json:"object"`
			Name              string  `json:"name"`
			Description       string  `json:"description"`
			Price             int     `json:"price"`
			Currency          string  `json:"currency"`
			BillingType       string  `json:"billing_type"`
			BillingPeriod     string  `json:"billing_period"`
			Status            string  `json:"status"`
			TaxMode           string  `json:"tax_mode"`
			TaxCategory       string  `json:"tax_category"`
			DefaultSuccessUrl *string `json:"default_success_url"`
			CreatedAt         string  `json:"created_at"`
			UpdatedAt         string  `json:"updated_at"`
			Mode              string  `json:"mode"`
		} `json:"product"`
		Units    int `json:"units"`
		Customer struct {
			Id        string `json:"id"`
			Object    string `json:"object"`
			Email     string `json:"email"`
			Name      string `json:"name"`
			Country   string `json:"country"`
			CreatedAt string `json:"created_at"`
			UpdatedAt string `json:"updated_at"`
			Mode      string `json:"mode"`
		} `json:"customer"`
		Status   string            `json:"status"`
		Metadata map[string]string `json:"metadata"`
		Mode     string            `json:"mode"`
	} `json:"object"`
}

func CreemWebhook(c *gin.Context) {
	// 璇诲彇body鍐呭鐢ㄤ簬鎵撳嵃锛屽悓鏃朵繚鐣欏師濮嬫暟鎹緵鍚庣画浣跨敤
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("璇诲彇Creem Webhook璇锋眰body澶辫触: %v", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	signature := c.GetHeader(CreemSignatureHeader)

	log.Printf("Creem Webhook - URI: %s", c.Request.RequestURI)
	if setting.CreemTestMode {
		log.Printf("Creem Webhook - Signature: %s , Body: %s", signature, bodyBytes)
	} else if signature == "" {
		log.Printf("Creem Webhook missing signature")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	// 楠岃瘉绛惧悕
	if !verifyCreemSignature(string(bodyBytes), signature, setting.CreemWebhookSecret) {
		log.Printf("Creem Webhook missing signature")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	log.Printf("Creem Webhook missing signature")

	// 閲嶆柊璁剧疆body渚涘悗缁殑ShouldBindJSON浣跨敤
	c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	// 瑙ｆ瀽鏂版牸寮忕殑webhook鏁版嵁
	var webhookEvent CreemWebhookEvent
	if err := c.ShouldBindJSON(&webhookEvent); err != nil {
		log.Printf("parse Creem webhook payload err: %v", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	log.Printf("Creem Webhook missing signature")

	// 鏍规嵁浜嬩欢绫诲瀷澶勭悊涓嶅悓鐨剋ebhook
	switch webhookEvent.EventType {
	case "checkout.completed":
		handleCheckoutCompleted(c, &webhookEvent)
	default:
		log.Printf("ignore unsupported Creem webhook event: %s", webhookEvent.EventType)
		c.Status(http.StatusOK)
	}
}

// 澶勭悊鏀粯瀹屾垚浜嬩欢
func handleCheckoutCompleted(c *gin.Context, event *CreemWebhookEvent) {
	if event.Object.Order.Status != "paid" {
		log.Printf("skip Creem order with status %s", event.Object.Order.Status)
		c.Status(http.StatusOK)
		return
	}

	referenceId := event.Object.RequestId
	if referenceId == "" {
		log.Println("Creem webhook missing request_id")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	LockOrder(referenceId)
	defer UnlockOrder(referenceId)
	if err := model.CompleteSubscriptionOrder(referenceId, common.GetJsonString(event)); err == nil {
		c.Status(http.StatusOK)
		return
	} else if err != nil && !errors.Is(err, model.ErrSubscriptionOrderNotFound) {
		log.Printf("failed to complete subscription order %s: %v", referenceId, err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if event.Object.Order.Type != "onetime" {
		log.Printf("skip unsupported Creem order type %s", event.Object.Order.Type)
		c.Status(http.StatusOK)
		return
	}

	topUp := model.GetTopUpByTradeNo(referenceId)
	if topUp == nil {
		log.Printf("Creem topup not found: %s", referenceId)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if topUp.Status != common.TopUpStatusPending {
		log.Printf("Creem topup %s already handled with status %s", referenceId, topUp.Status)
		c.Status(http.StatusOK)
		return
	}

	customerEmail := event.Object.Customer.Email
	customerName := event.Object.Customer.Name
	if err := model.RechargeCreem(referenceId, customerEmail, customerName); err != nil {
		log.Printf("failed to recharge Creem topup %s: %v", referenceId, err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	log.Printf("Creem recharge completed, trade_no=%s amount=%d money=%.2f", referenceId, topUp.Amount, topUp.Money)
	c.Status(http.StatusOK)
}

type CreemCheckoutRequest struct {
	ProductId string `json:"product_id"`
	RequestId string `json:"request_id"`
	Customer  struct {
		Email string `json:"email"`
	} `json:"customer"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type CreemCheckoutResponse struct {
	CheckoutUrl string `json:"checkout_url"`
	Id          string `json:"id"`
}

func genCreemLink(referenceId string, product *CreemProduct, email string, username string) (string, error) {
	if setting.CreemApiKey == "" {
		return "", fmt.Errorf("Creem API key is not configured")
	}

	apiURL := "https://api.creem.io/v1/checkouts"
	if setting.CreemTestMode {
		apiURL = "https://test-api.creem.io/v1/checkouts"
		log.Printf("using Creem test endpoint: %s", apiURL)
	}

	requestData := CreemCheckoutRequest{
		ProductId: product.ProductId,
		RequestId: referenceId,
		Customer: struct {
			Email string `json:"email"`
		}{
			Email: email,
		},
		Metadata: map[string]string{
			"username":     username,
			"reference_id": referenceId,
			"product_name": product.Name,
			"quota":        fmt.Sprintf("%d", product.Quota),
		},
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal Creem request: %v", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create Creem request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", setting.CreemApiKey)

	log.Printf("send Creem checkout request, url=%s product=%s email=%s trade_no=%s", apiURL, product.ProductId, email, referenceId)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send Creem request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read Creem response: %v", err)
	}

	log.Printf("Creem API resp - status code: %d, resp: %s", resp.StatusCode, string(body))
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("Creem API http status %d", resp.StatusCode)
	}

	var checkoutResp CreemCheckoutResponse
	if err := json.Unmarshal(body, &checkoutResp); err != nil {
		return "", fmt.Errorf("failed to parse Creem response: %v", err)
	}
	if checkoutResp.CheckoutUrl == "" {
		return "", fmt.Errorf("Creem response missing checkout url")
	}

	log.Printf("Creem checkout url created for %s", referenceId)
	return checkoutResp.CheckoutUrl, nil
}
