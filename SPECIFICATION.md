

| DRAFT: This implementation guide is being shared for feedback and discussion. Based on integration experience and partner input, final requirements and behaviors may change. |
| :---- |

# OpenXBuild Enrichment Services: API Integration Guide for Partners

Welcome to the technical integration guide for **OpenXBuild Enrichment Services**. This document describes how to build real-time enrichment services that integrate into OpenXBuild to enhance OpenRTB BidRequests before DSP bidding.

---

## **Overview**

The platform enables real-time augmentation of OpenRTB `BidRequest` objects on OpenXBuild, within a secure Kubernetes environment. These services have access to an object store (currently supporting Google Cloud Storage), allowing storage of lookup tables, ML models, or other artifacts used by your service.

Enriched BidRequests are used internally by the OpenX SSP platform (e.g., to attach segments or apply deal matching logic) before soliciting bids from DSPs.

### Integration Paths

OpenX supports two integration paths. The OpenX team will confirm which is configured for your deployment.

| Path | Protocol | When to use |
| :---- | :---- | :---- |
| **OpenXBuild HTTP API** | `POST /openrtb25` (JSON over HTTP/1.1 or H2C) | Standard integration; simpler to implement and test |
| **IAB ARTF** | `GetMutations` gRPC (protobuf over HTTP/2) | Preferred for compliance with the [IAB Agentic RTB Framework](https://github.com/IABTechLab/agentic-rtb-framework) standard |

Regardless of integration path, all services must expose `GET /healthz` or `GET /health/ready` (either is accepted), and should expose `GET /metrics`.

## Getting Started

Before integrating your service, and pending privacy review, our support team will work with you to define:

* Which OpenRTB BidRequest fields will be sent to your service (based on criteria such as country code, device type, etc)

* Which fields will be populated in outbound requests to your enrichment service. This is referred to in this document as a “projection” of the OpenRTB BidRequest schema.

A reference implementation is available at https://github.com/openx/openx-enrichment-service-template

---

## **Enrichment API Specification**

### **Endpoint: `POST /openrtb25`**

#### **Request**

* Content-Type: `application/json`  
* Body: A **projection** of a valid [OpenRTB 2.5 BidRequest](https://github.com/InteractiveAdvertisingBureau/openrtb/blob/master/OpenRTB%20v2.5/README.md)  
* Only the declared subset of fields will be included (e.g., `user.ext.eids`, `site.page`)

The fields currently available are:

* `device.devicetype`
* `device.dnt`  
* `device.language`
* `device.lmt`
* `device.ua`  
* `device.ext.sua`  
* `device.h`  
* `device.w`  
* `device.geo.country`  
* `device.geo.region`
* `device.geo.type`
* `device.geo.metro`
* `device.geo.city`
* `device.geo.zip`
* `imp.banner` (only for Banner media type, contents will be empty but reserved for future use)  
* `imp.video` (only for Video media type, contents will be empty but reserved for future use)  
* `imp.native` (only for Native media type, contents will be empty but reserved for future use)  
* `imp.tagid`
* `imp.ext.gpid`
* `regs.ext.gdpr`  
* `regs.ext.us_privacy`  
* `regs.ext.gpp`  
* `regs.ext.gpp_sid`  
* `regs.coppa`  
* `site.domain`
* `site.cat`
* `site.page`  
* `app.bundle`
* `app.content.language`
* `app.content.url`
* `app.content.contentrating`
* `app.content.genre`
* `app.domain`
* `app.storeurl`
* `app.cat`
* `app.publisher.id`
* `app.publisher.cat`
* `user.consent`  
* `user.ext.eids.*` (only where the `source` is authorized to be consumed by your service)
* `source.ext.schain.*`

We will partner with you to identify the set of fields being sent to your service and identify additional fields that could improve your response accuracy.

Example Request:

```
{
  "id": "<unique-request-id>",
  "imp": [
    {
      "id": "<impression-id>",
      "banner": {}
    }
  ],
  "device": {
    "geo": {
      "country": "US"
    },
    "dnt": 0,
    "ua": "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Mobile Safari/537.36"
  },
  "site": {
    "page": "https://example.com"
  },
  "user": {
    "ext": {
      "eids": [
        {"source": "id-provider-1.com", "uids": [{"id": "xyz"}]}
      ]
    }
  }
}
```

#### **Response**

* Content-Type: `application/json`
* Body: A fragment of a `BidRequest`\-compatible object  
* Additions to the BidRequest object will be validated and merged back into the original `BidRequest`

Enrichment services can return different types of enrichments, each serving a specific intent:

* **Activate Segments**: Attach segment IDs to enable audience targeting
* **Add Extended Identifier (EID)**: Provide additional user identifiers for identity resolution
* **Propose Deal Floor Override**: Specify minimum bid floors for deals

##### **Activate Segments**

Use `user.data` to attach segment IDs that can be targeted by deals.

Example Response:

```
{
  "id": "<original-request-id>",
  "user": {
    "data": [
      {"name": "segment-provider.com", "segment": [{ "id": "123" }]}
    ]
  }
}
```

##### **Add Extended Identifier (EID)**

Use `user.ext.eids` to provide additional user identifiers for audience-based targeting.

Example Response:

```
{
  "id": "<original-request-id>",
  "user": {
    "ext": {
      "eids": [
        {"source": "id-provider-2.com", "uids": [{ "id": "abc" }]}
      ]
    }
  }
}
```

##### **Propose Deal Floor Override**

Use `imp[].pmp.deals` to propose bid floor overrides for deals.

Each deal object must include:

* `id` (required): The deal's alphanumeric identifier
* `bidfloor` (required): Minimum bid for this deal expressed in CPM
* `bidfloorcur` (optional): Currency specified using ISO-4217 alpha codes (default: "USD")

The `id` field must match a deal configured in the OpenX platform, and the floor override will only take effect
if the request also matches the eligibility criteria of that deal (i.e. it will not append this deal ID to the request
if it is not otherwise eligible).  It is safe to return deal IDs in cases where you are not sure whether the request
is eligible.

The `bidfloorcur` field must be set if the deal is not priced in USD, and the currency specified here must match
the currency on the deal.

The floor will only be adjusted if it is higher than other applicable floors, such as publisher floors and the floor
price configured on the deal.

Example Response:

```
{
  "id": "<original-request-id>",
  "imp": [
    {
      "id": "<impression-id>",
      "pmp": {
        "deals": [
          {
            "id": "OX-deal-123",
            "bidfloor": 2.50,
            "bidfloorcur": "USD"
          }
        ]
      }
    }
  ]
}
```

##### **Multiple Enrichments**

You can combine multiple enrichment types in a single response. For example, you can activate segments and add extended identifiers together:

Example Response:

```
{
  "id": "<original-request-id>",
  "user": {
    "data": [
      {"name": "segment-provider.com", "segment": [{ "id": "123" }]}
    ],
    "ext": {
      "eids": [
        {"source": "id-provider-2.com", "uids": [{ "id": "abc" }]}
      ]
    }
  }
}
```

## **HTTP Response Codes**

| Status Code | Description |
| :---- | :---- |
| `200 OK` | Request successful; mutations returned. |
| `204 No Content` | Request successful; no mutations needed. |
| `400 Bad Request` | Invalid JSON or malformed request. |
| `429 Too Many Requests` | Service overloaded; throttling in effect. |
| `500 Internal Server Error` | Error processing the request on the server. |

---

### **Health Check: `GET /healthz`**

* Returns `200 OK` if service is healthy

### **Optional Metrics: `GET /metrics`**

* Prometheus-compatible format  

---

## **OpenXBuild Deployment (BYO Container)**

### **Runtime Environment**

Enrichment services are hosted on the OpenXBuild platform. Your service runs as a container within a Kubernetes environment:

* **Container(s) will run as a Kubernetes pod in its own namespace**  
* **Object store available through the GCS API** (used for loading models, lookup tables, etc)  
* **Container(s) can be preempted at any time.**  Services must be **stateless**; each instance may be restarted at any time, and you may use the object store for externalized state  
* **CPU/memory limits will be determined by your OpenX team during integration of your service**, and autoscaling behavior is managed by the platform

### **Object Store**

* You will be provided with a pair of Google Cloud Storage buckets:
  * An “inbox” bucket, which is read-only from your container, and read-write from outside. This is to store model artifacts, lookup tables, and any other assets required by your service
  * A read-only “outbox” bucket, for exported logs and metrics
  * Both buckets will have lifecycle management enabled to delete files older than 14 days  
* You may provide us with one or more Google Cloud Platform principals that have access to these buckets, to allow you to update these assets and read from the outbox bucket.

### **Environment Variables**

These environment variables are automatically set by OpenX in a hosted deployment, and should be used by your service:

* `PORT`: HTTP port to listen on (default: 8080)
* `LOG_LEVEL`: Log level (default: "info")
* `GCS_INBOX_BUCKET`: Name of the GCS bucket for incoming files
* `GCP_REGION` and `GCP_ZONE`: the Google Cloud region and zone where your workload is running

### **Security Constraints**

* No outbound internet unless authorized (egress blocked).  If a pinhole is required, please provide IP addresses and port numbers during setup
* All file writes must go to a temporary volume which gets discarded at the end of a pod lifecycle  
* Your object store will be readable by any process running within your container, using the GCS API and [workload identity](https://cloud.google.com/kubernetes-engine/docs/concepts/workload-identity)

### **Observability**

* Logs must be written in plaintext to stdout/stderr, must not include any sensitive or personally-identifying information, and are subject to audit and redaction  
** **Important:** None of the data from inbound BidRequest objects may be included in logs without prior approval. Exceptions will be made for limited logging of exceptions/errors with BidRequest details.
* `/metrics` endpoint is optional but encouraged

---

## **GCP Regions**

In order of average QPS, from largest to smallest:

* us-east1
* us-east4
* us-west1
* europe-west4
* europe-west2
* us-central1
* europe-west1
* asia-southeast1
* asia-east1
* europe-west1
* us-west2
* northamerica-northeast1
* asia-south1


---

## **Performance Expectations**

* **Target latency**: \<2ms average; \<5ms @ p95  
* **Timeout**: 10ms
* The platform may shed traffic dynamically to maintain performance. Requests most likely to benefit from enrichment may be prioritized.

---

## **Getting Started Checklist**

* Package your service as a docker image, and push to a docker image registry accessible by OpenX ([Google Artifact Registry](https://pkg.dev) is recommended)  
* Provide:
  * Fully-qualified image URI (in the format `[REGISTRY_HOSTNAME]/[PROJECT]/[IMAGE_NAME]`)  
  * The docker image tag (eg, `prod`, `latest`) that labels the production-ready version of the image  
  * Authentication credentials (if required) - a username/password or Docker `config.json` file  
* OpenX will perform a security scan and distribute your image to each region where your service is hosted  
* Create a service account or other Google Cloud Platform IAM principal that will be granted access to Google Cloud Storage for assets used by your enrichment service  
* Contact your support team to receive the name of your Google Cloud Storage bucket  
* If your service requires outbound network access, provide the IP addresses and port number(s) so we can open a pinhole. This is subject to security and privacy review.

---

---

## **IAB ARTF Integration (gRPC/GetMutations)**

This section describes the gRPC integration path, which implements the [IAB Agentic RTB Framework (ARTF)](https://github.com/IABTechLab/agentic-rtb-framework) `RTBExtensionPoint` service.

### **Service**

```protobuf
service RTBExtensionPoint {
  rpc GetMutations (RTBRequest) returns (RTBResponse);
}
```

Protocol: gRPC over HTTP/2 (cleartext). The port is assigned by OpenX during deployment configuration.

### **Request**

OpenX sends an `RTBRequest` containing a projection of the OpenRTB BidRequest encoded as IAB proto types. The same field projection rules as the HTTP API apply — only declared fields will be populated.

| Field | Type | Description |
| :---- | :---- | :---- |
| `id` | string | Unique request ID assigned by OpenX |
| `lifecycle` | Lifecycle | Always `LIFECYCLE_PUBLISHER_BID_REQUEST` |
| `tmax` | int32 | Maximum time in milliseconds OpenX will wait for a response (including network latency) |
| `bid_request` | BidRequest | OpenRTB BidRequest projection (proto-encoded, using the IAB OpenRTB 2.6 proto schema) |
| `applicable_intents` | Intent[] | Intents OpenX will process from this service's response; mutations with other intents are ignored |

### **Response**

Return an `RTBResponse` containing zero or more `Mutation` objects. Each mutation specifies an `intent`, an `op`, a semantic `path`, and a typed `value`.

| Field | Type | Description |
| :---- | :---- | :---- |
| `id` | string | Must match `RTBRequest.id` |
| `mutations` | Mutation[] | Proposed changes; may be empty |
| `metadata` | Metadata | Optional. `api_version` and `model_version` strings for observability |

### **Supported Intents and Paths**

OpenX processes the following mutation intents. Unrecognised intents are ignored.

| Intent | Path | Value type | Description |
| :---- | :---- | :---- | :---- |
| `ACTIVATE_SEGMENTS` | `/user/data/segment` | `IDsPayload` | Segment IDs to attach for audience targeting |
| `ADJUST_DEAL_FLOOR` | `/imp/{impId}/pmp/deals/{dealId}` | `AdjustDealPayload` | Bid floor override for a specific deal on a specific impression |

`{impId}` must match an impression ID present in the request. `{dealId}` must match a deal configured in the OpenX platform (same eligibility rules as the HTTP API apply).

### **Example Request (proto JSON representation)**

```json
{
  "id": "req-abc123",
  "lifecycle": "LIFECYCLE_PUBLISHER_BID_REQUEST",
  "tmax": 10,
  "applicableIntents": ["ACTIVATE_SEGMENTS", "ADJUST_DEAL_FLOOR"],
  "bidRequest": {
    "id": "req-abc123",
    "imp": [
      {"id": "imp1", "banner": {}}
    ],
    "device": {"devicetype": 4},
    "site": {"page": "https://example.com"}
  }
}
```

### **Example Response**

```json
{
  "id": "req-abc123",
  "mutations": [
    {
      "intent": "ACTIVATE_SEGMENTS",
      "op": "OPERATION_ADD",
      "path": "/user/data/segment",
      "ids": {"id": ["sports-fan", "premium-user"]}
    },
    {
      "intent": "ADJUST_DEAL_FLOOR",
      "op": "OPERATION_REPLACE",
      "path": "/imp/imp1/pmp/deals/OX-deal-456",
      "adjustDeal": {"bidfloor": 2.50}
    }
  ]
}
```

### **Health and Metrics**

All services must expose a readiness health check and should expose `GET /metrics` (Prometheus format). These are served on the HTTP port alongside (or instead of) `POST /openrtb25` — no separate health port is required for OpenX-hosted deployments.

OpenX accepts either the OpenX convention or the ARTF convention for health endpoints:

| Endpoint | Description |
| :---- | :---- |
| `GET /healthz` | OpenX convention — readiness check (returns `200 OK` when ready) |
| `GET /health/ready` | ARTF convention — readiness check (returns `200 OK` when ready) |
| `GET /health/live` | ARTF convention — liveness check (returns `200 OK` when process is alive) |

At minimum, implement `/healthz` **or** `/health/ready`. The reference implementation exposes all three.

---

## **FAQ**

**Q: What happens if my service exceeds the latency budget?** A: The platform may dynamically shed traffic. Services are expected to be highly performant and stateless.

**Q: What happens if my service exceeds the CPU and memory budget?** A: Autoscaling may be paused, and traffic may be shed if latency rises.

**Q: Can I use an external datastore?** A: Only the provided object store is supported for hosted services. This is both a performance and security consideration. If your service requires outbound network access (for example, to look up decryption keys for an encrypted user identifier), this must be specified in advance when we configure your enrichment service.

**Q: What OpenRTB version is supported?** A: The HTTP API (`POST /openrtb25`) uses OpenRTB 2.5 field names and types. The ARTF gRPC path uses the IAB OpenRTB 2.6 proto schema — field names and locations may differ slightly from 2.5 JSON.

---

