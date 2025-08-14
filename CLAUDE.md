# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Chinese IoT device Go SDK (基于 Go 语言的物联网设备 SDK) designed for connecting IoT devices to IoT platforms with **C SDK compatibility** as a core design principle. The SDK provides triple authentication, MQTT connectivity, TLS security, dynamic registration, RRPC functionality, and a high-level event-driven framework.

## Development Commands

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
cd examples/framework/simple && go run .

# Build and run framework example (electric oven)
cd examples/framework/simple
go build -o oven .
./oven 2>&1 | tee /tmp/oven_rrpc.log

# Test RRPC functionality
cd examples/framework/simple/test_scripts
./test_rrpc_api.sh  # Test with actual API
python3 test_rrpc.py  # Python test suite (requires: pip install requests)
./test_rrpc_debug.sh  # Debug connection issues
```

## Architecture Overview

The SDK follows a layered architecture with clear separation of concerns:

### Core SDK Layers

**Configuration Layer** (`pkg/config/`): Centralized configuration with environment variable support (`IOT_*` prefixed). Automatically detects secure mode (TLS → securemode=2, non-TLS → securemode=3).

**Authentication Layer** (`pkg/auth/`): C SDK-compatible credential generation using HMAC-SHA256. Generates ClientID in exact C SDK format: `ProductKey.DeviceName|timestamp=xxx,_ss=1,_v=xxx,securemode=x,signmethod=hmacsha256,ext=3,|`

**MQTT Layer** (`pkg/mqtt/`): Thread-safe MQTT client built on Paho with automatic reconnection, TLS support, and topic-specific message routing. Supports wildcard subscriptions (`+`, `#`).

**Dynamic Registration** (`pkg/dynreg/`): Both HTTP and MQTT-based device registration for automatic credential acquisition.

**RRPC Layer** (`pkg/rrpc/`): Remote procedure call implementation with method handlers, request/response correlation, and timeout handling. Integrated with framework via MQTT plugin.

**TLS Layer** (`pkg/tls/`): Certificate management with built-in CA certificate and custom certificate support.

### Framework Architecture (`pkg/framework/`)

The framework provides a higher-level abstraction on top of the SDK:

**Event-Driven Core** (`pkg/framework/core/`): 
- `Framework`: Main orchestrator managing devices, plugins, and lifecycle
- `EventBus`: Asynchronous event processing with worker pools (default 10 workers)
- `DeviceManager`: Device registration and lifecycle management
- `PluginManager`: Dynamic plugin loading and management

**Event System** (`pkg/framework/event/`):
- Predefined event types: `system.*`, `property.*`, `service.*`, `event.*`
- Priority-based handler execution
- Synchronous and asynchronous event processing

**Device Abstraction** (`pkg/framework/device/`):
- `BaseDevice`: Common device implementation with property/service/event support
- Thing Model integration: Properties (R/W), Services (invoke/response), Events (report)

**Plugin System** (`pkg/framework/plugins/`):
- `mqtt/`: MQTT plugin integrating SDK client with framework
  - Handles Thing Model topics (`$SYS/` prefix)
  - RRPC integration via embedded client
  - Service routing fix for correct topic parsing (parts[4] not parts[5])

## Key Design Patterns

**C SDK Compatibility**: All authentication algorithms, ClientID formats, and message structures exactly match the C SDK implementation. Uses fixed timestamp (`2524608000000`) for consistency.

**Thread Safety**: Extensive use of `sync.RWMutex` throughout all components, with channel-based async communication.

**Event-Driven Pattern**: Framework uses event bus for loose coupling between components. Events flow: Device → Framework → EventBus → Plugin → IoT Platform.

**Plugin Lifecycle**: Plugins follow Initialize → Start → Stop lifecycle, managed by PluginManager. MQTT plugin auto-starts RRPC client during Start phase.

**Thing Model Mapping**:
- Properties: Auto-tracked with R/W modes, periodic reporting (30s idle, 2s active)
- Services: Registered handlers with request/response pattern
- Events: Custom event types with timestamp and data payload

## IoT Framework Thing Model

The framework implements standard Thing Model with `$SYS/` prefix topics:

### Property Operations
- **Upload**: `$SYS/{ProductKey}/{DeviceName}/property/post`
- **Set**: `$SYS/{ProductKey}/{DeviceName}/property/set`

**Property Message Format**:
```json
{
  "id": "1754475911",
  "version": "1.0",
  "params": {
    "temperature": {
      "value": "25.5",
      "time": 1754475911
    }
  }
}
```

### Event Operations
- **Upload**: `$SYS/{ProductKey}/{DeviceName}/event/post`

**Event Format**: `{"eventType": "timer_complete", "timestamp": 1234567890, "data": {}}`

### Service Operations
- **Invoke**: `$SYS/{ProductKey}/{DeviceName}/service/{serviceName}/invoke`

**Service Routing Fix**: Service topic parsing was fixed in `mqtt_plugin.go:248` to use `parts[4]` instead of `parts[5]`.

### RRPC Integration

RRPC is integrated into MQTT plugin and auto-starts after framework initialization:

**Topics**:
- Request: `/sys/{productKey}/{deviceName}/rrpc/request/{requestId}`
- Response: `/sys/{productKey}/{deviceName}/rrpc/response/{requestId}`

**Handler Registration** (MUST be after `framework.Start()`)
```go
mqttPlugin.RegisterRRPCHandler("MethodName", func(requestId string, payload []byte) ([]byte, error) {
    // Handle request and return response
})
```

**API Integration**: RRPC uses Base64 encoding for API calls. Platform handles encoding/decoding automatically.

### Framework Example - Electric Oven

`examples/framework/simple/` contains a complete smart oven implementation:
- **Properties**: temperature, heater_status, door_status, timer_setting, firmware_version, firmware_module
- **Services**: set_temperature, start_timer, toggle_door
- **Events**: timer_complete, overheat_warning
- **RRPC Methods**: GetOvenStatus, SetOvenTemperature, EmergencyStop
- **OTA Integration**: Framework OTA plugin with multi-module support, automatic version checking
- **Dynamic Behavior**: Adjusts reporting frequency based on activity

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

## TLS Configuration

**Important Network Configuration**:
- Port 1883 (non-TLS): Use IP `121.40.253.224`
- Port 8883 (TLS): Use IP `121.41.43.80` (SSL offload configuration)

**TLS Certificate Issues**:
1. Server uses self-signed certificate for CN="IoT"
2. Go's TLS validation is stricter than C SDK
3. IP connections to domain certificates will fail validation

**Recommended TLS Configurations**:

Testing/Development:
```go
cfg.TLS.SkipVerify = true  // Skip certificate verification
```

Production:
```go
cfg.TLS.SkipVerify = false
cfg.TLS.ServerName = "IoT"  // Must match certificate CN
```

## Debugging and Process Management

### ⚠️ Critical: ClientID Conflict Issue

**Problem**: Background processes with same ClientID cause connection kick-off loops.

**Solution**: Kill existing connections before running:
```bash
# Quick cleanup
lsof -i -P | grep -E "(1883|121\.40\.253\.224)" | grep -v LISTEN | awk '{print $2}' | xargs -r kill -9 2>/dev/null || true

# Then run
go run main.go
```

### Finding Background Connections

```bash
# Check active IoT connections
lsof -i -P | grep -E "(8883|1883|121\.41\.43\.80|121\.40\.253\.224)" | grep -v LISTEN

# Find Go processes by device ID
ps aux | grep -E "(QLTMkOfW|S4Wj7RZ5TO)" | grep -v grep

# Kill specific process
kill -15 <PID>  # Graceful
kill -9 <PID>   # Force
```

## Dynamic Registration (MQTT)

### Authentication Format
1. **ClientID**: `deviceName.productKey|random={random},authType={authType},securemode=2,signmethod=hmacsha256|`
   - authType: "register" (whitelist) or "regnwl" (non-whitelist)
   - random: Random number, NOT timestamp

2. **Username**: `deviceName&productKey`

3. **Password**: HMAC-SHA256 signature with **UPPERCASE** hex
   - Sign: `deviceName{deviceName}productKey{productKey}random{random}`
   - Use ProductSecret (NOT DeviceSecret)

### Message Flow
1. Connect with special credentials
2. Server auto-subscribes to `/ext/register/{productKey}/{deviceName}`
3. Server pushes result: `{"deviceSecret":"xxx"}`
4. No manual subscription needed!

## Current Status

### Completed Features (v1.0.0)
- ✅ Full C SDK compatibility
- ✅ MQTT with TLS and auto-reconnect
- ✅ HTTP/MQTT dynamic registration
- ✅ RRPC with Base64 encoding
- ✅ Event-driven framework with Thing Model
- ✅ Service routing fixed
- ✅ RRPC framework integration
- ✅ OTA framework plugin with multi-module support

### Pending Framework Features
- **P1**: Service response mechanism
- **P2**: Batch property operations, property query
- **P3**: Shadow device, device grouping
- **P4**: Rule engine, data persistence, offline caching

### Known Issues
- No automated test suite
- No CI/CD pipeline
- Topics with `$` prefix may cause issues with some brokers
- Background process management requires manual checking

## Dependencies

- Go 1.21+
- `github.com/eclipse/paho.mqtt.golang v1.4.3`