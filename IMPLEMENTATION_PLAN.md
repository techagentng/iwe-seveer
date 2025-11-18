# 🚀 Job Queue + WebSocket + AI - Implementation Plan

## Overview
Add Redis job queue, WebSocket real-time updates, and OpenAI analysis to your current file upload system.

---

## 📦 Phase 1: Setup (2 hours)

### Install Dependencies
```bash
go get github.com/redis/go-redis/v9
go get github.com/gorilla/websocket
go get github.com/sashabaranov/go-openai
go mod tidy
```

### Install Redis
```bash
brew install redis
brew services start redis
```

### Update .env
```bash
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
OPENAI_API_KEY=sk-your-key
OPENAI_MODEL=gpt-4o-mini
```

---

## 📁 Phase 2: New Structure (4 hours)

### Create Directories
```
queue/      - Job queue management
websocket/  - WebSocket handlers  
ai/         - OpenAI integration
```

### New Model: `models/job.go`
```go
type ProcessingJob struct {
    ID            uuid.UUID
    UserID        uuid.UUID
    FileID        uuid.UUID
    Prompt        string    // User's question
    Status        JobStatus // queued/processing/completed/failed
    Progress      int       // 0-100
    ExtractedText string
    AIResponse    string
    CreatedAt     time.Time
}
```

---

## 🔧 Phase 3: Core Components (8 hours)

### 1. Queue Manager (`queue/queue.go`)
- `EnqueueJob()` - Add job to Redis queue
- `DequeueJob()` - Get next job (blocking)
- `UpdateJobStatus()` - Update progress
- `PublishJobUpdate()` - Notify via Redis pub/sub

### 2. WebSocket Hub (`websocket/hub.go`)
- Manage connected clients by user ID
- Broadcast messages to specific users
- Handle connect/disconnect

### 3. OpenAI Client (`ai/openai.go`)
- `AnalyzeDocument()` - Send to GPT-4o-mini
- `StreamAnalyzeDocument()` - Stream response chunks

### 4. Worker (`queue/worker.go`)
```go
func (w *Worker) processJob(job *ProcessingJob) {
    // 1. Extract text (OCR) - 20-60%
    text := w.extractText(job)
    
    // 2. AI Analysis - 60-90%
    response := w.analyzeWithAI(text, job.Prompt)
    
    // 3. Complete - 100%
    w.completeJob(job, response)
}
```

---

## 🔌 Phase 4: API Integration (4 hours)

### New Endpoints

**POST /api/v1/analyze**
```go
// Upload file + prompt → immediate 202 response
{
    "job_id": "uuid",
    "status": "queued",
    "message": "Analyzing your document..."
}
```

**GET /api/v1/jobs/:job_id**
```go
// Get job status and results
{
    "status": "completed",
    "progress": 100,
    "ai_response": "..."
}
```

**GET /ws?user_id=uuid**
```
WebSocket connection for real-time updates
```

---

## 🎯 Phase 5: Flow Integration (4 hours)

### Current Flow (Keep)
```
POST /upload → Save file → Async OCR → Store chunks
```

### New Flow (Add)
```
POST /analyze → Save file → Queue job → 202 Response
    ↓
Worker picks job → OCR (progress 20-60%)
    ↓
AI Analysis (progress 60-90%, stream chunks via WS)
    ↓
Complete (progress 100%, send final result via WS)
```

### WebSocket Updates
```javascript
{
    "type": "job_update",
    "progress": 30,
    "message": "Extracting text..."
}

{
    "type": "ai_chunk",
    "chunk": "The document discusses...",
    "partial": "full text so far"
}

{
    "type": "job_completed",
    "ai_response": "complete answer"
}
```

---

## 📝 Phase 6: Database Migration (1 hour)

### Add to `db/db.go`
```go
db.AutoMigrate(&models.ProcessingJob{})
```

### New Repository Methods
```go
CreateJob(job *ProcessingJob)
GetJobByID(id uuid.UUID)
GetUserJobs(userID uuid.UUID)
UpdateJob(job *ProcessingJob)
```

---

## 🧪 Phase 7: Testing (4 hours)

### Test 1: Queue System
```bash
# Start worker
go run cmd/worker/main.go

# Upload file with prompt
curl -X POST http://localhost:8080/api/v1/analyze \
  -H "Authorization: Bearer token" \
  -F "file=@test.pdf" \
  -F "prompt=What is this document about?"
```

### Test 2: WebSocket
```javascript
const ws = new WebSocket('ws://localhost:8080/ws?user_id=uuid');
ws.onmessage = (event) => {
    console.log(JSON.parse(event.data));
};
```

### Test 3: End-to-End
1. Upload PDF with question
2. Verify 202 response with job_id
3. Connect WebSocket
4. Watch progress updates
5. Receive AI response chunks
6. Get final completed status

---

## 🚀 Phase 8: Deployment (2 hours)

### Render Setup
1. Add Redis addon
2. Set environment variables
3. Deploy worker as background service
4. Deploy main app

### Environment Variables
```bash
REDIS_URL=redis://...
OPENAI_API_KEY=sk-...
OPENAI_MODEL=gpt-4o-mini
```

---

## 📊 Timeline Summary

| Phase | Task | Time |
|-------|------|------|
| 1 | Setup dependencies | 2h |
| 2 | Create structure | 4h |
| 3 | Core components | 8h |
| 4 | API integration | 4h |
| 5 | Flow integration | 4h |
| 6 | Database migration | 1h |
| 7 | Testing | 4h |
| 8 | Deployment | 2h |
| **Total** | | **29 hours** (~4 days) |

---

## ✅ Success Criteria

- [ ] Redis queue working
- [ ] WebSocket real-time updates
- [ ] OCR extraction (Google Vision/Textract)
- [ ] OpenAI analysis with streaming
- [ ] Progress tracking (0-100%)
- [ ] Error handling
- [ ] Database persistence
- [ ] Frontend integration
- [ ] Production deployment

---

## 🎯 Next Steps

1. **Day 1**: Phases 1-2 (Setup + Structure)
2. **Day 2**: Phase 3 (Core Components)
3. **Day 3**: Phases 4-5 (API + Integration)
4. **Day 4**: Phases 6-8 (DB + Testing + Deploy)

Ready to start? Let me know which phase to begin with!
