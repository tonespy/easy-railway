package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tonespy/easy-railway/internal/api"
	"github.com/tonespy/easy-railway/internal/auth"
	"github.com/tonespy/easy-railway/internal/log"
	"github.com/tonespy/easy-railway/internal/prompt"
)

// testMeServer returns an httptest.Server that responds to the "me" query.
func testMeServer(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)

		resp := `{"data":{"me":{"id":"usr-1","name":"Test User","email":"test@example.com"}}}`

		if _, err := w.Write([]byte(resp)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
}

// testProjectTokenServer returns an httptest.Server that responds to the "projectToken" query.
func testProjectTokenServer(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Project-Access-Token") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			if _, err := w.Write([]byte("unauthorized")); err != nil {
				t.Errorf("write response: %v", err)
			}

			return
		}

		w.WriteHeader(http.StatusOK)

		resp := `{"data":{"projectToken":{"id":"pt-1","name":"My Token","projectId":"proj-1","environmentId":"env-1"}}}`

		if _, err := w.Write([]byte(resp)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
}

// testWorkspaceServer returns an httptest.Server that responds to both "me" and "workspace" queries.
func testWorkspaceServer(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Query string `json:"query"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		w.WriteHeader(http.StatusOK)

		if strings.Contains(req.Query, "workspace") {
			resp := `{"data":{"workspace":{"id":"ws-123","name":"My Workspace"}}}`
			if _, err := w.Write([]byte(resp)); err != nil {
				t.Errorf("write response: %v", err)
			}
		} else {
			resp := `{"data":{"me":{"id":"usr-1","name":"Test User","email":"test@example.com"}}}`
			if _, err := w.Write([]byte(resp)); err != nil {
				t.Errorf("write response: %v", err)
			}
		}
	}))
}

// testFailServer returns an httptest.Server that always returns a GraphQL error.
func testFailServer(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)

		resp := `{"data":null,"errors":[{"message":"Not authenticated"}]}`

		if _, err := w.Write([]byte(resp)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
}

type testOutput struct {
	promptOut *strings.Builder
	logOut    *strings.Builder
}

func newTestDeps(t *testing.T, input string, srv *httptest.Server, envMap map[string]string) (*authDeps, *testOutput) {
	t.Helper()

	var promptBuf strings.Builder
	var logBuf strings.Builder

	return &authDeps{
		prompter: &prompt.Prompter{
			Stdin:  strings.NewReader(input),
			Stdout: &promptBuf,
		},
		apiClient: api.NewWithEndpoint(srv.URL),
		logger:    log.New(&logBuf),
		getenv: func(key string) string {
			if envMap == nil {
				return ""
			}

			return envMap[key]
		},
	}, &testOutput{promptOut: &promptBuf, logOut: &logBuf}
}

// Login tests now include token type selection.
// Prompt sequence for account token login with --api-key and env token:
//   "1\n" → select "Account token" (index 0+1=1), then "1\n" → "No" to encrypt.

func TestLoginAccountTokenFromEnv(t *testing.T) {
	t.Parallel()

	srv := testMeServer(t)
	defer srv.Close()

	credPath := filepath.Join(t.TempDir(), "creds.json")

	// Prompt: "1" for Account token, "1" for No encrypt.
	deps, _ := newTestDeps(t, "1\n1\n", srv, map[string]string{
		"EASY_RAILWAY_API_KEY": "env-token",
	})

	err := authLogin(deps, []string{"--api-key", "--credentials-path", credPath})
	if err != nil {
		t.Fatalf("authLogin: %v", err)
	}

	store := auth.NewFileStore(credPath)

	creds, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if creds.Token != "env-token" {
		t.Fatalf("expected token %q, got %q", "env-token", creds.Token)
	}

	if creds.Type != auth.TypeAccount {
		t.Fatalf("expected type %q, got %q", auth.TypeAccount, creds.Type)
	}

	if creds.UserName != "Test User" {
		t.Fatalf("expected user name %q, got %q", "Test User", creds.UserName)
	}

	if creds.UserEmail != "test@example.com" {
		t.Fatalf("expected user email %q, got %q", "test@example.com", creds.UserEmail)
	}

	if creds.UserID != "usr-1" {
		t.Fatalf("expected user ID %q, got %q", "usr-1", creds.UserID)
	}
}

func TestLoginAccountTokenFromPrompt(t *testing.T) {
	t.Parallel()

	srv := testMeServer(t)
	defer srv.Close()

	credPath := filepath.Join(t.TempDir(), "creds.json")

	// Prompt: "1" Account, token value, "1" No encrypt.
	deps, _ := newTestDeps(t, "1\nprompted-token\n1\n", srv, nil)

	err := authLogin(deps, []string{"--api-key", "--credentials-path", credPath})
	if err != nil {
		t.Fatalf("authLogin: %v", err)
	}

	store := auth.NewFileStore(credPath)

	creds, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if creds.Token != "prompted-token" {
		t.Fatalf("expected token %q, got %q", "prompted-token", creds.Token)
	}
}

func TestLoginInvalidKey(t *testing.T) {
	t.Parallel()

	srv := testFailServer(t)
	defer srv.Close()

	// Prompt: "1" Account token.
	deps, _ := newTestDeps(t, "1\n", srv, map[string]string{
		"EASY_RAILWAY_API_KEY": "bad-token",
	})

	err := authLogin(deps, []string{"--api-key", "--credentials-path", filepath.Join(t.TempDir(), "creds.json")})
	if err == nil {
		t.Fatal("expected error for invalid API key")
	}

	if !strings.Contains(err.Error(), "invalid API key") {
		t.Fatalf("expected 'invalid API key' error, got %q", err.Error())
	}
}

func TestLoginEncrypted(t *testing.T) {
	t.Parallel()

	srv := testMeServer(t)
	defer srv.Close()

	credPath := filepath.Join(t.TempDir(), "creds.json")

	// Prompt: "1" Account token.
	deps, _ := newTestDeps(t, "1\n", srv, map[string]string{
		"EASY_RAILWAY_API_KEY": "enc-token",
	})

	err := authLogin(deps, []string{
		"--api-key",
		"--credentials-path", credPath,
		"--encrypt", "my-secret-pass",
	})
	if err != nil {
		t.Fatalf("authLogin: %v", err)
	}

	if !auth.IsEncrypted(credPath) {
		t.Fatal("expected encrypted file")
	}

	inner := auth.NewFileStore(credPath)
	store := auth.NewEncryptedStore(inner, []byte("my-secret-pass"))

	creds, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if creds.Token != "enc-token" {
		t.Fatalf("expected token %q, got %q", "enc-token", creds.Token)
	}
}

func TestLoginInteractiveSelect(t *testing.T) {
	t.Parallel()

	srv := testMeServer(t)
	defer srv.Close()

	credPath := filepath.Join(t.TempDir(), "creds.json")

	// Prompt: "1" API key method, "1" Account token, token value, "1" No encrypt.
	deps, _ := newTestDeps(t, "1\n1\ninteractive-token\n1\n", srv, nil)

	err := authLogin(deps, []string{"--credentials-path", credPath})
	if err != nil {
		t.Fatalf("authLogin: %v", err)
	}

	store := auth.NewFileStore(credPath)

	creds, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if creds.Token != "interactive-token" {
		t.Fatalf("expected token %q, got %q", "interactive-token", creds.Token)
	}
}

func TestLoginEmptyToken(t *testing.T) {
	t.Parallel()

	srv := testMeServer(t)
	defer srv.Close()

	// Prompt: "1" Account token, empty line.
	deps, _ := newTestDeps(t, "1\n\n", srv, nil)

	err := authLogin(deps, []string{"--api-key"})
	if err == nil {
		t.Fatal("expected error for empty token")
	}

	if !strings.Contains(err.Error(), "cannot be empty") {
		t.Fatalf("expected 'cannot be empty' error, got %q", err.Error())
	}
}

func TestLoginProjectToken(t *testing.T) {
	t.Parallel()

	srv := testProjectTokenServer(t)
	defer srv.Close()

	credPath := filepath.Join(t.TempDir(), "creds.json")

	// Prompt: "3" Project token, "1" No encrypt.
	deps, _ := newTestDeps(t, "3\n1\n", srv, map[string]string{
		"EASY_RAILWAY_API_KEY": "proj-token",
	})

	err := authLogin(deps, []string{"--api-key", "--credentials-path", credPath})
	if err != nil {
		t.Fatalf("authLogin: %v", err)
	}

	store := auth.NewFileStore(credPath)

	creds, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if creds.Type != auth.TypeProject {
		t.Fatalf("expected type %q, got %q", auth.TypeProject, creds.Type)
	}

	if creds.ProjectTokenID != "pt-1" {
		t.Fatalf("expected project token ID %q, got %q", "pt-1", creds.ProjectTokenID)
	}

	if creds.ProjectTokenName != "My Token" {
		t.Fatalf("expected project token name %q, got %q", "My Token", creds.ProjectTokenName)
	}

	if creds.ProjectID != "proj-1" {
		t.Fatalf("expected project ID %q, got %q", "proj-1", creds.ProjectID)
	}

	if creds.EnvironmentID != "env-1" {
		t.Fatalf("expected environment ID %q, got %q", "env-1", creds.EnvironmentID)
	}
}

func TestLoginWorkspaceToken(t *testing.T) {
	t.Parallel()

	srv := testWorkspaceServer(t)
	defer srv.Close()

	credPath := filepath.Join(t.TempDir(), "creds.json")

	// Prompt: "2" Workspace token, workspace ID, "1" No encrypt.
	deps, _ := newTestDeps(t, "2\nws-123\n1\n", srv, map[string]string{
		"EASY_RAILWAY_API_KEY": "ws-token",
	})

	err := authLogin(deps, []string{"--api-key", "--credentials-path", credPath})
	if err != nil {
		t.Fatalf("authLogin: %v", err)
	}

	store := auth.NewFileStore(credPath)

	creds, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if creds.Type != auth.TypeWorkspace {
		t.Fatalf("expected type %q, got %q", auth.TypeWorkspace, creds.Type)
	}

	if creds.UserName != "" {
		t.Fatalf("expected no user name for workspace token, got %q", creds.UserName)
	}

	if creds.WorkspaceID != "ws-123" {
		t.Fatalf("expected workspace ID %q, got %q", "ws-123", creds.WorkspaceID)
	}

	if creds.WorkspaceName != "My Workspace" {
		t.Fatalf("expected workspace name %q, got %q", "My Workspace", creds.WorkspaceName)
	}
}

func TestLogout(t *testing.T) {
	t.Parallel()

	credPath := filepath.Join(t.TempDir(), "creds.json")
	store := auth.NewFileStore(credPath)

	if err := store.Save(&auth.Credentials{Type: auth.TypeAccount, Token: "x"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	srv := testMeServer(t)
	defer srv.Close()

	deps, _ := newTestDeps(t, "", srv, map[string]string{
		"EASY_RAILWAY_CREDENTIALS_PATH": credPath,
	})

	if err := authLogout(deps); err != nil {
		t.Fatalf("authLogout: %v", err)
	}

	if _, err := os.Stat(credPath); !os.IsNotExist(err) {
		t.Fatal("expected credential file to be deleted")
	}
}

func TestLogoutNoSession(t *testing.T) {
	t.Parallel()

	srv := testMeServer(t)
	defer srv.Close()

	deps, out := newTestDeps(t, "", srv, map[string]string{
		"EASY_RAILWAY_CREDENTIALS_PATH": filepath.Join(t.TempDir(), "nonexistent.json"),
	})

	if err := authLogout(deps); err != nil {
		t.Fatalf("authLogout: %v", err)
	}

	if !strings.Contains(out.logOut.String(), "No active session found") {
		t.Fatalf("expected 'No active session found' warning, got %q", out.logOut.String())
	}
}

func TestWhoamiEnvVarAloneNotSufficient(t *testing.T) {
	t.Parallel()

	srv := testMeServer(t)
	defer srv.Close()

	deps, _ := newTestDeps(t, "", srv, map[string]string{
		"EASY_RAILWAY_API_KEY":          "env-token",
		"EASY_RAILWAY_CREDENTIALS_PATH": filepath.Join(t.TempDir(), "nonexistent.json"),
	})

	err := authWhoami(deps)
	if err == nil {
		t.Fatal("expected error when no credentials file exists, even with env var set")
	}

	if !strings.Contains(err.Error(), "not logged in") {
		t.Fatalf("expected 'not logged in' error, got %q", err.Error())
	}
}

func TestWhoamiAccountFromFile(t *testing.T) {
	t.Parallel()

	srv := testMeServer(t)
	defer srv.Close()

	credPath := filepath.Join(t.TempDir(), "creds.json")
	store := auth.NewFileStore(credPath)

	if err := store.Save(&auth.Credentials{
		Type:      auth.TypeAccount,
		Token:     "file-token",
		UserID:    "usr-1",
		UserName:  "Test User",
		UserEmail: "test@example.com",
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	deps, out := newTestDeps(t, "", srv, map[string]string{
		"EASY_RAILWAY_CREDENTIALS_PATH": credPath,
	})

	if err := authWhoami(deps); err != nil {
		t.Fatalf("authWhoami: %v", err)
	}

	output := out.logOut.String()
	if !strings.Contains(output, "Test User") {
		t.Fatalf("expected user name in output, got %q", output)
	}

	if !strings.Contains(output, "test@example.com") {
		t.Fatalf("expected email in output, got %q", output)
	}

	if !strings.Contains(output, "account") {
		t.Fatalf("expected type in output, got %q", output)
	}
}

func TestWhoamiProjectToken(t *testing.T) {
	t.Parallel()

	srv := testProjectTokenServer(t)
	defer srv.Close()

	credPath := filepath.Join(t.TempDir(), "creds.json")
	store := auth.NewFileStore(credPath)

	if err := store.Save(&auth.Credentials{
		Type:             auth.TypeProject,
		Token:            "proj-token",
		ProjectTokenID:   "pt-1",
		ProjectTokenName: "My Token",
		ProjectID:        "proj-1",
		EnvironmentID:    "env-1",
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	deps, out := newTestDeps(t, "", srv, map[string]string{
		"EASY_RAILWAY_CREDENTIALS_PATH": credPath,
	})

	if err := authWhoami(deps); err != nil {
		t.Fatalf("authWhoami: %v", err)
	}

	output := out.logOut.String()
	if !strings.Contains(output, "proj-1") {
		t.Fatalf("expected project ID in output, got %q", output)
	}

	if !strings.Contains(output, "env-1") {
		t.Fatalf("expected environment ID in output, got %q", output)
	}

	if !strings.Contains(output, "My Token") {
		t.Fatalf("expected token name in output, got %q", output)
	}
}

func TestWhoamiWorkspaceToken(t *testing.T) {
	t.Parallel()

	srv := testWorkspaceServer(t)
	defer srv.Close()

	credPath := filepath.Join(t.TempDir(), "creds.json")
	store := auth.NewFileStore(credPath)

	if err := store.Save(&auth.Credentials{
		Type:          auth.TypeWorkspace,
		Token:         "ws-token",
		WorkspaceID:   "ws-123",
		WorkspaceName: "My Workspace",
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	deps, out := newTestDeps(t, "", srv, map[string]string{
		"EASY_RAILWAY_CREDENTIALS_PATH": credPath,
	})

	if err := authWhoami(deps); err != nil {
		t.Fatalf("authWhoami: %v", err)
	}

	output := out.logOut.String()
	if !strings.Contains(output, "workspace") {
		t.Fatalf("expected type in output, got %q", output)
	}

	if !strings.Contains(output, "My Workspace") {
		t.Fatalf("expected workspace name in output, got %q", output)
	}

	if !strings.Contains(output, "ws-123") {
		t.Fatalf("expected workspace ID in output, got %q", output)
	}
}

func TestWhoamiEncrypted(t *testing.T) {
	t.Parallel()

	srv := testMeServer(t)
	defer srv.Close()

	credPath := filepath.Join(t.TempDir(), "creds.json")
	inner := auth.NewFileStore(credPath)
	encStore := auth.NewEncryptedStore(inner, []byte("test-pass"))

	if err := encStore.Save(&auth.Credentials{
		Type:      auth.TypeAccount,
		Token:     "enc-token",
		UserID:    "usr-1",
		UserName:  "Test User",
		UserEmail: "test@example.com",
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Prompt provides the password.
	deps, out := newTestDeps(t, "test-pass\n", srv, map[string]string{
		"EASY_RAILWAY_CREDENTIALS_PATH": credPath,
	})

	if err := authWhoami(deps); err != nil {
		t.Fatalf("authWhoami: %v", err)
	}

	output := out.logOut.String()
	if !strings.Contains(output, "Test User") {
		t.Fatalf("expected user name in output, got %q", output)
	}
}

func TestWhoamiNotLoggedIn(t *testing.T) {
	t.Parallel()

	srv := testMeServer(t)
	defer srv.Close()

	deps, _ := newTestDeps(t, "", srv, map[string]string{
		"EASY_RAILWAY_CREDENTIALS_PATH": filepath.Join(t.TempDir(), "nonexistent.json"),
	})

	err := authWhoami(deps)
	if err == nil {
		t.Fatal("expected error when not logged in")
	}

	if !strings.Contains(err.Error(), "not logged in") {
		t.Fatalf("expected 'not logged in' error, got %q", err.Error())
	}
}

func TestLoginEncryptViaEnvVar(t *testing.T) {
	t.Parallel()

	srv := testMeServer(t)
	defer srv.Close()

	credPath := filepath.Join(t.TempDir(), "creds.json")

	// Prompt: "1" Account token.
	deps, _ := newTestDeps(t, "1\n", srv, map[string]string{
		"EASY_RAILWAY_API_KEY":      "env-token",
		"EASY_RAILWAY_SESSION_PASS": "env-password",
	})

	err := authLogin(deps, []string{
		"--api-key",
		"--credentials-path", credPath,
	})
	if err != nil {
		t.Fatalf("authLogin: %v", err)
	}

	if !auth.IsEncrypted(credPath) {
		t.Fatal("expected encrypted file when EASY_RAILWAY_SESSION_PASS is set")
	}
}

func TestLoginEncryptViaEnvVarName(t *testing.T) {
	t.Parallel()

	srv := testMeServer(t)
	defer srv.Close()

	credPath := filepath.Join(t.TempDir(), "creds.json")

	// Prompt: "1" Account token.
	deps, _ := newTestDeps(t, "1\n", srv, map[string]string{
		"EASY_RAILWAY_API_KEY": "env-token",
		"MY_CUSTOM_PASS":      "custom-password",
	})

	err := authLogin(deps, []string{
		"--api-key",
		"--credentials-path", credPath,
		"--encrypt", "MY_CUSTOM_PASS",
	})
	if err != nil {
		t.Fatalf("authLogin: %v", err)
	}

	if !auth.IsEncrypted(credPath) {
		t.Fatal("expected encrypted file")
	}

	inner := auth.NewFileStore(credPath)
	store := auth.NewEncryptedStore(inner, []byte("custom-password"))

	creds, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if creds.Token != "env-token" {
		t.Fatalf("expected token %q, got %q", "env-token", creds.Token)
	}
}

func TestLoginPlainFileIsValidJSON(t *testing.T) {
	t.Parallel()

	srv := testMeServer(t)
	defer srv.Close()

	credPath := filepath.Join(t.TempDir(), "creds.json")

	// Prompt: "1" Account token, "1" No encrypt.
	deps, _ := newTestDeps(t, "1\n1\n", srv, map[string]string{
		"EASY_RAILWAY_API_KEY": "json-token",
	})

	err := authLogin(deps, []string{"--api-key", "--credentials-path", credPath})
	if err != nil {
		t.Fatalf("authLogin: %v", err)
	}

	data, err := os.ReadFile(credPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	if !json.Valid(data) {
		t.Fatalf("expected valid JSON, got %q", string(data))
	}
}
