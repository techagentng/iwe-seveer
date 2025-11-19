# ⚡ Quick Reference Card

## 🚀 Start Locally

```bash
# 1. Start services
brew services start redis
brew services start postgresql

# 2. Run server
go run main.go

# 3. Test
curl http://localhost:8080/health
```

---

## 🔑 Environment Variables (Add to .env)

```bash
# Essential
DATABASE_URL=postgresql://user:pass@localhost/iwe_db
REDIS_HOST=localhost
REDIS_PORT=6379
JWT_SECRET=your-secret-key
OPENAI_API_KEY=sk-proj-xxxxxxxxxxxxx

# AWS S3
AWS_BUCKET=your-bucket
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=AKIAXXXXX
AWS_SECRET_ACCESS_KEY=xxxxxxxx

# Optional
GOOGLE_APPLICATION_CREDENTIALS=/path/to/creds.json
```

---

## 📡 API Quick Reference

### Auth
```bash
# Signup
POST /api/v1/auth/signup
{ "email": "user@example.com", "password": "Pass123!", "fullname": "User" }

# Login
POST /api/v1/auth/login
{ "email": "user@example.com", "password": "Pass123!" }
```

### Upload
```bash
# Normal upload
POST /api/v1/upload
-H "Authorization: Bearer TOKEN"
-F "file=@document.pdf"
-F "prompt=Summarize this"

# Priority upload
-F "priority=8"

# Scheduled upload
-F "scheduled_at=2025-11-20T09:00:00Z"
```

### Status
```bash
# Check job
GET /api/v1/upload/status/{job_id}

# List uploads
GET /api/v1/upload/my-uploads
```

---

## 🔌 WebSocket

```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    if (data.type === 'ai_chunk') {
        console.log(data.chunk);
    }
};
```

---

## 🛠️ Redis Commands

```bash
# Queue length
redis-cli LLEN iwe:jobs:queue

# Priority queue
redis-cli ZCARD iwe:jobs:priority_queue

# View job
redis-cli GET iwe:job:{uuid}

# Monitor
redis-cli MONITOR
```

---

## 📊 Database Queries

```sql
-- Job stats
SELECT status, COUNT(*) FROM processing_jobs GROUP BY status;

-- Recent jobs
SELECT id, status, progress FROM processing_jobs ORDER BY created_at DESC LIMIT 10;

-- Failed jobs
SELECT id, error_msg FROM processing_jobs WHERE status = 'failed';
```

---

## 🚀 Deploy to Render

```bash
# 1. Create services
- PostgreSQL database
- Redis instance  
- Web service

# 2. Add env vars
DATABASE_URL, REDIS_URL, OPENAI_API_KEY, etc.

# 3. Deploy
git push origin main
```

---

## 🔍 Troubleshooting

```bash
# Check Redis
redis-cli ping

# Check database
psql $DATABASE_URL -c "SELECT 1"

# View logs
tail -f logs/app.log

# Test OpenAI
curl https://api.openai.com/v1/models \
  -H "Authorization: Bearer $OPENAI_API_KEY"
```

---

## 📈 Monitoring

```bash
# Queue length
redis-cli LLEN iwe:jobs:queue

# Active workers
ps aux | grep iwe-server

# Database connections
SELECT count(*) FROM pg_stat_activity;

# OpenAI usage
https://platform.openai.com/usage
```

---

## 🎯 Priority Levels

```
0  = Normal (default)
1-3 = Low
4-6 = Medium
7-9 = High
10 = Critical
```

---

## 💰 Costs

```
OpenAI GPT-4o-mini:
- 1K analyses: ~$0.70
- 10K analyses: ~$7

Render (monthly):
- Web service: $7
- PostgreSQL: $7
- Redis: $10
Total: ~$24/month
```

---

## ✅ Health Checks

```bash
# Server
curl http://localhost:8080/health

# Redis
redis-cli ping

# Database
psql $DATABASE_URL -c "SELECT 1"

# Workers
# Check logs for "Worker #1 started"
```

---

## 🔐 Get API Keys

```
OpenAI: https://platform.openai.com/api-keys
AWS: https://console.aws.amazon.com/iam/
Google Cloud: https://console.cloud.google.com/apis/credentials
```

---

## 📚 Documentation

```
COMPLETE_SYSTEM_OVERVIEW.md  - Full system details
OPENAI_SETUP_GUIDE.md        - OpenAI integration
DEPLOYMENT_TESTING_GUIDE.md  - Production deployment
JOB_QUEUE_COMPLETE.md        - Job queue system
REDIS_SETUP.md               - Redis configuration
```

---

## 🎉 Success Indicators

```
✅ Server starts without errors
✅ Redis connected
✅ Workers started (3)
✅ WebSocket hub running
✅ OpenAI initialized
✅ Jobs processing
✅ Real-time updates working
```

---

**Keep this card handy for quick reference!** 📌
