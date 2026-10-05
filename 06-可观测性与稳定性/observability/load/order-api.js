import http from 'k6/http';
import { check, sleep } from 'k6';

const baseURL = __ENV.BASE_URL || 'http://localhost:8080';

export const options = {
  stages: [
    { duration: '1m', target: 5 },
    { duration: '2m', target: 20 },
    { duration: '5m', target: 20 },
    { duration: '1m', target: 0 },
  ],
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<500'],
  },
};

export default function () {
  const response = http.post(
    `${baseURL}/orders`,
    JSON.stringify({ source: 'load-test' }),
    { headers: { 'Content-Type': 'application/json' }, tags: { route: 'orders' } },
  );
  check(response, { 'order accepted': (res) => res.status === 201 });
  sleep(0.1);
}
