# WhatsGo - WhatsApp Multi Session API

WhatsGo adalah aplikasi API untuk mengelola multiple session WhatsApp menggunakan library whatsmeow dan Go. Aplikasi ini dilengkapi dengan Docker support untuk deployment yang mudah.

## Fitur

- Multi-session WhatsApp management
- QR Code authentication
- REST API untuk mengelola session
- Message sending capabilities
- Docker containerization
- Session persistence dengan SQLite
- Logging dan health checks

## Prerequisites

- Go 1.21+
- Docker & Docker Compose (untuk containerization)
- SQLite3

## Installation

### Using Docker (Recommended)

1. Clone repository:
```bash
git clone https://github.com/indrawanalghifary/whatsgo.git
cd whatsgo
```

2. Build dan jalankan dengan docker-compose:
```bash
docker-compose up -d
```

3. API akan tersedia di `http://localhost:8080`

### Manual Installation

1. Clone repository:
```bash
git clone https://github.com/indrawanalghifary/whatsgo.git
cd whatsgo
```

2. Install dependencies:
```bash
go mod download
```

3. Copy environment configuration:
```bash
cp .env.example .env
```

4. Build aplikasi:
```bash
go build -o whatsgo .
```

5. Jalankan aplikasi:
```bash
./whatsgo
```

## API Documentation

### Base URL
```
http://localhost:8080/api/v1
```

### Endpoints

#### 1. Health Check
```
GET /health
```

#### 2. Create Session
```
POST /sessions
Content-Type: application/json

{
  "session_id": "session1"
}
```

#### 3. List Sessions
```
GET /sessions
```

#### 4. Get Session Info
```
GET /sessions/{sessionId}
```

#### 5. Get QR Code for Authentication
```
GET /sessions/{sessionId}/qr
```

#### 6. Send Message
```
POST /sessions/{sessionId}/send
Content-Type: application/json

{
  "jid": "6281234567890@s.whatsapp.net",
  "message": "Hello from WhatsGo!"
}
```

#### 7. Get Session Status
```
GET /sessions/{sessionId}/status
```

#### 8. Logout Session
```
POST /sessions/{sessionId}/logout
```

#### 9. Delete Session
```
DELETE /sessions/{sessionId}
```

#### 10. Get Chats (Placeholder)
```
GET /sessions/{sessionId}/chats
```

## Usage Example

### 1. Membuat Session Baru
```bash
curl -X POST http://localhost:8080/api/v1/sessions \
  -H "Content-Type: application/json" \
  -d '{"session_id": "my_session"}'
```

### 2. Mendapatkan QR Code untuk Authentication
```bash
curl http://localhost:8080/api/v1/sessions/my_session/qr
```

Scan QR code yang diberikan dengan WhatsApp di ponsel Anda.

### 3. Mengirim Pesan
```bash
curl -X POST http://localhost:8080/api/v1/sessions/my_session/send \
  -H "Content-Type: application/json" \
  -d '{
    "jid": "6281234567890@s.whatsapp.net",
    "message": "Hello from WhatsGo API!"
  }'
```

### 4. Memeriksa Status Session
```bash
curl http://localhost:8080/api/v1/sessions/my_session/status
```

## Configuration

Environment variables yang dapat dikonfigurasi:

| Variable | Default | Description |
|----------|---------|-------------|
| PORT | 8080 | Server port |
| HOST | 0.0.0.0 | Server host |
| DB_TYPE | sqlite | Database type |
| DB_PATH | ./sessions.db | Database path |
| MAX_SESSIONS | 10 | Maximum concurrent sessions |
| QR_CODE_TIMEOUT | 60 | QR code timeout in seconds |
| LOG_LEVEL | info | Logging level |
| LOG_FILE | ./logs/whatsgo.log | Log file path |

## Docker Compose Configuration

File `docker-compose.yml` sudah dikonfigurasi dengan:
- Volume persistence untuk database dan logs
- Health checks
- Restart policy
- Port mapping
- Environment variables

## File Structure

```
whatsgo/
├── main.go                 # Entry point aplikasi
├── internal/
│   ├── config/            # Configuration management
│   ├── session/           # Session management & whatsmeow integration
│   └── handlers/          # HTTP handlers & API endpoints
├── Dockerfile             # Docker configuration
├── docker-compose.yml     # Docker Compose configuration
├── .env.example          # Environment configuration template
├── .gitignore            # Git ignore rules
└── README.md             # Documentation
```

## Development

### Running in Development Mode
```bash
go run main.go
```

### Building for Production
```bash
go build -ldflags="-w -s" -o whatsgo .
```

### Running Tests
```bash
go test ./...
```

## Contributing

1. Fork repository
2. Buat feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit changes (`git commit -m 'Add some AmazingFeature'`)
4. Push ke branch (`git push origin feature/AmazingFeature`)
5. Buat Pull Request

## License

This project is licensed under the MIT License.

## Troubleshooting

### Common Issues

1. **QR Code tidak muncul**: Pastikan session belum login dan dalam status connecting
2. **Database locked**: Pastikan tidak ada instance lain yang menggunakan database yang sama
3. **Container tidak start**: Periksa logs dengan `docker-compose logs whatsgo`

### Logs

Untuk melihat logs aplikasi:
```bash
# Docker
docker-compose logs -f whatsgo

# Manual
tail -f logs/whatsgo.log
```

## Support

Jika Anda menemukan bug atau memiliki pertanyaan, silakan buat issue di repository GitHub.
