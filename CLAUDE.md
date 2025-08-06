# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Chinese IoT device Go SDK (基于 Go 语言的物联网设备 SDK) designed for connecting IoT devices to IoT platforms with **C SDK compatibility** as a core design principle. The SDK provides triple authentication, MQTT connectivity, TLS security, dynamic registration, and RRPC functionality.

## Development Commands

This project uses standard Go tooling:

```bash
# Build the entire SDK
go build ./...

# Run tests 
go test ./...

# Install/update dependencies
go mod download && go mod tidy

# Run specific examples
cd examples/basic_mqtt && go run main.go
cd examples/tls_mqtt && go run main.go
cd examples/dynreg_http && go run main.go
cd examples/dynreg_mqtt && go run main.go
cd examples/rrpc && go run main.go
```

## Architecture Overview

The SDK follows a layered architecture with clear separation of concerns:

**Configuration Layer** (`pkg/config/`): Centralized configuration with environment variable support (`IOT_*` prefixed). Automatically detects secure mode (TLS → securemode=2, non-TLS → securemode=3).

**Authentication Layer** (`pkg/auth/`): C SDK-compatible credential generation using HMAC-SHA256. Generates ClientID in exact C SDK format: `ProductKey.DeviceName|timestamp=xxx,_ss=1,_v=xxx,securemode=x,signmethod=hmacsha256,ext=3,|`

**MQTT Layer** (`pkg/mqtt/`): Thread-safe MQTT client built on Paho with automatic reconnection, TLS support, and topic-specific message routing.

**Dynamic Registration** (`pkg/dynreg/`): Both HTTP and MQTT-based device registration for automatic credential acquisition.

**RRPC Layer** (`pkg/rrpc/`): Remote procedure call implementation with method handlers, request/response correlation, and timeout handling.

**TLS Layer** (`pkg/tls/`): Certificate management with built-in CA certificate and custom certificate support.

## Key Design Patterns

**C SDK Compatibility**: All authentication algorithms, ClientID formats, and message structures exactly match the C SDK implementation. Uses fixed timestamp (`2524608000000`) for consistency.

**Thread Safety**: Extensive use of `sync.RWMutex` throughout all components, with channel-based async communication.

**Configuration-Driven**: Single config struct flows through all components, with comprehensive environment variable override support.

**Callback Architecture**: Function-based handlers (`MessageHandler`, `RequestHandler`) for flexible message processing and RPC method registration.

## Required Configuration

Device credentials are mandatory:
- `ProductKey` and `DeviceName` (always required)
- Either `DeviceSecret` OR `ProductSecret` (for dynamic registration)

Environment variables supported:
```
IOT_PRODUCT_KEY, IOT_DEVICE_NAME, IOT_DEVICE_SECRET
IOT_MQTT_HOST, IOT_MQTT_PORT, IOT_MQTT_USE_TLS
IOT_MQTT_SECURE_MODE, IOT_TLS_SKIP_VERIFY
```

## Dependencies

- Go 1.21+
- `github.com/eclipse/paho.mqtt.golang v1.4.3` (core MQTT functionality)

## TLS Configuration

The SDK supports TLS connections with special handling for production environments:

**Important Network Configuration**:
- Port 1883 (non-TLS): Use IP `121.40.253.224`
- Port 8883 (TLS): Use IP `121.41.43.80` (SSL offload configuration)

**TLS Certificate Issues**:
1. The server uses a **self-signed certificate** issued for CN="IoT", not for IP addresses
2. Go's TLS validation is stricter than C SDK
3. When connecting via IP to a certificate issued for a domain name, certificate validation will fail

**Recommended TLS Configurations**:

1. **For Testing/Development** (with self-signed certificates):
```go
cfg.TLS.SkipVerify = true  // Skip certificate verification
```

2. **For Production** (with proper certificates):
```go
cfg.TLS.SkipVerify = false
cfg.TLS.ServerName = "IoT"  // Must match certificate CN
```

**Known Issues**:
- The SDK's TLS logic in `pkg/mqtt/client.go` attempts to handle IP connections with domain certificates by setting `InsecureSkipVerify=true` when `ServerName` is set, but this may not work correctly with self-signed certificates
- For self-signed certificates, always use `SkipVerify = true`

## MQTT Topic Wildcard Support

**Critical Issue Fixed**: The MQTT client now supports wildcard topic subscriptions (`+` and `#`). Without this, RRPC and other wildcard-based subscriptions will fail.

**Implementation**: The client implements a `topicMatches` function that handles MQTT wildcard matching for message routing to the correct handlers.

## Debugging and Process Management

### ⚠️ Critical: ClientID Conflict Issue

**Problem**: When debugging IoT applications, if you start a program in the background (using `&`) and don't properly kill it, the device will remain connected to the IoT platform. When you run the program again with the same ClientID, the two instances will kick each other offline repeatedly, causing connection instability.

**Symptoms**:
- Connection immediately drops after subscribing to topics
- Error messages like "Connection lost: EOF" or "Disconnected from MQTT broker"
- Device status flashing online/offline on IoT platform
- Subscribe failures with "not currently connected and ResumeSubs not set"
- Topics with `$` prefix causing immediate disconnection

**Root Cause**: MQTT brokers only allow one connection per ClientID. When a new connection with the same ClientID connects, the broker disconnects the old one. If the old client has auto-reconnect enabled, it will reconnect and kick off the new one, creating a kick-off loop.

**Solution**: Always check for and kill existing connections before starting a new instance:

```bash
# Quick cleanup before running
lsof -i -P | grep -E "(1883|121\.40\.253\.224)" | grep -v LISTEN | awk '{print $2}' | xargs -r kill -9 2>/dev/null || true

# Then run your program
go run main.go
```

### Finding Background IoT Connections

**IMPORTANT**: Always check for existing connections before running a new instance. When IoT devices appear to stay online after closing programs, use these commands to find and terminate background processes:

**1. Find processes by IoT platform ports:**
```bash
# Check active connections to IoT platform (most useful)
lsof -i -P | grep -E "(8883|1883|121\.41\.43\.80|121\.40\.253\.224)" | grep -v LISTEN

# Check network connections by port
netstat -tulpn | grep -E "(:8883|:1883)"

# Alternative using ss command
ss -tulpn | grep -E "(:8883|:1883)"
```

**2. Find Go processes by device identifiers:**
```bash
# Search for processes containing device credentials
ps aux | grep -E "(QLTMkOfW|WjJjXbP0X1|THYYENG5wd)" | grep -v grep

# Find all Go-related processes
ps aux | grep "go" | grep -v grep | grep -v gopls
```

**3. Find all running Go programs:**
```bash
# Find go run processes
ps aux | grep -E "(go run|main\.go)" | grep -v grep

# Find compiled Go binaries in cache
ps aux | grep "/go-build/" | grep -v grep
```

**4. Kill processes safely:**
```bash
# Kill specific process by PID
kill -15 <PID>  # Try graceful termination first
kill -9 <PID>   # Force kill if needed

# Kill all Go processes (use with caution)
pkill -f "go run"
```

**5. Comprehensive IoT connection check:**
```bash
# One-liner to find all IoT-related connections
lsof -i -P | grep -E "(8883|1883)" | grep -v LISTEN && ps aux | grep -E "(go run|/go-build/)" | grep -v grep
```

### Common Issues

- Programs started from editors (VSCode, Cursor) may continue running in background
- Go programs compiled to `/home/pi/.cache/go-build/` may not be obvious from process names
- Always use `Ctrl+C` to properly exit programs instead of closing terminal windows

## Dynamic Registration (MQTT)

**Important Implementation Details**:

The MQTT dynamic registration process differs from regular MQTT connections and has specific requirements:

### Authentication Format
1. **ClientID**: `deviceName.productKey|random={random},authType={authType},securemode=2,signmethod=hmacsha256|`
   - Note: deviceName comes FIRST, then productKey
   - authType: "register" for whitelist mode, "regnwl" for non-whitelist mode
   - random: Use random number, NOT timestamp

2. **Username**: `deviceName&productKey`

3. **Password**: HMAC-SHA256 signature with **UPPERCASE** hex encoding
   - Sign content: `deviceName{deviceName}productKey{productKey}random{random}`
   - Sign with ProductSecret (NOT DeviceSecret)

### Message Flow
**Critical**: Dynamic registration does NOT require manual topic subscription!

1. Client connects to MQTT broker with special credentials
2. Server automatically subscribes client to `/ext/register/{productKey}/{deviceName}`
3. Server pushes registration result immediately after connection
4. Response format: `{"deviceSecret":"xxx"}` (direct JSON, no wrapper)

### Common Pitfalls
- **Wrong auth type**: Make sure skipPreRegist flag matches your platform configuration
- **Case sensitivity**: Password MUST be uppercase hex (C SDK compatibility)
- **No manual subscription**: Server handles subscription automatically
- **ProductSecret vs DeviceSecret**: Use ProductSecret for dynamic registration authentication

### Example Success Log
```
Dynamic registration connecting with ClientID: deviceName.productKey|random=xxx,authType=register,securemode=2,signmethod=hmacsha256|
Connected to MQTT broker for dynamic registration: ssl://121.41.43.80:8883
Received message on topic /ext/register: {"deviceSecret":"xxx"}
```

## IoT Framework (New)

The project now includes an event-driven framework in `pkg/framework/` that provides higher-level abstractions:

### Framework vs SDK
- **SDK**: Direct MQTT/HTTP operations, full control, more code
- **Framework**: Event-driven, plugin-based, business logic focused, minimal code

### Thing Model Topics

The framework uses standard Thing Model topics with `$SYS/` prefix:

**Property Operations**:
- Upload: `$SYS/{ProductKey}/{DeviceName}/property/post`
- Upload Reply: `$SYS/{ProductKey}/{DeviceName}/property/post/reply`
- Set: `$SYS/{ProductKey}/{DeviceName}/property/set`
- Set Reply: `$SYS/{ProductKey}/{DeviceName}/property/set/reply`

**Property Message Format**:
```json
{
  "id": "1754475911",
  "version": "1.0",
  "params": {
    "temperature": {
      "value": "25.5",
      "time": 1754475911
    },
    "humidity": {
      "value": "60.0",
      "time": 1754475911
    }
  }
}
```

**Event Operations**:
- Upload: `$SYS/{ProductKey}/{DeviceName}/event/post`
- Upload Reply: `$SYS/{ProductKey}/{DeviceName}/event/post/reply`

**Service Operations**:
- Invoke: `$SYS/{ProductKey}/{DeviceName}/service/{serviceName}/invoke`
- Invoke Reply: `$SYS/{ProductKey}/{DeviceName}/service/{serviceName}/invoke/reply`

### Framework Example

See `examples/framework/simple/` for a complete smart sensor example using the framework.

## Current Limitations

- No automated test suite exists
- No CI/CD pipeline configured
- Documentation primarily in Chinese
- No custom build scripts or Makefile
- Background process management requires manual checking
- Topics with `$` prefix may cause connection issues with some MQTT broker configurations