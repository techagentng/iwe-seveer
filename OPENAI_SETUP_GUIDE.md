# 🤖 OpenAI GPT-4o-mini Integration Guide

## ✅ What's New

### 1. OpenAI GPT-4o-mini Integration
- Real AI document analysis
- Streaming responses via WebSocket
- Fallback to placeholder if API key not set

### 2. Priority Queue
- High-priority jobs processed first
- Redis sorted sets for priority management
- Priority levels: 0 (normal) to 10 (highest)

### 3. Job Scheduling
- Schedule jobs for future execution
- Delayed job processing
- Automatic execution when time arrives

---

## 🔑 Get OpenAI API Key

### Step 1: Create OpenAI Account
1. Go to https://platform.openai.com/signup
2. Sign up or log in
3. Verify your email

### Step 2: Get API Key
1. Go to https://platform.openai.com/api-keys
2. Click **"Create new secret key"**
3. Name it: `iwe-server-production`
4. Copy the key (starts with `sk-proj-...`)
5. **Save it immediately** - you won't see it again!

### Step 3: Add Credits
1. Go to https://platform.openai.com/settings/organization/billing
2. Add payment method
3. Add at least $5 in credits
4. GPT-4o-mini is very cheap: ~$0.15 per 1M input tokens

---

## 💰 OpenAI Pricing

**GPT-4o-mini** (Recommended):
- Input: $0.150 / 1M tokens (~$0.0001 per 1K tokens)
- Output: $0.600 / 1M tokens (~$0.0006 per 1K tokens)

**Example Costs:**
- 1,000 document analyses (1K tokens each): ~$0.70
- 10,000 analyses: ~$7.00
- Very affordable for production use!

---

## ⚙️ Configuration

### Local Development

Add to your `.env` file:

```bash
# OpenAI
OPENAI_API_KEY=sk-proj-xxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

### Production (Render)

Add environment variable in Render dashboard:

```bash
Key: OPENAI_API_KEY
Value: sk-proj-xxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

---

## 🧪 Testing

### Test 1: Without API Key (Placeholder Mode)

```bash
# Don't set OPENAI_API_KEY
go run main.go

# You'll see:
# ⚠️  OpenAI API key not set - AI features will use placeholder responses
```

Upload a file - you'll get a placeholder response.

### Test 2: With API Key (Real AI)

```bash
# Set API key in .env
OPENAI_API_KEY=sk-proj-your-real-key

# Run server
go run main.go

# You'll see:
# ✅ OpenAI service initialized with GPT-4o-mini
```

Upload a file - you'll get real AI analysis!

---

## 📊 Priority Queue Usage

### Create High-Priority Job

```bash
POST /api/v1/upload
Authorization: Bearer {token}

{
  "file": <file>,
  "prompt": "Urgent: Analyze this contract",
  "priority": 10  # 0-10, higher = more important
}
```

**Priority Levels:**
- `0`: Normal (default)
- `1-3`: Low priority
- `4-6`: Medium priority
- `7-9`: High priority
- `10`: Critical/Urgent

### How It Works

```
Priority Queue (Sorted Set):
┌─────────────────────────────┐
│ Priority 10 → Job A (first) │
│ Priority 8  → Job B          │
│ Priority 5  → Job C          │
└─────────────────────────────┘

Regular Queue (FIFO):
┌─────────────────────────────┐
│ Job D → Job E → Job F       │
└─────────────────────────────┘

Worker checks:
1. Priority queue first
2. Then regular queue
```

---

## ⏰ Job Scheduling Usage

### Schedule Job for Later

```bash
POST /api/v1/upload
Authorization: Bearer {token}

{
  "file": <file>,
  "prompt": "Analyze this report",
  "scheduled_at": "2025-11-19T14:00:00Z"  # ISO 8601 format
}
```

### Schedule Examples

```javascript
// Schedule for 1 hour from now
const scheduledTime = new Date(Date.now() + 3600000);
scheduled_at: scheduledTime.toISOString()

// Schedule for tomorrow 9 AM
const tomorrow9am = new Date();
tomorrow9am.setDate(tomorrow9am.getDate() + 1);
tomorrow9am.setHours(9, 0, 0, 0);
scheduled_at: tomorrow9am.toISOString()

// Schedule for specific date/time
scheduled_at: "2025-12-01T10:30:00Z"
```

### How It Works

1. Job created with `scheduled_at` timestamp
2. Enqueued to Redis
3. Worker picks job
4. Checks if `time.Now() >= scheduled_at`
5. If not ready, re-enqueues
6. If ready, processes immediately

---

## 🔄 Complete API Flow

### 1. Upload with All Features

```bash
curl -X POST http://localhost:8080/api/v1/upload \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -F "file=@document.pdf" \
  -F "prompt=What are the key findings in this document?" \
  -F "priority=8" \
  -F "scheduled_at=2025-11-19T14:00:00Z"

Response (202 Accepted):
{
  "message": "File uploaded successfully",
  "data": {
    "job_id": "uuid",
    "status": "queued",
    "priority": 8,
    "scheduled_at": "2025-11-19T14:00:00Z"
  }
}
```

### 2. Connect WebSocket

```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    
    switch(data.type) {
        case 'job_update':
            console.log(`Progress: ${data.progress}%`);
            break;
            
        case 'ai_chunk':
            // Real-time AI streaming!
            console.log('AI says:', data.chunk);
            document.getElementById('response').innerHTML += data.chunk;
            break;
            
        case 'job_completed':
            console.log('Done!', data.ai_response);
            break;
    }
};
```

### 3. Real-Time Updates

```
10% - Starting job processing...
20% - Extracting text from file...
30% - Running OCR on document...
60% - Text extracted successfully
70% - Analyzing with AI...
     ↓
[AI Streaming Chunks]
"Based on the document..."
"The key findings are..."
"1. First finding..."
"2. Second finding..."
     ↓
90% - AI analysis complete
100% - Job completed!
```

---

## 🎯 Frontend Integration Example

### React Hook

```javascript
import { useEffect, useState } from 'react';

function useAIDocumentAnalysis(token) {
    const [ws, setWs] = useState(null);
    const [progress, setProgress] = useState(0);
    const [aiResponse, setAIResponse] = useState('');
    const [isStreaming, setIsStreaming] = useState(false);

    useEffect(() => {
        const websocket = new WebSocket('ws://localhost:8080/ws');
        
        websocket.onmessage = (event) => {
            const data = JSON.parse(event.data);
            
            if (data.type === 'job_update') {
                setProgress(data.progress);
            }
            
            if (data.type === 'ai_chunk') {
                setIsStreaming(true);
                setAIResponse(data.partial);
            }
            
            if (data.type === 'job_completed') {
                setIsStreaming(false);
                setProgress(100);
                setAIResponse(data.ai_response);
            }
        };
        
        setWs(websocket);
        return () => websocket.close();
    }, [token]);

    const uploadFile = async (file, prompt, priority = 0) => {
        const formData = new FormData();
        formData.append('file', file);
        formData.append('prompt', prompt);
        formData.append('priority', priority);

        const response = await fetch('http://localhost:8080/api/v1/upload', {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${token}`
            },
            body: formData
        });

        return response.json();
    };

    return { uploadFile, progress, aiResponse, isStreaming };
}

// Usage in component
function DocumentAnalyzer() {
    const { uploadFile, progress, aiResponse, isStreaming } = useAIDocumentAnalysis(token);
    
    const handleUpload = async (file) => {
        const result = await uploadFile(
            file, 
            "Summarize the key points",
            8 // High priority
        );
        console.log('Job created:', result.data.job_id);
    };
    
    return (
        <div>
            <input type="file" onChange={(e) => handleUpload(e.target.files[0])} />
            
            <div>Progress: {progress}%</div>
            
            {isStreaming && <div className="streaming">AI is thinking...</div>}
            
            <div className="ai-response">
                {aiResponse}
            </div>
        </div>
    );
}
```

---

## 📈 Monitoring

### Redis Commands

```bash
# Check queue lengths
redis-cli LLEN iwe:jobs:queue
redis-cli ZCARD iwe:jobs:priority_queue

# View priority jobs
redis-cli ZRANGE iwe:jobs:priority_queue 0 -1 WITHSCORES

# Check specific job
redis-cli GET iwe:job:{uuid}
```

### Application Logs

```bash
# Watch for AI activity
tail -f logs/app.log | grep "AI"

# Monitor job processing
tail -f logs/app.log | grep "Worker"

# Check OpenAI status
tail -f logs/app.log | grep "OpenAI"
```

---

## 🚨 Troubleshooting

### Issue: "OpenAI API key not set"

**Solution:**
```bash
# Check if key is in .env
cat .env | grep OPENAI_API_KEY

# If not, add it:
echo "OPENAI_API_KEY=sk-proj-your-key" >> .env

# Restart server
```

### Issue: "OpenAI API error: 401 Unauthorized"

**Solution:**
- API key is invalid or expired
- Get a new key from https://platform.openai.com/api-keys
- Update `.env` file

### Issue: "OpenAI API error: 429 Rate limit"

**Solution:**
- You've hit rate limits
- Add more credits to your account
- Wait a few minutes and try again
- Consider upgrading your OpenAI tier

### Issue: Priority jobs not processing first

**Solution:**
```bash
# Check priority queue
redis-cli ZRANGE iwe:jobs:priority_queue 0 -1 WITHSCORES

# Ensure priority > 0 when creating job
# Priority 0 goes to regular queue
```

---

## 🎉 Success Checklist

```bash
✅ OpenAI API key obtained
✅ Added to .env file
✅ Server shows "OpenAI service initialized"
✅ File upload creates job
✅ WebSocket connected
✅ AI streaming chunks received
✅ Job completes with AI response
✅ Priority queue working
✅ Scheduled jobs executing on time
```

---

## 💡 Tips

1. **Start with placeholder mode** - Test without API key first
2. **Monitor costs** - Check OpenAI usage dashboard regularly
3. **Use priority wisely** - Reserve high priority for urgent jobs
4. **Cache responses** - Store AI responses in database
5. **Set rate limits** - Prevent abuse in production
6. **Test scheduling** - Schedule jobs 1 minute ahead for testing

---

## 🚀 Ready for Production!

Your system now has:
- ✅ Real AI analysis with GPT-4o-mini
- ✅ Priority queue for urgent jobs
- ✅ Job scheduling for delayed execution
- ✅ Real-time streaming via WebSocket
- ✅ Fallback to placeholder mode
- ✅ Cost-effective pricing

**Deploy to Render and start analyzing documents with AI!** 🎉
