package submission

import (
	"archive/tar"
	"bytes"
	"io"
	"testing"
)

func TestWriteArchive(t *testing.T) {
	var output bytes.Buffer
	bundle := []byte("bundle contents")
	if err := WriteArchive(&output, "change.bundle", bytes.NewReader(bundle), int64(len(bundle))); err != nil {
		t.Fatal(err)
	}

	reader := tar.NewReader(bytes.NewReader(output.Bytes()))
	assertTarFile(t, reader, ManifestName, "format_version: 1\nbundle: change.bundle\n")
	assertTarFile(t, reader, "change.bundle", string(bundle))
	if _, err := reader.Next(); err != io.EOF {
		t.Fatalf("third tar entry error = %v, want EOF", err)
	}
}

func TestWriteArchiveRejectsUnsafeBundleName(t *testing.T) {
	for _, name := range []string{"bundle", "../change.bundle", "directory/change.bundle", "change file.bundle"} {
		if err := WriteArchive(io.Discard, name, bytes.NewReader([]byte("x")), 1); err == nil {
			t.Errorf("WriteArchive accepted %q", name)
		}
	}
}

func assertTarFile(t *testing.T, reader *tar.Reader, name, contents string) {
	t.Helper()
	header, err := reader.Next()
	if err != nil {
		t.Fatal(err)
	}
	if header.Name != name {
		t.Fatalf("entry name = %q, want %q", header.Name, name)
	}
	actual, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if string(actual) != contents {
		t.Fatalf("%s = %q, want %q", name, actual, contents)
	}
}
