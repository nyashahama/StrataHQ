function envFlag(value) {
  return ["1", "true", "yes", "on"].includes(String(value || "").toLowerCase());
}

function positiveInt(value, fallback) {
  const parsed = Number.parseInt(String(value || ""), 10);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback;
}

export function emailForVU(baseEmail, vu, env = {}) {
  if (!envFlag(env.UNIQUE_TEST_USERS)) {
    return baseEmail;
  }

  if (!baseEmail.includes("@")) {
    return baseEmail;
  }

  return baseEmail.replace("@", `+${vu}@`);
}

export function protectedEndpointPaths(schemeID) {
  const endpoints = [
    "/api/v1/auth/me",
    "/api/v1/schemes",
    "/api/v1/levies/attention",
  ];

  if (schemeID) {
    endpoints.push(`/api/v1/levies/${schemeID}`);
    endpoints.push(`/api/v1/maintenance/${schemeID}`);
  }

  return endpoints;
}

export function buildLoadOptions(env = {}) {
  const vus = positiveInt(env.LOAD_VUS || env.VUS, 10);
  const duration = env.LOAD_DURATION || env.DURATION || "2m";
  const httpReqDurationP95MS = positiveInt(env.HTTP_REQ_DURATION_P95_MS, 1000);

  return {
    scenarios: {
      auth_and_dashboard: {
        executor: "constant-vus",
        vus,
        duration,
        tags: { scenario: `${vus}_users` },
      },
    },
    thresholds: {
      login_success: ["rate>0.99"],
      refresh_blocked: ["rate==0"],
      repeat_login_fail: ["rate==0"],
      get_endpoints_fail: ["rate<0.01"],
      http_req_duration: [`p(95)<${httpReqDurationP95MS}`],
    },
  };
}

export function shouldPostSummary(env = {}) {
  return Boolean(env.SUMMARY_URL);
}
