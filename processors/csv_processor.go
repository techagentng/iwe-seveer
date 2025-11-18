package processors

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/techagentng/iweapp/db"
	"github.com/techagentng/iweapp/models"
)

// CSVProcessor handles CSV file processing
type CSVProcessor struct {
	uploadRepo db.UploadRepository
}

// NewCSVProcessor creates a new CSVProcessor instance
func NewCSVProcessor(uploadRepo db.UploadRepository) *CSVProcessor {
	return &CSVProcessor{
		uploadRepo: uploadRepo,
	}
}

// ProcessBankStatementCSV processes a bank statement CSV file asynchronously
func (p *CSVProcessor) ProcessBankStatementCSV(fileID uuid.UUID, reader io.Reader) error {
	log.Printf("Starting CSV processing for file: %s", fileID)

	// Update file status to processing
	file, err := p.uploadRepo.GetUploadedFileByID(fileID)
	if err != nil {
		return fmt.Errorf("failed to get file: %w", err)
	}
	
	file.Status = models.FileStatusProcessing
	if err := p.uploadRepo.UpdateUploadedFile(file); err != nil {
		log.Printf("Failed to update file status to processing: %v", err)
	}

	// Parse CSV
	csvReader := csv.NewReader(reader)
	csvReader.LazyQuotes = true
	csvReader.TrimLeadingSpace = true

	// Read header row
	headers, err := csvReader.Read()
	if err != nil {
		return p.handleProcessingError(file, fmt.Errorf("failed to read CSV headers: %w", err))
	}

	log.Printf("CSV Headers: %v", headers)

	// Validate headers (flexible matching)
	headerMap := p.mapHeaders(headers)
	if !p.validateHeaders(headerMap) {
		return p.handleProcessingError(file, fmt.Errorf("invalid CSV format: missing required columns"))
	}

	// Process rows in batches
	statements := []models.BankStatement{}
	batchSize := 100
	rowCount := 0

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Error reading CSV row %d: %v", rowCount+1, err)
			continue // Skip invalid rows
		}

		// Parse row into BankStatement
		statement, err := p.parseCSVRow(record, headerMap, fileID)
		if err != nil {
			log.Printf("Error parsing CSV row %d: %v", rowCount+1, err)
			continue // Skip invalid rows
		}

		statements = append(statements, *statement)
		rowCount++

		// Batch insert when batch size is reached
		if len(statements) >= batchSize {
			if err := p.uploadRepo.BatchCreateBankStatements(statements); err != nil {
				return p.handleProcessingError(file, fmt.Errorf("failed to insert batch: %w", err))
			}
			log.Printf("Inserted batch of %d statements", len(statements))
			statements = []models.BankStatement{} // Reset batch
		}
	}

	// Insert remaining statements
	if len(statements) > 0 {
		if err := p.uploadRepo.BatchCreateBankStatements(statements); err != nil {
			return p.handleProcessingError(file, fmt.Errorf("failed to insert final batch: %w", err))
		}
		log.Printf("Inserted final batch of %d statements", len(statements))
	}

	// Update file status to completed
	now := time.Now()
	file.Status = models.FileStatusCompleted
	file.ProcessedAt = &now
	if err := p.uploadRepo.UpdateUploadedFile(file); err != nil {
		log.Printf("Failed to update file status to completed: %v", err)
	}

	log.Printf("CSV processing completed for file: %s. Total rows: %d", fileID, rowCount)
	return nil
}

// mapHeaders creates a map of header names to column indices
func (p *CSVProcessor) mapHeaders(headers []string) map[string]int {
	headerMap := make(map[string]int)
	for i, header := range headers {
		normalized := strings.ToLower(strings.TrimSpace(header))
		headerMap[normalized] = i
	}
	return headerMap
}

// validateHeaders checks if required headers are present
func (p *CSVProcessor) validateHeaders(headerMap map[string]int) bool {
	// Check for common variations of required columns
	hasDate := p.hasAnyHeader(headerMap, []string{"date", "transaction date", "trans date", "transaction_date"})
	hasDescription := p.hasAnyHeader(headerMap, []string{"description", "details", "narration", "particulars"})
	hasBalance := p.hasAnyHeader(headerMap, []string{"balance", "running balance", "closing balance"})
	
	return hasDate && hasDescription && hasBalance
}

// hasAnyHeader checks if any of the given header variations exist
func (p *CSVProcessor) hasAnyHeader(headerMap map[string]int, variations []string) bool {
	for _, v := range variations {
		if _, exists := headerMap[v]; exists {
			return true
		}
	}
	return false
}

// getHeaderIndex returns the index of the first matching header variation
func (p *CSVProcessor) getHeaderIndex(headerMap map[string]int, variations []string) int {
	for _, v := range variations {
		if idx, exists := headerMap[v]; exists {
			return idx
		}
	}
	return -1
}

// parseCSVRow parses a CSV row into a BankStatement
func (p *CSVProcessor) parseCSVRow(record []string, headerMap map[string]int, fileID uuid.UUID) (*models.BankStatement, error) {
	statement := &models.BankStatement{
		FileID: fileID,
	}

	// Parse transaction date
	dateIdx := p.getHeaderIndex(headerMap, []string{"date", "transaction date", "trans date", "transaction_date"})
	if dateIdx >= 0 && dateIdx < len(record) {
		date, err := p.parseDate(record[dateIdx])
		if err != nil {
			return nil, fmt.Errorf("invalid date: %w", err)
		}
		statement.TransactionDate = date
	}

	// Parse description
	descIdx := p.getHeaderIndex(headerMap, []string{"description", "details", "narration", "particulars"})
	if descIdx >= 0 && descIdx < len(record) {
		statement.Description = strings.TrimSpace(record[descIdx])
	}

	// Parse debit amount
	debitIdx := p.getHeaderIndex(headerMap, []string{"debit", "debit amount", "withdrawal", "withdrawals"})
	if debitIdx >= 0 && debitIdx < len(record) {
		if amount, err := p.parseAmount(record[debitIdx]); err == nil && amount > 0 {
			statement.DebitAmount = &amount
		}
	}

	// Parse credit amount
	creditIdx := p.getHeaderIndex(headerMap, []string{"credit", "credit amount", "deposit", "deposits"})
	if creditIdx >= 0 && creditIdx < len(record) {
		if amount, err := p.parseAmount(record[creditIdx]); err == nil && amount > 0 {
			statement.CreditAmount = &amount
		}
	}

	// Parse balance
	balanceIdx := p.getHeaderIndex(headerMap, []string{"balance", "running balance", "closing balance"})
	if balanceIdx >= 0 && balanceIdx < len(record) {
		balance, err := p.parseAmount(record[balanceIdx])
		if err != nil {
			return nil, fmt.Errorf("invalid balance: %w", err)
		}
		statement.Balance = balance
	}

	// Parse currency (optional)
	currencyIdx := p.getHeaderIndex(headerMap, []string{"currency", "ccy"})
	if currencyIdx >= 0 && currencyIdx < len(record) {
		statement.Currency = strings.TrimSpace(record[currencyIdx])
	}
	if statement.Currency == "" {
		statement.Currency = "NGN" // Default to Nigerian Naira
	}

	return statement, nil
}

// parseDate parses a date string in various formats
func (p *CSVProcessor) parseDate(dateStr string) (time.Time, error) {
	dateStr = strings.TrimSpace(dateStr)
	
	// Try common date formats
	formats := []string{
		"2006-01-02",
		"02/01/2006",
		"01/02/2006",
		"2-Jan-2006",
		"02-Jan-2006",
		"2006/01/02",
		"Jan 2, 2006",
		"January 2, 2006",
	}

	for _, format := range formats {
		if date, err := time.Parse(format, dateStr); err == nil {
			return date, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

// parseAmount parses an amount string, handling commas and currency symbols
func (p *CSVProcessor) parseAmount(amountStr string) (float64, error) {
	// Remove common currency symbols and whitespace
	amountStr = strings.TrimSpace(amountStr)
	amountStr = strings.ReplaceAll(amountStr, ",", "")
	amountStr = strings.ReplaceAll(amountStr, "₦", "")
	amountStr = strings.ReplaceAll(amountStr, "$", "")
	amountStr = strings.ReplaceAll(amountStr, "NGN", "")
	amountStr = strings.TrimSpace(amountStr)

	if amountStr == "" || amountStr == "-" {
		return 0, nil
	}

	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount: %s", amountStr)
	}

	return amount, nil
}

// handleProcessingError updates file status to failed and logs the error
func (p *CSVProcessor) handleProcessingError(file *models.UploadedFile, err error) error {
	log.Printf("CSV processing failed for file %s: %v", file.ID, err)
	
	file.Status = models.FileStatusFailed
	file.ErrorMsg = err.Error()
	now := time.Now()
	file.ProcessedAt = &now
	
	if updateErr := p.uploadRepo.UpdateUploadedFile(file); updateErr != nil {
		log.Printf("Failed to update file status to failed: %v", updateErr)
	}
	
	return err
}

// ConvertStatementsToText converts bank statements to readable text format
func (p *CSVProcessor) ConvertStatementsToText(statements []models.BankStatement) string {
	if len(statements) == 0 {
		return "No bank statements found."
	}

	var text string
	text += fmt.Sprintf("Bank Statement Summary\n")
	text += fmt.Sprintf("======================\n\n")
	text += fmt.Sprintf("Total Transactions: %d\n\n", len(statements))

	for i, stmt := range statements {
		text += fmt.Sprintf("Transaction #%d\n", i+1)
		text += fmt.Sprintf("Date: %s\n", stmt.TransactionDate.Format("2006-01-02"))
		text += fmt.Sprintf("Description: %s\n", stmt.Description)
		
		if stmt.DebitAmount != nil && *stmt.DebitAmount > 0 {
			text += fmt.Sprintf("Debit: %.2f %s\n", *stmt.DebitAmount, stmt.Currency)
		}
		if stmt.CreditAmount != nil && *stmt.CreditAmount > 0 {
			text += fmt.Sprintf("Credit: %.2f %s\n", *stmt.CreditAmount, stmt.Currency)
		}
		
		text += fmt.Sprintf("Balance: %.2f %s\n", stmt.Balance, stmt.Currency)
		text += "\n"
	}

	return text
}
