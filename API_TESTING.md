# API Testing Examples

## Test Commands

### 1. Health Check
```bash
curl http://localhost:8080/health
```

### 2. Create a new session
```bash
curl -X POST http://localhost:8080/api/v1/sessions \
  -H "Content-Type: application/json" \
  -d '{"session_id": "my_whatsapp_session"}'
```

### 3. List all sessions
```bash
curl http://localhost:8080/api/v1/sessions
```

### 4. Get session info
```bash
curl http://localhost:8080/api/v1/sessions/my_whatsapp_session
```

### 5. Get QR code for authentication
```bash
curl http://localhost:8080/api/v1/sessions/my_whatsapp_session/qr
```

### 6. Check session status
```bash
curl http://localhost:8080/api/v1/sessions/my_whatsapp_session/status
```

### 7. Send a message (requires authenticated session)
```bash
curl -X POST http://localhost:8080/api/v1/sessions/my_whatsapp_session/send \
  -H "Content-Type: application/json" \
  -d '{
    "jid": "6281234567890@s.whatsapp.net",
    "message": "Hello from WhatsGo!"
  }'
```

### 8. Logout session
```bash
curl -X POST http://localhost:8080/api/v1/sessions/my_whatsapp_session/logout
```

### 9. Delete session
```bash
curl -X DELETE http://localhost:8080/api/v1/sessions/my_whatsapp_session
```

## Response Format

All API responses follow this format:

```json
{
  "success": true,
  "message": "Optional success message",
  "data": {},
  "error": "Optional error message"
}
```

## Session States

- `disconnected`: Session is not connected to WhatsApp
- `connecting`: Session is attempting to connect
- `connected`: Session is connected and ready to use
- `logged_out`: Session has been logged out

## JID Format

WhatsApp JID (Jabber ID) formats:
- Individual: `6281234567890@s.whatsapp.net`
- Group: `120363000000000000@g.us`

Replace the phone number with the actual number including country code (without + sign).