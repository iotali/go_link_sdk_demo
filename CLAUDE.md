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

## Current Limitations

- No automated test suite exists
- No CI/CD pipeline configured
- Documentation primarily in Chinese
- No custom build scripts or Makefile