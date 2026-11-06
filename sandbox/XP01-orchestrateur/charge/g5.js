import http from 'k6/http';
import { check } from 'k6';

const cible = __ENV.CIBLE || 'https://collector.qvl-project.com/';
const debit = Number(__ENV.DEBIT || 30);

export const options = {
  discardResponseBodies: true,
  scenarios: {
    paliers: {
      executor: 'ramping-arrival-rate',
      startRate: 1,
      timeUnit: '1s',
      preAllocatedVUs: 50,
      maxVUs: 300,
      stages: [
        { target: debit, duration: __ENV.MONTEE || '2m' },
        { target: debit, duration: __ENV.PLATEAU || '5m' },
        { target: 0, duration: __ENV.DESCENTE || '1m' },
      ],
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.01'],
  },
};

export default function () {
  check(http.get(cible, { timeout: '10s' }), { 'statut 200': (r) => r.status === 200 });
}
