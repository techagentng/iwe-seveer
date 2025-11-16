# 🚨 Fix Git Secrets - Step by Step

## Problem
GitHub blocked your push because `.env` file with real AWS credentials was committed in previous commits (268c1eb, 368ad87, 2f48127).

## ✅ Quick Solution (Recommended)

### Option 1: Create New Clean Branch

```bash
cd /Users/nnahnnamdi/Desktop/iwe-server

# Create a new orphan branch (no history)
git checkout --orphan clean-main

# Add all files (except .env which is now in .gitignore)
git add -A

# Commit everything fresh
git commit -m "Initial commit - production ready with hybrid OCR"

# Delete old main branch
git branch -D main

# Rename clean-main to main
git branch -m main

# Force push to GitHub (this replaces all history)
git push origin main --force
```

**This creates a completely new history without any secrets!**

---

## Option 2: Use GitHub's Secret Bypass (Temporary)

If you need to push urgently and will rotate keys later:

1. Go to the URLs GitHub provided:
   - https://github.com/techagentng/iwe-seveer/security/secret-scanning/unblock-secret/35Y5pgjUvuaJQk2NWG7T6flW1JD
   - https://github.com/techagentng/iwe-seveer/security/secret-scanning/unblock-secret/35Y5phkt8W2wgicdaJtqkqKPRPy
   - https://github.com/techagentng/iwe-seveer/security/secret-scanning/unblock-secret/35Y5piuWwhGqBRhj6jxrzAldItV

2. Click "Allow secret" for each one

3. Push again:
   ```bash
   git push origin main --force
   ```

**⚠️ WARNING: This leaves your secrets exposed in git history! You MUST rotate your AWS keys after!**

---

## Option 3: Use BFG Repo-Cleaner (Advanced)

```bash
# Install BFG
brew install bfg

# Clone a fresh copy
cd ~/Desktop
git clone --mirror https://github.com/techagentng/iwe-seveer.git

# Remove .env from all history
bfg --delete-files .env iwe-seveer.git

# Clean up
cd iwe-seveer.git
git reflog expire --expire=now --all
git gc --prune=now --aggressive

# Push cleaned history
git push --force
```

---

## 🔒 After Pushing - Rotate Your Keys!

### 1. Rotate AWS Keys

**In AWS Console:**
1. Go to IAM → Users → Your User
2. Security Credentials tab
3. Create new access key
4. Update `.env` with new keys
5. Delete old access key

### 2. Rotate Mailgun API Key

**In Mailgun Dashboard:**
1. Settings → API Security
2. Create new API key
3. Update `.env`
4. Delete old key

### 3. Update `.env` Locally

```bash
# Edit .env with new credentials
nano .env

# Verify it's in .gitignore
cat .gitignore | grep .env
```

---

## ✅ Prevention Checklist

- [x] `.env` added to `.gitignore`
- [x] Binaries (`iwe-server`, `iwe-server-upload`) added to `.gitignore`
- [x] Documentation files sanitized (no real credentials)
- [ ] AWS keys rotated
- [ ] Mailgun key rotated
- [ ] Clean git history pushed

---

## 🎯 Recommended Action

**Use Option 1 (Clean Branch)** - It's the safest and cleanest solution:

```bash
# Run these commands now:
cd /Users/nnahnnamdi/Desktop/iwe-server
git checkout --orphan clean-main
git add -A
git commit -m "Initial commit - production ready with hybrid OCR and S3 upload"
git branch -D main
git branch -m main
git push origin main --force
```

This will:
✅ Remove all secret history
✅ Keep all your current code
✅ Push successfully to GitHub
✅ Start with a clean slate

**Then rotate your AWS keys for extra security!**
