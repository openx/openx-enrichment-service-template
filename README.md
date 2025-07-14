# OpenX Enrichment Service Template

This is a reference implementation of an enrichment service that follows the [OpenX BYO Container specification](SPECIFICATION.md). It demonstrates how to implement a containerized enrichment service that can be deployed on OpenX infrastructure.

## Features

- Accepts OpenRTB BidRequest objects at `/openrtb25`
- Returns mock segments in the response
- Exposes health check endpoint at `/healthz`
- Exposes Prometheus metrics at `/metrics`
- Follows OpenX's standard patterns for metrics, logging, and configuration
- Includes a certification suite to validate implementations

## Configuration

### OpenX-Managed Configuration

These environment variables are automatically set by OpenX in a hosted deployment, and should be used by your service:

- `PORT`: HTTP port to listen on (default: 8080)
- `LOG_LEVEL`: Log level (default: "info")
- `GCS_INBOX_BUCKET`: Name of the GCS bucket for incoming files
- `GCP_REGION` and `GCP_ZONE`: the Google Cloud region and zone where your workload is running

### Service-Specific Configuration

These environment variables are used in this example service, for testing and local development:

#### GCS Access
- `GCS_CREDENTIALS`: Path to GCP credentials file (OpenX deployments will use [workload identity](https://cloud.google.com/kubernetes-engine/docs/concepts/workload-identity), so this will not be set in a hosted deployment)

#### Latency Simulation
- `SIMULATE_LATENCY`: Enable latency simulation (default: false)
- `LATENCY_MEAN_MS`: Mean latency in milliseconds (default: 2)
- `LATENCY_STDDEV_MS`: Standard deviation of latency in milliseconds (default: 1)

#### CPU Load Simulation
- `SIMULATE_CPU_LOAD`: Enable CPU load simulation (default: false)
- `CPU_LOAD_PERCENTAGE`: Percentage of CPU to use (default: 50)

#### Request Logging
- `REQUEST_LOG_THROTTLE`: Fraction of requests to log (0.0 to 1.0, default: 0.0)
  - Set to 0.01 to log 1% of requests
  - Set to 1.0 to log all requests
  - Set to 0.0 to disable request logging (default)

#### Service Behavior
- `DISABLE_ENRICHMENT`: If set to "true", always return 204 No Content (default: false)

## Local Development

For local development, we use Docker Compose to run the service with a standard configuration. The `docker-compose.yml` file includes:

- Port mapping: `8082:8080`
- GCP credentials mounted from your local machine
- Default simulation settings:
  - Latency simulation enabled with 5ms mean and 5ms standard deviation
  - CPU load simulation enabled at 50% utilization
- Debug logging enabled

### Modifying Defaults

To modify the default configuration, you can:

1. Edit the `docker-compose.yml` file directly:
   ```yaml
   services:
     enrichment-service:
       environment:
         # Adjust simulation parameters
         - SIMULATE_LATENCY=true
         - LATENCY_MEAN_MS=5
         - LATENCY_STDDEV_MS=5
         - SIMULATE_CPU_LOAD=true
         - CPU_LOAD_PERCENTAGE=50
         
         # Configure request logging
         - REQUEST_LOG_THROTTLE=0.01
         
         # Disable enrichment responses
         - DISABLE_ENRICHMENT=true
         
         # Change logging level
         - LOG_LEVEL=debug
   ```

2. Override settings using environment variables:
   ```bash
   SIMULATE_LATENCY=false docker compose up
   ```

3. Use a `.env` file to set environment variables:
   ```bash
   # Create .env file
   echo "SIMULATE_LATENCY=false" > .env
   docker compose up
   ```

### Production Deployment

For production deployment, you'll need to build and provide a container image that OpenX can deploy. Here's how:

1. Build the container image:
   ```bash
   docker build -t openx-enrichment-service-template .
   ```

2. Tag and push the image to your container registry:
   ```bash
   # Tag the image
   docker tag openx-enrichment-service-template your-registry.com/openx-enrichment-service-template:1.0.0
   
   # Push to registry
   docker push your-registry.com/openx-enrichment-service-template:1.0.0
   ```

3. Provide the following to OpenX:
   - Container image URL (e.g., `your-registry.com/openx-enrichment-service-template:1.0.0`)
   - Docker image tag that labels the production-ready version of the image (eg, `prod`, `latest`)
   - If your registry requires authentication, a username/password or Docker `config.json` file
   - Resource requirements (CPU, memory)
   - Google Cloud principal(s) that will be given access to GCS buckets
   - Enrichment endpoint (if not `/openrtb25`)
   - Health check endpoint (if not `/healthz`)
   - Metrics endpoint (if not `/metrics`)

The service will be deployed in OpenX's Kubernetes clusters with:
- Workload identity for GCS authentication
- Automatic scaling based on load
- Monitoring and alerting
- Log aggregation

For local testing of the production image, you can run:
```bash
docker run -p 8082:8080 \
  -v ~/.config/gcloud/application_default_credentials.json:/gcp/creds.json:ro \
  -e GOOGLE_APPLICATION_CREDENTIALS=/gcp/creds.json \
  -e SIMULATE_LATENCY=true \
  -e LATENCY_MEAN_MS=5 \
  -e LATENCY_STDDEV_MS=5 \
  -e SIMULATE_CPU_LOAD=true \
  -e CPU_LOAD_PERCENTAGE=50 \
  -e REQUEST_LOG_THROTTLE=0.01 \
  -e DISABLE_ENRICHMENT=true \
  -e LOG_LEVEL=debug \
  your-registry.com/openx-enrichment-service-template:1.0.0
```

## Kubernetes Deployment

When deployed to Kubernetes, the service will use workload identity for GCS authentication. No additional credential configuration is needed.

## API Documentation

For a full description of the request/response format, required endpoints, and runtime expectations, see the [OpenX Enrichment Service Specification](SPECIFICATION.md).

## Certification Suite

The repository includes a certification suite to validate that your implementation meets OpenX's requirements. The suite uses k6 for load testing and validation.

### Running the Certification Suite

1. Against a local service based on this reference implementation:
   ```bash
   make cert
   ```

2. Against a remote service:
   ```bash
   make cert-remote CERT_HOST=your-service-host CERT_PORT=your-service-port
   ```

3. Against a service using HTTPS:
   ```bash
   make cert-remote CERT_HOST=your-service-host CERT_PORT=443 USE_HTTPS=true
   ```

The certification suite validates:
- Response codes and structure
- Performance requirements (latency thresholds)
- Response format compliance
- Basic error handling

Note: The load test targets 1000 requests per second for the `/openrtb25` endpoint. This is intended to be a moderate load suitable for testing on a developer's machine, before deploying to a distributed Kubernetes environment with auto-scaling.

For more details about the certification suite, see the [certification README](certification/README.md).