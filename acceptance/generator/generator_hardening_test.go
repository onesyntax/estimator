//go:build hardening

package generator

import (
	"os"
	"path/filepath"
	"testing"
)

// Kills writeMetadata `schema_version: 1 -> 0`: the metadata must record schema
// version 1. JSON numbers decode into float64.
func TestHardeningMetadataRecordsSchemaVersionOne(t *testing.T) {
	outDir := t.TempDir()
	irPath := filepath.Join(t.TempDir(), "wbs_approval.json")
	if err := os.WriteFile(irPath, []byte(sampleIR), 0o644); err != nil {
		t.Fatalf("write IR: %v", err)
	}

	if code := Run([]string{irPath, outDir}, noEnv, os.Stderr); code != 0 {
		t.Fatalf("Run exit code = %d, want 0", code)
	}

	meta := readOnlyMetadata(t, outDir)
	if got, ok := meta["schema_version"].(float64); !ok || got != 1 {
		t.Fatalf("schema_version = %v (%T), want 1", meta["schema_version"], meta["schema_version"])
	}
}
