package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRateLimitUsesAuthenticatedSubjectBehindSharedIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		if subject := c.GetHeader("X-Test-Subject"); subject != "" {
			c.Set("cognito_context", &CognitoContext{Sub: subject})
		}
		c.Next()
	})
	router.Use(RateLimitMiddleware(NewMemoryRateLimitStore(1, 60), "", ""))
	router.GET("/api/v1/collections/activities", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	request := func(subject, remoteAddr string) int {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/collections/activities", nil)
		req.RemoteAddr = remoteAddr
		req.Header.Set("X-Test-Subject", subject)
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		return res.Code
	}

	if got := request("agent-one", "203.0.113.10:1001"); got != http.StatusOK {
		t.Fatalf("first agent request status = %d, want 200", got)
	}
	if got := request("agent-two", "203.0.113.10:1002"); got != http.StatusOK {
		t.Fatalf("second agent sharing IP status = %d, want 200", got)
	}
	if got := request("agent-one", "198.51.100.20:1003"); got != http.StatusTooManyRequests {
		t.Fatalf("same agent from another IP status = %d, want 429", got)
	}
}

func TestRateLimitPrincipalHashesAuthenticatedSubject(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/collections/activities", nil)
	c.Set("cognito_context", &CognitoContext{Sub: "private-subject"})

	principal := rateLimitPrincipal(c, nil)
	if !strings.HasPrefix(principal, "sub:") {
		t.Fatalf("authenticated principal = %q, want sub prefix", principal)
	}
	if strings.Contains(principal, "private-subject") {
		t.Fatal("rate-limit principal must not expose the Cognito subject")
	}
}
