# 🔄 Hybrid OCR System - Complete Guide

## 📋 Overview

The file upload system now uses a **hybrid OCR approach** that intelligently routes documents to the most appropriate OCR service:

- **AWS Textract** → PDFs, bank statements, invoices, receipts (structured documents)
- **Google Cloud Vision** → Images, handwritten notes, scans, signatures (unstructured/handwritten)

---

## 🎯 Routing Strategy

### Automatic Provider Selection

The system analyzes the **filename** and **file type** to determine the best OCR provider:

```go
// AWS Textract (for structured documents)
✓ File type: PDF
✓ Filename contains: "statement", "bank", "invoice", "receipt"

// Google Cloud Vision (for handwriting & images)
✓ File type: Image (jpg, png, gif, etc.)
✓ Filename contains: "handwritten", "note", "scan", "signature"
```

### Examples

| Filename | File Type | OCR Provider | Reason |
|----------|-----------|--------------|--------|
| `bank_statement_jan2024.pdf` | PDF | **AWS Textract** | Contains "statement" + PDF |
| `invoice_12345.pdf` | PDF | **AWS Textract** | Contains "invoice" + PDF |
| `handwritten_note.jpg` | Image | **Google Vision** | Contains "handwritten" |
| `signature_scan.png` | Image | **Google Vision** | Contains "signature" + "scan" |
| `receipt.pdf` | PDF | **AWS Textract** | Contains "receipt" |
| `document.jpg` | Image | **Google Vision** | Default for images |
| `report.pdf` | PDF | **AWS Textract** | Default for PDFs |

---

## 🚀 Current Status (Mock Mode)

The system is currently running in **mock mode** with realistic sample outputs:

### Mock AWS Textract Output
```
═══════════════════════════════════════════════════════════
              AWS TEXTRACT - MOCK OCR RESULT
═══════════════════════════════════════════════════════════

DOCUMENT TYPE: Bank Statement / Invoice / Receipt
OCR PROVIDER: AWS Textract (Optimized for structured documents)

BANK STATEMENT
Account Number: 1234567890
Statement Period: January 1, 2024 - January 31, 2024

TRANSACTION DETAILS:
Date       | Description     | Debit    | Credit   | Balance
-----------|-----------------|----------|----------|----------
2024-01-05 | Salary Deposit  |          | 5,000.00 | 15,000.00
...

TABLES DETECTED: 1 table with 8 rows and 5 columns
CONFIDENCE SCORE: 98.5%
```

### Mock Google Vision Output
```
═══════════════════════════════════════════════════════════
          GOOGLE CLOUD VISION - MOCK OCR RESULT
═══════════════════════════════════════════════════════════

DOCUMENT TYPE: Image / Handwritten Note / Scan
OCR PROVIDER: Google Cloud Vision (Optimized for handwriting)

HANDWRITTEN NOTE:
Dear Team,
Please review the following action items...

TEXT ANNOTATIONS:
  - Language detected: English (en)
  - Handwriting confidence: 94.2%
  - Signatures: 1 instance
```

---

## 🔧 Production Integration

### Step 1: Install Dependencies

```bash
# AWS Textract
go get github.com/aws/aws-sdk-go/service/textract

# Google Cloud Vision
go get cloud.google.com/go/vision/apiv1
```

### Step 2: Configure Environment Variables

#### For AWS Textract
```bash
# .env file
AWS_ACCESS_KEY_ID=your_access_key_here
AWS_SECRET_ACCESS_KEY=your_secret_key_here
AWS_REGION=us-east-1
AWS_BUCKET=citizenx  # Already configured
```

#### For Google Cloud Vision
```bash
# .env file
GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account-key.json
```

**Get Google Cloud credentials:**
1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select existing
3. Enable Cloud Vision API
4. Create service account key (JSON)
5. Download and save the JSON file
6. Set path in `.env`

### Step 3: Enable Production OCR

In `processors/media_processor.go`, update:

```go
func NewMediaProcessor(uploadRepo db.UploadRepository) *MediaProcessor {
    return &MediaProcessor{
        uploadRepo:      uploadRepo,
        textractEnabled: true,  // ← Change to true
        visionEnabled:   true,  // ← Change to true
    }
}
```

### Step 4: Implement Real OCR Functions

#### AWS Textract Implementation

```go
func (p *MediaProcessor) performTextractOCR(fileURL string) (string, error) {
    // Parse S3 URL to extract bucket and key
    // Example: https://citizenx.s3.eu-north-1.amazonaws.com/uploads/user-123/file-456.pdf
    parts := strings.Split(fileURL, "/")
    bucket := "citizenx"
    key := strings.Join(parts[4:], "/") // uploads/user-123/file-456.pdf
    
    // Create AWS session
    sess := session.Must(session.NewSession(&aws.Config{
        Region: aws.String("eu-north-1"),
    }))
    svc := textract.New(sess)
    
    // Call Textract
    input := &textract.DetectDocumentTextInput{
        Document: &textract.Document{
            S3Object: &textract.S3Object{
                Bucket: aws.String(bucket),
                Name:   aws.String(key),
            },
        },
    }
    
    result, err := svc.DetectDocumentText(input)
    if err != nil {
        return "", fmt.Errorf("textract error: %w", err)
    }
    
    // Extract text from blocks
    var extractedText strings.Builder
    for _, block := range result.Blocks {
        if *block.BlockType == "LINE" {
            extractedText.WriteString(*block.Text + "\n")
        }
    }
    
    return extractedText.String(), nil
}
```

#### Google Cloud Vision Implementation

```go
func (p *MediaProcessor) performVisionOCR(fileURL string) (string, error) {
    ctx := context.Background()
    
    // Create Vision client
    client, err := vision.NewImageAnnotatorClient(ctx)
    if err != nil {
        return "", fmt.Errorf("vision client error: %w", err)
    }
    defer client.Close()
    
    // Create image from URL
    image := vision.NewImageFromURI(fileURL)
    
    // Detect document text (best for handwriting)
    annotation, err := client.DetectDocumentText(ctx, image, nil)
    if err != nil {
        return "", fmt.Errorf("vision detection error: %w", err)
    }
    
    if annotation.Text != "" {
        return annotation.Text, nil
    }
    
    return "", fmt.Errorf("no text found in image")
}
```

---

## 📊 Testing the Hybrid System

### Test 1: Upload PDF (Should use Textract)

**Postman:**
```
POST http://localhost:8080/api/v1/upload
Headers: Authorization: Bearer YOUR_TOKEN
Body (form-data):
  - file: bank_statement.pdf
  - type: pdf
```

**Check logs:**
```
Selected OCR provider: aws_textract for file: bank_statement.pdf
[MOCK AWS TEXTRACT] Processing structured document...
```

### Test 2: Upload Image (Should use Vision)

**Postman:**
```
POST http://localhost:8080/api/v1/upload
Headers: Authorization: Bearer YOUR_TOKEN
Body (form-data):
  - file: handwritten_note.jpg
  - type: image
```

**Check logs:**
```
Selected OCR provider: google_vision for file: handwritten_note.jpg
[MOCK GOOGLE VISION] Processing image/handwritten document...
```

### Test 3: Verify Routing Logic

Upload files with different names to test routing:

| Test File | Expected Provider |
|-----------|-------------------|
| `invoice_2024.pdf` | AWS Textract |
| `receipt_grocery.pdf` | AWS Textract |
| `scan_document.jpg` | Google Vision |
| `signature_page.png` | Google Vision |
| `random_doc.pdf` | AWS Textract (default for PDF) |
| `photo.jpg` | Google Vision (default for image) |

---

## 💰 Cost Analysis

### AWS Textract Pricing
- **DetectDocumentText**: $1.50 per 1,000 pages
- **AnalyzeDocument** (tables/forms): $50-65 per 1,000 pages

**Example:**
- 10,000 bank statements/month = $15/month
- With table extraction = $500-650/month

### Google Cloud Vision Pricing
- **Text Detection**: $1.50 per 1,000 images
- **Document Text Detection**: $1.50 per 1,000 images
- **First 1,000 images/month**: FREE

**Example:**
- 10,000 handwritten notes/month = $13.50/month
- First 1,000 free = $13.50 - $1.50 = $12/month

### Cost Optimization Tips

1. **Use DetectDocumentText** for simple text extraction
2. **Use AnalyzeDocument** only when you need tables/forms/key-value pairs
3. **Cache results** to avoid re-processing same documents
4. **Batch processing** for better throughput
5. **Monitor usage** with AWS CloudWatch and Google Cloud Monitoring

---

## 🔍 Monitoring & Debugging

### Check Which Provider Was Used

```sql
-- Query database to see processing logs
SELECT 
    file_name,
    file_type,
    status,
    created_at,
    processed_at,
    EXTRACT(EPOCH FROM (processed_at - created_at)) as processing_seconds
FROM uploaded_files
ORDER BY created_at DESC
LIMIT 10;
```

### Server Logs

```bash
# Watch logs for OCR provider selection
tail -f server.log | grep "Selected OCR provider"

# Output:
# Selected OCR provider: aws_textract for file: bank_statement.pdf
# Selected OCR provider: google_vision for file: handwritten_note.jpg
```

### Check Extracted Text

```sql
-- View extracted text chunks
SELECT 
    uf.file_name,
    dc.chunk_index,
    LEFT(dc.content, 200) as preview
FROM document_chunks dc
JOIN uploaded_files uf ON dc.file_id = uf.id
WHERE uf.file_name = 'bank_statement.pdf'
ORDER BY dc.chunk_index;
```

---

## 🎯 Use Cases

### Use Case 1: Bank Statement Processing
**File**: `bank_statement_jan2024.pdf`  
**Provider**: AWS Textract  
**Why**: Excellent at extracting tables, account numbers, transaction details  
**Output**: Structured data with high accuracy for amounts and dates

### Use Case 2: Handwritten Meeting Notes
**File**: `meeting_notes_handwritten.jpg`  
**Provider**: Google Cloud Vision  
**Why**: Superior handwriting recognition, works with cursive  
**Output**: Text from handwritten notes with entity detection

### Use Case 3: Invoice Processing
**File**: `invoice_12345.pdf`  
**Provider**: AWS Textract  
**Why**: Can extract key-value pairs (invoice number, total, date)  
**Output**: Structured invoice data ready for accounting system

### Use Case 4: Signature Verification
**File**: `signature_scan.png`  
**Provider**: Google Cloud Vision  
**Why**: Better at detecting and analyzing signatures  
**Output**: Signature detection with confidence scores

---

## 🚨 Error Handling

The system handles errors gracefully:

```go
// If OCR fails, file status is set to "failed"
// Error message is stored in database
// User can retry or check error details

// Example error response:
{
  "status": "failed",
  "error_msg": "AWS Textract error: InvalidS3ObjectException"
}
```

### Common Errors & Solutions

| Error | Cause | Solution |
|-------|-------|----------|
| `AWS Textract not yet implemented` | Production mode not enabled | Set `textractEnabled: true` |
| `Google Cloud Vision not yet implemented` | Production mode not enabled | Set `visionEnabled: true` |
| `InvalidS3ObjectException` | File not in S3 | Check S3 upload succeeded |
| `ProvisionedThroughputExceededException` | Too many requests | Implement rate limiting |
| `InvalidParameterException` | Invalid file format | Check file is valid PDF/image |

---

## 📚 Additional Resources

### AWS Textract Documentation
- [Getting Started](https://docs.aws.amazon.com/textract/latest/dg/getting-started.html)
- [API Reference](https://docs.aws.amazon.com/textract/latest/dg/API_Reference.html)
- [Best Practices](https://docs.aws.amazon.com/textract/latest/dg/best-practices.html)

### Google Cloud Vision Documentation
- [Getting Started](https://cloud.google.com/vision/docs/quickstart)
- [OCR Tutorial](https://cloud.google.com/vision/docs/ocr)
- [Handwriting Detection](https://cloud.google.com/vision/docs/handwriting)

### Code Examples
- [AWS Textract Go Examples](https://github.com/aws/aws-sdk-go/tree/main/service/textract)
- [Google Vision Go Examples](https://github.com/GoogleCloudPlatform/golang-samples/tree/main/vision)

---

## ✅ Summary

**Current State:**
- ✅ Hybrid OCR routing implemented
- ✅ Mock responses for both providers
- ✅ Intelligent file type detection
- ✅ Filename-based routing
- ✅ Error handling and logging
- ✅ Database integration
- ✅ Status tracking

**To Enable Production:**
1. Install AWS Textract SDK
2. Install Google Cloud Vision SDK
3. Configure credentials in `.env`
4. Set `textractEnabled: true` and `visionEnabled: true`
5. Implement `performTextractOCR()` and `performVisionOCR()`
6. Test with real documents
7. Monitor costs and performance

**Benefits:**
- 🎯 Best OCR for each document type
- 💰 Cost-effective (use right tool for right job)
- 🚀 High accuracy for both printed and handwritten text
- 📊 Structured data extraction (tables, forms, key-value pairs)
- 🔄 Automatic routing (no manual selection needed)

---

**Ready to process any document type with optimal OCR!** 🎉
