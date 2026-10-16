import http from 'k6/http';
import { check } from 'k6';
import { Counter } from 'k6/metrics';

const successfulRequests = new Counter('quota_successful_requests');
const rejectedRequests = new Counter('quota_rejected_requests');

export const options = {
  scenarios: {
    quota_burst: {
      executor: 'per-vu-iterations',
      vus: 20,
      iterations: 1,
      maxDuration: '10s',
    },
  },
};

export default function () {
  const url = __ENV.TARGET_URL || 'http://localhost:8081/api/v1/ai/chat';
  const payload = JSON.stringify({
    message: 'Halo FinTrack, hitung ringkasan pengeluaran saya.',
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${__ENV.TEST_USER_TOKEN}`,
      'X-Request-Id': `req-race-${__VU}-${Date.now()}`,
    },
  };

  const res = http.post(url, payload, params);

  if (res.status === 200) {
    successfulRequests.add(1);
  } else if (res.status === 429) {
    rejectedRequests.add(1);
  }

  check(res, {
    'status is 200 or 429': (r) => r.status === 200 || r.status === 429,
  });
}
