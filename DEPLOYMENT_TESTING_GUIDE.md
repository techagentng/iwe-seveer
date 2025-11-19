# 🚀 Deployment & Testing Guide

## 📋 Pre-Deployment Checklist

### Local Testing
- [ ] Redis running (`redis-cli ping` returns PONG)
- [ ] PostgreSQL running
- [ ] `.env` file configured
- [ ] `go run main.go` starts successfully
- [ ] All services initialized:
  - [ ] ✅ Connected to Redis
  - [ ] ✅ Queue Manager initialized
  - [ ] ✅ WebSocket Hub started
  - [ ] ✅ Worker Pool started with 3 workers
  - [ ] ✅ OpenAI service initialized (or placeholder warning)

---

## 🌐 Deploy to Render

### Step 1: Prepare Repository

```bash
# Ensure all changes are committed
git status
git add -A
git commit -m "Ready for production deployment"
git push origin main
```

### Step 2: Create PostgreSQL Database

1. Go to https://dashboard.render.com
2. Click **"New +"** → **"PostgreSQL"**
3. Settings:
   - Name: `iwe-database`
   - Database: `iwe_db`
   - User: `iwe_user`
   - Region: Choose closest to you
   - Plan: Free or Starter
4. Click **"Create Database"**
5. **Copy the Internal Database URL** (starts with `postgresql://`)

### Step 3: Create Redis Instance

1. Click **"New +"** → **"Redis"**
2. Settings:
   - Name: `iwe-redis`
   - Region: **Same as PostgreSQL**
   - Plan: Free or Starter
3. Click **"Create Redis"**
4. **Copy the Internal Redis URL** (starts with `redis://`)

### Step 4: Create Web Service

1. Click **"New +"** → **"Web Service"**
2. Connect your GitHub repository
3. Settings:
   - Name: `iwe-server`
   - Region: **Same as database and Redis**
   - Branch: `main`
   - Runtime: **Go**
   - Build Command: `go build -o iwe-server`
   - Start Command: `./iwe-server`
   - Plan: Free or Starter

### Step 5: Configure Environment Variables

Add these in the **Environment** tab:

```bash
# Database (from Step 2)
DATABASE_URL=postgresql://iwe_user:password@host/iwe_db

# Redis (from Step 3)
REDIS_URL=redis://red-xxxxxxxxxxxxx:6379

# JWT
JWT_SECRET=your-super-secret-production-jwt-key-change-this

# AWS S3
AWS_BUCKET=your-bucket-name
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=AKIAXXXXXXXXXXXXX
AWS_SECRET_ACCESS_KEY=your-secret-key

# Google Cloud Vision
GOOGLE_APPLICATION_CREDENTIALS=/etc/secrets/google-credentials.json
GOOGLE_CLOUD_PROJECT=your-project-id

# OpenAI (Get from https://platform.openai.com/api-keys)
OPENAI_API_KEY=sk-proj-xxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# Mailgun
MAILGUN_DOMAIN=your-domain.mailgun.org
MAILGUN_API_KEY=your-mailgun-api-key

# Server
PORT=8080
GIN_MODE=release
```

### Step 6: Deploy

1. Click **"Create Web Service"**
2. Wait for deployment (5-10 minutes)
3. Check logs for:
   ```
   ✅ Connected to Redis at red-xxxxx:6379
   ✅ Queue Manager initialized
   ✅ WebSocket Hub started
   ✅ Worker Pool started with 3 workers
   ✅ OpenAI service initialized
   Server started on :8080
   ```

### Step 7: Get Your URL

Your app will be at: `https://iwe-server.onrender.com`

---

## 🧪 Testing Guide

### Test 1: Health Check

```bash
curl https://iwe-server.onrender.com/health

# Expected: 200 OK
```

### Test 2: User Registration

```bash
curl -X POST https://iwe-server.onrender.com/api/v1/auth/signup \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "Test123!@#",
    "fullname": "Test User"
  }'

# Expected: 201 Created with user data
```

### Test 3: Login

```bash
curl -X POST https://iwe-server.onrender.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "Test123!@#"
  }'

# Expected: 200 OK with access_token
# Save the token for next tests
```

### Test 4: Upload File (Normal Priority)

```bash
TOKEN="your-access-token-from-login"

curl -X POST https://iwe-server.onrender.com/api/v1/upload \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@sample_bank_statement.csv" \
  -F "prompt=Summarize my spending patterns"

# Expected: 202 Accepted with job_id
```

### Test 5: Upload with Priority

```bash
curl -X POST https://iwe-server.onrender.com/api/v1/upload \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@document.pdf" \
  -F "prompt=What are the key findings?" \
  -F "priority=8"

# Expected: 202 Accepted with priority=8
```

### Test 6: Upload with Scheduling

```bash
# Schedule for 5 minutes from now
SCHEDULED_TIME=$(date -u -v+5M +"%Y-%m-%dT%H:%M:%SZ")

curl -X POST https://iwe-server.onrender.com/api/v1/upload \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@document.pdf" \
  -F "prompt=Analyze this report" \
  -F "scheduled_at=$SCHEDULED_TIME"

# Expected: 202 Accepted with scheduled_at timestamp
```

### Test 7: WebSocket Connection

```javascript
// In browser console or Node.js
const token = "your-access-token";
const ws = new WebSocket('wss://iwe-server.onrender.com/ws');

ws.onopen = () => {
    console.log('✅ Connected to WebSocket');
};

ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    console.log('📨 Received:', data);
    
    if (data.type === 'ai_chunk') {
        console.log('🤖 AI:', data.chunk);
    }
};

ws.onerror = (error) => {
    console.error('❌ Error:', error);
};

// Expected: Connection successful, real-time updates received
```

### Test 8: Check Job Status

```bash
JOB_ID="uuid-from-upload-response"

curl https://iwe-server.onrender.com/api/v1/upload/status/$JOB_ID \
  -H "Authorization: Bearer $TOKEN"

# Expected: Job details with status, progress, ai_response
```

### Test 9: List User Uploads

```bash
curl https://iwe-server.onrender.com/api/v1/upload/my-uploads \
  -H "Authorization: Bearer $TOKEN"

# Expected: Array of user's uploaded files and jobs
```

---

## 📊 Monitoring Production

### Render Dashboard

1. **Logs**: Click on your service → "Logs" tab
2. **Metrics**: View CPU, memory, requests
3. **Events**: See deployments, restarts

### Redis Monitoring

```bash
# Install Redis CLI locally
brew install redis

# Connect to production Redis
redis-cli -u redis://red-xxxxx:6379

# Check queue lengths
LLEN iwe:jobs:queue
ZCARD iwe:jobs:priority_queue

# View jobs
KEYS iwe:job:*

# Monitor in real-time
MONITOR
```

### Database Monitoring

```bash
# Connect to production database
psql postgresql://user:pass@host/db

# Check tables
\dt

# Count jobs
SELECT status, COUNT(*) FROM processing_jobs GROUP BY status;

# Recent jobs
SELECT id, status, progress, created_at 
FROM processing_jobs 
ORDER BY created_at DESC 
LIMIT 10;
```

### OpenAI Usage

1. Go to https://platform.openai.com/usage
2. View daily usage and costs
3. Set up usage limits if needed

---

## 🔍 Performance Testing

### Load Test with Apache Bench

```bash
# Test 100 requests, 10 concurrent
ab -n 100 -c 10 \
  -H "Authorization: Bearer $TOKEN" \
  https://iwe-server.onrender.com/api/v1/upload/my-uploads

# Expected: 
# - Requests per second: 50+
# - Mean response time: < 200ms
```

### Stress Test Job Queue

```bash
# Upload 50 files rapidly
for i in {1..50}; do
  curl -X POST https://iwe-server.onrender.com/api/v1/upload \
    -H "Authorization: Bearer $TOKEN" \
    -F "file=@test.pdf" \
    -F "prompt=Test $i" &
done

# Monitor worker logs
# Expected: All jobs processed, no errors
```

---

## 🐛 Troubleshooting

### Issue: "Failed to connect to Redis"

**Check:**
```bash
# Verify REDIS_URL is set
echo $REDIS_URL

# Test Redis connection
redis-cli -u $REDIS_URL ping
```

**Solution:**
- Ensure Redis and web service in same region
- Use Internal Redis URL, not External
- Check Redis instance is running

### Issue: "Database connection failed"

**Check:**
```bash
# Verify DATABASE_URL is set
echo $DATABASE_URL

# Test database connection
psql $DATABASE_URL -c "SELECT 1"
```

**Solution:**
- Ensure database is running
- Use Internal Database URL
- Check credentials are correct

### Issue: "OpenAI API error: 401"

**Solution:**
- API key is invalid
- Get new key from OpenAI dashboard
- Update OPENAI_API_KEY environment variable
- Redeploy

### Issue: "Worker not processing jobs"

**Check logs:**
```
# Should see:
🔧 Worker #1 started
🔧 Worker #2 started
🔧 Worker #3 started
```

**Solution:**
- Check Redis connection
- Verify worker pool initialized
- Check for errors in logs
- Restart service

### Issue: "WebSocket connection failed"

**Solution:**
- Use `wss://` not `ws://` for production
- Check CORS settings
- Verify authentication token
- Check firewall/proxy settings

---

## 📈 Scaling Guide

### Increase Workers

In `main.go`:
```go
workerPool := queue.NewWorkerPool(queue.WorkerPoolConfig{
    NumWorkers: 5, // Increase from 3 to 5
    // ...
})
```

Redeploy.

### Upgrade Render Plan

1. Go to service settings
2. Click "Change Plan"
3. Select higher tier
4. More CPU/RAM = more concurrent jobs

### Add Redis Memory

1. Go to Redis instance
2. Click "Change Plan"
3. Select plan with more memory
4. Supports more queued jobs

### Database Scaling

1. Upgrade PostgreSQL plan
2. Add read replicas if needed
3. Enable connection pooling

---

## ✅ Production Checklist

```bash
✅ All environment variables set
✅ Redis running and connected
✅ PostgreSQL running and migrated
✅ OpenAI API key configured
✅ AWS S3 credentials valid
✅ Google Cloud Vision configured
✅ Workers processing jobs
✅ WebSocket connections working
✅ SSL/TLS enabled (wss://)
✅ CORS configured for frontend
✅ Error logging enabled
✅ Monitoring set up
✅ Backup strategy in place
✅ Rate limiting configured
✅ API documentation updated
```

---

## 🎯 Next Steps

1. **Set up monitoring** - Use Render metrics + custom logging
2. **Configure alerts** - Get notified of errors
3. **Add rate limiting** - Prevent abuse
4. **Implement caching** - Cache AI responses
5. **Add analytics** - Track usage patterns
6. **Create admin dashboard** - Monitor jobs
7. **Set up CI/CD** - Automated testing and deployment
8. **Document API** - Swagger/OpenAPI spec
9. **Load testing** - Test with realistic traffic
10. **Security audit** - Review authentication, authorization

---

## 🎉 Success!

Your production system is now running with:
- ✅ Asynchronous job processing
- ✅ Real-time WebSocket updates
- ✅ AI-powered document analysis
- ✅ Priority queue for urgent jobs
- ✅ Job scheduling for delayed execution
- ✅ Scalable worker pool
- ✅ Production-grade infrastructure

**Start processing documents with AI!** 🚀
