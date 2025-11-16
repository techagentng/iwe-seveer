# Quick Start Guide - File Upload System

## 🚀 Get Started in 3 Steps

### Step 1: Install Dependencies & Start Server
```bash
cd /Users/nnahnnamdi/Desktop/iwe-server

# Dependencies are already installed, just run:
go run main.go
```

Server will start on `http://localhost:8080`

### Step 2: Get Authentication Token
```bash
# Option A: Create new account
curl -X POST http://localhost:8080/api/v1/auth/signup \
  -H 'Content-Type: application/json' \
  -d '{
    "email": "test@example.com",
    "password": "password123",
    "fullname": "Test User",
    "username": "testuser"
  }'

# Option B: Login with existing account
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{
    "email": "test@example.com",
    "password": "password123"
  }'
```

Copy the `access_token` from the response.

### Step 3: Upload a File
```bash
# Using the test script (easiest)
./test_upload.sh YOUR_ACCESS_TOKEN

# Or manually upload the sample CSV
curl -X POST http://localhost:8080/api/v1/upload \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -F "file=@sample_bank_statement.csv" \
  -F "type=csv"
```

## 📋 Available Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/upload` | Upload CSV/PDF/Image file |
| GET | `/api/v1/upload/status/:id` | Check file processing status |
| GET | `/api/v1/upload/my-uploads` | List all your uploads |

## 📁 Test Files Included

- **`sample_bank_statement.csv`** - Sample CSV with 10 transactions
- **`test_upload.sh`** - Automated test script

## 🔍 Verify Upload in Database

```bash
# Connect to PostgreSQL
psql -U postgres -d iwe_dev

# Check uploaded files
SELECT id, file_name, file_type, status, created_at FROM uploaded_files;

# Check bank statements (for CSV uploads)
SELECT * FROM bank_statements ORDER BY transaction_date DESC LIMIT 10;

# Check processing status
SELECT status, COUNT(*) FROM uploaded_files GROUP BY status;
```

## 📊 Example Response

### Upload Response (202 Accepted)
```json
{
  "message": "File uploaded successfully and is being processed",
  "data": {
    "file_id": "123e4567-e89b-12d3-a456-426614174000",
    "file_name": "sample_bank_statement.csv",
    "file_type": "csv",
    "status": "pending"
  }
}
```

### Status Check Response (200 OK)
```json
{
  "message": "File status retrieved",
  "data": {
    "id": "123e4567-e89b-12d3-a456-426614174000",
    "user_id": "987e6543-e21b-12d3-a456-426614174000",
    "file_name": "sample_bank_statement.csv",
    "file_type": "csv",
    "file_size": 1024,
    "status": "completed",
    "created_at": "2024-01-15T10:30:00Z",
    "processed_at": "2024-01-15T10:30:05Z"
  }
}
```

## 🎯 Supported File Types

### CSV Files
- **Extension**: `.csv`
- **Processing**: Parsed and inserted into `bank_statements` table
- **Batch Size**: 100 rows per insert
- **Required Columns**: Date, Description, Balance

### PDF Files
- **Extension**: `.pdf`
- **Processing**: Uploaded to S3, OCR performed, text chunked
- **Storage**: `document_chunks` table
- **OCR**: Mock implementation (ready for real integration)

### Image Files
- **Extensions**: `.jpg`, `.jpeg`, `.png`, `.gif`, `.bmp`, `.webp`
- **Processing**: Same as PDF
- **Max Size**: 50MB

## 🔧 Configuration

All configuration is in `.env` file:

```env
# AWS S3 Configuration (already set)
AWS_BUCKET=your-bucket-name
AWS_REGION=eu-north-1
AWS_ACCESS_KEY_ID=YOUR_AWS_ACCESS_KEY_ID
AWS_SECRET_ACCESS_KEY=YOUR_AWS_SECRET_ACCESS_KEY

# Database (already set)
POSTGRES_HOST=localhost
POSTGRES_USER=postgres
POSTGRES_DB=iwe_dev
POSTGRES_PASSWORD=ken
POSTGRES_PORT=5432
```

## 📚 Documentation

- **`FILE_UPLOAD_README.md`** - Complete API documentation
- **`IMPLEMENTATION_SUMMARY.md`** - Technical implementation details
- **`QUICK_START.md`** - This file

## 🐛 Troubleshooting

### Server won't start
```bash
# Check if port 8080 is in use
lsof -i :8080

# Check database connection
psql -U postgres -d iwe_dev -c "SELECT 1"
```

### Upload fails
- Verify JWT token is valid (not expired)
- Check file size is under 50MB
- Ensure file extension matches type parameter
- Check server logs for detailed errors

### Processing stuck in "pending"
- Check server logs for goroutine errors
- Verify database connection is stable
- Check if migrations ran successfully

## 💡 Tips

1. **Use the test script** - It's the easiest way to test: `./test_upload.sh TOKEN`
2. **Check logs** - Server logs show detailed processing information
3. **Monitor database** - Use the SQL queries above to verify data
4. **File size** - Keep files under 50MB for best performance
5. **CSV format** - Use the sample CSV as a template

## 🎉 What's Working

✅ File upload with authentication  
✅ CSV parsing and batch insert  
✅ Async processing with goroutines  
✅ Status tracking  
✅ Error handling  
✅ S3 integration (ready to use)  
✅ OCR placeholder (ready for integration)  

## 📞 Need Help?

Check the comprehensive documentation:
- API details: `FILE_UPLOAD_README.md`
- Implementation: `IMPLEMENTATION_SUMMARY.md`
- Server logs: Watch terminal output when running server

---

**Ready to go!** Just run `go run main.go` and start uploading files! 🚀
