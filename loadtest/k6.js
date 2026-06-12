import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

const errorRate = new Rate('errors');
const enqueueTrend = new Trend('enqueue_duration');

export const options = {
  stages: [
    { duration: '10s', target: 50 },
    { duration: '30s', target: 100 },
    { duration: '10s', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<50'],
    errors: ['rate<0.01'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://127.0.0.1:8080';

export default function () {
  const payload = JSON.stringify({
    task: {
      name: 'k6-load-test',
      queue: 'default',
      max_retries: 3,
    },
  });

  const params = { headers: { 'Content-Type': 'application/json' } };
  const res = http.post(`${BASE_URL}/api/jobs`, payload, params);

  const ok = check(res, {
    'status is 201': (r) => r.status === 201,
    'has job id': (r) => r.json('id') !== '',
  });

  errorRate.add(!ok);
  enqueueTrend.add(res.timings.duration);

  sleep(0.01);
}
