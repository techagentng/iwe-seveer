# ⚡ Render Quick Fix - 2 Minutes

## 🎯 The Issue
- Database name mismatch
- AWS credentials are swapped

## ✅ Fix in Render Dashboard

### 1. Fix AWS Credentials (CRITICAL)

**Go to:** Render → Your Service → Environment

**Find and fix these two variables:**

```bash
# Currently WRONG:
AWS_ACCESS_KEY_ID = 3542246689-jutm6p6ctc8he0k9ec4rg4f2eid0krmb.apps.googleusercontent.com
AWS_SECRET_ACCESS_KEY = AKIA... (starts with AKIA)

# Change to CORRECT:
AWS_ACCESS_KEY_ID = AKIA... (the one that starts with AKIA)
AWS_SECRET_ACCESS_KEY = ... (the long secret key)
```

**They're swapped!** The Google OAuth ID is in the wrong place.

### 2. Verify DATABASE_URL Exists

**Check:** Render should auto-set `DATABASE_URL` when you created the PostgreSQL database.

**If missing:** 
- Go to your PostgreSQL database
- Copy "Internal Database URL"  
- Add as environment variable: `DATABASE_URL`

### 3. Redeploy

Click: **"Manual Deploy" → "Deploy latest commit"**

---

## ✅ Expected Result

After fixing and redeploying, logs should show:
```
✓ Connected to database successfully
✓ Running migrations...
✓ Server starting on port 8080
```

---

## 📋 All Required Render Environment Variables

```bash
GIN_MODE=release
PORT=8080
ENV=production
JWT_SECRET=thesecretetowelth
DATABASE_URL=(auto-set by Render)
AWS_BUCKET=citizenx
AWS_REGION=eu-north-1
AWS_ACCESS_KEY_ID=(your AKIA... key)
AWS_SECRET_ACCESS_KEY=(your secret key)
GOOGLE_CLOUD_PROJECT=iwe-ocr
GOOGLE_APPLICATION_CREDENTIALS=./credentials/google-vision-key.json
```

---

## 🚀 That's It!

The code fix is already deployed. Just fix those two AWS environment variables in Render and redeploy.

**Full guide:** See `RENDER_DEPLOYMENT_FIX.md` for detailed instructions.
