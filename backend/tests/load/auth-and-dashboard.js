import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate } from 'k6/metrics';
import {
  buildLoadOptions,
  emailForVU,
  protectedEndpointPaths,
  shouldPostSummary,
} from './auth-and-dashboard-config.js';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const TEST_EMAIL = __ENV.TEST_EMAIL || 'agent@demo.stratahq.test';
const TEST_PASSWORD = __ENV.TEST_PASSWORD || __ENV.SEED_DEMO_PASSWORD || 'StrataDemo!2026';

const loginSuccessRate = new Rate('login_success');
const refreshBlockedRate = new Rate('refresh_blocked');
const repeatLoginFailRate = new Rate('repeat_login_fail');
const getEndpointsFailRate = new Rate('get_endpoints_fail');

export const options = buildLoadOptions(__ENV);

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

function firstSchemeID(accessToken) {
  const res = getWithAuth(`${BASE_URL}/api/v1/schemes`, accessToken);
  if (res.status < 200 || res.status >= 300) {
    return '';
  }

  try {
    const schemes = JSON.parse(res.body).data;
    return Array.isArray(schemes) && schemes.length > 0 ? schemes[0].id || '' : '';
  } catch {
    return '';
  }
}

function testProtectedEndpoints(accessToken) {
  const endpoints = protectedEndpointPaths(firstSchemeID(accessToken));

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

  sleep(Math.random() * 2 + 1);
}

export default function () {
  group('Auth and Dashboard Flow', () => {
    const email = emailForVU(TEST_EMAIL, __VU, __ENV);
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

    testProtectedEndpoints(accessToken);

    sleep(Math.random() * 2 + 1);

    testTokenExpiryScenario(email, password);
  });
}

export function handleSummary(data) {
  const output = {
    stdout: textSummary(data, { indent: ' ', enableColors: true }),
  };

  if (shouldPostSummary(__ENV)) {
    http.post(`${__ENV.SUMMARY_URL}/metrics`, JSON.stringify({
      timestamp: new Date().toISOString(),
      vus: __VU,
      data,
    }), {
      headers: { 'Content-Type': 'application/json' },
    });
  }

  return output;
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
