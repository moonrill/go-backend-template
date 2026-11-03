/*
Usage:
API_BASE_URL=<URL> k6 run loadtest.js
*/

import http from 'k6/http';
import { check, sleep, group } from 'k6';

// Load Test Configuration
export const options = {
    stages: [
        { duration: '30s', target: 10 }, // ramp up to 10 VUs
        { duration: '1m', target: 10 },  // hold 10 VUs
        { duration: '30s', target: 0 },  // ramp down to 0 VUs
    ],
    thresholds: {
        http_req_duration: ['p(95)<500'], // 95% of requests must be under 500ms
        http_req_failed: ['rate<0.01'],   // error rate must be under 1%
    },
};

// Configure BASE_URL from environment variable
const BASE_URL = __ENV.API_BASE_URL || 'http://localhost:8080';

export default function () {
    // 1) Health Live Check
    group('Health Live', function () {
        const res = http.get(`${BASE_URL}/api/v1/health/live`, {
            tags: { name: 'HealthLive' },
        });

        check(res, {
            'live status is 200': (r) => r.status === 200,
            'status is live': (r) => r.json() && r.json().data && r.json().data.status === 'live',
        });
    });

    // 2) Health Ready Check
    group('Health Ready', function () {
        const res = http.get(`${BASE_URL}/api/v1/health/ready`, {
            tags: { name: 'HealthReady' },
        });

        check(res, {
            'ready status is 200': (r) => r.status === 200,
        });
    });

    // 3) Metrics Endpoint Check
    group('Metrics', function () {
        const res = http.get(`${BASE_URL}/metrics`, {
            tags: { name: 'Metrics' },
        });

        check(res, {
            'metrics status is 200': (r) => r.status === 200,
        });
    });

    sleep(1);
}

