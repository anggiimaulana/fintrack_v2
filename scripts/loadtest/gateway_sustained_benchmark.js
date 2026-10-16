import http from 'k6/http';
import { check } from 'k6';
import { Trend, Rate } from 'k6/metrics';

const gatewayLatency = new Trend('ai_gateway_latency_ms');
const errorRate = new Rate('ai_gateway_error_rate');

export const options = {
  scenarios: {
    sustained_gateway_load: {
      executor: 'constant-arrival-rate',
      rate: 500, // 500 RPS
      timeUnit: '1s',
      duration: '30s', // burst sustained test
      preAllocatedVUs: 100,
      maxVUs: 500,
    },
  },
  thresholds: {
    'ai_gateway_latency_ms': ['p(95)<60'],
    'ai_gateway_error_rate': ['rate<0.0001'],
  },
};

export default function () {
  const url = __ENV.TARGET_URL || 'http://localhost:8081/api/v1/ai/chat';
  const payload = JSON.stringify({
    message: 'Halo FinTrack test load',
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${__ENV.TEST_USER_TOKEN}`,
      'X-Request-Id': `load-${__VU}-${Date.now()}`,
    },
  };

  const start = Date.now();
  const res = http.post(url, payload, params);
  const duration = Date.now() - start;

  gatewayLatency.add(duration);
  const isOk = res.status === 200 || res.status === 429;
  errorRate.add(!isOk);

  check(res, {
    'status is acceptable': () => isOk,
  });
}
