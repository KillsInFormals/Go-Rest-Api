package sqlconnect

import (
	"strings"
	"testing"

	"restapi/internal/models"
	"restapi/pkg/utils"
)

func TestGenerateInsertQueryUsesDbColumnNames(t *testing.T) {
	query := utils.GenerateInsertQuery("teachers",models.Teacher{})

	if strings.Contains(query, "omitempty") {
		t.Fatalf("expected query to omit omitempty suffixes, got %q", query)
	}

	if !strings.Contains(query, "first_name") {
		t.Fatalf("expected query to include first_name column, got %q", query)
	}
}

func TestGetStructValuesSkipsEmptyOmitEmptyFields(t *testing.T) {
	teacher := models.Teacher{FirstName: "Ana"}
	values := utils.GetStructValues(teacher)

	if len(values) != 1 {
		t.Fatalf("expected only one non-empty value, got %d values", len(values))
	}

	if got, ok := values[0].(string); !ok || got != "Ana" {
		t.Fatalf("expected first non-empty value to be 'Ana', got %#v", values[0])
	}
}
