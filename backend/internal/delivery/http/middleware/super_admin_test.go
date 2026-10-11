package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func setupSuperAdminRouter(role string, resolve RoleResolver) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/super", func(c *gin.Context) {
		c.Set("user_id", uuid.New())
		c.Set("role", role)
		c.Next()
	}, RequireSuperAdmin(resolve), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

func doGet(r *gin.Engine) int {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/super", nil)
	r.ServeHTTP(w, req)
	return w.Code
}

// A plain `user` must be rejected with 403 even before the role source is hit.
func TestRequireSuperAdmin_RejectsUserRole(t *testing.T) {
	called := false
	resolve := RoleResolverFunc(func(context.Context, uuid.UUID) (string, error) {
		called = true
		return "super_admin", nil
	})
	if code := doGet(setupSuperAdminRouter("user", resolve)); code != http.StatusForbidden {
		t.Fatalf("user role: got %d, want 403", code)
	}
	if called {
		t.Fatalf("role source must not be consulted for a non-super_admin claim")
	}
}

// A JWT claiming super_admin is allowed when the live role agrees.
func TestRequireSuperAdmin_AllowsSuperAdmin(t *testing.T) {
	resolve := RoleResolverFunc(func(context.Context, uuid.UUID) (string, error) {
		return "super_admin", nil
	})
	if code := doGet(setupSuperAdminRouter("super_admin", resolve)); code != http.StatusOK {
		t.Fatalf("super_admin: got %d, want 200", code)
	}
}

// A stale JWT (super_admin) must be rejected once the live role is demoted.
func TestRequireSuperAdmin_RejectsStaleJWT(t *testing.T) {
	resolve := RoleResolverFunc(func(context.Context, uuid.UUID) (string, error) {
		return "user", nil
	})
	if code := doGet(setupSuperAdminRouter("super_admin", resolve)); code != http.StatusForbidden {
		t.Fatalf("stale jwt: got %d, want 403", code)
	}
}

// A resolver failure fails closed.
func TestRequireSuperAdmin_ResolverErrorFailsClosed(t *testing.T) {
	resolve := RoleResolverFunc(func(context.Context, uuid.UUID) (string, error) {
		return "", errors.New("cache down")
	})
	if code := doGet(setupSuperAdminRouter("super_admin", resolve)); code != http.StatusForbidden {
		t.Fatalf("resolver error: got %d, want 403", code)
	}
}

// RequireAdmin accepts both admin and super_admin.
func TestRequireAdmin_AcceptsSuperAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, role := range []string{"admin", SuperAdminRole} {
		r := gin.New()
		r.GET("/super", func(c *gin.Context) { c.Set("role", role); c.Next() },
			RequireAdmin(), func(c *gin.Context) { c.Status(http.StatusOK) })
		if code := doGet(r); code != http.StatusOK {
			t.Fatalf("role %q: got %d, want 200", role, code)
		}
	}
}
