# Modbus PI Aveva Connector IIoT

![Go](https://img.shields.io/badge/Go-1.25.4-007D9C?style=for-the-badge&logo=go&logoColor=white)
![PI Web API](https://img.shields.io/badge/PI_Web_API-2025-blue?style=for-the-badge)
![Modbus](https://img.shields.io/badge/Modbus-TCP-green?style=for-the-badge)
![License](https://img.shields.io/badge/license-MIT-blue.svg?style=for-the-badge)

**Modbus PI Aveva Connector IIoT** is a high-performance, production-ready data connector built with Go that bridges Modbus TCP devices to AVEVA PI System. It enables seamless data ingestion from industrial sensors and PLCs into the PI Data Archive and PI Cloud Services.

## 🚀 Features

- **Dual-Protocol Support**: Native Modbus TCP client with automatic data type detection (int16, uint16, int32, uint32, float32, float64)
- **PI System Integration**: Full support for PI Web API with automatic tag creation and data ingestion
- **High Availability**: Automatic failover between primary and secondary PI Web API servers
- **Smart Tag Management**: Automatic tag creation with proper naming conventions and data type mapping
- **Error Handling**: Comprehensive error handling with retry mechanisms and circuit breaker patterns
- **Configuration**: Flexible configuration via YAML files with environment variable overrides
- **Monitoring**: Built-in health checks and metrics collection
- **Security**: Secure credential management and TLS support

## 📋 Prerequisites

- **Go 1.25.4** or higher
- **AVEVA PI Web API** (2025 recommended) - accessible from the connector host
- **Modbus TCP Devices** - with static IP addresses
- **Network Connectivity** - between connector and both PI Web API servers

## 🛠️ Installation

### Option 1: Download Pre-compiled Binary

1. Download the latest release from [GitHub Releases](https://github.com/yourusername/modbus-pi-aveva-connector-IIoT/releases)
2. Extract the binary to your desired location

### Option 2: Build from Source

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/modbus-pi-aveva-connector-IIoT.git
   cd modbus-pi-aveva-connector-IIoT
   ```

2. Build the binary:
   ```bash
   go build -o modbus-connector ./cmd/main.go
   ```

## ⚙️ Configuration

The connector uses a YAML configuration file located at `config/config.yaml`. You can override any setting using environment variables.

### Configuration Options

```yaml
# PI System Configuration
pisystem:
  primary:
    url: "https://piwebapi.example.com"
    username: "admin"
    password: "[PASSWORD]"
    verify_ssl: false
  secondary:
    url: "https://piwebapi-secondary.example.com"
    username: "admin"
    password: "[PASSWORD]"
    verify_ssl: false

# Modbus Configuration
modbus:
  gateways:
    - address: "[IP_ADDRESS]"
      port: 502
      unit_id: 1
      scan_rate: 1s
      timeout: 5s
      retries: 3
      data_types:
        - register: 100
          name: "Temperature"
          data_type: "float32"
          description: "Process temperature"
          units: "°C"
        - register: 102
          name: "Pressure"
          data_type: "float32"
          description: "Process pressure"
          units: "PSI"
    - address: "[IP_ADDRESS]"
      port: 502
      unit_id: 1
      scan_rate: 2s
      timeout: 5s
      retries: 3
      data_types:
        - register: 200
          name: "FlowRate"
          data_type: "float32"
          description: "Fluid flow rate"
          units: "m³/h"

# Logging Configuration
logging:
  level: "info"
  format: "json"
  output: "stdout"
```

### Environment Variable Overrides

You can override any configuration value using environment variables with the prefix `MODBUS_PI_`:

```bash
export MODBUS_PI_PISYSTEM_PRIMARY_URL="https://new-piwebapi.example.com"
export MODBUS_PI_MODBUS_GATEWAYS_0_ADDRESS="[IP_ADDRESS]"
export MODBUS_PI_MODBUS_GATEWAYS_0_DATA_TYPES_0_NAME="NewTagName"

./modbus-connector
```

## 🏃 Usage

### Start the Connector

```bash
./modbus-connector
```

### Run in Background (Linux/macOS)

```bash
nohup ./modbus-connector > connector.log 2>&1 &
```

### Stop the Connector

Press `Ctrl+C` to stop the connector gracefully.

## 📂 Project Structure

```
modbus-pi-aveva-connector-IIoT/
├── cmd/
│   └── main.go             # Application entry point
├── config/
│   └── config.yaml         # Default configuration
├── internal/
│   ├── domain/             # Core business logic
│   │   ├── processor.go    # Data processing pipeline
│   │   ├── modbus.go       # Modbus client implementation
│   │   └── pi.go           # PI Web API client implementation
│   ├── infrastructure/     # Infrastructure components
│   │   ├── config.go       # Configuration loader
│   │   ├── logger.go       # Logging setup
│   │   └── health.go       # Health check implementation
│   └── shared/             # Shared utilities
│       └── types.go        # Data structures
├── go.mod                  # Go module definition
├── go.sum                  # Go module checksums
└── README.md               # Project documentation
```

## 🧪 Testing

### Run Unit Tests

```bash
go test ./internal/... -v
```

### Run Integration Tests

Integration tests require a running PI Web API instance. Ensure your configuration is set up correctly.

```bash
go test ./tests/integration -v
```

## 🔐 Security

- **Credential Management**: Use environment variables or a secure secrets manager for PI Web API credentials
- **TLS Verification**: Disable SSL verification (`verify_ssl: false`) only in development environments. In production, use valid certificates.
- **Network Security**: Restrict network access to the connector host only
- **Input Validation**: All configuration values are validated at startup

## 📊 Monitoring

The connector exposes a health check endpoint at `/health` that returns:

```json
{
  "status": "healthy",
  "timestamp": "2023-10-27T10:00:00Z",
  "pi_primary": {
    "status": "healthy",
    "url": "https://piwebapi.example.com"
  },
  "pi_secondary": {
    "status": "healthy",
    "url": "https://piwebapi-secondary.example.com"
  },
  "modbus_gateways": [
    {
      "address": "[IP_ADDRESS]",
      "status": "healthy"
    }
  ]
}
```

## 📝 License

This project is licensed under the **MIT License** - see the [LICENSE](LICENSE) file for details.

## 🤝 Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 👨‍💻 Authors

- **AlbusDD** - [Your GitHub Profile](https://github.com/yourusername)

## 📞 Support

For issues, questions, or feature requests, please open an issue on the [GitHub Issues](https://github.com/yourusername/modbus-pi-aveva-connector-IIoT/issues) page.

## 🙏 Acknowledgments

- **AVEVA PI System** - For providing the PI Web API platform
- **Modbus Organization** - For the Modbus TCP protocol
- **Go Community** - For the excellent Go ecosystem

---

**Built with ❤️ for Industrial IoT**