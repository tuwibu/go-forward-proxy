# Go Forward Proxy

Hệ thống proxy trung gian để duy trì kết nối ổn định (IP:port cố định) khi upstream proxy services (TMProxy, KiotProxy) thay đổi IP.

## Tính năng

- **Proxy Middleware**: Mỗi proxy trong database có một dumbproxy instance chạy trên port = id + 10000
- **Auto-Reset**: Tự động reset proxy theo khoảng thời gian cấu hình (min_time_reset)
- **Multiple Services**: Hỗ trợ TMProxy và KiotProxy
- **REST API**: Quản lý proxies qua HTTP API với Basic Authentication
- **Export**: Export danh sách proxies dưới dạng text

## Cài đặt

### Yêu cầu

- Go 1.24+
- SQLite3

### Build

```bash
go build -o bin/server.exe ./cmd/server
```

## Cấu hình

Copy `.env.example` thành `.env` và cấu hình:

```env
# Server Configuration
SERVER_IP=localhost
API_PORT=8080

# Authentication (for API and Proxy instances)
USERNAME=admin
PASSWORD=secure123

# Database
DATABASE_PATH=./data/proxies.db

# Optional - Auto-reset check interval (seconds)
AUTO_RESET_INTERVAL=10
```

## Chạy

```bash
./bin/server.exe
```

Hoặc trực tiếp:

```bash
go run ./cmd/server/main.go
```

## API Endpoints

Tất cả endpoints yêu cầu Basic Authentication (USERNAME:PASSWORD từ .env).

### 1. Tạo Proxy

```bash
POST /api/proxies
Content-Type: application/json
Authorization: Basic base64(admin:secure123)

{
  "api_key": "your_api_key_here",
  "service_type": "tmproxy",
  "min_time_reset": 3600
}
```

**Response:**

```json
{
  "id": 1,
  "proxy_str": "1.2.3.4:8080:username:password",
  "api_key": "your_api_key_here",
  "service_type": "tmproxy",
  "min_time_reset": 3600,
  "last_reset_at": "2025-12-12T14:00:00Z",
  "created_at": "2025-12-12T14:00:00Z"
}
```

**Service Types:**
- `tmproxy` - TMProxy service
- `kiotproxy` - KiotProxy service

### 2. Xóa Proxy

```bash
DELETE /api/proxies/:id
Authorization: Basic base64(admin:secure123)
```

### 3. Danh sách Proxies

```bash
GET /api/proxies
Authorization: Basic base64(admin:secure123)
```

**Response:**

```json
[
  {
    "id": 1,
    "proxy_str": "1.2.3.4:8080:username:password",
    "api_key": "your_api_key_here",
    "service_type": "tmproxy",
    "min_time_reset": 3600,
    "last_reset_at": "2025-12-12T14:00:00Z",
    "created_at": "2025-12-12T14:00:00Z"
  }
]
```

### 4. Export Text

```bash
GET /api/export
Authorization: Basic base64(admin:secure123)
```

**Response (text/plain):**

```
localhost:10001
localhost:10002
localhost:10003
```

Format: `{SERVER_IP}:{id+10000}`

## Sử dụng Proxy

Sau khi tạo proxy với ID = 1, bạn có thể sử dụng proxy tại:

- **Host**: `localhost` (hoặc SERVER_IP từ .env)
- **Port**: `10001` (ID + 10000)
- **Username**: `admin` (từ .env)
- **Password**: `secure123` (từ .env)

### Ví dụ với curl:

```bash
curl -x http://admin:secure123@localhost:10001 https://api.ipify.org
```

### Ví dụ với Python:

```python
import requests

proxies = {
    'http': 'http://admin:secure123@localhost:10001',
    'https': 'http://admin:secure123@localhost:10001',
}

response = requests.get('https://api.ipify.org', proxies=proxies)
print(response.text)
```

## Auto-Reset

Hệ thống tự động kiểm tra mỗi `AUTO_RESET_INTERVAL` giây (mặc định: 10 giây).

Với mỗi proxy:
- Nếu `(now - last_reset_at) >= min_time_reset`: Gọi API GetNewProxy
- Update `proxy_str` và `last_reset_at` trong database
- Update upstream của running dumbproxy instance

## Cấu trúc Project

```
go-forward-proxy/
├── cmd/
│   └── server/
│       └── main.go              # Entry point
├── internal/
│   ├── api/                     # REST API
│   ├── config/                  # Configuration loader
│   ├── database/                # Database layer
│   ├── proxymanager/            # Proxy instance management
│   └── proxyservices/           # TMProxy/KiotProxy clients
├── pkg/
│   └── dumbproxy/               # Embedded dumbproxy library
├── data/                        # SQLite database
├── .env                         # Environment variables
└── README.md
```

## Troubleshooting

### Port đã được sử dụng

Nếu port (id + 10000) đã được sử dụng, xóa proxy đó và tạo lại với ID khác.

### Proxy không hoạt động

- Kiểm tra logs để xem lỗi
- Verify API key của TMProxy/KiotProxy còn hợp lệ
- Kiểm tra proxy_str trong database có đúng format không

### Auto-reset không hoạt động

- Kiểm tra `AUTO_RESET_INTERVAL` và `min_time_reset`
- Xem logs để biết lỗi khi gọi GetNewProxy API

## License

MIT License
