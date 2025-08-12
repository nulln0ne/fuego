# Fuego API Testing Framework

Fuego is a comprehensive API testing framework written in Go, designed to be a modern alternative to tools like Postman, Insomnia, and curl.

## Features

- **Declarative test scenarios** written in YAML or JSON
- **HTTP/REST support** with full method support, headers, query parameters, and body handling
- **Variable system** with global, local, and step-scoped variables
- **Template interpolation** for dynamic request values
- **Comprehensive assertions** including status codes, headers, JSON path, regex, and more
- **Multiple output formats** - console, JSON, HTML, and Markdown reports
- **Environment support** for dev/staging/prod configurations
- **CLI interface** designed for CI/CD integration

## Quick Start

### Installation

```bash
go build -o fuego ./cmd/fuego
```

### Basic Usage

Create a test scenario file (`test.yaml`):

```yaml
version: "1.0"
name: "Simple API Test"
description: "Test JSON Placeholder API"

variables:
  base_url: "https://jsonplaceholder.typicode.com"
  user_id: 1

steps:
  - name: "Get user information"
    type: http
    request:
      method: GET
      url: "{{base_url}}/users/{{user_id}}"
      headers:
        Accept: "application/json"
    assertions:
      - type: status
        operator: eq
        value: 200
        description: "Should return HTTP 200"
      - type: json_path
        field: "id"
        operator: eq
        value: 1
        description: "User ID should be 1"
```

Run the test:

```bash
./fuego run test.yaml
```

### Command Line Options

```bash
# Run with verbose output
./fuego run --verbose test.yaml

# Run tests from a directory
./fuego run tests/

# Generate JSON report
./fuego run --format json --output report.json test.yaml

# Use specific environment
./fuego run --env development test.yaml
```

## Scenario Structure

### Basic Structure

```yaml
version: "1.0"
name: "Scenario Name"
description: "Optional description"

variables:
  key: "value"
  
steps:
  - name: "Step Name"
    type: http
    request:
      method: GET
      url: "https://api.example.com/endpoint"
    assertions:
      - type: status
        operator: eq
        value: 200
```

### Variable Interpolation

Use `{{variable_name}}` syntax in requests:

```yaml
request:
  url: "{{base_url}}/users/{{user_id}}"
  headers:
    Authorization: "Bearer {{token}}"
```

### Supported Assertion Types

- `status` - HTTP status code
- `header` - Response header value
- `body` - Response body text
- `json_path` - JSON path extraction (e.g., `user.id`, `items.0.name`)
- `regex` - Regular expression matching
- `response_time` - Response time validation
- `size` - Response size validation

### Assertion Operators

- `eq` / `equals` / `==` - Equality
- `ne` / `not_equals` / `!=` - Inequality
- `gt` / `>` - Greater than
- `gte` / `>=` - Greater than or equal
- `lt` / `<` - Less than
- `lte` / `<=` - Less than or equal
- `contains` - String contains
- `matches` / `regex` - Regular expression match
- `starts_with` - String starts with
- `ends_with` - String ends with

## Configuration

Create a `.fuego.yaml` configuration file:

```yaml
global:
  headers:
    User-Agent: "Fuego API Testing Tool/1.0"
  timeout: 30s

defaults:
  http_timeout: 30s
  max_retries: 3
  verify_ssl: true

environments:
  development:
    base_url: "https://dev.api.example.com"
  production:
    base_url: "https://api.example.com"
```

## Project Structure

```
fuego/
├── cmd/fuego/           # Main application entry point
├── pkg/
│   ├── config/          # Configuration management
│   ├── scenario/        # Test scenario definitions
│   ├── execution/       # Test execution engine
│   ├── protocols/       # Protocol implementations (HTTP, etc.)
│   ├── variables/       # Variable system and templating
│   ├── assertions/      # Assertion framework
│   └── reporting/       # Report generation
├── internal/cli/        # CLI command implementation
└── examples/            # Example scenarios and configs
```

## Development Status

This is a basic implementation of the Fuego API testing framework. Currently implemented:

- ✅ HTTP/REST protocol support
- ✅ YAML scenario parsing
- ✅ Variable interpolation
- ✅ Comprehensive assertion framework
- ✅ Multiple output formats (console, JSON, HTML, Markdown)
- ✅ CLI interface with environment support

## Future Enhancements

See `TASK.md` for the complete feature roadmap including:

- GraphQL, gRPC, WebSocket support
- Load testing capabilities
- Security testing features
- Plugin system
- Web UI interface
- Advanced monitoring and alerting