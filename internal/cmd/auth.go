// Package cmd contains top-level command handlers for easy-railway.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tonespy/easy-railway/internal/api"
	"github.com/tonespy/easy-railway/internal/auth"
	"github.com/tonespy/easy-railway/internal/log"
	"github.com/tonespy/easy-railway/internal/prompt"
)

// authDeps holds injectable dependencies for auth commands.
type authDeps struct {
	prompter    *prompt.Prompter
	apiClient   *api.Client
	logger      *log.Logger
	getenv      func(string) string
	oauthConfig *auth.OAuthConfig
}

func defaultAuthDeps() *authDeps {
	return &authDeps{
		prompter:    prompt.Default,
		apiClient:   api.New(),
		logger:      log.Default,
		getenv:      os.Getenv,
		oauthConfig: auth.DefaultOAuthConfig(),
	}
}

// Token type options presented during login.
var tokenTypeOptions = []string{
	"Account token",
	"Workspace token",
	"Project token",
}

// tokenTypeValues maps Select index → Credentials.Type value.
var tokenTypeValues = []string{
	auth.TypeAccount,
	auth.TypeWorkspace,
	auth.TypeProject,
}

// Login handles the login command.
func Login(args []string) error {
	return authLogin(defaultAuthDeps(), args)
}

// Logout handles the logout command.
func Logout(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("logout: additional arguments are not expected: %q", args)
	}

	return authLogout(defaultAuthDeps())
}

// Whoami handles the whoami command.
func Whoami(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("whoami: additional arguments are not expected: %q", args)
	}

	return authWhoami(defaultAuthDeps())
}

func authLogin(deps *authDeps, args []string) error {
	fs := newFlagSet("login")

	var (
		apiKeyFlag      bool
		credentialsPath string
		encryptFlag     string
		tokenTypeFlag   string
		workspaceIDFlag string
	)

	fs.BoolVar(&apiKeyFlag, "api-key", false, "use API key authentication")
	fs.BoolVar(&apiKeyFlag, "apk", false, "use API key authentication (shorthand)")
	fs.StringVar(&credentialsPath, "credentials-path", "", "override credential file path")
	fs.StringVar(&credentialsPath, "cdp", "", "override credential file path (shorthand)")
	fs.StringVar(&encryptFlag, "encrypt", "", "encrypt credentials with password")
	fs.StringVar(&encryptFlag, "enc", "", "encrypt credentials with password (shorthand)")
	fs.StringVar(&tokenTypeFlag, "token-type", "", "token type: account, workspace, project")
	fs.StringVar(&tokenTypeFlag, "tkt", "", "token type (shorthand)")
	fs.StringVar(&workspaceIDFlag, "workspace-id", "", "workspace ID for workspace tokens")
	fs.StringVar(&workspaceIDFlag, "wid", "", "workspace ID (shorthand)")

	if err := parseFlags(fs, args); err != nil {
		if errors.Is(err, errHelp) {
			printLoginHelp(deps.logger)
			return nil
		}

		return err
	}

	// --api-key flag opts into API key flow; otherwise default to browser.
	if apiKeyFlag {
		return authAPIKeyLogin(deps, credentialsPath, encryptFlag, tokenTypeFlag, workspaceIDFlag)
	}

	return authBrowserLogin(deps, credentialsPath, encryptFlag)
}

// authAPIKeyLogin handles the API key login flow.
func authAPIKeyLogin(deps *authDeps, credentialsPath, encryptFlag, tokenTypeFlag, workspaceIDFlag string) error {
	// Resolve token type: flag → prompt.
	tokenType, err := resolveTokenType(deps, tokenTypeFlag)
	if err != nil {
		return fmt.Errorf("login: %w", err)
	}

	// Resolve API key: env var → prompt.
	token := deps.getenv("EASY_RAILWAY_API_KEY")
	if token == "" {
		token, err = deps.prompter.ReadSecret("Enter your Railway API key: ")
		if err != nil {
			return fmt.Errorf("login: %w", err)
		}
	}

	if token == "" {
		return fmt.Errorf("login: API key cannot be empty")
	}

	// Resolve workspace ID for workspace tokens: flag → env → prompt.
	workspaceID, err := resolveWorkspaceID(deps, tokenType, workspaceIDFlag)
	if err != nil {
		return fmt.Errorf("login: %w", err)
	}

	// Validate against Railway API.
	deps.logger.Info("Validating API key...")

	meta, err := validateToken(deps, tokenType, token, workspaceID)
	if err != nil {
		return fmt.Errorf("login: invalid API key: %w", err)
	}

	// Build credential store.
	store, err := buildStore(deps, credentialsPath, encryptFlag)
	if err != nil {
		return err
	}

	// Save credentials with validated metadata.
	creds := &auth.Credentials{
		Type:             tokenType,
		Token:            token,
		UserID:           meta.UserID,
		UserName:         meta.UserName,
		UserEmail:        meta.UserEmail,
		WorkspaceID:      meta.WorkspaceID,
		WorkspaceName:    meta.WorkspaceName,
		ProjectTokenID:   meta.ProjectTokenID,
		ProjectTokenName: meta.ProjectTokenName,
		ProjectID:        meta.ProjectID,
		EnvironmentID:    meta.EnvironmentID,
	}

	if err := store.Save(creds); err != nil {
		return fmt.Errorf("login: %w", err)
	}

	deps.logger.Success("Logged in with %s token.", tokenType)

	return nil
}

// tokenMetadata holds the validated metadata for each token type.
type tokenMetadata struct {
	// Account fields.
	UserID    string
	UserName  string
	UserEmail string

	// Workspace fields.
	WorkspaceID   string
	WorkspaceName string

	// Project fields.
	ProjectTokenID   string
	ProjectTokenName string
	ProjectID        string
	EnvironmentID    string
}

// resolveTokenType resolves the token type from flag value or interactive prompt.
func resolveTokenType(deps *authDeps, flagValue string) (string, error) {
	if flagValue != "" {
		switch flagValue {
		case auth.TypeAccount, auth.TypeWorkspace, auth.TypeProject:
			return flagValue, nil
		default:
			return "", fmt.Errorf("invalid token type %q (must be account, workspace, or project)", flagValue)
		}
	}

	tokenTypeIdx, err := deps.prompter.Select(
		"Select token type (see https://docs.railway.com/integrations/api#choosing-a-token-type):",
		tokenTypeOptions,
	)
	if err != nil {
		return "", fmt.Errorf("select token type: %w", err)
	}

	return tokenTypeValues[tokenTypeIdx], nil
}

// resolveWorkspaceID resolves the workspace ID using flag → env → prompt.
// Returns empty string for non-workspace token types.
func resolveWorkspaceID(deps *authDeps, tokenType, flagValue string) (string, error) {
	if tokenType != auth.TypeWorkspace {
		return "", nil
	}

	if flagValue != "" {
		return flagValue, nil
	}

	if v := deps.getenv("EASY_RAILWAY_WORKSPACE_ID"); v != "" {
		return v, nil
	}

	id, err := deps.prompter.ReadLine("Enter workspace ID: ")
	if err != nil {
		return "", fmt.Errorf("resolve workspace ID: %w", err)
	}

	if id == "" {
		return "", fmt.Errorf("resolve workspace ID: workspace ID cannot be empty")
	}

	return id, nil
}

// validateToken validates the token by calling the appropriate Railway API endpoint.
// Returns metadata to be stored in credentials.
func validateToken(deps *authDeps, tokenType, token, workspaceID string) (*tokenMetadata, error) {
	ctx := context.Background()

	meta := &tokenMetadata{}

	switch tokenType {
	case auth.TypeAccount:
		user, err := deps.apiClient.Me(ctx, token)
		if err != nil {
			return nil, fmt.Errorf("validate token: %w", err)
		}

		meta.UserID = user.ID
		meta.UserName = user.Name
		meta.UserEmail = user.Email

		deps.logger.Info("Authenticated as %s (%s)", user.Name, user.Email)
	case auth.TypeWorkspace:
		// Validate workspace directly — workspace tokens cannot call the me query.
		ws, err := deps.apiClient.ValidateWorkspaceToken(ctx, token, workspaceID)
		if err != nil {
			return nil, fmt.Errorf("validate token: %w", err)
		}

		meta.WorkspaceID = ws.ID
		meta.WorkspaceName = ws.Name

		deps.logger.Info("Workspace: %s (%s)", ws.Name, ws.ID)
	case auth.TypeProject:
		info, err := deps.apiClient.ValidateProjectToken(ctx, token)
		if err != nil {
			return nil, fmt.Errorf("validate token: %w", err)
		}

		meta.ProjectTokenID = info.ID
		meta.ProjectTokenName = info.Name
		meta.ProjectID = info.ProjectID
		meta.EnvironmentID = info.EnvironmentID

		deps.logger.Info("Project token: %s, Project: %s, Environment: %s", info.Name, info.ProjectID, info.EnvironmentID)
	default:
		return nil, fmt.Errorf("unknown token type: %s", tokenType)
	}

	return meta, nil
}

const browserLoginTimeout = 2 * time.Minute

func authBrowserLogin(deps *authDeps, credentialsPath, encryptFlag string) error {
	cfg := deps.oauthConfig

	// Ask about long-lived sessions (offline_access scope).
	scopes := "openid email profile"

	keepIdx, err := deps.prompter.Select(
		"Keep me logged in? (uses refresh tokens for long-lived sessions)",
		[]string{"Yes (recommended)", "No (session expires in ~1 hour)"},
	)
	if err != nil {
		return fmt.Errorf("login: %w", err)
	}

	if keepIdx == 0 {
		scopes += " offline_access"
	}

	// Generate PKCE and state.
	pkce, err := auth.GeneratePKCE()
	if err != nil {
		return fmt.Errorf("login: %w", err)
	}

	state, err := auth.GenerateState()
	if err != nil {
		return fmt.Errorf("login: %w", err)
	}

	authURL := auth.BuildAuthURL(cfg, pkce, state, scopes)

	deps.logger.Info("Opening browser for authentication...")
	deps.logger.Info("If the browser doesn't open, visit:\n  %s", authURL)

	if err := cfg.OpenBrowser(authURL); err != nil {
		deps.logger.Warn("Could not open browser: %v", err)
		deps.logger.Info("Please open the URL above manually.")
	}

	// Wait for the OAuth callback.
	ctx, cancel := context.WithTimeout(context.Background(), browserLoginTimeout)
	defer cancel()

	code, err := auth.ListenForCallback(ctx, cfg, state)
	if err != nil {
		return fmt.Errorf("login: %w", err)
	}

	// Exchange the authorization code for tokens.
	deps.logger.Info("Exchanging authorization code...")

	result, err := auth.ExchangeCode(cfg, code, pkce)
	if err != nil {
		return fmt.Errorf("login: %w", err)
	}

	// Validate by calling the me endpoint with the access token.
	user, err := deps.apiClient.Me(context.Background(), result.AccessToken)
	if err != nil {
		return fmt.Errorf("login: validate OAuth token: %w", err)
	}

	// Build credential store.
	store, err := buildStore(deps, credentialsPath, encryptFlag)
	if err != nil {
		return err
	}

	now := cfg.Now
	if now == nil {
		now = time.Now
	}

	creds := &auth.Credentials{
		Type:         auth.TypeAccount,
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresAt:    now().Add(time.Duration(result.ExpiresIn) * time.Second),
		UserID:       user.ID,
		UserName:     user.Name,
		UserEmail:    user.Email,
	}

	if err := store.Save(creds); err != nil {
		return fmt.Errorf("login: %w", err)
	}

	deps.logger.Success("Logged in as %s (%s)", user.Name, user.Email)

	if result.RefreshToken != "" {
		deps.logger.Info("Session will auto-refresh when expired.")
	} else {
		deps.logger.Info("Session expires in %d seconds.", result.ExpiresIn)
	}

	return nil
}

func authLogout(deps *authDeps) error {
	credPath, err := resolveCredPath(deps, "")
	if err != nil {
		return err
	}

	if _, statErr := os.Stat(credPath); errors.Is(statErr, os.ErrNotExist) {
		deps.logger.Warn("No active session found.")
		return nil
	}

	store := auth.NewFileStore(credPath)
	if err := store.Delete(); err != nil {
		return fmt.Errorf("logout: %w", err)
	}

	deps.logger.Success("Logged out.")

	return nil
}

func authWhoami(deps *authDeps) error {
	creds, err := loadCredentials(deps)
	if err != nil {
		return err
	}

	switch creds.Type {
	case auth.TypeAccount:
		deps.logger.Print("Type:   %s\n", creds.Type)
		deps.logger.Print("Name:   %s\n", creds.UserName)
		deps.logger.Print("Email:  %s\n", creds.UserEmail)
		deps.logger.Print("ID:     %s\n", creds.UserID)

		if creds.AccessToken != "" {
			deps.logger.Print("Auth:   OAuth (browser)\n")

			if !creds.ExpiresAt.IsZero() {
				deps.logger.Print("Expires: %s\n", creds.ExpiresAt.Format(time.RFC3339))
			}

			if creds.RefreshToken != "" {
				deps.logger.Print("Refresh: enabled\n")
			}
		} else {
			deps.logger.Print("Auth:   API key\n")
		}
	case auth.TypeWorkspace:
		deps.logger.Print("Type:       %s\n", creds.Type)
		deps.logger.Print("Workspace:  %s (%s)\n", creds.WorkspaceName, creds.WorkspaceID)
	case auth.TypeProject:
		deps.logger.Print("Type:         %s\n", creds.Type)
		deps.logger.Print("Token:        %s (%s)\n", creds.ProjectTokenName, creds.ProjectTokenID)
		deps.logger.Print("Project:      %s\n", creds.ProjectID)
		deps.logger.Print("Environment:  %s\n", creds.EnvironmentID)
	default:
		deps.logger.Print("Type:   %s\n", creds.Type)
	}

	return nil
}

// buildStore creates the appropriate Store based on flags and env vars.
func buildStore(deps *authDeps, credentialsPath, encryptFlag string) (auth.Store, error) {
	credPath, err := resolveCredPath(deps, credentialsPath)
	if err != nil {
		return nil, err
	}

	fileStore := auth.NewFileStore(credPath)

	password, err := resolvePassword(deps, encryptFlag)
	if err != nil {
		return nil, err
	}

	if len(password) > 0 {
		return auth.NewEncryptedStore(fileStore, password), nil
	}

	return fileStore, nil
}

// resolveCredPath resolves the credential file path using flag → env → default.
func resolveCredPath(deps *authDeps, flagValue string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}

	if v := deps.getenv("EASY_RAILWAY_CREDENTIALS_PATH"); v != "" {
		return v, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve credentials path: %w", err)
	}

	return filepath.Join(home, ".easy-railway", "credentials.json"), nil
}

// resolvePassword resolves the encryption password using flag → env → interactive prompt.
func resolvePassword(deps *authDeps, encryptFlag string) ([]byte, error) {
	// --encrypt was explicitly given.
	if encryptFlag != "" {
		// Check if it's an env var name (read from that env var).
		if v := deps.getenv(encryptFlag); v != "" {
			return []byte(v), nil
		}

		// Otherwise treat the flag value itself as the password.
		return []byte(encryptFlag), nil
	}

	// Check EASY_RAILWAY_SESSION_PASS env var.
	if v := deps.getenv("EASY_RAILWAY_SESSION_PASS"); v != "" {
		return []byte(v), nil
	}

	// Ask interactively.
	idx, err := deps.prompter.Select("Secure session with a password?", []string{"No", "Yes"})
	if err != nil {
		return nil, fmt.Errorf("resolve password: %w", err)
	}

	if idx == 0 {
		return nil, nil
	}

	pw, err := deps.prompter.ReadSecret("Choose a session password: ")
	if err != nil {
		return nil, fmt.Errorf("resolve password: %w", err)
	}

	if pw == "" {
		return nil, fmt.Errorf("resolve password: password cannot be empty")
	}

	return []byte(pw), nil
}

func printLoginHelp(l *log.Logger) {
	l.Print("Usage: easy-railway login [flags]\n\n")
	l.Print("Authentication methods:\n")
	l.Print("  Browser (default)   OAuth2 login via your browser\n")
	l.Print("  API key             Use --api-key flag for token-based auth\n")
	l.Print("\nFlags:\n")
	l.Print("  --api-key, -apk                   Use API key authentication\n")
	l.Print("  --token-type, -tkt TYPE            Token type: account, workspace, project\n")
	l.Print("  --workspace-id, -wid ID            Workspace ID (for workspace tokens)\n")
	l.Print("  --credentials-path, -cdp PATH     Override credential file path\n")
	l.Print("  --encrypt, -enc PASSWORD           Encrypt credentials with password\n")
	l.Print("\nEnvironment variables:\n")
	l.Print("  EASY_RAILWAY_API_KEY              API key (skips interactive prompt)\n")
	l.Print("  EASY_RAILWAY_WORKSPACE_ID         Workspace ID (for workspace tokens)\n")
	l.Print("  EASY_RAILWAY_CREDENTIALS_PATH     Credential file path\n")
	l.Print("  EASY_RAILWAY_SESSION_PASS         Encryption password\n")
}

// loadCredentials loads credentials from the store, handling encrypted files.
// If the credential is an expired OAuth token with a refresh token, it
// automatically refreshes and re-saves the updated credentials.
func loadCredentials(deps *authDeps) (*auth.Credentials, error) {
	credPath, err := resolveCredPath(deps, "")
	if err != nil {
		return nil, err
	}

	fileStore := auth.NewFileStore(credPath)

	var store auth.Store = fileStore

	if auth.IsEncrypted(credPath) {
		pw, err := deps.prompter.ReadSecret("Session password: ")
		if err != nil {
			return nil, fmt.Errorf("auth: %w", err)
		}

		store = auth.NewEncryptedStore(fileStore, []byte(pw))
	}

	creds, err := store.Load()
	if err != nil {
		return nil, fmt.Errorf("auth: %w", err)
	}

	if creds == nil {
		return nil, fmt.Errorf("auth: not logged in — run 'easy-railway login'")
	}

	// Auto-refresh expired OAuth tokens.
	if creds.NeedsRefresh() {
		if err := refreshCredentials(deps, creds, store); err != nil {
			return nil, err
		}
	} else if creds.IsExpired() {
		return nil, fmt.Errorf("auth: session expired — run 'easy-railway login' to re-authenticate")
	}

	return creds, nil
}

// refreshCredentials refreshes an expired OAuth token and persists the updated credentials.
func refreshCredentials(deps *authDeps, creds *auth.Credentials, store auth.Store) error {
	cfg := deps.oauthConfig
	if cfg == nil {
		cfg = auth.DefaultOAuthConfig()
	}

	deps.logger.Debug("Access token expired, refreshing...")

	result, err := auth.RefreshAccessToken(cfg, creds.RefreshToken)
	if err != nil {
		return fmt.Errorf("auth: refresh token: %w", err)
	}

	now := cfg.Now
	if now == nil {
		now = time.Now
	}

	creds.AccessToken = result.AccessToken
	creds.ExpiresAt = now().Add(time.Duration(result.ExpiresIn) * time.Second)

	if result.RefreshToken != "" {
		creds.RefreshToken = result.RefreshToken
	}

	if err := store.Save(creds); err != nil {
		return fmt.Errorf("auth: save refreshed credentials: %w", err)
	}

	deps.logger.Debug("Token refreshed successfully.")

	return nil
}
