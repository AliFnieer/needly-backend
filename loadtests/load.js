import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

const errorRate = new Rate('errors');
const requestDuration = new Trend('request_duration');

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export const options = {
  stages: [
    { duration: '30s', target: 20 },  // Ramp up
    { duration: '1m', target: 20 },   // Stay at 20
    { duration: '30s', target: 50 },  // Ramp up to 50
    { duration: '1m', target: 50 },   // Stay at 50
    { duration: '30s', target: 100 }, // Ramp up to 100
    { duration: '1m', target: 100 },  // Stay at 100
    { duration: '30s', target: 0 },   // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'],
    errors: ['rate<0.1'],
  },
};

// Shared test user credentials
const TEST_EMAIL = 'loadtest@example.com';
const TEST_PASSWORD = 'LoadTest123!';

export function setup() {
  // Register a test user
  const registerRes = http.post(`${BASE_URL}/api/v1/auth/register`, JSON.stringify({
    first_name: 'Load',
    last_name: 'Test',
    email: TEST_EMAIL,
    password: TEST_PASSWORD,
  }), { headers: { 'Content-Type': 'application/json' } });

  if (registerRes.status === 201 || registerRes.status === 409) {
    const loginRes = http.post(`${BASE_URL}/api/v1/auth/login`, JSON.stringify({
      email: TEST_EMAIL,
      password: TEST_PASSWORD,
    }), { headers: { 'Content-Type': 'application/json' } });

    if (loginRes.status === 200) {
      const body = JSON.parse(loginRes.body);
      return { token: body.access_token };
    }
  }
  return { token: '' };
}

export default function (data) {
  const headers = {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${data.token}`,
  };

  // Health check (public)
  const healthRes = http.get(`${BASE_URL}/health`);
  check(healthRes, { 'health status 200': (r) => r.status === 200 });

  // Get user profile
  const meRes = http.get(`${BASE_URL}/api/v1/auth/me`, { headers });
  check(meRes, { 'me status 200': (r) => r.status === 200 });

  // List households
  const listsRes = http.get(`${BASE_URL}/api/v1/households`, { headers });
  check(listsRes, { 'households status 200': (r) => r.status === 200 });

  // Create a household
  const createRes = http.post(`${BASE_URL}/api/v1/households`, JSON.stringify({
    name: `Load Test Household ${Date.now()}`,
  }), { headers });
  check(createRes, { 'create household status 201': (r) => r.status === 201 });

  sleep(1);
}

export function teardown(data) {
  // Cleanup is optional - test users will be overwritten
}
