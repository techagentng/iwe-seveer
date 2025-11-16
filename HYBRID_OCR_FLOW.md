# 🔄 Hybrid OCR Flow Diagram

## System Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         CLIENT UPLOADS FILE                          │
│                    (PDF, Image, Bank Statement)                      │
└────────────────────────────┬────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      UPLOAD HANDLER                                  │
│  • Validate file type & size                                        │
│  • Create DB record (status: pending)                               │
│  • Return 202 Accepted                                              │
│  • Spawn goroutine for async processing                             │
└────────────────────────────┬────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    MEDIA PROCESSOR                                   │
│                  (Hybrid OCR Router)                                │
│                                                                      │
│  ┌────────────────────────────────────────────────────────────┐    │
│  │  INTELLIGENT ROUTING LOGIC                                 │    │
│  │  ─────────────────────────────────────────────────────────│    │
│  │                                                            │    │
│  │  Analyze:                                                  │    │
│  │    • File type (PDF vs Image)                             │    │
│  │    • Filename keywords                                     │    │
│  │                                                            │    │
│  │  Keywords for AWS Textract:                               │    │
│  │    ✓ "statement", "bank", "invoice", "receipt"           │    │
│  │    ✓ File type: PDF                                       │    │
│  │                                                            │    │
│  │  Keywords for Google Vision:                              │    │
│  │    ✓ "handwritten", "note", "scan", "signature"          │    │
│  │    ✓ File type: Image (jpg, png, etc.)                   │    │
│  └────────────────────────────────────────────────────────────┘    │
└──────────────┬─────────────────────────────┬────────────────────────┘
               │                             │
               │                             │
      PDF / Structured                  Image / Handwritten
      Documents                         Documents
               │                             │
               ▼                             ▼
┌──────────────────────────┐    ┌──────────────────────────────┐
│   AWS TEXTRACT           │    │   GOOGLE CLOUD VISION        │
│   ─────────────          │    │   ────────────────────       │
│                          │    │                              │
│  Best for:               │    │  Best for:                   │
│  ✓ Bank statements       │    │  ✓ Handwritten notes         │
│  ✓ Invoices              │    │  ✓ Scanned images            │
│  ✓ Receipts              │    │  ✓ Photos of documents       │
│  ✓ Forms with tables     │    │  ✓ Signatures                │
│  ✓ Multi-column PDFs     │    │  ✓ Cursive writing           │
│  ✓ Printed documents     │    │  ✓ Rotated/skewed images     │
│                          │    │                              │
│  Features:               │    │  Features:                   │
│  • Table extraction      │    │  • Handwriting recognition   │
│  • Key-value pairs       │    │  • Multi-language support    │
│  • Form detection        │    │  • Entity detection          │
│  • High accuracy         │    │  • Signature detection       │
│                          │    │                              │
│  Cost:                   │    │  Cost:                       │
│  $1.50 / 1,000 pages     │    │  $1.50 / 1,000 images        │
│                          │    │  (First 1,000 FREE)          │
└────────────┬─────────────┘    └────────────┬─────────────────┘
             │                               │
             │  Extracted Text               │  Extracted Text
             │                               │
             └───────────────┬───────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    TEXT PROCESSING                                   │
│                                                                      │
│  1. Chunk text (1000 characters per chunk)                          │
│  2. Create document chunks with index                               │
│  3. Batch insert to database (50 chunks/batch)                      │
│  4. Update file status to "completed"                               │
│  5. Set processed_at timestamp                                      │
└────────────────────────────┬────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    POSTGRESQL DATABASE                               │
│                                                                      │
│  ┌──────────────────┐        ┌──────────────────────────────┐      │
│  │ uploaded_files   │        │ document_chunks              │      │
│  │ ────────────────│        │ ────────────────────────────│      │
│  │ • id             │◄───────│ • id                         │      │
│  │ • user_id        │        │ • file_id (FK)               │      │
│  │ • file_name      │        │ • chunk_index                │      │
│  │ • file_type      │        │ • content (extracted text)   │      │
│  │ • file_url       │        │ • created_at                 │      │
│  │ • status         │        └──────────────────────────────┘      │
│  │ • created_at     │                                               │
│  │ • processed_at   │                                               │
│  └──────────────────┘                                               │
└─────────────────────────────────────────────────────────────────────┘
```

## Decision Flow

```
                    ┌─────────────────┐
                    │  File Uploaded  │
                    └────────┬────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │ Check File Type │
                    └────────┬────────┘
                             │
                ┌────────────┴────────────┐
                │                         │
                ▼                         ▼
         ┌──────────┐              ┌──────────┐
         │   PDF?   │              │  Image?  │
         └─────┬────┘              └─────┬────┘
               │                         │
               ▼                         ▼
        ┌─────────────┐          ┌─────────────┐
        │ Check Name  │          │ Check Name  │
        └──────┬──────┘          └──────┬──────┘
               │                        │
    ┌──────────┴──────────┐  ┌─────────┴──────────┐
    │                     │  │                    │
    ▼                     ▼  ▼                    ▼
Contains:           Default  Contains:        Default
"statement"         for PDF  "handwritten"    for Image
"bank"                      "note"
"invoice"                   "scan"
"receipt"                   "signature"
    │                     │  │                    │
    └──────────┬──────────┘  └─────────┬──────────┘
               │                       │
               ▼                       ▼
        ┌─────────────┐        ┌─────────────┐
        │AWS TEXTRACT │        │GOOGLE VISION│
        └─────────────┘        └─────────────┘
```

## Example Routing

```
┌──────────────────────────────┬──────────┬─────────────────┬──────────────────┐
│ Filename                     │ Type     │ OCR Provider    │ Reason           │
├──────────────────────────────┼──────────┼─────────────────┼──────────────────┤
│ bank_statement_jan2024.pdf   │ PDF      │ AWS Textract    │ Contains "bank"  │
│                              │          │                 │ + "statement"    │
├──────────────────────────────┼──────────┼─────────────────┼──────────────────┤
│ invoice_12345.pdf            │ PDF      │ AWS Textract    │ Contains         │
│                              │          │                 │ "invoice"        │
├──────────────────────────────┼──────────┼─────────────────┼──────────────────┤
│ receipt_grocery.pdf          │ PDF      │ AWS Textract    │ Contains         │
│                              │          │                 │ "receipt"        │
├──────────────────────────────┼──────────┼─────────────────┼──────────────────┤
│ handwritten_note.jpg         │ Image    │ Google Vision   │ Contains         │
│                              │          │                 │ "handwritten"    │
├──────────────────────────────┼──────────┼─────────────────┼──────────────────┤
│ signature_scan.png           │ Image    │ Google Vision   │ Contains         │
│                              │          │                 │ "signature"      │
├──────────────────────────────┼──────────┼─────────────────┼──────────────────┤
│ document_scan.jpg            │ Image    │ Google Vision   │ Contains "scan"  │
├──────────────────────────────┼──────────┼─────────────────┼──────────────────┤
│ meeting_notes.png            │ Image    │ Google Vision   │ Contains "note"  │
├──────────────────────────────┼──────────┼─────────────────┼──────────────────┤
│ random_document.pdf          │ PDF      │ AWS Textract    │ Default for PDF  │
├──────────────────────────────┼──────────┼─────────────────┼──────────────────┤
│ photo.jpg                    │ Image    │ Google Vision   │ Default for      │
│                              │          │                 │ Image            │
└──────────────────────────────┴──────────┴─────────────────┴──────────────────┘
```

## Processing Timeline

```
Time    Event
─────   ──────────────────────────────────────────────────────────────
0:00    Client uploads file
        ↓
0:01    Handler validates & creates DB record
        Status: "pending"
        ↓
0:01    HTTP 202 Accepted returned to client
        ↓
0:01    Goroutine spawned for async processing
        ↓
0:02    Status updated to "processing"
        ↓
0:02    OCR provider selected based on filename/type
        ↓
0:03    File uploaded to S3 (if not already)
        ↓
0:04    OCR service called (Textract or Vision)
        ↓
0:05    Text extracted from document
        ↓
0:06    Text chunked into 1000-character segments
        ↓
0:07    Chunks batch inserted to database
        ↓
0:08    Status updated to "completed"
        ProcessedAt timestamp set
        ↓
0:08    Client polls status endpoint
        Receives "completed" status
```

## Mock vs Production Mode

```
┌─────────────────────────────────────────────────────────────────────┐
│                        CURRENT: MOCK MODE                            │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  textractEnabled: false  ──►  Uses mockTextractOCR()                │
│  visionEnabled: false    ──►  Uses mockVisionOCR()                  │
│                                                                      │
│  Benefits:                                                           │
│  ✓ Test system without API costs                                   │
│  ✓ Realistic sample outputs                                        │
│  ✓ Fast processing (no external API calls)                         │
│  ✓ No credentials needed                                           │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│                      PRODUCTION MODE (Future)                        │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  textractEnabled: true   ──►  Uses performTextractOCR()            │
│  visionEnabled: true     ──►  Uses performVisionOCR()              │
│                                                                      │
│  Requirements:                                                       │
│  • AWS credentials configured                                       │
│  • Google Cloud service account key                                 │
│  • API access enabled                                               │
│  • Billing account set up                                           │
│                                                                      │
│  Benefits:                                                           │
│  ✓ Real text extraction                                            │
│  ✓ High accuracy OCR                                               │
│  ✓ Production-ready                                                │
│  ✓ Handles all document types                                      │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

## Cost Comparison

```
Scenario: 10,000 documents/month

┌─────────────────────────────────────────────────────────────────────┐
│                    HYBRID APPROACH (Optimized)                       │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  6,000 PDFs (bank statements, invoices)                             │
│    → AWS Textract: 6,000 × $1.50/1000 = $9.00                      │
│                                                                      │
│  4,000 Images (handwritten notes, scans)                            │
│    → Google Vision: 4,000 × $1.50/1000 = $6.00                     │
│    → First 1,000 FREE = $6.00 - $1.50 = $4.50                      │
│                                                                      │
│  TOTAL: $9.00 + $4.50 = $13.50/month                               │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│                  SINGLE PROVIDER (Not Optimized)                     │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Option A: Only AWS Textract                                        │
│    10,000 documents × $1.50/1000 = $15.00/month                    │
│    ✗ Poor handwriting recognition                                  │
│                                                                      │
│  Option B: Only Google Vision                                       │
│    10,000 documents × $1.50/1000 = $15.00/month                    │
│    (First 1,000 FREE = $13.50/month)                               │
│    ✗ Less accurate for structured PDFs                             │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘

SAVINGS: $1.50 - $13.50 = Up to $1.50/month
BENEFIT: Better accuracy for each document type
```

## Summary

✅ **Intelligent Routing**: Automatic provider selection based on file type and name  
✅ **Cost Optimized**: Use the right tool for the right job  
✅ **High Accuracy**: Best OCR for each document type  
✅ **Mock Mode**: Test without API costs  
✅ **Production Ready**: Easy to enable real OCR  
✅ **Scalable**: Handles any document volume  
✅ **Error Handling**: Graceful failures with detailed logging  

**The hybrid approach gives you the best of both worlds!** 🎉
