#!/bin/bash
# 调试登录脚本 — 调用后端 ticket 接口，打印可在浏览器打开的 URL
# 用法: ./scripts/dev-login.sh [id]

set -e

USER_ID="${1:-user_001}"
API_KEY="${JWT_API_KEY:-${IM_API_KEY:-im-api-key-change-me}}"
BASE="${IM_BASE:-http://localhost:8080}"

echo "→ 请求 ticket (ID=$USER_ID, API_KEY=${API_KEY:0:8}...)"
RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE/api/v1/auth/ticket" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d "{\"id\": \"$USER_ID\"}") || {
  echo "❌ 无法连接后端 ($BASE)，请确认 gateway 已启动"
  exit 1
}

HTTP_CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | sed '$d')

if [ "$HTTP_CODE" != "200" ]; then
  echo "❌ 后端返回 HTTP $HTTP_CODE: $BODY"
  exit 1
fi

TICKET=$(echo "$BODY" | grep -o '"ticket":"[^"]*"' | cut -d'"' -f4)
REDIRECT=$(echo "$BODY" | grep -o '"redirect_url":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TICKET" ]; then
  echo "❌ 响应中未找到 ticket: $BODY"
  exit 1
fi

FRONTEND="http://localhost:5173${REDIRECT}"
echo ""
echo "====================================="
echo "  ID:      $USER_ID"
echo "  Ticket:  ${TICKET:0:40}..."
echo "====================================="
echo ""
echo "  浏览器打开:"
echo "  $FRONTEND"
echo ""
