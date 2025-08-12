# Fuego API Testing Framework

Fuego is a comprehensive API testing framework written in Go, designed to be a modern, powerful tool for API testing and validation.

## Features

- **Flexible Test Scenarios** - Support for both simple linear scenarios and complex test suites with parallel execution
- **Multiple Formats** - Both legacy single-scenario format and modern test group format
- **HTTP/REST Support** - Full method support, headers, query parameters, authentication, and body handling
- **Variable System** - Global, local, and step-scoped variables with template interpolation
- **Request Chaining** - Capture data from responses and use in subsequent requests
- **Comprehensive Assertions** - Status codes, headers, JSON path, regex, performance checks, and more
- **Multiple Output Formats** - Console, JSON, HTML, and Markdown reports
- **Environment Support** - Environment-specific configurations for dev/staging/prod
- **Parallel Execution** - Run test groups concurrently for faster feedback
- **CI/CD Ready** - Designed for seamless integration into pipelines

## Quick Start

### Installation

```bash
go build -o fuego ./cmd/fuego
```

### Basic Usage

#### Simple Scenario Format
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

#### Advanced Test Suite Format
Create a test suite with parallel execution and request chaining:

```yaml
version: "1.1"
name: "API Testing Suite"

env:
  host: jsonplaceholder.typicode.com
  protocol: https
  userID: 1

config:
  parallel: true
  http:
    timeout: 30s

before:
  steps:
    - name: API Health Check
      http:
        url: ${{env.protocol}}://${{env.host}}/posts/1
        method: GET
      check:
        status: 200

tests:
  user_operations:
    name: "User Operations"
    steps:
      - name: Get User Profile
        http:
          url: ${{env.protocol}}://${{env.host}}/users/${{env.userID}}
          method: GET
        capture:
          userName:
            jsonpath: $.name
        check:
          status: 200

  crud_operations:
    name: "CRUD Operations"
    steps:
      - name: Create Post
        http:
          url: ${{env.protocol}}://${{env.host}}/posts
          method: POST
          json:
            title: "Test Post"
            body: "Test content"
            userId: ${{env.userID}}
        capture:
          postID:
            jsonpath: $.id
        check:
          status: 201

after:
  steps:
    - name: Cleanup Check
      http:
        url: ${{env.protocol}}://${{env.host}}/posts/1
        method: GET
      check:
        status: 200
```

Run the tests:

```bash
./fuego run test.yaml
./fuego run api-suite.yaml
```

### Command Line Options

```bash
# Run with verbose output
./fuego run --verbose test.yaml

# Run tests from a directory
./fuego run tests/

# Generate JSON report
./fuego run --format json --output report.json test.yaml

# Generate HTML report
./fuego run --format html --output report.html test.yaml

# Use specific environment
./fuego run --env development test.yaml
```

## Scenario Structure

### Simple Scenario Structure (Legacy Format)

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

### Test Suite Structure (Modern Format)

```yaml
version: "1.1"
name: "Test Suite Name"

env:
  host: api.example.com
  
config:
  parallel: true
  http:
    timeout: 30s

before:
  steps:
    - name: "Setup Step"
      http:
        url: "https://{{env.host}}/health"
        method: GET

tests:
  test_group_1:
    name: "User Tests"
    steps:
      - name: "Get User"
        http:
          url: "https://{{env.host}}/users/1"
          method: GET
        capture:
          userID:
            jsonpath: $.id
        check:
          status: 200

after:
  steps:
    - name: "Cleanup"
      http:
        url: "https://{{env.host}}/cleanup"
        method: POST
```

### Variable Interpolation

Fuego supports two variable interpolation syntaxes:

```yaml
# Standard syntax
request:
  url: "{{base_url}}/users/{{user_id}}"
  headers:
    Authorization: "Bearer {{token}}"

# Environment variable syntax (modern format)
request:
  url: "${{env.protocol}}://${{env.host}}/users/${{env.userID}}"
  headers:
    Authorization: "Bearer ${{capturedToken}}"
```

### Request Chaining with Captures

Extract data from responses for use in subsequent requests:

```yaml
- name: "Login"
  http:
    url: "/auth/login"
    method: POST
    json:
      username: "user"
      password: "pass"
  capture:
    authToken:
      jsonpath: $.token
    userID:
      jsonpath: $.user.id

- name: "Get User Profile"
  http:
    url: "/users/${{userID}}"
    method: GET
    headers:
      Authorization: "Bearer ${{authToken}}"
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
