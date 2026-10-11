package render

import (
	"bytes"
	"strings"
	"testing"
)

func TestHeader_uppercase(t *testing.T) {
	var buf bytes.Buffer
	tbl := TableTo(&buf)
	tbl.Header("name", "namespace", "status")

	// Verify headers are uppercased
	if tbl.header[0] != "NAME" {
		t.Errorf("expected NAME, got %s", tbl.header[0])
	}
	if tbl.header[1] != "NAMESPACE" {
		t.Errorf("expected NAMESPACE, got %s", tbl.header[1])
	}
	if tbl.header[2] != "STATUS" {
		t.Errorf("expected STATUS, got %s", tbl.header[2])
	}
}

func TestAppend_rows(t *testing.T) {
	var buf bytes.Buffer
	tbl := TableTo(&buf)
	tbl.Append("pod1", "default", "Running")
	tbl.Append("pod2", "kube-system", "Pending")

	if len(tbl.rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(tbl.rows))
	}
	if tbl.rows[0][0] != "pod1" {
		t.Errorf("expected pod1, got %s", tbl.rows[0][0])
	}
	if tbl.rows[1][1] != "kube-system" {
		t.Errorf("expected kube-system, got %s", tbl.rows[1][1])
	}
}

func TestFooter_set(t *testing.T) {
	var buf bytes.Buffer
	tbl := TableTo(&buf)
	tbl.Footer("total", "2")

	if tbl.footer == nil {
		t.Fatal("expected footer to be set")
	}
	if tbl.footer[0] != "total" {
		t.Errorf("expected total, got %s", tbl.footer[0])
	}
}

func TestRender_full_table(t *testing.T) {
	var buf bytes.Buffer
	tbl := TableTo(&buf)
	tbl.Header("NAME", "STATUS")
	tbl.Append("pod1", "Running")
	tbl.Append("pod2", "Pending")
	tbl.Footer("total", "2")

	if err := tbl.Render(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "NAME") {
		t.Error("expected output to contain NAME header")
	}
	if !strings.Contains(output, "pod1") {
		t.Error("expected output to contain pod1")
	}
	if !strings.Contains(output, "total") {
		t.Error("expected output to contain footer")
	}
}

func TestRender_empty_table(t *testing.T) {
	var buf bytes.Buffer
	tbl := TableTo(&buf)

	if err := tbl.Render(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if output != "" {
		t.Errorf("expected empty output, got %q", output)
	}
}

func TestRender_header_only(t *testing.T) {
	var buf bytes.Buffer
	tbl := TableTo(&buf)
	tbl.Header("NAME", "STATUS")

	if err := tbl.Render(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "NAME") {
		t.Error("expected output to contain NAME header")
	}
	if strings.Contains(output, "pod1") {
		t.Error("expected no rows in output")
	}
}

func TestRender_rows_without_header(t *testing.T) {
	var buf bytes.Buffer
	tbl := TableTo(&buf)
	tbl.Append("pod1", "Running")

	if err := tbl.Render(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "pod1") {
		t.Error("expected output to contain pod1")
	}
}
