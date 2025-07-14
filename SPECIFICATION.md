

| DRAFT: This implementation guide is being shared for feedback and discussion. Based on integration experience and partner input, final requirements and behaviors may change. |
| :---- |

# Hosted RTB Enrichment Services: API Integration Guide for Partners

Welcome to the technical integration guide for **Hosted RTB Enrichment Services**. This document describes how to build real-time enrichment services that integrate into the OpenX Hosted Enrichment Platform to enhance OpenRTB BidRequests before DSP bidding.

---

## **Overview**

The platform enables real-time augmentation of OpenRTB `BidRequest` objects on the OpenX Hosted Enrichment platform, within a secure Kubernetes environment. These services have access to an object store (currently supporting Google Cloud Storage), allowing storage of lookup tables, ML models, or other artifacts used by your service.

Enriched BidRequests are used internally by the OpenX SSP platform (e.g., to attach segments or apply deal matching logic) before soliciting bids from DSPs.

## Getting Started

Before integrating your service, and pending privacy review, our support team will work with you to define:

* Which OpenRTB BidRequest fields will be sent to your service (based on criteria such as country code, device type, etc)

* Which fields will be populated in outbound requests to your enrichment service. This is referred to in this document as a “projection” of the OpenRTB 2.5 BidRequest schema.

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
* `regs.gdpr`  
* `regs.us_privacy`  
* `regs.gpp`  
* `regs.gpp_sid`  
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
* `user.ext.eids` (only where the `source` is authorized to be consumed by your service)
* `source.ext.schain`

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

* Body: A fragment of a `BidRequest`\-compatible object  
* Additions to the BidRequest object will be validated and merged back into the original `BidRequest`

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

The fields that you may include in a response are limited to the following paths, unless arranged in advance with OpenX:

* `user.data` (see example above, must follow the OpenRTB 2.5 definition of `Data` objects used for segments, and only the `id` field will be used for targeting)  
* `user.ext.eids` (see example above, must follow the OpenRTB 2.6 definition of `EID` objects)

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

## **Hosted Mode (BYO Container)**

### **Runtime Environment**

Enrichment services are hosted on the OpenX platform. Your service runs as a container within a Kubernetes environment:

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

## **FAQ**

**Q: What happens if my service exceeds the latency budget?** A: The platform may dynamically shed traffic. Services are expected to be highly performant and stateless.

**Q: What happens if my service exceeds the CPU and memory budget?** A: Autoscaling may be paused, and traffic may be shed if latency rises.

**Q: Can I use an external datastore?** A: Only the provided object store is supported for hosted services. This is both a performance and security consideration. If your service requires outbound network access (for example, to look up decryption keys for an encrypted user identifier), this must be specified in advance when we configure your enrichment service.

**Q: What OpenRTB version is supported?** A: OpenRTB 2.5. Compatibility with later versions may be added in the future.

---

