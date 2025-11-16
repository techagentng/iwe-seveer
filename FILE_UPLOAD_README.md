# File Upload System Documentation

## Overview

This is an enterprise-grade file upload system built with Go and Gin framework that handles CSV, PDF, and Image files with asynchronous processing using goroutines.

## Features

- ✅ **Multi-format Support**: CSV, PDF, and Image files
- ✅ **Asynchronous Processing**: Uses goroutines for non-blocking uploads
- ✅ **S3 Integration**: Uploads media files to AWS S3
- ✅ **CSV Batch Processing**: Streams CSV files and batch inserts to PostgreSQL
- ✅ **OCR Processing**: Placeholder for PDF/Image text extraction
- ✅ **Status Tracking**: Real-time file processing status
- ✅ **Authentication**: Protected endpoints with JWT middleware
- ✅ **Error Handling**: Comprehensive error tracking and logging

## Architecture

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │ POST /api/v1/upload
       ▼
┌─────────────────────┐
│  Upload Handler     │
│  (upload_handler.go)│
└──────┬──────────────┘
       │
       ├─── CSV ────────► CSV Processor ────► Batch Insert to DB
       │                  (csv_processor.go)   (bank_statements)
       │
       └─── PDF/Image ──► S3 Upload ────────► Media Processor ────► OCR ────► Chunks to DB
                                               (media_processor.go)            (document_chunks)
```

## Database Schema

### uploaded_files
```sql
CREATE TABLE uploaded_files (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    file_type VARCHAR(50) NOT NULL,
    file_url TEXT,
    file_size BIGINT NOT NULL,
    status VARCHAR(50) DEFAULT 'pending',
    error_msg TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    processed_at TIMESTAMP
);
```

### bank_statements
```sql
CREATE TABLE bank_statements (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    file_id UUID REFERENCES uploaded_files(id),
    transaction_date DATE NOT NULL,
    description TEXT,
    debit_amount NUMERIC(14,2),
    credit_amount NUMERIC(14,2),
    balance NUMERIC(14,2) NOT NULL,
    currency VARCHAR(10) DEFAULT 'NGN',
    created_at TIMESTAMP DEFAULT NOW()
);
```

### document_chunks
```sql
CREATE TABLE document_chunks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    file_id UUID REFERENCES uploaded_files(id),
    chunk_index INT NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);
```

## API Endpoints

### 1. Upload File
**POST** `/api/v1/upload`

**Headers:**
```
Authorization: Bearer <jwt_token>
Content-Type: multipart/form-data
```

**Body (form-data):**
- `file`: File to upload (required)
- `type`: File type - "csv", "pdf", or "image" (required)

**Response (202 Accepted):**
```json
{
  "message": "File uploaded successfully and is being processed",
  "data": {
    "file_id": "123e4567-e89b-12d3-a456-426614174000",
    "file_name": "bank_statement.csv",
    "file_type": "csv",
    "status": "pending"
  }
}
```

### 2. Get Upload Status
**GET** `/api/v1/upload/status/:id`

**Headers:**
```
Authorization: Bearer <jwt_token>
```

**Response (200 OK):**
```json
{
  "message": "File status retrieved",
  "data": {
    "id": "123e4567-e89b-12d3-a456-426614174000",
    "user_id": "987e6543-e21b-12d3-a456-426614174000",
    "file_name": "bank_statement.csv",
    "file_type": "csv",
    "file_url": "",
    "file_size": 1024000,
    "status": "completed",
    "created_at": "2024-01-15T10:30:00Z",
    "processed_at": "2024-01-15T10:30:15Z"
  }
}
```

### 3. Get User Uploads
**GET** `/api/v1/upload/my-uploads`

**Headers:**
```
Authorization: Bearer <jwt_token>
```

**Response (200 OK):**
```json
{
  "message": "Uploads retrieved successfully",
  "data": [
    {
      "id": "123e4567-e89b-12d3-a456-426614174000",
      "file_name": "bank_statement.csv",
      "file_type": "csv",
      "status": "completed",
      "created_at": "2024-01-15T10:30:00Z"
    }
  ]
}
```

## File Processing Flow

### CSV Files
1. File uploaded via API
2. Record created in `uploaded_files` with status `pending`
3. Goroutine spawned for async processing
4. CSV parsed row-by-row (streaming, not full memory load)
5. Rows batch inserted (100 rows per batch) into `bank_statements`
6. Status updated to `completed` or `failed`

### PDF/Image Files
1. File uploaded via API
2. Record created in `uploaded_files` with status `pending`
3. Goroutine spawned for async processing
4. File uploaded to S3
5. File URL stored in `uploaded_files`
6. OCR performed (mock implementation provided)
7. Text extracted and chunked (1000 chars per chunk)
8. Chunks batch inserted into `document_chunks`
9. Status updated to `completed` or `failed`

## Installation

### 1. Install Dependencies

```bash
# Install AWS SDK for S3 uploads
go get github.com/aws/aws-sdk-go/aws
go get github.com/aws/aws-sdk-go/aws/credentials
go get github.com/aws/aws-sdk-go/aws/session
go get github.com/aws/aws-sdk-go/service/s3

# Tidy up modules
go mod tidy
```

### 2. Environment Variables

Ensure your `.env` file has the following AWS credentials:

```env
AWS_BUCKET=your-bucket-name
AWS_REGION=your-region
AWS_ACCESS_KEY_ID=your-access-key
AWS_SECRET_ACCESS_KEY=your-secret-key
```

### 3. Run Migrations

The migrations will run automatically on server start. The following tables will be created:
- `uploaded_files`
- `bank_statements`
- `document_chunks`

### 4. Start Server

```bash
go run main.go
```

## CSV Format Requirements

Your CSV file should have the following columns (flexible header matching):

**Required Columns:**
- Date/Transaction Date/Trans Date
- Description/Details/Narration
- Balance/Running Balance

**Optional Columns:**
- Debit/Debit Amount/Withdrawal
- Credit/Credit Amount/Deposit
- Currency/CCY

**Example CSV:**
```csv
Date,Description,Debit,Credit,Balance,Currency
2024-01-15,ATM Withdrawal,5000.00,,45000.00,NGN
2024-01-16,Salary Deposit,,150000.00,195000.00,NGN
2024-01-17,Transfer to John,25000.00,,170000.00,NGN
```

## Testing

### Using cURL

**1. Upload CSV File:**
```bash
curl -X POST http://localhost:8080/api/v1/upload \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -F "file=@bank_statement.csv" \
  -F "type=csv"
```

**2. Upload PDF File:**
```bash
curl -X POST http://localhost:8080/api/v1/upload \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -F "file=@document.pdf" \
  -F "type=pdf"
```

**3. Check Upload Status:**
```bash
curl -X GET http://localhost:8080/api/v1/upload/status/FILE_ID \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

**4. Get All Uploads:**
```bash
curl -X GET http://localhost:8080/api/v1/upload/my-uploads \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### Using Postman

1. Create a new POST request to `http://localhost:8080/api/v1/upload`
2. Set Authorization header with Bearer token
3. In Body tab, select "form-data"
4. Add key "file" with type "File" and select your file
5. Add key "type" with value "csv", "pdf", or "image"
6. Send request

## File Structure

```
iwe-server/
├── models/
│   ├── uploaded_file.go      # UploadedFile model
│   ├── bank_statement.go     # BankStatement model
│   └── document_chunk.go     # DocumentChunk model
├── db/
│   └── upload_repository.go  # Database operations
├── storage/
│   └── s3.go                 # S3 upload helper
├── processors/
│   ├── csv_processor.go      # CSV parsing and batch insert
│   └── media_processor.go    # PDF/Image OCR processing
├── server/
│   ├── upload_handler.go     # HTTP handlers
│   ├── router.go             # Route definitions
│   └── server.go             # Server struct
└── main.go                   # Application entry point
```

## Configuration

### File Size Limits
- Maximum file size: **50 MB**
- Configurable in `upload_handler.go`: `MaxFileSize = 50 << 20`

### Batch Sizes
- CSV batch insert: **100 rows** per batch
- Document chunks batch: **50 chunks** per batch
- Chunk size: **1000 characters** per chunk

### Supported File Extensions
- **CSV**: `.csv`
- **PDF**: `.pdf`
- **Images**: `.jpg`, `.jpeg`, `.png`, `.gif`, `.bmp`, `.webp`

## OCR Integration

The current implementation uses mock OCR. To integrate real OCR services:

### Option 1: Google Cloud Vision API
```go
import "cloud.google.com/go/vision/apiv1"

client, err := vision.NewImageAnnotatorClient(ctx)
text, err := client.DetectDocumentText(ctx, image, nil)
```

### Option 2: AWS Textract
```go
import "github.com/aws/aws-sdk-go/service/textract"

svc := textract.New(session)
result, err := svc.DetectDocumentText(input)
```

### Option 3: Tesseract (Local)
```go
import "github.com/otiai10/gosseract/v2"

client := gosseract.NewClient()
client.SetImage("image.png")
text, err := client.Text()
```

## Error Handling

All errors are logged and tracked:
- File processing errors are stored in `uploaded_files.error_msg`
- Status is updated to `failed` on errors
- Detailed logs available in server output

## Performance Considerations

1. **Streaming CSV**: Files are read row-by-row, not loaded into memory
2. **Batch Inserts**: Database writes are batched for efficiency
3. **Goroutines**: Processing happens asynchronously
4. **Connection Pooling**: GORM handles database connection pooling

## Security

- ✅ JWT authentication required for all endpoints
- ✅ File type validation
- ✅ File extension validation
- ✅ File size limits
- ✅ User isolation (users can only access their own files)

## Monitoring

Monitor file processing status:
```sql
-- Check processing status
SELECT status, COUNT(*) 
FROM uploaded_files 
GROUP BY status;

-- Check failed uploads
SELECT id, file_name, error_msg, created_at 
FROM uploaded_files 
WHERE status = 'failed' 
ORDER BY created_at DESC;

-- Check processing time
SELECT 
  file_type,
  AVG(EXTRACT(EPOCH FROM (processed_at - created_at))) as avg_processing_seconds
FROM uploaded_files 
WHERE status = 'completed'
GROUP BY file_type;
```

## Troubleshooting

### Issue: AWS SDK Import Errors
**Solution:** Run `go get github.com/aws/aws-sdk-go` and `go mod tidy`

### Issue: File Processing Stuck in "pending"
**Solution:** Check server logs for goroutine errors. Ensure database connection is stable.

### Issue: S3 Upload Fails
**Solution:** Verify AWS credentials in `.env` file. Check bucket permissions and region.

### Issue: CSV Parsing Errors
**Solution:** Ensure CSV has required columns. Check date format compatibility.

## Future Enhancements

- [ ] Real OCR integration (Google Vision, AWS Textract)
- [ ] Webhook notifications on processing completion
- [ ] File preview/download endpoints
- [ ] Advanced CSV column mapping
- [ ] File compression before S3 upload
- [ ] Retry mechanism for failed uploads
- [ ] Rate limiting per user
- [ ] File virus scanning
- [ ] Multi-region S3 support

## Support

For issues or questions, check the server logs or contact the development team.
