// Package submission defines the public submission archive emitted by the
// OEFP CLI. Server-specific storage, authentication, and queueing do not
// belong here.
package submission

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
)

const (
	ManifestName  = "submission.yaml"
	FormatVersion = 1
)

// WriteArchive writes a deterministic tar containing submission.yaml and the
// supplied Git bundle. bundleName must be a top-level .bundle filename.
func WriteArchive(destination io.Writer, bundleName string, bundle io.Reader, bundleSize int64) error {
	if !safeBundleName(bundleName) {
		return errors.New("bundle name must be a top-level .bundle file")
	}
	if bundleSize <= 0 {
		return errors.New("bundle size must be positive")
	}

	manifest := []byte(fmt.Sprintf("format_version: %d\nbundle: %s\n", FormatVersion, bundleName))
	archive := tar.NewWriter(destination)
	if err := writeFile(archive, ManifestName, bytes.NewReader(manifest), int64(len(manifest))); err != nil {
		_ = archive.Close()
		return err
	}
	if err := writeFile(archive, bundleName, bundle, bundleSize); err != nil {
		_ = archive.Close()
		return err
	}
	if err := archive.Close(); err != nil {
		return fmt.Errorf("close submission archive: %w", err)
	}
	return nil
}

func writeFile(archive *tar.Writer, name string, source io.Reader, size int64) error {
	header := &tar.Header{
		Name: name,
		Mode: 0o644,
		Size: size,
	}
	if err := archive.WriteHeader(header); err != nil {
		return fmt.Errorf("write %s header: %w", name, err)
	}
	written, err := io.CopyN(archive, source, size)
	if err != nil {
		return fmt.Errorf("write %s: copied %d of %d bytes: %w", name, written, size, err)
	}
	return nil
}

func safeBundleName(name string) bool {
	if name == "" || name == "." || path.Base(name) != name || strings.Contains(name, "\\") {
		return false
	}
	if !strings.HasSuffix(name, ".bundle") {
		return false
	}
	for _, character := range name {
		if character >= 'a' && character <= 'z' ||
			character >= 'A' && character <= 'Z' ||
			character >= '0' && character <= '9' ||
			strings.ContainsRune("._-", character) {
			continue
		}
		return false
	}
	return true
}
