package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckValidEntries(t *testing.T) {
	root := t.TempDir()
	writeEntry(t, root, "OEFP/T12/T1234/T123456/T12345678.lean", `import Std
/-!
id: T12345678
title: Example theorem
description: An example.
tags: [example]
status: open
-/
`)
	writeEntry(t, root, "OEFP/D12/D1234/D123456/D12345678.lean", `/-!
id: D12345678
title: Example definition
description: An example.
tags:
  - example
-/
`)

	result := Check(root)
	if result.Files != 2 || len(result.Errors) != 0 {
		t.Fatalf("Check() = %+v", result)
	}
}

func TestCheckRejectsLegacySixDigitDefinitionID(t *testing.T) {
	root := t.TempDir()
	writeEntry(t, root, "OEFP/D12/D1234/D123456.lean", `/-!
id: D123456
title: Legacy definition
description: Uses the old ID width.
tags: [example]
-/
`)

	result := Check(root)
	if len(result.Errors) != 1 || !strings.Contains(result.Errors[0], "T or D followed by 8 digits") {
		t.Fatalf("Check() errors = %v", result.Errors)
	}
}

func TestExpectedDefinitionPathSupportsEightDigits(t *testing.T) {
	want := filepath.FromSlash("OEFP/D99/D9999/D999999/D99999999.lean")
	if got, ok := expectedPath("D99999999"); !ok || got != want {
		t.Fatalf("expectedPath(D99999999) = %q, %v; want %q, true", got, ok, want)
	}
	if _, ok := expectedPath("D100000000"); ok {
		t.Fatal("expectedPath accepted a nine-digit definition ID")
	}
}

func TestCheckReportsPathAndMetadataErrors(t *testing.T) {
	root := t.TempDir()
	writeEntry(t, root, "OEFP/T00/T00000001.lean", `/-!
id: T00000002
title: ""
description: Example
tags: []
extra: nope
-/
`)

	result := Check(root)
	joined := strings.Join(result.Errors, "\n")
	for _, want := range []string{"entry belongs at", "unknown field", "non-empty string", "at least one", "missing required field \"status\"", "does not match filename"} {
		if !strings.Contains(joined, want) {
			t.Errorf("errors do not contain %q:\n%s", want, joined)
		}
	}
}

func writeEntry(t *testing.T, root, name, contents string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}
