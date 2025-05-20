# Enrichment Service Certification Suite

This directory contains a test suite to validate implementations of the OpenX Enrichment Service specification. The suite uses k6 for load testing and validation.

## Prerequisites

- Docker
- Docker Compose

## Running the Tests

### Testing a Local Service

If your service is running locally on port 8080:

```bash
make cert
```

### Testing a Remote Service

To test a service running on a different host or port:

```bash
make cert-remote CERT_HOST=your-service-host CERT_PORT=your-service-port
```

For example, to test a service running on `192.168.1.100:9090`:

```bash
make cert-remote CERT_HOST=192.168.1.100 CERT_PORT=9090
```

### Testing with HTTPS

To test a service using HTTPS:

```bash
make cert-remote CERT_HOST=your-service-host CERT_PORT=443 USE_HTTPS=true
```

Note: The test suite will skip TLS verification in the test environment. In production, proper TLS verification should be enabled.

### Load Testing Options (Test Harness Only)

When using our reference implementation, you can simulate various load conditions:

```bash
# Enable latency simulation
make cert SIMULATE_LATENCY=true

# Enable CPU load simulation
make cert SIMULATE_CPU_LOAD=true

# Customize latency parameters
make cert SIMULATE_LATENCY=true LATENCY_MEAN_MS=10 LATENCY_STDDEV_MS=2

# Customize CPU load
make cert SIMULATE_CPU_LOAD=true CPU_LOAD_PERCENTAGE=75
```

Available options:
- `SIMULATE_LATENCY`: Enable/disable latency simulation (default: false)
- `LATENCY_MEAN_MS`: Mean latency in milliseconds (default: 5)
- `LATENCY_STDDEV_MS`: Standard deviation of latency in milliseconds (default: 5)
- `SIMULATE_CPU_LOAD`: Enable/disable CPU load simulation (default: false)
- `CPU_LOAD_PERCENTAGE`: Target CPU load percentage (default: 50)

Note: These options are specific to our reference implementation and are not part of the actual service specification. They are provided to help test the certification suite under various conditions.

## Test Scenarios

The suite includes the following test scenarios:

### OpenRTB Validation (`openrtb.js`)
- Validates the `/openrtb25` endpoint
- Checks response time < 5ms p95 (as per spec)
- Verifies response structure and allowed fields
- Tests with synthetic bid requests
- Includes periodic health checks (`/healthz`) to monitor service availability
  - Verifies 200 OK response
  - Checks response time < 100ms p95

## Performance Requirements

The test suite validates against these requirements from the specification:
- Average latency < 2ms
- p95 latency < 5ms
- 10ms timeout

## Customization

You can modify the test parameters in each `.js` file:
- `vus`: Virtual Users (concurrent connections)
- `duration`: Test duration
- `thresholds`: Performance thresholds

## Environment Variables

- `TARGET_HOST`: Hostname or IP of the service to test (default: localhost)
- `TARGET_PORT`: Port number of the service to test (default: 8080)
- `USE_HTTPS`: Whether to use HTTPS for connections (default: false) 