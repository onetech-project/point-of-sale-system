package services

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/pos/backend/product-service/src/config"
	"github.com/pos/backend/product-service/src/models"
	"github.com/pos/backend/product-service/src/utils"
	"github.com/xuri/excelize/v2"
)

const (
	ProductImportMaxFileSizeBytes int64 = 5 * 1024 * 1024
	ProductImportMaxRows                = 1000
	productImportMaxPhotoURLs           = 5
	productImportTimeout                = 15 * time.Minute
	productImportPhotoTimeout           = 10 * time.Second
	defaultImportPhotoMaxBytes    int64 = 10 * 1024 * 1024
)

var ProductImportTemplateHeaders = []string{
	"SKU",
	"Name",
	"Category",
	"Selling Price",
	"Cost Price",
	"Tax Rate",
	"Stock Quantity",
	"Description",
	"Photos",
}

type ProductImportService struct {
	db           *sql.DB
	photoService *PhotoService
	httpClient   *http.Client
}

type productImportJobWithFile struct {
	models.ProductImportJob
	SourceFileBytes []byte
}

type parsedProductImportRow struct {
	RowNumber     int
	SKU           string
	Name          string
	Category      string
	SellingPrice  float64
	CostPrice     float64
	TaxRate       float64
	StockQuantity int
	Description   string
	PhotoURLs     []string
	ProductID     uuid.UUID
}

func NewProductImportService(db *sql.DB, photoService *PhotoService) *ProductImportService {
	return &ProductImportService{
		db:           db,
		photoService: photoService,
		httpClient:   newProductImportPhotoClient(),
	}
}

func NormalizeProductImportFormat(filename, contentType, requestedFormat string) (string, error) {
	format := strings.ToLower(strings.TrimSpace(requestedFormat))
	if format == "" {
		ext := strings.ToLower(filepath.Ext(filename))
		switch ext {
		case ".csv":
			format = "csv"
		case ".xlsx":
			format = "xlsx"
		}
	}

	if format == "" {
		mediaType, _, err := mime.ParseMediaType(contentType)
		if err == nil {
			switch strings.ToLower(mediaType) {
			case "text/csv", "application/csv", "application/vnd.ms-excel":
				format = "csv"
			case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
				format = "xlsx"
			}
		}
	}

	switch format {
	case "csv", "xlsx":
		return format, nil
	default:
		return "", fmt.Errorf("unsupported import format")
	}
}

func BuildProductImportTemplate(format string) (string, string, []byte, error) {
	switch strings.ToLower(format) {
	case "csv":
		var buf bytes.Buffer
		writer := csv.NewWriter(&buf)
		if err := writer.Write(ProductImportTemplateHeaders); err != nil {
			return "", "", nil, err
		}
		writer.Flush()
		if err := writer.Error(); err != nil {
			return "", "", nil, err
		}
		return "product_import_template.csv", "text/csv", buf.Bytes(), nil
	case "xlsx":
		file := excelize.NewFile()
		defer file.Close()

		sheet := file.GetSheetName(0)
		for idx, header := range ProductImportTemplateHeaders {
			cell, err := excelize.CoordinatesToCellName(idx+1, 1)
			if err != nil {
				return "", "", nil, err
			}
			if err := file.SetCellValue(sheet, cell, header); err != nil {
				return "", "", nil, err
			}
		}

		buf, err := file.WriteToBuffer()
		if err != nil {
			return "", "", nil, err
		}
		return "product_import_template.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes(), nil
	default:
		return "", "", nil, fmt.Errorf("unsupported template format")
	}
}

func (s *ProductImportService) StartImport(ctx context.Context, tenantID, userID uuid.UUID, filename, format string, fileBytes []byte) (*models.ProductImportJob, error) {
	if len(fileBytes) == 0 {
		return nil, fmt.Errorf("import file cannot be empty")
	}
	if int64(len(fileBytes)) > ProductImportMaxFileSizeBytes {
		return nil, fmt.Errorf("import file exceeds %d byte limit", ProductImportMaxFileSizeBytes)
	}

	job, err := s.createImportJob(ctx, tenantID, userID, filename, format, fileBytes)
	if err != nil {
		return nil, err
	}

	go func(importID uuid.UUID) {
		jobCtx, cancel := context.WithTimeout(context.Background(), productImportTimeout)
		defer cancel()

		if err := s.ProcessImport(jobCtx, importID); err != nil {
			utils.Log.Error("Product import job failed to process: import_id=%s error=%v", importID, err)
		}
	}(job.ID)

	return job, nil
}

func (s *ProductImportService) GetImport(ctx context.Context, tenantID, importID uuid.UUID) (*models.ProductImportJob, error) {
	query := `
		SELECT id, tenant_id, user_id, status, failure_code, source_file_name, source_format,
		       source_file_size, row_count, created_count, error_count, warning_count,
		       errors, warnings, started_at, completed_at, created_at, updated_at
		FROM product_import_jobs
		WHERE id = $1 AND tenant_id = $2
	`

	job := &models.ProductImportJob{}
	err := s.db.QueryRowContext(ctx, query, importID, tenantID).Scan(
		&job.ID,
		&job.TenantID,
		&job.UserID,
		&job.Status,
		&job.FailureCode,
		&job.SourceFileName,
		&job.SourceFormat,
		&job.SourceFileSize,
		&job.RowCount,
		&job.CreatedCount,
		&job.ErrorCount,
		&job.WarningCount,
		&job.Errors,
		&job.Warnings,
		&job.StartedAt,
		&job.CompletedAt,
		&job.CreatedAt,
		&job.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return job, nil
}

func (s *ProductImportService) ResumeQueuedImports(ctx context.Context, limit int) {
	importIDs, err := s.findQueuedImportIDs(ctx, limit)
	if err != nil {
		utils.Log.Warn("Failed to resume queued product imports: %v", err)
		return
	}

	for _, importID := range importIDs {
		go func(id uuid.UUID) {
			jobCtx, cancel := context.WithTimeout(context.Background(), productImportTimeout)
			defer cancel()
			if err := s.ProcessImport(jobCtx, id); err != nil {
				utils.Log.Error("Queued product import failed to process: import_id=%s error=%v", id, err)
			}
		}(importID)
	}
}

func (s *ProductImportService) ProcessImport(ctx context.Context, importID uuid.UUID) error {
	job, err := s.getImportJobForProcessing(ctx, importID)
	if err != nil {
		return err
	}

	claimed, err := s.markImportProcessing(ctx, importID)
	if err != nil {
		return err
	}
	if !claimed {
		return nil
	}

	rows, rowCount, validationIssues, err := parseProductImportRows(job.SourceFormat, job.SourceFileBytes)
	if err != nil {
		issue := models.ProductImportIssue{Message: err.Error()}
		return s.completeImportFailed(ctx, importID, rowCount, models.ProductImportFailureParse, []models.ProductImportIssue{issue}, nil)
	}

	existingSKUs, err := s.findExistingSKUs(ctx, job.TenantID, rows)
	if err != nil {
		issue := models.ProductImportIssue{Message: "failed to validate existing SKUs"}
		return s.completeImportFailed(ctx, importID, rowCount, models.ProductImportFailureInternal, []models.ProductImportIssue{issue}, nil)
	}
	validationIssues = append(validationIssues, validateDuplicateSKUs(rows, existingSKUs)...)

	if len(validationIssues) > 0 {
		return s.completeImportFailed(ctx, importID, rowCount, models.ProductImportFailureValidation, validationIssues, nil)
	}

	categoryCreated, createdRows, err := s.createProductsInTransaction(ctx, job.TenantID, rows)
	if err != nil {
		issue := models.ProductImportIssue{Message: "failed to create imported products"}
		utils.Log.Error("Product import transaction failed: import_id=%s error=%v", importID, err)
		return s.completeImportFailed(ctx, importID, rowCount, models.ProductImportFailureMutation, []models.ProductImportIssue{issue}, nil)
	}

	if categoryCreated {
		s.invalidateCategoryCache(ctx, job.TenantID)
	}

	warnings := s.uploadImportedPhotos(ctx, job.TenantID, createdRows)
	return s.completeImportSucceeded(ctx, importID, rowCount, len(createdRows), warnings)
}

func (s *ProductImportService) findQueuedImportIDs(ctx context.Context, limit int) ([]uuid.UUID, error) {
	if limit <= 0 {
		limit = 10
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id
		FROM product_import_jobs
		WHERE status = $1
		ORDER BY created_at ASC
		LIMIT $2
	`, models.ProductImportStatusQueued, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	importIDs := make([]uuid.UUID, 0)
	for rows.Next() {
		var importID uuid.UUID
		if err := rows.Scan(&importID); err != nil {
			return nil, err
		}
		importIDs = append(importIDs, importID)
	}
	return importIDs, rows.Err()
}

func (s *ProductImportService) createImportJob(ctx context.Context, tenantID, userID uuid.UUID, filename, format string, fileBytes []byte) (*models.ProductImportJob, error) {
	query := `
		INSERT INTO product_import_jobs (
			tenant_id, user_id, status, source_file_name, source_format,
			source_file_size, source_file_bytes, errors, warnings
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, '[]'::jsonb, '[]'::jsonb)
		RETURNING id, tenant_id, user_id, status, failure_code, source_file_name, source_format,
		          source_file_size, row_count, created_count, error_count, warning_count,
		          errors, warnings, started_at, completed_at, created_at, updated_at
	`

	job := &models.ProductImportJob{}
	err := s.db.QueryRowContext(
		ctx,
		query,
		tenantID,
		userID,
		models.ProductImportStatusQueued,
		sanitizeImportSourceFilename(filename),
		format,
		int64(len(fileBytes)),
		fileBytes,
	).Scan(
		&job.ID,
		&job.TenantID,
		&job.UserID,
		&job.Status,
		&job.FailureCode,
		&job.SourceFileName,
		&job.SourceFormat,
		&job.SourceFileSize,
		&job.RowCount,
		&job.CreatedCount,
		&job.ErrorCount,
		&job.WarningCount,
		&job.Errors,
		&job.Warnings,
		&job.StartedAt,
		&job.CompletedAt,
		&job.CreatedAt,
		&job.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return job, nil
}

func (s *ProductImportService) getImportJobForProcessing(ctx context.Context, importID uuid.UUID) (*productImportJobWithFile, error) {
	query := `
		SELECT id, tenant_id, user_id, status, failure_code, source_file_name, source_format,
		       source_file_size, source_file_bytes, row_count, created_count, error_count,
		       warning_count, errors, warnings, started_at, completed_at, created_at, updated_at
		FROM product_import_jobs
		WHERE id = $1
	`

	job := &productImportJobWithFile{}
	err := s.db.QueryRowContext(ctx, query, importID).Scan(
		&job.ID,
		&job.TenantID,
		&job.UserID,
		&job.Status,
		&job.FailureCode,
		&job.SourceFileName,
		&job.SourceFormat,
		&job.SourceFileSize,
		&job.SourceFileBytes,
		&job.RowCount,
		&job.CreatedCount,
		&job.ErrorCount,
		&job.WarningCount,
		&job.Errors,
		&job.Warnings,
		&job.StartedAt,
		&job.CompletedAt,
		&job.CreatedAt,
		&job.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return job, nil
}

func (s *ProductImportService) markImportProcessing(ctx context.Context, importID uuid.UUID) (bool, error) {
	result, err := s.db.ExecContext(ctx, `
		UPDATE product_import_jobs
		SET status = $2, started_at = COALESCE(started_at, NOW()), failure_code = NULL
		WHERE id = $1 AND status = $3
	`, importID, models.ProductImportStatusProcessing, models.ProductImportStatusQueued)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected > 0, nil
}

func (s *ProductImportService) completeImportFailed(ctx context.Context, importID uuid.UUID, rowCount int, failureCode string, issues []models.ProductImportIssue, warnings []models.ProductImportIssue) error {
	errorBytes, err := marshalProductImportIssues(issues)
	if err != nil {
		return err
	}
	warningBytes, err := marshalProductImportIssues(warnings)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE product_import_jobs
		SET status = $2,
		    failure_code = $3,
		    row_count = $4,
		    created_count = 0,
		    error_count = $5,
		    warning_count = $6,
		    errors = $7,
		    warnings = $8,
		    completed_at = NOW()
		WHERE id = $1
	`, importID, models.ProductImportStatusFailed, failureCode, rowCount, len(issues), len(warnings), errorBytes, warningBytes)
	return err
}

func (s *ProductImportService) completeImportSucceeded(ctx context.Context, importID uuid.UUID, rowCount, createdCount int, warnings []models.ProductImportIssue) error {
	warningBytes, err := marshalProductImportIssues(warnings)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE product_import_jobs
		SET status = $2,
		    failure_code = NULL,
		    row_count = $3,
		    created_count = $4,
		    error_count = 0,
		    warning_count = $5,
		    errors = '[]'::jsonb,
		    warnings = $6,
		    completed_at = NOW()
		WHERE id = $1
	`, importID, models.ProductImportStatusCompleted, rowCount, createdCount, len(warnings), warningBytes)
	return err
}

func marshalProductImportIssues(issues []models.ProductImportIssue) ([]byte, error) {
	if issues == nil {
		issues = []models.ProductImportIssue{}
	}
	return json.Marshal(issues)
}

func sanitizeImportSourceFilename(filename string) string {
	filename = SanitizeFilename(filename)
	if filename == "." || filename == "" {
		filename = "product-import"
	}
	filenameRunes := []rune(filename)
	if len(filenameRunes) > 255 {
		ext := filepath.Ext(filename)
		extRunes := []rune(ext)
		limit := 255 - len(extRunes)
		if limit < 1 {
			return string(filenameRunes[:255])
		}
		stem := strings.TrimSuffix(filename, ext)
		stemRunes := []rune(stem)
		if len(stemRunes) > limit {
			stem = string(stemRunes[:limit])
		}
		return stem + ext
	}
	return filename
}

func parseProductImportRows(format string, fileBytes []byte) ([]parsedProductImportRow, int, []models.ProductImportIssue, error) {
	switch format {
	case "csv":
		return parseCSVProductImport(fileBytes)
	case "xlsx":
		return parseXLSXProductImport(fileBytes)
	default:
		return nil, 0, nil, fmt.Errorf("unsupported import format")
	}
}

func parseCSVProductImport(fileBytes []byte) ([]parsedProductImportRow, int, []models.ProductImportIssue, error) {
	reader := csv.NewReader(bytes.NewReader(fileBytes))
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	header, err := reader.Read()
	if err == io.EOF {
		return nil, 0, []models.ProductImportIssue{{Row: 1, Message: "import file is empty"}}, nil
	}
	if err != nil {
		return nil, 0, nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	if len(header) > 0 {
		header[0] = strings.TrimPrefix(header[0], "\ufeff")
	}

	issues := validateProductImportHeader(header)
	rows := make([]parsedProductImportRow, 0)
	rowCount := 0

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			rowNumber := rowCount + 2
			var parseErr *csv.ParseError
			if errors.As(err, &parseErr) && parseErr.Line > 0 {
				rowNumber = parseErr.Line
			}
			issues = append(issues, models.ProductImportIssue{
				Row:     rowNumber,
				Message: fmt.Sprintf("invalid CSV row: %v", err),
			})
			continue
		}
		if isBlankImportRecord(record) {
			continue
		}

		line, _ := reader.FieldPos(0)
		rowNumber := line
		if rowNumber == 0 {
			rowNumber = rowCount + 2
		}
		rowCount++
		if rowCount > ProductImportMaxRows {
			if rowCount == ProductImportMaxRows+1 {
				issues = append(issues, models.ProductImportIssue{
					Row:     rowNumber,
					Message: fmt.Sprintf("maximum row count of %d exceeded", ProductImportMaxRows),
				})
			}
			continue
		}

		row, rowIssues := parseProductImportRecord(record, rowNumber)
		issues = append(issues, rowIssues...)
		if len(rowIssues) == 0 {
			rows = append(rows, row)
		}
	}

	return rows, rowCount, issues, nil
}

func parseXLSXProductImport(fileBytes []byte) ([]parsedProductImportRow, int, []models.ProductImportIssue, error) {
	file, err := excelize.OpenReader(bytes.NewReader(fileBytes))
	if err != nil {
		return nil, 0, nil, fmt.Errorf("failed to open XLSX file: %w", err)
	}
	defer file.Close()

	sheet := file.GetSheetName(0)
	if sheet == "" {
		return nil, 0, []models.ProductImportIssue{{Row: 1, Message: "XLSX file has no sheets"}}, nil
	}

	records, err := file.GetRows(sheet)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("failed to read XLSX rows: %w", err)
	}
	if len(records) == 0 {
		return nil, 0, []models.ProductImportIssue{{Row: 1, Message: "import file is empty"}}, nil
	}

	issues := validateProductImportHeader(records[0])
	rows := make([]parsedProductImportRow, 0)
	rowCount := 0

	for idx, record := range records[1:] {
		rowNumber := idx + 2
		if isBlankImportRecord(record) {
			continue
		}

		rowCount++
		if rowCount > ProductImportMaxRows {
			if rowCount == ProductImportMaxRows+1 {
				issues = append(issues, models.ProductImportIssue{
					Row:     rowNumber,
					Message: fmt.Sprintf("maximum row count of %d exceeded", ProductImportMaxRows),
				})
			}
			continue
		}

		row, rowIssues := parseProductImportRecord(record, rowNumber)
		issues = append(issues, rowIssues...)
		if len(rowIssues) == 0 {
			rows = append(rows, row)
		}
	}

	return rows, rowCount, issues, nil
}

func validateProductImportHeader(record []string) []models.ProductImportIssue {
	issues := make([]models.ProductImportIssue, 0)
	for idx, expected := range ProductImportTemplateHeaders {
		if idx >= len(record) {
			issues = append(issues, models.ProductImportIssue{
				Row:     1,
				Field:   expected,
				Message: "missing required header",
			})
			continue
		}
		if strings.TrimSpace(record[idx]) != expected {
			issues = append(issues, models.ProductImportIssue{
				Row:     1,
				Field:   expected,
				Message: fmt.Sprintf("expected header %q", expected),
			})
		}
	}

	for idx := len(ProductImportTemplateHeaders); idx < len(record); idx++ {
		if strings.TrimSpace(record[idx]) != "" {
			issues = append(issues, models.ProductImportIssue{
				Row:     1,
				Message: "unexpected extra header column",
			})
		}
	}

	return issues
}

func parseProductImportRecord(record []string, rowNumber int) (parsedProductImportRow, []models.ProductImportIssue) {
	cells := make([]string, len(ProductImportTemplateHeaders))
	copy(cells, record)
	for idx := range cells {
		cells[idx] = strings.TrimSpace(cells[idx])
	}

	issues := make([]models.ProductImportIssue, 0)
	if len(record) > len(ProductImportTemplateHeaders) {
		for _, value := range record[len(ProductImportTemplateHeaders):] {
			if strings.TrimSpace(value) != "" {
				issues = append(issues, models.ProductImportIssue{
					Row:     rowNumber,
					Message: "unexpected extra column",
				})
				break
			}
		}
	}

	row := parsedProductImportRow{
		RowNumber:   rowNumber,
		SKU:         cells[0],
		Name:        cells[1],
		Category:    cells[2],
		Description: cells[7],
	}

	if row.SKU == "" {
		issues = append(issues, rowIssue(rowNumber, "SKU", row.SKU, "SKU is required"))
	} else if len(row.SKU) > 50 {
		issues = append(issues, rowIssue(rowNumber, "SKU", row.SKU, "SKU must be at most 50 characters"))
	}

	if row.Name == "" {
		issues = append(issues, rowIssue(rowNumber, "Name", row.SKU, "name is required"))
	} else if len(row.Name) > 255 {
		issues = append(issues, rowIssue(rowNumber, "Name", row.SKU, "name must be at most 255 characters"))
	}

	if len(row.Category) > 100 {
		issues = append(issues, rowIssue(rowNumber, "Category", row.SKU, "category must be at most 100 characters"))
	}

	if cells[3] == "" {
		issues = append(issues, rowIssue(rowNumber, "Selling Price", row.SKU, "selling price is required"))
	} else if price, err := parseNonNegativeFloat(cells[3]); err != nil {
		issues = append(issues, rowIssue(rowNumber, "Selling Price", row.SKU, err.Error()))
	} else {
		row.SellingPrice = price
	}

	if cells[4] == "" {
		issues = append(issues, rowIssue(rowNumber, "Cost Price", row.SKU, "cost price is required"))
	} else if price, err := parseNonNegativeFloat(cells[4]); err != nil {
		issues = append(issues, rowIssue(rowNumber, "Cost Price", row.SKU, err.Error()))
	} else {
		row.CostPrice = price
	}

	if cells[5] != "" {
		taxRate, err := parseNonNegativeFloat(cells[5])
		if err != nil {
			issues = append(issues, rowIssue(rowNumber, "Tax Rate", row.SKU, err.Error()))
		} else if taxRate > 100 {
			issues = append(issues, rowIssue(rowNumber, "Tax Rate", row.SKU, "tax rate must be between 0 and 100"))
		} else {
			row.TaxRate = taxRate
		}
	}

	if cells[6] != "" {
		stockQuantity, err := strconv.Atoi(cells[6])
		if err != nil {
			issues = append(issues, rowIssue(rowNumber, "Stock Quantity", row.SKU, "stock quantity must be a whole number"))
		} else if stockQuantity < 0 {
			issues = append(issues, rowIssue(rowNumber, "Stock Quantity", row.SKU, "stock quantity cannot be negative"))
		} else {
			row.StockQuantity = stockQuantity
		}
	}

	if cells[8] != "" {
		photoURLs, photoIssues := parseImportPhotoURLs(cells[8], rowNumber, row.SKU, productImportMaxPhotoURLs)
		issues = append(issues, photoIssues...)
		row.PhotoURLs = photoURLs
	}

	return row, issues
}

func parseNonNegativeFloat(value string) (float64, error) {
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("must be a valid number")
	}
	if parsed < 0 {
		return 0, fmt.Errorf("must be greater than or equal to 0")
	}
	return parsed, nil
}

func parseImportPhotoURLs(value string, rowNumber int, sku string, maxPhotos int) ([]string, []models.ProductImportIssue) {
	parts := strings.Split(value, ",")
	urls := make([]string, 0, len(parts))
	issues := make([]models.ProductImportIssue, 0)
	for _, part := range parts {
		photoURL := strings.TrimSpace(part)
		if photoURL == "" {
			continue
		}
		if len(urls) >= maxPhotos {
			issues = append(issues, rowIssue(rowNumber, "Photos", sku, fmt.Sprintf("maximum of %d photo URLs allowed", maxPhotos)))
			continue
		}
		if err := validateImportPhotoURL(photoURL); err != nil {
			issues = append(issues, rowIssue(rowNumber, "Photos", sku, err.Error()))
			continue
		}
		urls = append(urls, photoURL)
	}
	return urls, issues
}

func validateImportPhotoURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed == nil {
		return fmt.Errorf("photo URL is invalid")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("photo URL must use http or https")
	}
	if parsed.Hostname() == "" {
		return fmt.Errorf("photo URL host is required")
	}
	if parsed.User != nil {
		return fmt.Errorf("photo URL credentials are not allowed")
	}
	return nil
}

func rowIssue(rowNumber int, field, sku, message string) models.ProductImportIssue {
	return models.ProductImportIssue{
		Row:     rowNumber,
		SKU:     sku,
		Field:   field,
		Message: message,
	}
}

func isBlankImportRecord(record []string) bool {
	for _, value := range record {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}

func validateDuplicateSKUs(rows []parsedProductImportRow, existingSKUs map[string]struct{}) []models.ProductImportIssue {
	counts := make(map[string]int)
	for _, row := range rows {
		counts[row.SKU]++
	}

	issues := make([]models.ProductImportIssue, 0)
	for _, row := range rows {
		if counts[row.SKU] > 1 {
			issues = append(issues, rowIssue(row.RowNumber, "SKU", row.SKU, "duplicate SKU in import file"))
			continue
		}
		if _, exists := existingSKUs[row.SKU]; exists {
			issues = append(issues, rowIssue(row.RowNumber, "SKU", row.SKU, "SKU already exists"))
		}
	}
	return issues
}

func (s *ProductImportService) findExistingSKUs(ctx context.Context, tenantID uuid.UUID, rows []parsedProductImportRow) (map[string]struct{}, error) {
	skuSet := make(map[string]struct{})
	for _, row := range rows {
		skuSet[row.SKU] = struct{}{}
	}
	if len(skuSet) == 0 {
		return map[string]struct{}{}, nil
	}

	skus := make([]string, 0, len(skuSet))
	for sku := range skuSet {
		skus = append(skus, sku)
	}

	query := `SELECT sku FROM products WHERE tenant_id = $1 AND sku = ANY($2)`
	dbRows, err := s.db.QueryContext(ctx, query, tenantID, pq.Array(skus))
	if err != nil {
		return nil, err
	}
	defer dbRows.Close()

	existing := make(map[string]struct{})
	for dbRows.Next() {
		var sku string
		if err := dbRows.Scan(&sku); err != nil {
			return nil, err
		}
		existing[sku] = struct{}{}
	}
	return existing, dbRows.Err()
}

func (s *ProductImportService) createProductsInTransaction(ctx context.Context, tenantID uuid.UUID, rows []parsedProductImportRow) (bool, []parsedProductImportRow, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, nil, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `SELECT set_config('app.current_tenant_id', $1, true)`, tenantID.String()); err != nil {
		return false, nil, err
	}

	categories, maxDisplayOrder, err := loadCategoryMap(ctx, tx, tenantID)
	if err != nil {
		return false, nil, err
	}

	categoryCreated := false
	createdRows := make([]parsedProductImportRow, 0, len(rows))
	for idx, row := range rows {
		var categoryID *uuid.UUID
		if row.Category != "" {
			id, exists := categories[row.Category]
			if !exists {
				maxDisplayOrder++
				id, err = createCategoryInImport(ctx, tx, tenantID, row.Category, maxDisplayOrder)
				if err != nil {
					return false, nil, err
				}
				categories[row.Category] = id
				categoryCreated = true
			}
			categoryID = &id
		}

		productID, err := createProductInImport(ctx, tx, tenantID, row, categoryID)
		if err != nil {
			return false, nil, err
		}
		rows[idx].ProductID = productID
		createdRows = append(createdRows, rows[idx])
	}

	if err := tx.Commit(); err != nil {
		return false, nil, err
	}

	return categoryCreated, createdRows, nil
}

func loadCategoryMap(ctx context.Context, tx *sql.Tx, tenantID uuid.UUID) (map[string]uuid.UUID, int, error) {
	query := `SELECT id, name, display_order FROM categories WHERE tenant_id = $1`
	rows, err := tx.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	categories := make(map[string]uuid.UUID)
	maxDisplayOrder := 0
	for rows.Next() {
		var id uuid.UUID
		var name string
		var displayOrder int
		if err := rows.Scan(&id, &name, &displayOrder); err != nil {
			return nil, 0, err
		}
		categories[name] = id
		if displayOrder > maxDisplayOrder {
			maxDisplayOrder = displayOrder
		}
	}

	return categories, maxDisplayOrder, rows.Err()
}

func createCategoryInImport(ctx context.Context, tx *sql.Tx, tenantID uuid.UUID, name string, displayOrder int) (uuid.UUID, error) {
	query := `
		WITH inserted AS (
			INSERT INTO categories (tenant_id, name, display_order)
			VALUES ($1, $2, $3)
			ON CONFLICT (tenant_id, name) DO NOTHING
			RETURNING id
		)
		SELECT id FROM inserted
		UNION ALL
		SELECT id FROM categories WHERE tenant_id = $1 AND name = $2
		LIMIT 1
	`

	var id uuid.UUID
	if err := tx.QueryRowContext(ctx, query, tenantID, name, displayOrder).Scan(&id); err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func createProductInImport(ctx context.Context, tx *sql.Tx, tenantID uuid.UUID, row parsedProductImportRow, categoryID *uuid.UUID) (uuid.UUID, error) {
	var description interface{}
	if row.Description != "" {
		description = row.Description
	}

	query := `
		INSERT INTO products (
			tenant_id, sku, name, description, category_id, selling_price,
			cost_price, tax_rate, stock_quantity
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`

	var productID uuid.UUID
	err := tx.QueryRowContext(
		ctx,
		query,
		tenantID,
		row.SKU,
		row.Name,
		description,
		categoryID,
		row.SellingPrice,
		row.CostPrice,
		row.TaxRate,
		row.StockQuantity,
	).Scan(&productID)
	return productID, err
}

func (s *ProductImportService) invalidateCategoryCache(ctx context.Context, tenantID uuid.UUID) {
	if config.RedisClient == nil {
		return
	}
	cacheKey := fmt.Sprintf("categories:tenant:%s", tenantID.String())
	if err := config.RedisClient.Del(ctx, cacheKey).Err(); err != nil {
		utils.Log.Warn("Failed to invalidate category cache after product import: tenant_id=%s error=%v", tenantID, err)
	}
}

func (s *ProductImportService) uploadImportedPhotos(ctx context.Context, tenantID uuid.UUID, rows []parsedProductImportRow) []models.ProductImportIssue {
	if s.photoService == nil {
		return nil
	}

	warnings := make([]models.ProductImportIssue, 0)
	for _, row := range rows {
		successCount := 0
		for idx, photoURL := range row.PhotoURLs {
			photoCtx, cancel := context.WithTimeout(ctx, productImportPhotoTimeout)
			imageBytes, mimeType, err := s.downloadPhoto(photoCtx, photoURL)
			cancel()
			if err != nil {
				warnings = append(warnings, rowIssue(row.RowNumber, "Photos", row.SKU, fmt.Sprintf("photo %d download failed: %v", idx+1, err)))
				continue
			}

			filename := productImportPhotoFilename(photoURL, row.RowNumber, idx+1, mimeType)
			isPrimary := successCount == 0
			_, err = s.photoService.UploadPhoto(ctx, row.ProductID, tenantID, filename, bytes.NewReader(imageBytes), successCount, isPrimary)
			if err != nil {
				warnings = append(warnings, rowIssue(row.RowNumber, "Photos", row.SKU, fmt.Sprintf("photo %d upload failed: %v", idx+1, err)))
				continue
			}
			successCount++
		}
	}

	return warnings
}

func (s *ProductImportService) downloadPhoto(ctx context.Context, rawURL string) ([]byte, string, error) {
	if err := validateImportPhotoURL(rawURL); err != nil {
		return nil, "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, "", fmt.Errorf("photo URL returned status %d", resp.StatusCode)
	}

	headerMime := normalizeContentType(resp.Header.Get("Content-Type"))
	if headerMime != "" && !isAllowedProductImportPhotoMIME(headerMime) {
		return nil, "", fmt.Errorf("unsupported photo content type %s", headerMime)
	}

	maxBytes := s.maxPhotoBytes()
	imageBytes, err := readLimited(resp.Body, maxBytes)
	if err != nil {
		return nil, "", err
	}

	detectedMime := detectImageMIME(imageBytes)
	if detectedMime == "" || !isAllowedProductImportPhotoMIME(detectedMime) {
		return nil, "", fmt.Errorf("downloaded file is not a supported image")
	}
	if headerMime != "" && headerMime != detectedMime {
		return nil, "", fmt.Errorf("photo content type mismatch")
	}

	return imageBytes, detectedMime, nil
}

func (s *ProductImportService) maxPhotoBytes() int64 {
	if s.photoService != nil && s.photoService.imageProcessor != nil && s.photoService.imageProcessor.maxSizeBytes > 0 {
		return s.photoService.imageProcessor.maxSizeBytes
	}
	return defaultImportPhotoMaxBytes
}

func readLimited(reader io.Reader, maxBytes int64) ([]byte, error) {
	limited := io.LimitReader(reader, maxBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("photo exceeds %d byte limit", maxBytes)
	}
	return data, nil
}

func normalizeContentType(contentType string) string {
	if contentType == "" {
		return ""
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return strings.ToLower(strings.TrimSpace(contentType))
	}
	return strings.ToLower(mediaType)
}

func detectImageMIME(data []byte) string {
	if len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return "image/webp"
	}
	detected := http.DetectContentType(data)
	switch detected {
	case "image/jpeg", "image/png", "image/gif":
		return detected
	default:
		return ""
	}
}

func isAllowedProductImportPhotoMIME(mimeType string) bool {
	switch strings.ToLower(mimeType) {
	case "image/jpeg", "image/png", "image/webp", "image/gif":
		return true
	default:
		return false
	}
}

func productImportPhotoFilename(rawURL string, rowNumber, index int, mimeType string) string {
	filename := ""
	if parsed, err := url.Parse(rawURL); err == nil {
		filename = path.Base(parsed.Path)
	}
	if filename == "" || filename == "." || filename == "/" {
		filename = fmt.Sprintf("import-row-%d-photo-%d%s", rowNumber, index, extensionForMIME(mimeType))
	}
	if filepath.Ext(filename) == "" {
		filename += extensionForMIME(mimeType)
	}
	return SanitizeFilename(filename)
}

func extensionForMIME(mimeType string) string {
	switch mimeType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		return ".jpg"
	}
}

func newProductImportPhotoClient() *http.Client {
	resolver := net.DefaultResolver
	dialer := &net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	transport := &http.Transport{
		Proxy: nil,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			ip, err := resolvePublicIP(ctx, resolver, host)
			if err != nil {
				return nil, err
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		},
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConns:          10,
		IdleConnTimeout:       30 * time.Second,
	}

	return &http.Client{
		Timeout:   productImportPhotoTimeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("too many redirects")
			}
			return validateImportPhotoURL(req.URL.String())
		},
	}
}

func resolvePublicIP(ctx context.Context, resolver *net.Resolver, host string) (netip.Addr, error) {
	host = strings.Trim(host, "[]")
	if ip, err := netip.ParseAddr(host); err == nil {
		ip = ip.Unmap()
		if isUnsafeImportPhotoIP(ip) {
			return netip.Addr{}, fmt.Errorf("photo URL resolves to a private or unsafe address")
		}
		return ip, nil
	}

	ips, err := resolver.LookupIPAddr(ctx, host)
	if err != nil {
		return netip.Addr{}, err
	}
	if len(ips) == 0 {
		return netip.Addr{}, fmt.Errorf("photo URL host did not resolve")
	}

	var selected netip.Addr
	for _, ipAddr := range ips {
		ip, ok := netip.AddrFromSlice(ipAddr.IP)
		if !ok {
			continue
		}
		ip = ip.Unmap()
		if isUnsafeImportPhotoIP(ip) {
			return netip.Addr{}, fmt.Errorf("photo URL resolves to a private or unsafe address")
		}
		if !selected.IsValid() {
			selected = ip
		}
	}
	if !selected.IsValid() {
		return netip.Addr{}, fmt.Errorf("photo URL host did not resolve to a valid address")
	}
	return selected, nil
}

func isUnsafeImportPhotoIP(ip netip.Addr) bool {
	if !ip.IsValid() {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	if !ip.IsGlobalUnicast() {
		return true
	}
	cgnat := netip.MustParsePrefix("100.64.0.0/10")
	return cgnat.Contains(ip)
}
