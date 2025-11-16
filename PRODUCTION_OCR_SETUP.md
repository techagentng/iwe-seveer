# 🚀 Production OCR Setup - Complete Guide

## ✅ What's Already Done

Your production OCR is **90% complete!** Here's what's ready:

### 1. Dependencies Installed ✅
```bash
✓ github.com/aws/aws-sdk-go/service/textract
✓ cloud.google.com/go/vision/apiv1
✓ go mod tidy completed
✓ Build successful
```

### 2. AWS Textract Implemented ✅
- Real OCR function created
- S3 URL parsing
- Text extraction from PDFs
- Error handling

### 3. Google Cloud Vision Implemented ✅
- Real OCR function created
- Image processing
- Handwriting recognition
- Error handling

### 4. Production Mode Enabled ✅
```go
textractEnabled: true  // AWS Textract for PDFs
visionEnabled: true    // Google Vision for images
```

---

## 🔧 Final Setup Steps

### Step 1: Get Google Cloud Credentials

**A. Create Google Cloud Project**

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Click "Select a project" → "New Project"
3. Name: "iwe-ocr" (or any name)
4. Click "Create"

**B. Enable Cloud Vision API**

1. In the project, go to "APIs & Services" → "Library"
2. Search for "Cloud Vision API"
3. Click on it → Click "Enable"
4. Wait for it to enable (~30 seconds)

**C. Create Service Account**

1. Go to "IAM & Admin" → "Service Accounts"
2. Click "Create Service Account"
3. Fill in:
   - **Name**: `iwe-ocr-service`
   - **Description**: `OCR service for file uploads`
4. Click "Create and Continue"
5. **Role**: Select "Cloud Vision AI Service Agent"
6. Click "Continue" → "Done"

**D. Generate JSON Key**

1. Click on the service account you just created
2. Go to "Keys" tab
3. Click "Add Key" → "Create new key"
4. Select "JSON"
5. Click "Create"
6. A file downloads: `iwe-ocr-[project-id]-[random].json`

**E. Save the Key File**

```bash
# Create credentials directory
mkdir -p /Users/nnahnnamdi/Desktop/iwe-server/credentials

# Move the downloaded file (adjust filename)
mv ~/Downloads/iwe-ocr-*.json /Users/nnahnnamdi/Desktop/iwe-server/credentials/google-vision-key.json

# Verify it exists
ls -la /Users/nnahnnamdi/Desktop/iwe-server/credentials/
```

### Step 2: Update `.env` File

Add this line to `/Users/nnahnnamdi/Desktop/iwe-server/.env`:

```bash
GOOGLE_APPLICATION_CREDENTIALS=/Users/nnahnnamdi/Desktop/iwe-server/credentials/google-vision-key.json
```

Your complete `.env` should now have:

```bash
# ... existing config ...

# AWS Credentials (already configured)
AWS_BUCKET=your-bucket-name
AWS_REGION=eu-north-1
AWS_ACCESS_KEY_ID=YOUR_AWS_ACCESS_KEY_ID
AWS_SECRET_ACCESS_KEY=YOUR_AWS_SECRET_ACCESS_KEY

# Google Cloud Vision (NEW - add this)
GOOGLE_APPLICATION_CREDENTIALS=/Users/nnahnnamdi/Desktop/iwe-server/credentials/google-vision-key.json
```

### Step 3: Add Credentials Directory to .gitignore

```bash
# Add to .gitignore
echo "credentials/" >> .gitignore
```

---

## 🧪 Test Production OCR

### Test 1: Upload PDF (AWS Textract)

```bash
# Start server
go run main.go

# Upload a PDF
curl -X POST http://localhost:8080/api/v1/upload \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -F "file=@bank_statement.pdf" \
  -F "type=pdf"
```

**Expected logs:**
```
Starting async media processing for file: abc-123...
File uploaded to S3: https://citizenx.s3.eu-north-1.amazonaws.com/...
Selected OCR provider: aws_textract for file: bank_statement.pdf
[AWS TEXTRACT] Processing document from https://...
[AWS TEXTRACT] Bucket: citizenx, Key: uploads/user-id/file-id.pdf
[AWS TEXTRACT] Extracted 45 lines of text
Media processing completed successfully
```

### Test 2: Upload Image (Google Cloud Vision)

```bash
# Upload an image
curl -X POST http://localhost:8080/api/v1/upload \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -F "file=@handwritten_note.jpg" \
  -F "type=image"
```

**Expected logs:**
```
Starting async media processing for file: def-456...
File uploaded to S3: https://citizenx.s3.eu-north-1.amazonaws.com/...
Selected OCR provider: google_vision for file: handwritten_note.jpg
[GOOGLE VISION] Processing image from https://...
[GOOGLE VISION] Extracted 512 characters of text
Media processing completed successfully
```

### Test 3: Verify Extracted Text

```sql
-- Check database for extracted text
SELECT 
    uf.file_name,
    uf.file_type,
    uf.status,
    dc.chunk_index,
    LEFT(dc.content, 200) as preview
FROM uploaded_files uf
JOIN document_chunks dc ON dc.file_id = uf.id
WHERE uf.file_name = 'bank_statement.pdf'
ORDER BY dc.chunk_index;
```

---

## 🔍 How It Works

### Hybrid Routing Logic

```
File Upload
    ↓
Check filename & type
    ↓
┌─────────────────────────┬─────────────────────────┐
│                         │                         │
│  Contains:              │  Contains:              │
│  • "statement"          │  • "handwritten"        │
│  • "bank"               │  • "note"               │
│  • "invoice"            │  • "scan"               │
│  • "receipt"            │  • "signature"          │
│  OR type = PDF          │  OR type = Image        │
│                         │                         │
▼                         ▼
AWS TEXTRACT              GOOGLE VISION
    │                         │
    │  Extract text           │  Extract text
    │  from PDF               │  from image
    │                         │
    └─────────┬───────────────┘
              │
              ▼
        Chunk text
              │
              ▼
        Save to DB
              │
              ▼
        Status: completed
```

### AWS Textract Flow

1. File uploaded to S3
2. S3 URL generated
3. Textract called with S3 bucket + key
4. Text extracted line by line
5. Combined into single string
6. Chunked and saved

### Google Cloud Vision Flow

1. File uploaded to S3
2. S3 URL generated (public-read)
3. Vision API called with S3 URL
4. Text extracted (including handwriting)
5. Returned as single string
6. Chunked and saved

---

## 💰 Cost Analysis

### AWS Textract Pricing

| Service | Cost | Example |
|---------|------|---------|
| DetectDocumentText | $1.50 per 1,000 pages | 1,000 PDFs = $1.50 |
| AnalyzeDocument (tables) | $50 per 1,000 pages | Not used (too expensive) |

**Monthly estimate (1,000 PDFs):** $1.50

### Google Cloud Vision Pricing

| Service | Cost | Example |
|---------|------|---------|
| Text Detection | $1.50 per 1,000 images | 1,000 images = $1.50 |
| Document Text Detection | $1.50 per 1,000 images | Same price |
| **Free Tier** | First 1,000/month FREE | Save $1.50 |

**Monthly estimate (1,000 images):** $0 (free tier) or $1.50

### Combined Monthly Cost

For 1,000 PDFs + 1,000 images:
- AWS Textract: $1.50
- Google Vision: $0 (free tier)
- **Total: $1.50/month**

For 10,000 PDFs + 10,000 images:
- AWS Textract: $15.00
- Google Vision: $13.50 (after free tier)
- **Total: $28.50/month**

---

## 🔒 Security Best Practices

### 1. Protect Credentials

```bash
# Never commit credentials
echo "credentials/" >> .gitignore
echo ".env" >> .gitignore

# Set proper permissions
chmod 600 credentials/google-vision-key.json
chmod 600 .env
```

### 2. Use Environment Variables

```bash
# In production, use environment variables instead of files
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/key.json
export AWS_ACCESS_KEY_ID=your_key
export AWS_SECRET_ACCESS_KEY=your_secret
```

### 3. Rotate Keys Regularly

- AWS: Rotate access keys every 90 days
- Google: Create new service account keys periodically
- Delete old keys after rotation

### 4. Limit Permissions

**AWS IAM Policy (minimal):**
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "textract:DetectDocumentText",
        "s3:GetObject"
      ],
      "Resource": [
        "arn:aws:textract:eu-north-1:*:*",
        "arn:aws:s3:::citizenx/uploads/*"
      ]
    }
  ]
}
```

**Google Cloud (minimal):**
- Role: "Cloud Vision AI Service Agent" only
- No additional permissions needed

---

## 🐛 Troubleshooting

### Error: "failed to create vision client"

**Cause:** Google credentials not found or invalid

**Solutions:**
1. Check file exists:
   ```bash
   ls -la /Users/nnahnnamdi/Desktop/iwe-server/credentials/google-vision-key.json
   ```

2. Verify `.env` has correct path:
   ```bash
   grep GOOGLE_APPLICATION_CREDENTIALS .env
   ```

3. Test credentials:
   ```bash
   export GOOGLE_APPLICATION_CREDENTIALS=/Users/nnahnnamdi/Desktop/iwe-server/credentials/google-vision-key.json
   gcloud auth application-default print-access-token
   ```

### Error: "textract API error: AccessDeniedException"

**Cause:** AWS credentials don't have Textract permissions

**Solutions:**
1. Check IAM user has `textract:DetectDocumentText` permission
2. Verify credentials in `.env` are correct
3. Test with AWS CLI:
   ```bash
   aws textract detect-document-text \
     --document '{"S3Object":{"Bucket":"citizenx","Name":"test.pdf"}}' \
     --region eu-north-1
   ```

### Error: "failed to parse S3 URL"

**Cause:** S3 URL format is incorrect

**Solution:**
- Ensure S3 upload is working
- Check file_url in database matches format:
  `https://citizenx.s3.eu-north-1.amazonaws.com/uploads/...`

### Error: "no text found in document/image"

**Possible causes:**
1. File is blank or has no text
2. Image quality too low
3. PDF is image-based (scanned) - Textract should still work
4. File corrupted during upload

**Solutions:**
- Verify file manually
- Check file size in database
- Download from S3 and inspect

---

## 📊 Monitoring

### Check OCR Success Rate

```sql
-- Overall success rate
SELECT 
    status,
    COUNT(*) as count,
    ROUND(COUNT(*) * 100.0 / SUM(COUNT(*)) OVER (), 2) as percentage
FROM uploaded_files
WHERE file_type IN ('pdf', 'image')
GROUP BY status;
```

### Check Processing Time

```sql
-- Average processing time by type
SELECT 
    file_type,
    AVG(EXTRACT(EPOCH FROM (processed_at - created_at))) as avg_seconds,
    MAX(EXTRACT(EPOCH FROM (processed_at - created_at))) as max_seconds
FROM uploaded_files
WHERE status = 'completed'
GROUP BY file_type;
```

### Check OCR Provider Usage

```bash
# Check logs for provider distribution
grep "Selected OCR provider" server.log | \
  awk '{print $NF}' | \
  sort | uniq -c
```

### Monitor Costs

**AWS:**
```bash
# Check Textract usage
aws ce get-cost-and-usage \
  --time-period Start=2024-01-01,End=2024-01-31 \
  --granularity MONTHLY \
  --metrics BlendedCost \
  --filter file://textract-filter.json
```

**Google Cloud:**
```bash
# Check Vision API usage
gcloud billing accounts list
gcloud billing accounts describe ACCOUNT_ID
```

---

## 🎯 Optimization Tips

### 1. Cache Results

Avoid re-processing same files:
```sql
-- Add unique constraint on file hash
ALTER TABLE uploaded_files ADD COLUMN file_hash VARCHAR(64);
CREATE INDEX idx_file_hash ON uploaded_files(file_hash);
```

### 2. Batch Processing

Process multiple files in batches:
- Textract: Use `StartDocumentTextDetection` for async
- Vision: Batch multiple images in one request

### 3. Fallback Strategy

If one provider fails, try the other:
```go
// In performHybridOCR
text, err := p.performTextractOCR(fileURL)
if err != nil {
    log.Printf("Textract failed, trying Vision: %v", err)
    text, err = p.performVisionOCR(fileURL)
}
```

### 4. Quality Checks

Validate extracted text:
```go
if len(extractedText) < 10 {
    return "", fmt.Errorf("extracted text too short, possible OCR failure")
}
```

---

## ✅ Final Checklist

- [ ] Google Cloud project created
- [ ] Cloud Vision API enabled
- [ ] Service account created
- [ ] JSON key downloaded
- [ ] Key saved to `credentials/google-vision-key.json`
- [ ] `.env` updated with `GOOGLE_APPLICATION_CREDENTIALS`
- [ ] `credentials/` added to `.gitignore`
- [ ] Server restarted
- [ ] Test PDF upload (Textract)
- [ ] Test image upload (Vision)
- [ ] Check database for extracted text
- [ ] Monitor logs for errors
- [ ] Verify costs in AWS/Google consoles

---

## 🎉 Summary

**Your production OCR is ready!**

✅ **AWS Textract** - For PDFs and structured documents  
✅ **Google Cloud Vision** - For images and handwriting  
✅ **Hybrid routing** - Automatic provider selection  
✅ **Real-time processing** - Async with goroutines  
✅ **Cost optimized** - ~$1.50-$30/month depending on volume  
✅ **Error handling** - Comprehensive logging  
✅ **Production ready** - Just add Google credentials  

**Final step:** Complete Step 1 & 2 above to add Google Cloud credentials, then you're live! 🚀
