# OpenX Enrichment Service Template

This is a reference implementation of an enrichment service that follows the [OpenX BYO Container specification](SPECIFICATION.md). It demonstrates how to implement a containerized enrichment service that can be deployed on OpenX infrastructure.

## Features

- Accepts OpenRTB BidRequest objects at `/openrtb25`
- Returns mock segments in the response
- Exposes health check endpoint at `/healthz`
- Exposes Prometheus metrics at `/metrics`
- Follows OpenX's standard patterns for metrics, logging, and configuration

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
  -e LOG_LEVEL=debug \
  your-registry.com/openx-enrichment-service-template:1.0.0
```

## Kubernetes Deployment

When deployed to Kubernetes, the service will use workload identity for GCS authentication. No additional credential configuration is needed.

## API Documentation

For a full description of the request/response format, required endpoints, and runtime expectations, see the [OpenX Enrichment Service Specification](SPECIFICATION.md).