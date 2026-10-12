import http from 'k6/http';
import { check } from 'k6';
import { Counter } from 'k6/metrics';

const processedCounter = new Counter('webhook_processed');
const ignoredCounter = new Counter('webhook_ignored_duplicate');

export const options = {
  scenarios: {
    webhook_replay: {
      executor: 'per-vu-iterations',
      vus: 10,
      iterations: 1,
      maxDuration: '5s',
    },
  },
};

export default function () {
  const url = __ENV.TARGET_URL || 'http://localhost:8080/api/v1/payments/webhook/midtrans';
  const orderId = __ENV.ORDER_ID || 'INV-2026-TEST-998811';
  const payload = JSON.stringify({
    order_id: orderId,
    transaction_status: 'settlement',
    gross_amount: '150000.00',
    status_code: '200',
    signature_key: __ENV.PRECOMPUTED_VALID_SIGNATURE || 'valid_sig',
    payment_type: 'qris',
    transaction_time: '2026-10-11 14:00:00',
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'X-Idempotency-Key': `idem-midtrans-${orderId}`,
    },
  };

  const res = http.post(url, payload, params);

  check(res, {
    'status is 200': (r) => r.status === 200,
  });

  const body = res.json();
  if (body && body.already_processed === false) {
    processedCounter.add(1);
  } else if (body && body.already_processed === true) {
    ignoredCounter.add(1);
  }
}
