// 用户头像接口测试(计划 Task 4 Go 部分)
// 契约: 合法 data URI 原样保存;超过 500 字符仍保存(去除 binding max=500);
//
//	空字符串清除;非法协议返回 HTTP 400 + handler.user.avatar_validate,
//	响应不得泄露 Base64 payload。
//
// 真实 sqlite 内存库 + 真实 handler,不引入 mock
package user

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alchemy-furnace/server/model"
	"github.com/alchemy-furnace/server/server/http/router"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupUserProfileDB SQLite 内存库(共享缓存,允许多连接)并迁移 UserProfile
func setupUserProfileDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&_loc=Local&_fk=1"),
		&gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("打开 SQLite 失败: %v", err)
	}
	if err := db.AutoMigrate(&model.UserProfile{}); err != nil {
		t.Fatalf("迁移 UserProfile 失败: %v", err)
	}
	return db
}

// setupUserProfileRouter 挂载真实 PUT/GET /api/v1/user/profile
func setupUserProfileRouter(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := New(db)
	r.PUT("/api/v1/user/profile", router.Wrapper(h.Update))
	r.GET("/api/v1/user/profile", router.Wrapper(h.Get))
	return r
}

// putUserProfile 发送 PUT JSON 请求并解析响应包络
func putUserProfile(t *testing.T, r *gin.Engine, body string) (int, map[string]interface{}) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/user/profile", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var envelope map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("解析响应包络失败: %v, body: %s", err, w.Body.String())
	}
	return w.Code, envelope
}

// readAvatar 直读 DB 中整库唯一的用户档案头像
func readAvatar(t *testing.T, db *gorm.DB) string {
	t.Helper()
	var profile model.UserProfile
	if err := db.First(&profile, 1).Error; err != nil {
		t.Fatalf("读取档案失败: %v", err)
	}
	return profile.Avatar
}

func TestUserProfileAvatarValidDataURIPersistedVerbatim(t *testing.T) {
	db := setupUserProfileDB(t)
	r := setupUserProfileRouter(db)
	avatar := "data:image/png;base64,AAAA"

	status, body := putUserProfile(t, r, `{"display_name":"炉主","avatar":"data:image/png;base64,AAAA"}`)
	if status != http.StatusOK {
		t.Fatalf("valid avatar status = %d, body = %v", status, body)
	}
	if got := readAvatar(t, db); got != avatar {
		t.Fatalf("DB avatar = %q, want %q(原样保存)", got, avatar)
	}
}

func TestUserProfileAvatarOver500CharsStillSaved(t *testing.T) {
	db := setupUserProfileDB(t)
	r := setupUserProfileRouter(db)
	avatar := "data:image/png;base64," + strings.Repeat("A", 600)
	if len(avatar) < 500 {
		t.Fatal("测试用例构造错误: 头像未超过 500 字符")
	}

	status, body := putUserProfile(t, r, fmt.Sprintf(`{"avatar":%q}`, avatar))
	if status != http.StatusOK {
		t.Fatalf(">500 字头像期望保存, 实际 %d, body: %v", status, body)
	}
	if got := readAvatar(t, db); got != avatar {
		t.Fatalf("DB avatar = %q, want %q", got, avatar)
	}
}

func TestUserProfileAvatarEmptyStringClears(t *testing.T) {
	db := setupUserProfileDB(t)
	r := setupUserProfileRouter(db)
	if status, body := putUserProfile(t, r, `{"avatar":"data:image/png;base64,AAAA"}`); status != http.StatusOK {
		t.Fatalf("预置头像失败: %d %v", status, body)
	}

	status, body := putUserProfile(t, r, `{"avatar":""}`)
	if status != http.StatusOK {
		t.Fatalf("清空头像失败: %d %v", status, body)
	}
	if got := readAvatar(t, db); got != "" {
		t.Fatalf("DB avatar = %q, want empty", got)
	}
}

func TestUserProfileAvatarInvalidProtocolRejected(t *testing.T) {
	db := setupUserProfileDB(t)
	r := setupUserProfileRouter(db)

	status, body := putUserProfile(t, r, `{"avatar":"javascript:alert(1)"}`)
	if status != http.StatusBadRequest || body["error_code"] != "handler.user.avatar_validate" {
		t.Fatalf("invalid avatar response = %d %v", status, body)
	}
	if got := readAvatar(t, db); got != "" {
		t.Fatalf("非法头像不应落库, DB avatar = %q", got)
	}
}

func TestUserProfileAvatarInvalidPayloadNotLeakedInResponse(t *testing.T) {
	db := setupUserProfileDB(t)
	r := setupUserProfileRouter(db)
	// payload 尾部含非法字符 @,校验必然失败;前置 LEAKCHECK 标记用于检测响应泄露
	payload := strings.Repeat("A", 64) + "LEAKCHECK@@@"

	status, body := putUserProfile(t, r, fmt.Sprintf(`{"avatar":"data:image/png;base64,%s"}`, payload))
	if status != http.StatusBadRequest || body["error_code"] != "handler.user.avatar_validate" {
		t.Fatalf("invalid avatar response = %d %v", status, body)
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("序列化响应失败: %v", err)
	}
	if strings.Contains(string(raw), "LEAKCHECK") {
		t.Fatalf("响应泄露了头像 payload: %s", raw)
	}
}
