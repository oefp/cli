package submission

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const DefaultBase = "oefp/main"

// GenerateOptions configures a submission archive generated from a Git checkout.
type GenerateOptions struct {
	Root   string
	Base   string
	Output string
	Force  bool
}

// Generate creates a Git bundle for Base..HEAD and wraps it in an OEFP archive.
// Only committed work is included, so a dirty checkout is rejected.
func Generate(ctx context.Context, options GenerateOptions) error {
	if options.Base == "" {
		options.Base = DefaultBase
	}
	if options.Output == "" {
		options.Output = "submission.oefp"
	}
	root, err := filepath.Abs(options.Root)
	if err != nil {
		return fmt.Errorf("resolve repository root: %w", err)
	}
	top, err := git(ctx, root, "rev-parse", "--show-toplevel")
	if err != nil {
		return fmt.Errorf("open Git repository: %w", err)
	}
	top = strings.TrimSpace(top)
	resolvedTop, topErr := filepath.EvalSymlinks(top)
	resolvedRoot, rootErr := filepath.EvalSymlinks(root)
	if topErr == nil && rootErr == nil && resolvedTop != resolvedRoot {
		return fmt.Errorf("--root must name the Git repository root (%s)", resolvedTop)
	}
	status, err := git(ctx, root, "status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return fmt.Errorf("inspect working tree: %w", err)
	}
	if strings.TrimSpace(status) != "" {
		return errors.New("working tree is not clean; commit or stash changes before generating a submission")
	}
	base, err := git(ctx, root, "rev-parse", "--verify", options.Base+"^{commit}")
	if err != nil {
		return fmt.Errorf("resolve base %q: %w", options.Base, err)
	}
	base = strings.TrimSpace(base)
	if _, err := git(ctx, root, "merge-base", "--is-ancestor", base, "HEAD"); err != nil {
		return fmt.Errorf("base %q is not an ancestor of HEAD", options.Base)
	}
	countText, err := git(ctx, root, "rev-list", "--count", base+"..HEAD")
	if err != nil {
		return fmt.Errorf("count submission commits: %w", err)
	}
	count, err := strconv.Atoi(strings.TrimSpace(countText))
	if err != nil {
		return fmt.Errorf("parse submission commit count: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("HEAD contains no commits after base %q", options.Base)
	}

	output := options.Output
	if !filepath.IsAbs(output) {
		output = filepath.Join(root, output)
	}
	if _, err := os.Stat(output); err == nil && !options.Force {
		return fmt.Errorf("output %s already exists (use --force to replace it)", output)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect output: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	staging, err := os.MkdirTemp(filepath.Dir(output), ".oefp-generate-")
	if err != nil {
		return fmt.Errorf("create staging directory: %w", err)
	}
	defer os.RemoveAll(staging)

	bundlePath := filepath.Join(staging, "submission.bundle")
	if _, err := git(ctx, root, "bundle", "create", bundlePath, "HEAD", "^"+base); err != nil {
		return fmt.Errorf("create Git bundle: %w", err)
	}
	bundle, err := os.Open(bundlePath)
	if err != nil {
		return fmt.Errorf("open Git bundle: %w", err)
	}
	defer bundle.Close()
	info, err := bundle.Stat()
	if err != nil {
		return fmt.Errorf("inspect Git bundle: %w", err)
	}

	temporary, err := os.CreateTemp(filepath.Dir(output), ".oefp-archive-")
	if err != nil {
		return fmt.Errorf("create submission archive: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := WriteArchive(temporary, filepath.Base(bundlePath), bundle, info.Size()); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close submission archive: %w", err)
	}
	if err := os.Chmod(temporaryPath, 0o644); err != nil {
		return fmt.Errorf("set submission archive permissions: %w", err)
	}
	if options.Force {
		if err := os.Remove(output); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("replace output: %w", err)
		}
	}
	if err := os.Rename(temporaryPath, output); err != nil {
		return fmt.Errorf("publish submission archive: %w", err)
	}
	return nil
}

func git(ctx context.Context, directory string, args ...string) (string, error) {
	command := exec.CommandContext(ctx, "git", args...)
	command.Dir = directory
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return "", errors.New(message)
	}
	return stdout.String(), nil
}
