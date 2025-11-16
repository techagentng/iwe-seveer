# 🔧 Render Deployment Fix

## ✅ Code Fix Applied

I've updated `config/config.go` to support `DATABASE_URL` environment variable, which Render provides automatically.

**Changes:**
- Added `DatabaseURL` field to Config struct
- Modified `GetDBUrl()` to use `DATABASE_URL` if available
- Falls back to individual DB fields if not

---

## 🎯 Fix Your Render Deployment

### Option 1: Use Render's DATABASE_URL (Recommended)

Render automatically provides a `DATABASE_URL` environment variable when you create a PostgreSQL database.

**Steps:**

1. **Go to your Render Dashboard**
   - Navigate to your PostgreSQL database
   - Copy the **Internal Database URL**

2. **In your Web Service settings:**
   - Go to "Environment" tab
   - Render should already have `DATABASE_URL` set automatically
   - If not, add it manually from your database's connection info

3. **Remove these environment variables** (not needed with DATABASE_URL):
   - `POSTGRES_HOST`
   - `POSTGRES_USER`
   - `POSTGRES_DB`
   - `POSTGRES_PASSWORD`
   - `POSTGRES_PORT`

4. **Redeploy:**
   - Click "Manual Deploy" → "Deploy latest commit"
   - Or it will auto-deploy from the git push

---

### Option 2: Fix Database Name

If you want to keep using individual DB fields:

1. **Find the actual database name:**
   - Go to Render Dashboard → Your PostgreSQL database
   - Look for "Database" field (not "Name")
   - It's usually something like: `citizenx_c81e` or similar

2. **Update environment variable:**
   - In your Web Service → Environment
   - Change `POSTGRES_DB` from `iwe_production` to the actual database name
   - Example: `POSTGRES_DB=citizenx_c81e`

3. **Redeploy**

---

## 🚨 Other Issues I Noticed

### 1. Wrong AWS Credentials

Your logs show:
```
AWS_ACCESS_KEY_ID:3542246689-jutm6p6ctc8he0k9ec4rg4f2eid0krmb.apps.googleusercontent.com
AWS_SECRET_ACCESS_KEY:AKIA... (wrong key here)
```

**These are swapped!** The Google Client ID is in AWS_ACCESS_KEY_ID.

**Fix in Render Environment Variables:**
```bash
# WRONG (current):
AWS_ACCESS_KEY_ID=3542246689-jutm6p6ctc8he0k9ec4rg4f2eid0krmb.apps.googleusercontent.com
AWS_SECRET_ACCESS_KEY=AKIA... (wrong key)

# CORRECT (fix to):
AWS_ACCESS_KEY_ID=AKIA... (your actual AWS access key)
AWS_SECRET_ACCESS_KEY=... (your actual AWS secret key)
```

### 2. Missing Environment Variables

Add these to Render:
```bash
FRONTEND_URL=https://your-frontend.com
BASE_URL=https://your-api.onrender.com
ACCESS_CONTROL_ALLOW_ORIGIN=https://your-frontend.com
```

---

## 📋 Complete Render Environment Variables

Here's what you should have in Render:

### Required:
```bash
# Application
GIN_MODE=release
PORT=8080
ENV=production
JWT_SECRET=thesecretetowelth

# Database (Render provides this automatically)
DATABASE_URL=postgres://citizenx_c81e_user:mK4Aiwy1bJpWMjdXCyjn26TNB9i7YZuY@dpg-cq98utaju9rs73b237c0-a:5432/citizenx_c81e

# AWS S3
AWS_BUCKET=citizenx
AWS_REGION=eu-north-1
AWS_ACCESS_KEY_ID=YOUR_AWS_ACCESS_KEY_ID
AWS_SECRET_ACCESS_KEY=YOUR_AWS_SECRET_ACCESS_KEY

# Google Cloud Vision
GOOGLE_APPLICATION_CREDENTIALS=./credentials/google-vision-key.json
GOOGLE_CLOUD_PROJECT=iwe-ocr

# Frontend
FRONTEND_URL=https://your-frontend.com
BASE_URL=https://your-api.onrender.com
ACCESS_CONTROL_ALLOW_ORIGIN=https://your-frontend.com
```

### Optional (if using):
```bash
# Mailgun
MG_PUBLIC_API_KEY=your-mailgun-key
MG_DOMAIN=your-domain.mailgun.org
EMAIL_FROM=noreply@yourdomain.com

# Google OAuth
GOOGLE_CLIENT_ID=3542246689-jutm6p6ctc8he0k9ec4rg4f2eid0krmb.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=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
GOOGLE_REDIRECT_URL=https://your-api.onrender.com/api/v1/auth/google/callback
```

---

## 🔐 Google Cloud Credentials on Render

For the Google Vision API credentials file:

### Option 1: Base64 Encode (Recommended)

```bash
# On your local machine
base64 -i credentials/google-vision-key.json > encoded-key.txt

# Copy the output and add to Render as:
GOOGLE_CREDENTIALS_BASE64=<paste-encoded-content>
```

Then update your code to decode it on startup.

### Option 2: Use Render Disk

1. Create a persistent disk in Render
2. Upload the credentials file
3. Mount it to your service
4. Update path: `GOOGLE_APPLICATION_CREDENTIALS=/mnt/data/google-vision-key.json`

### Option 3: Use Secret Files (Render Feature)

1. Go to your service → "Secret Files"
2. Add file: `credentials/google-vision-key.json`
3. Paste the JSON content
4. Keep path: `GOOGLE_APPLICATION_CREDENTIALS=./credentials/google-vision-key.json`

---

## 🚀 Deploy Steps

1. **Fix environment variables in Render:**
   - Correct AWS credentials (swap them back)
   - Ensure `DATABASE_URL` is set
   - Add missing variables

2. **Add Google credentials:**
   - Use Secret Files feature
   - Or base64 encode method

3. **Redeploy:**
   ```bash
   # Push the code fix (already done)
   git push origin main
   
   # Or manually trigger in Render dashboard
   ```

4. **Check logs:**
   - Watch deployment logs
   - Look for successful database connection
   - Verify no more "database does not exist" error

---

## ✅ Verification Checklist

After deployment:

- [ ] Database connects successfully
- [ ] No "database does not exist" error
- [ ] AWS S3 credentials are correct
- [ ] Google Cloud credentials loaded
- [ ] Server starts on port 8080
- [ ] Health check passes
- [ ] Can access API endpoints

---

## 🐛 If Still Having Issues

### Check Database Connection

In Render logs, you should see:
```
Connecting to postgres: &{...DatabaseURL:postgres://...}
```

If `DatabaseURL` is empty, Render didn't set it automatically. Add it manually.

### Verify Database Name

```bash
# In Render Shell (or connect to DB)
psql $DATABASE_URL -c "\l"

# This lists all databases
# Use the one that exists (usually matches your service name)
```

### Test Locally with Render's DATABASE_URL

```bash
# Copy DATABASE_URL from Render
export DATABASE_URL="postgres://citizenx_c81e_user:password@host:5432/citizenx_c81e"

# Test connection
psql $DATABASE_URL -c "SELECT 1;"
```

---

## 📞 Need Help?

If you're still stuck:
1. Check Render logs for specific errors
2. Verify all environment variables are set correctly
3. Ensure the database is running and accessible
4. Check if migrations need to run

The code fix is deployed - now just fix the environment variables in Render! 🚀
