import { describe, expect, it } from "vitest";

import {
  buildLoadOptions,
  emailForVU,
  protectedEndpointPaths,
  shouldPostSummary,
} from "../backend/tests/load/auth-and-dashboard-config";

describe("production load test harness config", () => {
  it("uses one documented seeded user by default", () => {
    expect(emailForVU("demo@stratahq.com", 25, {})).toBe("demo@stratahq.com");
  });

  it("uses unique per-VU emails only when explicitly enabled", () => {
    expect(emailForVU("demo@stratahq.com", 25, { UNIQUE_TEST_USERS: "true" })).toBe(
      "demo+25@stratahq.com",
    );
  });

  it("runs one configurable load level per invocation", () => {
    expect(
      buildLoadOptions({
        LOAD_VUS: "25",
        LOAD_DURATION: "2m",
      }).scenarios.auth_and_dashboard,
    ).toMatchObject({
      executor: "constant-vus",
      vus: 25,
      duration: "2m",
    });
  });

  it("uses a configurable p95 HTTP duration threshold", () => {
    expect(
      buildLoadOptions({
        HTTP_REQ_DURATION_P95_MS: "30000",
      }).thresholds.http_req_duration,
    ).toEqual(["p(95)<30000"]);
  });

  it("posts summaries only when SUMMARY_URL is configured", () => {
    expect(shouldPostSummary({})).toBe(false);
    expect(shouldPostSummary({ SUMMARY_URL: "http://localhost:8081" })).toBe(true);
  });

  it("builds scheme-scoped dashboard endpoints from a discovered scheme id", () => {
    expect(protectedEndpointPaths("scheme-123")).toEqual([
      "/api/v1/auth/me",
      "/api/v1/schemes",
      "/api/v1/levies/attention",
      "/api/v1/levies/scheme-123",
      "/api/v1/maintenance/scheme-123",
    ]);
  });
});
