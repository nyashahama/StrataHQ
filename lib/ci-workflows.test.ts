import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

describe("CI workflows", () => {
  it("runs govulncheck in backend CI", () => {
    const workflow = readFileSync(".github/workflows/backend-ci.yml", "utf8");

    expect(workflow).toContain("govulncheck ./...");
  });

  it("runs pnpm audit in frontend CI", () => {
    const workflow = readFileSync(".github/workflows/frontend-ci.yml", "utf8");

    expect(workflow).toContain("pnpm audit --prod --audit-level=high");
  });

  it("installs pnpm via pnpm/action-setup", () => {
    const workflow = readFileSync(".github/workflows/frontend-ci.yml", "utf8");

    expect(workflow).toContain("pnpm/action-setup@v4");
  });
});
