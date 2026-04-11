package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

func TestIssueInvoiceRecordRequiresAdminRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/invoice/admin/records", bytes.NewBufferString(`{"application_ids":[1],"invoice_no":"INV-001"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("id", 100)
	ctx.Set("role", common.RoleCommonUser)

	IssueInvoiceRecord(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if !bytes.Contains(recorder.Body.Bytes(), []byte(`"success":false`)) {
		t.Fatalf("expected non-admin request to be rejected, got %s", recorder.Body.String())
	}
}
