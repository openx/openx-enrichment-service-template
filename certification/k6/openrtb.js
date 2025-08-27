import http from 'k6/http';
import { check, sleep } from 'k6';
import { randomString } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';
import { Trend } from 'k6/metrics';


const TARGET_HOST = __ENV.TARGET_HOST || 'localhost';
const TARGET_PORT = __ENV.TARGET_PORT || '8080';
const USE_HTTPS = __ENV.USE_HTTPS === 'true';
const VERBOSE = __ENV.VERBOSE === 'true';
const PROTOCOL = USE_HTTPS ? 'https' : 'http';
const BASE_URL = `${PROTOCOL}://${TARGET_HOST}:${TARGET_PORT}`;

// Custom metrics
const openrtbTrend = new Trend('openrtb_duration');
const healthTrend = new Trend('health_duration');

// Try to load sample redacted test data, fall back to synthetic data
let sampleTestData = null;
try {
  // Load JSONL data (one JSON object per line)
  const data = open('./openrtb_1000.jsonl');
  const lines = data.trim().split('\n');
  sampleTestData = lines.map(line => JSON.parse(line));
  if (VERBOSE) {
    console.log(`Loaded ${sampleTestData.length} sample test requests`);
  }
} catch (e) {
  console.log('Sample test data not available, using synthetic data');
  console.log('Error loading sample data:', e.message);
}

export const options = {
  scenarios: {
    // Main load test for OpenRTB endpoint
    openrtb_load: {
      executor: 'constant-arrival-rate',
      rate: 1000, // 1000 requests per second
      timeUnit: '1s',
      duration: '30s',
      preAllocatedVUs: 100,
      maxVUs: 200,
    },
    // Periodic health checks
    health_checks: {
      executor: 'constant-arrival-rate',
      rate: 1, // 1 check per second
      timeUnit: '1s',
      duration: '30s',
      preAllocatedVUs: 1,
      maxVUs: 1,
    },
  },
  thresholds: {
    'http_req_duration': ['p(95)<5'], // 5ms p95 threshold as per spec
    'openrtb_duration': ['p(95)<5'],
    'health_duration': ['p(95)<100'],
    'checks': ['rate>0.99'], // Overall check success rate
  },
};

const validBidRequest = {
  id: randomString(10),
  imp: [
    {
      id: randomString(10),
      banner: {}
    }
  ],
  device: {
    geo: {
      country: "US"
    },
    dnt: 0,
    ip: "192.168.2.123",
    ua: "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Mobile Safari/537.36"
  },
  site: {
    page: "https://example.com"
  },
  user: {
    ext: {
      eids: [
        {"source": "id-provider-1.com", "uids": [{"id": "xyz"}]}
      ]
    }
  }
};

// Variations of bid requests to test different scenarios
const bidRequestVariations = {
  // Minimal valid request with only required fields
  minimal: {
    id: randomString(10),
    imp: [
      {
        id: randomString(10),
        banner: {}
      }
    ]
  },

  // Request with video impression type
  videoImp: {
    id: randomString(10),
    imp: [
      {
        id: randomString(10),
        video: {}
      }
    ],
    device: {
      geo: {
        country: "US"
      }
    }
  },

  // Request with GDPR consent
  withGDPR: {
    id: randomString(10),
    imp: [
      {
        id: randomString(10),
        banner: {}
      }
    ],
    regs: {
      gdpr: 1
    },
    user: {
      consent: "BOEFEAyOEFEAyAHABDENAI4AAAB9vABAASA"
    }
  },

  // Request with US Privacy
  withUSPrivacy: {
    id: randomString(10),
    imp: [
      {
        id: randomString(10),
        banner: {}
      }
    ],
    regs: {
      us_privacy: "1YNN"
    }
  },

  // Request with COPPA flag
  withCOPPA: {
    id: randomString(10),
    imp: [
      {
        id: randomString(10),
        banner: {}
      }
    ],
    regs: {
      coppa: 1
    }
  },

  // Request with device screen dimensions
  withScreenSize: {
    id: randomString(10),
    imp: [
      {
        id: randomString(10),
        banner: {}
      }
    ],
    device: {
      w: 1920,
      h: 1080,
      geo: {
        country: "US"
      }
    }
  },

  // Invalid request - missing required fields
  invalidMissingFields: {
    imp: [
      {
        banner: {}
      }
    ]
  },

  // Invalid request - malformed impression
  invalidMalformedImp: {
    id: randomString(10),
    imp: [
      {
        id: randomString(10)
        // Missing banner/video/native
      }
    ]
  },

  // Invalid request - malformed device
  invalidMalformedDevice: {
    id: randomString(10),
    imp: [
      {
        id: randomString(10),
        banner: {}
      }
    ],
    device: {
      geo: {
        // Missing country
      }
    }
  }
};

function checkHealth() {
  const res = http.get(`${BASE_URL}/healthz`, {
    timeout: '10s',
    insecureSkipTLSVerify: true,
  });
  healthTrend.add(res.timings.duration);
  check(res, {
    'health check is status 200': (r) => r.status === 200,
  });
}

function checkOpenRTB() {
  // Use sample data for 70% of requests, fall back to synthetic data for 30%
  let request;
  let dataSource = '';
  if (sampleTestData && Math.random() < 0.7) {
    // Use a random sample from the loaded data
    const randomIndex = Math.floor(Math.random() * sampleTestData.length);
    request = sampleTestData[randomIndex];
    dataSource = 'sample';
  } else {
    // Fall back to synthetic data
    const variations = Object.values(bidRequestVariations);
    request = Math.random() < 0.5 ? validBidRequest : variations[Math.floor(Math.random() * variations.length)];
    dataSource = 'synthetic';
  }

  // Log which data source we're using (for debugging)
  if (VERBOSE) {
    console.log(`Using ${dataSource} data for request ID: ${request.id}`);
  }

  const res = http.post(`${BASE_URL}/openrtb25`, JSON.stringify(request), {
    headers: { 'Content-Type': 'application/json' },
    timeout: '10s',
    insecureSkipTLSVerify: true,
  });
  openrtbTrend.add(res.timings.duration);

  // Check response based on request type
  const invalidRequests = [
    bidRequestVariations.invalidMissingFields,
    bidRequestVariations.invalidMalformedImp,
    bidRequestVariations.invalidMalformedDevice
  ];

  if (invalidRequests.includes(request)) {
    check(res, {
      // it's ok to respond with 200/204 for invalid requests, but tolerate 400
      'is status 200, 204 or 400 for invalid request': (r) => r.status == 200 || r.status == 204 || r.status === 400,
    });
  } else {
    check(res, {
      'is status 200 or 204 for valid request': (r) => r.status === 200 || r.status === 204,
    });

    // Only validate response body for 200 responses
    if (res.status === 200) {
      try {
        const responseBody = JSON.parse(res.body);
        
        // Validate that response ID matches request ID
        if (responseBody.id !== request.id) {
          console.log('Validation failed - Response ID does not match request ID');
          console.log('Request ID:', request.id);
          console.log('Response ID:', responseBody.id);
        }
        check(res, {
          'response ID matches request ID': (r) => responseBody.id === request.id
        });
        
        // Check that response only contains allowed fields
        const allowedFields = ['id', 'user'];
        const hasOnlyAllowedFields = Object.keys(responseBody).every(key => allowedFields.includes(key));
        if (!hasOnlyAllowedFields) {
          console.log('Validation failed - Invalid top-level fields:', Object.keys(responseBody).filter(key => !allowedFields.includes(key)));
          console.log('Response body:', JSON.stringify(responseBody, null, 2));
        }
        
        if (responseBody.user) {
          // Check that user object only contains allowed fields
          const allowedUserFields = ['data', 'ext'];
          const hasOnlyAllowedUserFields = Object.keys(responseBody.user).every(key => allowedUserFields.includes(key));
          if (!hasOnlyAllowedUserFields) {
            console.log('Validation failed - Invalid user fields:', Object.keys(responseBody.user).filter(key => !allowedUserFields.includes(key)));
            console.log('User object:', JSON.stringify(responseBody.user, null, 2));
          }
          
          if (responseBody.user.data) {
            // Validate data array structure
            const isValidData = Array.isArray(responseBody.user.data) && 
              responseBody.user.data.every(item => 
                item.name && 
                Array.isArray(item.segment) && 
                item.segment.every(seg => seg.id)
              );
            if (!isValidData) {
              console.log('Validation failed - Invalid data structure:', JSON.stringify(responseBody.user.data, null, 2));
            }
            
            check(res, {
              'response data has valid structure': isValidData
            });
          }
          
          if (responseBody.user.ext) {
            // Validate eids structure
            const isValidEids = responseBody.user.ext.eids && 
              Array.isArray(responseBody.user.ext.eids) && 
              responseBody.user.ext.eids.every(eid => 
                eid.source && 
                Array.isArray(eid.uids) && 
                eid.uids.every(uid => uid.id)
              );
            if (!isValidEids) {
              console.log('Validation failed - Invalid eids structure:', JSON.stringify(responseBody.user.ext.eids, null, 2));
            }
            
            check(res, {
              'response eids has valid structure': isValidEids
            });
          }
          
          check(res, {
            'response user object has only allowed fields': hasOnlyAllowedUserFields
          });
        }
        
        check(res, {
          'response has only allowed fields': hasOnlyAllowedFields
        });
      } catch (e) {
        console.log('Validation failed - Invalid JSON:', e.message);
        console.log('Raw response:', res.body);
        check(res, {
          'response is valid JSON': false
        });
      }
    }
  }
}

export default function () {
  if (__ENV.SCENARIO === 'health_checks') {
    checkHealth();
  } else {
    checkOpenRTB();
  }
} 
