package submission

import (
	"archive/tar"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerate(t *testing.T) {
	repo := newGitRepo(t)
	base := strings.TrimSpace(gitTest(t, repo, "rev-parse", "HEAD"))
	if err := os.WriteFile(filepath.Join(repo, "entry.lean"), []byte("example"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitTest(t, repo, "add", "entry.lean")
	gitTest(t, repo, "commit", "-m", "add entry")

	output := filepath.Join(t.TempDir(), "change.oefp")
	if err := Generate(context.Background(), GenerateOptions{Root: repo, Base: base, Output: output}); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(output)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	reader := tar.NewReader(file)
	assertTarFile(t, reader, ManifestName, "format_version: 1\nbundle: submission.bundle\n")
	header, err := reader.Next()
	if err != nil {
		t.Fatal(err)
	}
	if header.Name != "submission.bundle" || header.Size == 0 {
		t.Fatalf("bundle header = %+v", header)
	}
	bundlePath := filepath.Join(t.TempDir(), "submission.bundle")
	bundle, err := os.Create(bundlePath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(bundle, reader); err != nil {
		t.Fatal(err)
	}
	if err := bundle.Close(); err != nil {
		t.Fatal(err)
	}
	gitTest(t, repo, "bundle", "verify", bundlePath)
}

func TestGenerateRejectsDirtyTree(t *testing.T) {
	repo := newGitRepo(t)
	base := strings.TrimSpace(gitTest(t, repo, "rev-parse", "HEAD"))
	if err := os.WriteFile(filepath.Join(repo, "dirty"), []byte("dirty"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := Generate(context.Background(), GenerateOptions{Root: repo, Base: base, Output: filepath.Join(t.TempDir(), "x.oefp")})
	if err == nil {
		t.Fatal("Generate accepted a dirty tree")
	}
}

func newGitRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	gitTest(t, repo, "init", "-b", "main")
	gitTest(t, repo, "config", "user.name", "Test")
	gitTest(t, repo, "config", "user.email", "test@example.com")
	if err := os.WriteFile(filepath.Join(repo, "README"), []byte("base"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitTest(t, repo, "add", "README")
	gitTest(t, repo, "commit", "-m", "base")
	return repo
}

func gitTest(t *testing.T, directory string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = directory
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
	return string(output)
}
