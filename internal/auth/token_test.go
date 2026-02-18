package auth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileStoreRoundTrip(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "creds.json")
	store := NewFileStore(path)

	want := &Credentials{Type: TypeAccount, Token: "test-token-123"}

	if err := store.Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got.Type != want.Type {
		t.Fatalf("Type: expected %q, got %q", want.Type, got.Type)
	}

	if got.Token != want.Token {
		t.Fatalf("Token: expected %q, got %q", want.Token, got.Token)
	}
}

func TestFileStoreLoadMissing(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "nonexistent.json")
	store := NewFileStore(path)

	creds, err := store.Load()
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if creds != nil {
		t.Fatalf("expected nil credentials, got %+v", creds)
	}
}

func TestFileStoreDelete(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "creds.json")
	store := NewFileStore(path)

	if err := store.Save(&Credentials{Type: TypeAccount, Token: "x"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := store.Delete(); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	creds, err := store.Load()
	if err != nil {
		t.Fatalf("Load after delete: %v", err)
	}

	if creds != nil {
		t.Fatalf("expected nil after delete, got %+v", creds)
	}
}

func TestFileStoreDeleteMissing(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "nonexistent.json")
	store := NewFileStore(path)

	if err := store.Delete(); err != nil {
		t.Fatalf("Delete nonexistent: expected nil, got %v", err)
	}
}

func TestFileStorePermissions(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "creds.json")
	store := NewFileStore(path)

	if err := store.Save(&Credentials{Type: TypeAccount, Token: "x"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}

	mode := info.Mode().Perm()
	if mode != 0o600 {
		t.Fatalf("expected file mode 0600, got %04o", mode)
	}
}

func TestFileStoreCreatesDirectory(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "sub", "dir", "creds.json")
	store := NewFileStore(path)

	if err := store.Save(&Credentials{Type: TypeAccount, Token: "x"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	info, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("Stat dir: %v", err)
	}

	mode := info.Mode().Perm()
	if mode != 0o700 {
		t.Fatalf("expected dir mode 0700, got %04o", mode)
	}
}

func TestEncryptedStoreRoundTrip(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "creds.enc")
	inner := NewFileStore(path)
	store := NewEncryptedStore(inner, []byte("strong-password"))

	want := &Credentials{Type: TypeAccount, Token: "secret-token-456"}

	if err := store.Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got.Type != want.Type {
		t.Fatalf("Type: expected %q, got %q", want.Type, got.Type)
	}

	if got.Token != want.Token {
		t.Fatalf("Token: expected %q, got %q", want.Token, got.Token)
	}
}

func TestEncryptedStoreWrongPassword(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "creds.enc")
	inner := NewFileStore(path)

	writer := NewEncryptedStore(inner, []byte("correct-password"))
	if err := writer.Save(&Credentials{Type: TypeAccount, Token: "x"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	reader := NewEncryptedStore(inner, []byte("wrong-password"))
	_, err := reader.Load()

	if err == nil {
		t.Fatal("expected error for wrong password")
	}

	if !strings.Contains(err.Error(), "wrong password") {
		t.Fatalf("expected wrong password error, got %q", err.Error())
	}
}

func TestEncryptedStoreLoadMissing(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "nonexistent.enc")
	inner := NewFileStore(path)
	store := NewEncryptedStore(inner, []byte("password"))

	creds, err := store.Load()
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if creds != nil {
		t.Fatalf("expected nil credentials, got %+v", creds)
	}
}

func TestEncryptedStoreDelete(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "creds.enc")
	inner := NewFileStore(path)
	store := NewEncryptedStore(inner, []byte("password"))

	if err := store.Save(&Credentials{Type: TypeAccount, Token: "x"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := store.Delete(); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	creds, err := store.Load()
	if err != nil {
		t.Fatalf("Load after delete: %v", err)
	}

	if creds != nil {
		t.Fatalf("expected nil after delete, got %+v", creds)
	}
}

func TestIsEncryptedPlainFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "plain.json")
	if err := os.WriteFile(path, []byte(`{"type":"account"}`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if IsEncrypted(path) {
		t.Fatal("expected plain file to not be detected as encrypted")
	}
}

func TestIsEncryptedEncFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "creds.enc")
	inner := NewFileStore(path)
	store := NewEncryptedStore(inner, []byte("password"))

	if err := store.Save(&Credentials{Type: TypeAccount, Token: "x"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if !IsEncrypted(path) {
		t.Fatal("expected encrypted file to be detected")
	}
}

func TestIsEncryptedMissing(t *testing.T) {
	t.Parallel()

	if IsEncrypted(filepath.Join(t.TempDir(), "nonexistent")) {
		t.Fatal("expected missing file to return false")
	}
}

func TestResolvePathFlag(t *testing.T) {
	t.Parallel()

	got, err := ResolvePath("/custom/path.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != "/custom/path.json" {
		t.Fatalf("expected %q, got %q", "/custom/path.json", got)
	}
}

func TestResolvePathEnv(t *testing.T) {
	t.Setenv("EASY_RAILWAY_CREDENTIALS_PATH", "/env/path.json")

	got, err := ResolvePath("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != "/env/path.json" {
		t.Fatalf("expected %q, got %q", "/env/path.json", got)
	}
}

func TestResolvePathDefault(t *testing.T) {
	t.Setenv("EASY_RAILWAY_CREDENTIALS_PATH", "")

	got, err := ResolvePath("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasSuffix(got, filepath.Join(".easy-railway", "credentials.json")) {
		t.Fatalf("expected default path ending with .easy-railway/credentials.json, got %q", got)
	}
}

func TestResolvePathFlagOverridesEnv(t *testing.T) {
	t.Setenv("EASY_RAILWAY_CREDENTIALS_PATH", "/env/path.json")

	got, err := ResolvePath("/flag/path.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != "/flag/path.json" {
		t.Fatalf("expected flag path to win, got %q", got)
	}
}
