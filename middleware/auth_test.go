package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupAuthTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}
	model.DB = db
	model.LOG_DB = db
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("failed to migrate users table: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func TestAdminAuthRejectsStaleAdminSessionAfterRoleDemotion(t *testing.T) {
	db := setupAuthTestDB(t)
	adminUser := &model.User{
		Username:    "stale-admin",
		Password:    "password123",
		DisplayName: "stale-admin",
		Role:        common.RoleAdminUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		AffCode:     "stale-admin-aff",
	}
	if err := db.Create(adminUser).Error; err != nil {
		t.Fatalf("failed to create admin user: %v", err)
	}

	router := gin.New()
	store := cookie.NewStore([]byte("auth-test-secret"))
	router.Use(sessions.Sessions("auth-test", store))
	router.GET("/seed", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("id", adminUser.Id)
		session.Set("username", adminUser.Username)
		session.Set("role", common.RoleAdminUser)
		session.Set("status", common.UserStatusEnabled)
		session.Set("group", "default")
		if err := session.Save(); err != nil {
			t.Fatalf("failed to save session: %v", err)
		}
		c.Status(http.StatusNoContent)
	})
	router.GET("/admin", AdminAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	seedReq := httptest.NewRequest(http.MethodGet, "/seed", nil)
	seedResp := httptest.NewRecorder()
	router.ServeHTTP(seedResp, seedReq)
	cookies := seedResp.Result().Cookies()

	if err := db.Model(&model.User{}).Where("id = ?", adminUser.Id).Updates(map[string]interface{}{
		"role": common.RoleCommonUser,
	}).Error; err != nil {
		t.Fatalf("failed to demote admin user: %v", err)
	}

	adminReq := httptest.NewRequest(http.MethodGet, "/admin", nil)
	adminReq.Header.Set("New-Api-User", fmt.Sprintf("%d", adminUser.Id))
	for _, cookie := range cookies {
		adminReq.AddCookie(cookie)
	}
	adminResp := httptest.NewRecorder()
	router.ServeHTTP(adminResp, adminReq)

	if !strings.Contains(adminResp.Body.String(), "\"success\":false") {
		t.Fatalf("expected stale admin session to be rejected, got %s", adminResp.Body.String())
	}
}
