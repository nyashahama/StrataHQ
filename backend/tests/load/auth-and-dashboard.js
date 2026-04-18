import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate } from 'k6/metrics';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const TEST_EMAIL = __ENV.TEST_EMAIL || 'demo@stratahq.com';
const TEST_PASSWORD = __ENV.TEST_PASSWORD || 'Demo2024!';

const loginSuccessRate = new Rate('login_success');
const refreshBlockedRate = new Rate('refresh_blocked');
const repeatLoginFailRate = new Rate('repeat_login_fail');
const getEndpointsFailRate = new Rate('get_endpoints_fail');

export const options = {
  scenarios: {
    ten_users: {
      executor: 'constant-vus',
      vus: 10,
      duration: '2m',
      tags: { scenario: '10_users' },
    },
    twentyfive_users: {
      executor: 'constant-vus',
      vus: 25,
      duration: '2m',
      tags: { scenario: '25_users' },
    },
    fifty_users: {
      executor: 'constant-vus',
      vus: 50,
      duration: '2m',
      tags: { scenario: '50_users' },
    },
  },
  thresholds: {
    login_success: ['rate>0.99'],
    refresh_blocked: ['rate==0'],
    repeat_login_fail: ['rate==0'],
    get_endpoints_fail: ['rate<0.01'],
    http_req_duration: ['p(95)<1000'],
  },
};

function login(email, password) {
  const url = `${BASE_URL}/api/v1/auth/login`;
  const payload = JSON.stringify({ email, password });
  const params = {
    headers: { 'Content-Type': 'application/json' },
  };

  const res = http.post(url, payload, params);

  check(res, {
    'login status 200': (r) => r.status === 200,
    'login has access_token': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.data && body.data.access_token;
      } catch {
        return false;
      }
    },
  }) ? loginSuccessRate.add(1) : loginSuccessRate.add(0);

  if (res.status === 429) {
    refreshBlockedRate.add(1);
  }

  return res;
}

function refresh(refreshToken) {
  const url = `${BASE_URL}/api/v1/auth/refresh`;
  const payload = JSON.stringify({ refresh_token: refreshToken });
  const params = {
    headers: { 'Content-Type': 'application/json' },
  };

  const res = http.post(url, payload, params);

  if (res.status === 429) {
    refreshBlockedRate.add(1);
  }

  return res;
}

function getWithAuth(url, accessToken) {
  const params = {
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${accessToken}`,
    },
  };

  const res = http.get(url, params);
  return res;
}

function testProtectedEndpoints(accessToken) {
  const endpoints = [
    `/api/v1/auth/me`,
    `/api/v1/schemes`,
    `/api/v1/levies`,
    `/api/v1/maintenance`,
  ];

  let failures = 0;

  for (const endpoint of endpoints) {
    const res = getWithAuth(`${BASE_URL}${endpoint}`, accessToken);
    const ok = check(res, {
      [`${endpoint} returns 2xx`]: (r) => r.status >= 200 && r.status < 300,
    });
    if (!ok) {
      failures++;
    }
  }

  getEndpointsFailRate.add(failures > 0 ? 1 : 0);
  return failures;
}

function testTokenExpiryScenario(email, password) {
  const loginRes = login(email, password);
  if (loginRes.status !== 200) {
    repeatLoginFailRate.add(1);
    return;
  }

  let body;
  try {
    body = JSON.parse(loginRes.body).data;
  } catch {
    repeatLoginFailRate.add(1);
    return;
  }

  const refreshToken = body.refresh_token;
  const accessToken = body.access_token;

  if (!refreshToken || !accessToken) {
    repeatLoginFailRate.add(1);
    return;
  }

  const refreshRes = refresh(refreshToken);
  if (refreshRes.status === 429) {
    refreshBlockedRate.add(1);
  }

  testProtectedEndpoints(accessToken);

  sleep(Math.random() * 2 + 1);
}

export default function () {
  group('Auth and Dashboard Flow', () => {
    const email = `${TEST_EMAIL.replace('@', `+${__VU}@`)}`;
    const password = TEST_PASSWORD;

    const loginRes = login(email, password);

    if (loginRes.status !== 200) {
      repeatLoginFailRate.add(1);
      sleep(1);
      return;
    }

    let body;
    try {
      body = JSON.parse(loginRes.body).data;
    } catch {
      repeatLoginFailRate.add(1);
      sleep(1);
      return;
    }

    const refreshToken = body.refresh_token;
    const accessToken = body.access_token;

    if (!refreshToken || !accessToken) {
      repeatLoginFailRate.add(1);
      sleep(1);
      return;
    }

    const refreshed = refresh(refreshToken);
    if (refreshed.status === 429) {
      refreshBlockedRate.add(1);
    }

    const failures = testProtectedEndpoints(accessToken);
    if (failures > 0) {
      getEndpointsFailRate.add(1);
    }

    sleep(Math.random() * 2 + 1);

    testTokenExpiryScenario(email, password);
  });
}

export function handleSummary(data) {
  const summary = {
    'auth-load-test-summary': http.post(`${__ENV.SUMMARY_URL || 'http://localhost:8081'}/metrics`, JSON.stringify({
      timestamp: new Date().toISOString(),
      vus: __VU,
      data,
    }), {
      headers: { 'Content-Type': 'application/json' },
    }),
  };

  return {
    stdout: textSummary(data, { indent: ' ', enableColors: true }),
  };
}

function textSummary(data, opts) {
  const indent = opts.indent || '';
  let output = `${indent}Test Results:\n`;
  output += `${indent}  HTTP Requests: ${data.metrics.http_reqs?.values?.count || 0}\n`;
  output += `${indent}  Failed Requests: ${data.metrics.http_req_failed?.values?.count || 0}\n`;
  output += `${indent}  Duration: ${data.metrics.http_req_duration?.values?.p95 || 0}ms (p95)\n`;
  output += `${indent}  Login Success Rate: ${(data.metrics.login_success?.values?.rate || 0) * 100}%\n`;
  output += `${indent}  Refresh Blocked Rate: ${data.metrics.refresh_blocked?.values?.rate || 0}\n`;
  output += `${indent}  Repeat Login Fail Rate: ${data.metrics.repeat_login_fail?.values?.rate || 0}\n`;
  output += `${indent}  GET Endpoints Fail Rate: ${data.metrics.get_endpoints_fail?.values?.rate || 0}\n`;
  return output;
}