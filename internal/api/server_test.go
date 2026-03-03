package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gin "github.com/gin-gonic/gin"
	proxyconfig "github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	internallogging "github.com/router-for-me/CLIProxyAPI/v6/internal/logging"
	sdkaccess "github.com/router-for-me/CLIProxyAPI/v6/sdk/access"
	"github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/auth"
	sdkconfig "github.com/router-for-me/CLIProxyAPI/v6/sdk/config"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()

	gin.SetMode(gin.TestMode)

	tmpDir := t.TempDir()
	authDir := filepath.Join(tmpDir, "auth")
	if err := os.MkdirAll(authDir, 0o700); err != nil {
		t.Fatalf("failed to create auth dir: %v", err)
	}

	cfg := &proxyconfig.Config{
		SDKConfig: sdkconfig.SDKConfig{
			APIKeys: []string{"test-key"},
		},
		Port:                   0,
		AuthDir:                authDir,
		Debug:                  true,
		LoggingToFile:          false,
		UsageStatisticsEnabled: false,
	}

	authManager := auth.NewManager(nil, nil, nil)
	accessManager := sdkaccess.NewManager()

	configPath := filepath.Join(tmpDir, "config.yaml")
	return NewServer(cfg, authManager, accessManager, configPath)
}

func TestAmpProviderModelRoutes(t *testing.T) {
	testCases := []struct {
		name         string
		path         string
		wantStatus   int
		wantContains string
	}{
		{
			name:         "openai root models",
			path:         "/api/provider/openai/models",
			wantStatus:   http.StatusOK,
			wantContains: `"object":"list"`,
		},
		{
			name:         "groq root models",
			path:         "/api/provider/groq/models",
			wantStatus:   http.StatusOK,
			wantContains: `"object":"list"`,
		},
		{
			name:         "openai models",
			path:         "/api/provider/openai/v1/models",
			wantStatus:   http.StatusOK,
			wantContains: `"object":"list"`,
		},
		{
			name:         "anthropic models",
			path:         "/api/provider/anthropic/v1/models",
			wantStatus:   http.StatusOK,
			wantContains: `"data"`,
		},
		{
			name:         "google models v1",
			path:         "/api/provider/google/v1/models",
			wantStatus:   http.StatusOK,
			wantContains: `"models"`,
		},
		{
			name:         "google models v1beta",
			path:         "/api/provider/google/v1beta/models",
			wantStatus:   http.StatusOK,
			wantContains: `"models"`,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			server := newTestServer(t)

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.Header.Set("Authorization", "Bearer test-key")

			rr := httptest.NewRecorder()
			server.engine.ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Fatalf("unexpected status code for %s: got %d want %d; body=%s", tc.path, rr.Code, tc.wantStatus, rr.Body.String())
			}
			if body := rr.Body.String(); !strings.Contains(body, tc.wantContains) {
				t.Fatalf("response body for %s missing %q: %s", tc.path, tc.wantContains, body)
			}
		})
	}
}

func TestDefaultRequestLoggerFactory_UsesResolvedLogDirectory(t *testing.T) {
	t.Setenv("WRITABLE_PATH", "")
	t.Setenv("writable_path", "")

	originalWD, errGetwd := os.Getwd()
	if errGetwd != nil {
		t.Fatalf("failed to get current working directory: %v", errGetwd)
	}

	tmpDir := t.TempDir()
	if errChdir := os.Chdir(tmpDir); errChdir != nil {
		t.Fatalf("failed to switch working directory: %v", errChdir)
	}
	defer func() {
		if errChdirBack := os.Chdir(originalWD); errChdirBack != nil {
			t.Fatalf("failed to restore working directory: %v", errChdirBack)
		}
	}()

	// Force ResolveLogDirectory to fallback to auth-dir/logs by making ./logs not a writable directory.
	if errWriteFile := os.WriteFile(filepath.Join(tmpDir, "logs"), []byte("not-a-directory"), 0o644); errWriteFile != nil {
		t.Fatalf("failed to create blocking logs file: %v", errWriteFile)
	}

	configDir := filepath.Join(tmpDir, "config")
	if errMkdirConfig := os.MkdirAll(configDir, 0o755); errMkdirConfig != nil {
		t.Fatalf("failed to create config dir: %v", errMkdirConfig)
	}
	configPath := filepath.Join(configDir, "config.yaml")

	authDir := filepath.Join(tmpDir, "auth")
	if errMkdirAuth := os.MkdirAll(authDir, 0o700); errMkdirAuth != nil {
		t.Fatalf("failed to create auth dir: %v", errMkdirAuth)
	}

	cfg := &proxyconfig.Config{
		SDKConfig: proxyconfig.SDKConfig{
			RequestLog: false,
		},
		AuthDir:           authDir,
		ErrorLogsMaxFiles: 10,
	}

	logger := defaultRequestLoggerFactory(cfg, configPath)
	fileLogger, ok := logger.(*internallogging.FileRequestLogger)
	if !ok {
		t.Fatalf("expected *FileRequestLogger, got %T", logger)
	}

	errLog := fileLogger.LogRequestWithOptions(
		"/v1/chat/completions",
		http.MethodPost,
		map[string][]string{"Content-Type": []string{"application/json"}},
		[]byte(`{"input":"hello"}`),
		http.StatusBadGateway,
		map[string][]string{"Content-Type": []string{"application/json"}},
		[]byte(`{"error":"upstream failure"}`),
		nil,
		nil,
		nil,
		true,
		"issue-1711",
		time.Now(),
		time.Now(),
	)
	if errLog != nil {
		t.Fatalf("failed to write forced error request log: %v", errLog)
	}

	authLogsDir := filepath.Join(authDir, "logs")
	authEntries, errReadAuthDir := os.ReadDir(authLogsDir)
	if errReadAuthDir != nil {
		t.Fatalf("failed to read auth logs dir %s: %v", authLogsDir, errReadAuthDir)
	}
	foundErrorLogInAuthDir := false
	for _, entry := range authEntries {
		if strings.HasPrefix(entry.Name(), "error-") && strings.HasSuffix(entry.Name(), ".log") {
			foundErrorLogInAuthDir = true
			break
		}
	}
	if !foundErrorLogInAuthDir {
		t.Fatalf("expected forced error log in auth fallback dir %s, got entries: %+v", authLogsDir, authEntries)
	}

	configLogsDir := filepath.Join(configDir, "logs")
	configEntries, errReadConfigDir := os.ReadDir(configLogsDir)
	if errReadConfigDir != nil && !os.IsNotExist(errReadConfigDir) {
		t.Fatalf("failed to inspect config logs dir %s: %v", configLogsDir, errReadConfigDir)
	}
	for _, entry := range configEntries {
		if strings.HasPrefix(entry.Name(), "error-") && strings.HasSuffix(entry.Name(), ".log") {
			t.Fatalf("unexpected forced error log in config dir %s", configLogsDir)
		}
	}
}

func runClientAuthMappingMiddleware(server *Server, apiKey string) (*httptest.ResponseRecorder, *gin.Context) {
	rr := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rr)
	ctx.Set("apiKey", apiKey)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	server.clientAuthMappingMiddleware()(ctx)
	return rr, ctx
}

func TestClientAuthMappingMiddleware_RejectsUnmappedKey(t *testing.T) {
	server := newTestServer(t)
	server.cfg.ClientAuthMappings = []proxyconfig.ClientAuthMappingEntry{{
		AuthIndex: "idx-a",
		APIKeys:   []string{"another-key"},
	}}

	rr, _ := runClientAuthMappingMiddleware(server, "test-key")

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status: got %d want %d; body=%s", rr.Code, http.StatusUnauthorized, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "not mapped to a dedicated auth") {
		t.Fatalf("expected unmapped error, got %s", rr.Body.String())
	}
}

func TestClientAuthMappingMiddleware_RejectsMissingMappedAuth(t *testing.T) {
	server := newTestServer(t)
	server.cfg.ClientAuthMappings = []proxyconfig.ClientAuthMappingEntry{{
		AuthIndex: "idx-missing",
		APIKeys:   []string{"test-key"},
	}}

	rr, _ := runClientAuthMappingMiddleware(server, "test-key")

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status: got %d want %d; body=%s", rr.Code, http.StatusUnauthorized, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "mapped auth-index not found") {
		t.Fatalf("expected mapped auth-index not found error, got %s", rr.Body.String())
	}
}

func TestClientAuthMappingMiddleware_PinsMappedAuth(t *testing.T) {
	server := newTestServer(t)
	registered, err := server.authManager.Register(context.Background(), &auth.Auth{
		ID:       "auth-1",
		FileName: "auth-a.json",
		Provider: "claude",
	})
	if err != nil {
		t.Fatalf("register auth: %v", err)
	}
	idx := registered.EnsureIndex()
	server.cfg.ClientAuthMappings = []proxyconfig.ClientAuthMappingEntry{{
		AuthIndex: idx,
		APIKeys:   []string{"test-key"},
	}}

	rr := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rr)
	ctx.Set("apiKey", "test-key")
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)

	originalReqCtx := ctx.Request.Context()
	server.clientAuthMappingMiddleware()(ctx)

	if rr.Code != http.StatusOK {
		// no response written on success; httptest recorder remains 200
		t.Fatalf("unexpected status on success path: got %d body=%s", rr.Code, rr.Body.String())
	}
	if got, ok := ctx.Get("pinnedAuthID"); !ok || got != "auth-1" {
		t.Fatalf("expected pinned auth ID auth-1, got %#v (ok=%v)", got, ok)
	}
	if ctx.Request == nil {
		t.Fatalf("expected request to remain available")
	}
	if ctx.Request.Context() == originalReqCtx {
		t.Fatalf("expected middleware to attach a derived request context")
	}
}

func TestClientAuthMappingMiddleware_RoundRobinAcrossMappedTargets(t *testing.T) {
	server := newTestServer(t)
	registeredA, err := server.authManager.Register(context.Background(), &auth.Auth{
		ID:       "auth-1",
		FileName: "auth-a.json",
		Provider: "claude",
	})
	if err != nil {
		t.Fatalf("register auth-a: %v", err)
	}
	registeredB, err := server.authManager.Register(context.Background(), &auth.Auth{
		ID:       "auth-2",
		FileName: "auth-b.json",
		Provider: "claude",
	})
	if err != nil {
		t.Fatalf("register auth-b: %v", err)
	}
	server.cfg.ClientAuthMappings = []proxyconfig.ClientAuthMappingEntry{
		{AuthIndex: registeredA.EnsureIndex(), APIKeys: []string{"test-key"}},
		{AuthIndex: registeredB.EnsureIndex(), APIKeys: []string{"test-key"}},
	}

	expected := []string{"auth-1", "auth-2", "auth-1", "auth-2"}
	for i := range expected {
		rr, ctx := runClientAuthMappingMiddleware(server, "test-key")
		if rr.Code != http.StatusOK {
			t.Fatalf("request %d unexpected status: got %d body=%s", i, rr.Code, rr.Body.String())
		}
		got, ok := ctx.Get("pinnedAuthID")
		if !ok {
			t.Fatalf("request %d expected pinnedAuthID", i)
		}
		if got != expected[i] {
			t.Fatalf("request %d expected pinnedAuthID %q, got %#v", i, expected[i], got)
		}
	}
}

func TestClientAuthMappingMiddleware_SkipsUnavailableTarget(t *testing.T) {
	t.Run("disabled first target", func(t *testing.T) {
		server := newTestServer(t)
		disabledAuth, err := server.authManager.Register(context.Background(), &auth.Auth{
			ID:       "auth-disabled",
			FileName: "auth-disabled.json",
			Provider: "claude",
			Disabled: true,
		})
		if err != nil {
			t.Fatalf("register disabled auth: %v", err)
		}
		activeAuth, err := server.authManager.Register(context.Background(), &auth.Auth{
			ID:       "auth-active",
			FileName: "auth-active.json",
			Provider: "claude",
		})
		if err != nil {
			t.Fatalf("register active auth: %v", err)
		}
		server.cfg.ClientAuthMappings = []proxyconfig.ClientAuthMappingEntry{
			{AuthIndex: disabledAuth.EnsureIndex(), APIKeys: []string{"test-key"}},
			{AuthIndex: activeAuth.EnsureIndex(), APIKeys: []string{"test-key"}},
		}

		rr, ctx := runClientAuthMappingMiddleware(server, "test-key")
		if rr.Code != http.StatusOK {
			t.Fatalf("unexpected status: got %d body=%s", rr.Code, rr.Body.String())
		}
		if got, ok := ctx.Get("pinnedAuthID"); !ok || got != "auth-active" {
			t.Fatalf("expected fallback to active auth, got %#v (ok=%v)", got, ok)
		}
	})

	t.Run("missing first target", func(t *testing.T) {
		server := newTestServer(t)
		activeAuth, err := server.authManager.Register(context.Background(), &auth.Auth{
			ID:       "auth-active",
			FileName: "auth-active.json",
			Provider: "claude",
		})
		if err != nil {
			t.Fatalf("register active auth: %v", err)
		}
		server.cfg.ClientAuthMappings = []proxyconfig.ClientAuthMappingEntry{
			{AuthIndex: "idx-missing", APIKeys: []string{"test-key"}},
			{AuthIndex: activeAuth.EnsureIndex(), APIKeys: []string{"test-key"}},
		}

		rr, ctx := runClientAuthMappingMiddleware(server, "test-key")
		if rr.Code != http.StatusOK {
			t.Fatalf("unexpected status: got %d body=%s", rr.Code, rr.Body.String())
		}
		if got, ok := ctx.Get("pinnedAuthID"); !ok || got != "auth-active" {
			t.Fatalf("expected fallback to active auth, got %#v (ok=%v)", got, ok)
		}
	})
}

func TestClientAuthMappingMiddleware_RejectsWhenAllMappedTargetsUnavailable(t *testing.T) {
	t.Run("all missing returns not found", func(t *testing.T) {
		server := newTestServer(t)
		server.cfg.ClientAuthMappings = []proxyconfig.ClientAuthMappingEntry{
			{AuthIndex: "idx-missing-a", APIKeys: []string{"test-key"}},
			{AuthIndex: "idx-missing-b", APIKeys: []string{"test-key"}},
		}

		rr, _ := runClientAuthMappingMiddleware(server, "test-key")
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("unexpected status: got %d want %d; body=%s", rr.Code, http.StatusUnauthorized, rr.Body.String())
		}
		if !strings.Contains(rr.Body.String(), "mapped auth-index not found") {
			t.Fatalf("expected not found error, got %s", rr.Body.String())
		}
	})

	t.Run("disabled has precedence over not found", func(t *testing.T) {
		server := newTestServer(t)
		disabledAuth, err := server.authManager.Register(context.Background(), &auth.Auth{
			ID:       "auth-disabled",
			FileName: "auth-disabled.json",
			Provider: "claude",
			Disabled: true,
		})
		if err != nil {
			t.Fatalf("register disabled auth: %v", err)
		}
		server.cfg.ClientAuthMappings = []proxyconfig.ClientAuthMappingEntry{
			{AuthIndex: "idx-missing", APIKeys: []string{"test-key"}},
			{AuthIndex: disabledAuth.EnsureIndex(), APIKeys: []string{"test-key"}},
		}

		rr, _ := runClientAuthMappingMiddleware(server, "test-key")
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("unexpected status: got %d want %d; body=%s", rr.Code, http.StatusUnauthorized, rr.Body.String())
		}
		if !strings.Contains(rr.Body.String(), "mapped auth-index is disabled") {
			t.Fatalf("expected disabled error precedence, got %s", rr.Body.String())
		}
	})
}

func TestClientAuthMappingMiddleware_RejectsDisabledAuth(t *testing.T) {
	server := newTestServer(t)
	registered, err := server.authManager.Register(context.Background(), &auth.Auth{
		ID:       "auth-disabled",
		FileName: "auth-disabled.json",
		Provider: "claude",
		Disabled: true,
	})
	if err != nil {
		t.Fatalf("register auth: %v", err)
	}
	server.cfg.ClientAuthMappings = []proxyconfig.ClientAuthMappingEntry{{
		AuthIndex: registered.EnsureIndex(),
		APIKeys:   []string{"test-key"},
	}}

	rr, _ := runClientAuthMappingMiddleware(server, "test-key")

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status: got %d want %d; body=%s", rr.Code, http.StatusUnauthorized, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "disabled") {
		t.Fatalf("expected disabled error, got %s", rr.Body.String())
	}
}

