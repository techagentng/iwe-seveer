#!/bin/bash

# File Upload Test Script
# This script tests the file upload endpoints

BASE_URL="http://localhost:8080/api/v1"

echo "==================================="
echo "File Upload System Test Script"
echo "==================================="
echo ""

# Check if JWT token is provided
if [ -z "$1" ]; then
    echo "Usage: ./test_upload.sh <JWT_TOKEN>"
    echo ""
    echo "Steps to get JWT token:"
    echo "1. First signup: curl -X POST $BASE_URL/auth/signup -H 'Content-Type: application/json' -d '{\"email\":\"test@example.com\",\"password\":\"password123\",\"fullname\":\"Test User\"}'"
    echo "2. Or login: curl -X POST $BASE_URL/auth/login -H 'Content-Type: application/json' -d '{\"email\":\"test@example.com\",\"password\":\"password123\"}'"
    echo "3. Copy the access_token from the response"
    echo ""
    exit 1
fi

JWT_TOKEN=$1

echo "Testing with JWT Token: ${JWT_TOKEN:0:20}..."
echo ""

# Test 1: Upload CSV file
echo "Test 1: Uploading CSV file..."
echo "-----------------------------------"
UPLOAD_RESPONSE=$(curl -s -X POST "$BASE_URL/upload" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -F "file=@sample_bank_statement.csv" \
  -F "type=csv")

echo "$UPLOAD_RESPONSE" | jq '.'
FILE_ID=$(echo "$UPLOAD_RESPONSE" | jq -r '.data.file_id')
echo ""
echo "File ID: $FILE_ID"
echo ""

# Wait for processing
echo "Waiting 3 seconds for processing..."
sleep 3

# Test 2: Check upload status
echo ""
echo "Test 2: Checking upload status..."
echo "-----------------------------------"
curl -s -X GET "$BASE_URL/upload/status/$FILE_ID" \
  -H "Authorization: Bearer $JWT_TOKEN" | jq '.'
echo ""

# Test 3: Get all user uploads
echo ""
echo "Test 3: Getting all user uploads..."
echo "-----------------------------------"
curl -s -X GET "$BASE_URL/upload/my-uploads" \
  -H "Authorization: Bearer $JWT_TOKEN" | jq '.'
echo ""

echo ""
echo "==================================="
echo "Tests completed!"
echo "==================================="
echo ""
echo "To test PDF/Image uploads, use:"
echo "curl -X POST $BASE_URL/upload -H 'Authorization: Bearer $JWT_TOKEN' -F 'file=@document.pdf' -F 'type=pdf'"
echo ""
