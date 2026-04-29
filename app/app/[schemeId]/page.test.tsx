import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { SchemeOverview } from "@/components/scheme/SchemeOverview";

describe("SchemeOverview", () => {
  it("renders core scheme details for trustees and admins", () => {
    render(
      <SchemeOverview
        isResident={false}
        attentionItems={[]}
        scheme={{
          id: "scheme-1",
          name: "Scheme 1",
          address: "Address 1",
          role: "admin",
          health: "good",
          health_score: 85,
          health_breakdown: {},
          unit_count: 10,
          total_members: 11,
          trustee_count: 2,
          resident_count: 9,
          levy_collection_pct: 96,
          open_maintenance_count: 2,
          notice_count: 1,
          unit_id: null,
          unit_identifier: null,
          next_agm_date: null,
          days_to_agm: null,
          units: [],
          recent_notices: [],
        }}
      />,
    );

    expect(screen.getByText("Scheme 1")).toBeInTheDocument();
    expect(screen.getByText("Scheme health")).toBeInTheDocument();
    expect(screen.getByText("Collections attention queue")).toBeInTheDocument();
  });

  it("hides attention queue for residents", () => {
    render(
      <SchemeOverview
        isResident={true}
        attentionItems={[]}
        scheme={{
          id: "scheme-1",
          name: "Scheme 1",
          address: "Address 1",
          role: "resident",
          health: "good",
          health_score: 85,
          health_breakdown: {},
          unit_count: 10,
          total_members: 11,
          trustee_count: 2,
          resident_count: 9,
          levy_collection_pct: 96,
          open_maintenance_count: 2,
          notice_count: 1,
          unit_id: "unit-1",
          unit_identifier: "1A",
          next_agm_date: null,
          days_to_agm: null,
          units: [],
          recent_notices: [],
        }}
      />,
    );

    expect(screen.queryByText("Collections attention queue")).not.toBeInTheDocument();
    expect(screen.getByText("Unit 1A · Welcome back.")).toBeInTheDocument();
  });
});
