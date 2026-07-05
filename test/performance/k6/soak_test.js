import http from 'k6/http';
import { check } from 'k6';

export const options = {
  vus: 100,
  duration: '10m',
};

export default function () {
  const res = http.get('https://api.helixterminator.dev/health');
  check(res, { 'status is 200': (r) => r.status === 200 });
}
