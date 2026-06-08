import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  vus: 1,
  duration: '10s',
};

export default function () {
  const response = http.get(__ENV.BASE_URL || 'http://127.0.0.1:8080/');
  check(response, {
    'status is 200': (r) => r.status === 200,
  });
  sleep(1);
}
