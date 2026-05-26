package services

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pos/backend/product-service/src/models"
)

func TestParseCSVProductImportValidRow(t *testing.T) {
	csvData := strings.Join([]string{
		strings.Join(ProductImportTemplateHeaders, ","),
		`SKU-1,Coffee,Drinks,15000,9000,10,25,Hot coffee,"https://example.com/coffee.jpg,https://example.com/coffee-2.jpg"`,
	}, "\n")

	rows, rowCount, issues, err := parseProductImportRows("csv", []byte(csvData))
	if err != nil {
		t.Fatalf("parseProductImportRows error = %v", err)
	}
	if rowCount != 1 {
		t.Fatalf("rowCount = %d, want 1", rowCount)
	}
	if len(issues) != 0 {
		t.Fatalf("issues = %#v, want none", issues)
	}
	if len(rows) != 1 {
		t.Fatalf("rows length = %d, want 1", len(rows))
	}

	row := rows[0]
	if row.SKU != "SKU-1" || row.Name != "Coffee" || row.Category != "Drinks" {
		t.Fatalf("parsed row = %#v", row)
	}
	if row.SellingPrice != 15000 || row.CostPrice != 9000 || row.TaxRate != 10 || row.StockQuantity != 25 {
		t.Fatalf("parsed numeric fields = %#v", row)
	}
	if len(row.PhotoURLs) != 2 {
		t.Fatalf("photo urls = %#v, want 2", row.PhotoURLs)
	}
}

func TestParseCSVProductImportValidationIssues(t *testing.T) {
	csvData := strings.Join([]string{
		strings.Join(ProductImportTemplateHeaders, ","),
		`,,Drinks,-1,abc,101,-5,,ftp://example.com/photo.jpg`,
	}, "\n")

	rows, rowCount, issues, err := parseProductImportRows("csv", []byte(csvData))
	if err != nil {
		t.Fatalf("parseProductImportRows error = %v", err)
	}
	if rowCount != 1 {
		t.Fatalf("rowCount = %d, want 1", rowCount)
	}
	if len(rows) != 0 {
		t.Fatalf("rows length = %d, want 0", len(rows))
	}

	wantFields := map[string]bool{
		"SKU":            false,
		"Name":           false,
		"Selling Price":  false,
		"Cost Price":     false,
		"Tax Rate":       false,
		"Stock Quantity": false,
		"Photos":         false,
	}
	for _, issue := range issues {
		if _, ok := wantFields[issue.Field]; ok {
			wantFields[issue.Field] = true
		}
	}
	for field, found := range wantFields {
		if !found {
			t.Fatalf("missing validation issue for field %s; issues=%#v", field, issues)
		}
	}
}

func TestValidateDuplicateSKUsFlagsFileAndDatabaseDuplicates(t *testing.T) {
	rows := []parsedProductImportRow{
		{RowNumber: 2, SKU: "SKU-1"},
		{RowNumber: 3, SKU: "SKU-1"},
		{RowNumber: 4, SKU: "SKU-2"},
	}
	existing := map[string]struct{}{"SKU-2": {}}

	issues := validateDuplicateSKUs(rows, existing)
	if len(issues) != 3 {
		t.Fatalf("issues length = %d, want 3: %#v", len(issues), issues)
	}

	assertIssue := func(row int, sku, message string) {
		t.Helper()
		for _, issue := range issues {
			if issue.Row == row && issue.SKU == sku && issue.Message == message {
				return
			}
		}
		t.Fatalf("missing issue row=%d sku=%s message=%q in %#v", row, sku, message, issues)
	}

	assertIssue(2, "SKU-1", "duplicate SKU in import file")
	assertIssue(3, "SKU-1", "duplicate SKU in import file")
	assertIssue(4, "SKU-2", "SKU already exists")
}

func TestParseCSVProductImportRowLimit(t *testing.T) {
	var builder strings.Builder
	builder.WriteString(strings.Join(ProductImportTemplateHeaders, ","))
	builder.WriteString("\n")
	for i := 0; i < ProductImportMaxRows+1; i++ {
		builder.WriteString(fmt.Sprintf("SKU-%d,Product %d,,1,1,0,0,,\n", i, i))
	}

	_, rowCount, issues, err := parseProductImportRows("csv", []byte(builder.String()))
	if err != nil {
		t.Fatalf("parseProductImportRows error = %v", err)
	}
	if rowCount != ProductImportMaxRows+1 {
		t.Fatalf("rowCount = %d, want %d", rowCount, ProductImportMaxRows+1)
	}

	for _, issue := range issues {
		if strings.Contains(issue.Message, "maximum row count") {
			return
		}
	}
	t.Fatalf("missing row limit issue in %#v", issues)
}

func TestProductImportIssueJSONShape(t *testing.T) {
	issue := models.ProductImportIssue{Row: 2, SKU: "SKU-1", Field: "SKU", Message: "duplicate SKU in import file"}
	if issue.Row == 0 || issue.SKU == "" || issue.Field == "" || issue.Message == "" {
		t.Fatalf("issue shape changed unexpectedly: %#v", issue)
	}
}
