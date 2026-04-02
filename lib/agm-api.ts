"use client";

import { apiFetch } from "@/lib/api";
import { readApiData, readApiError } from "@/lib/api-contract";
import type { AgmDashboard, AgmMeetingInfo, AgmResolutionInfo, AgmVoteChoice } from "@/lib/agm";

async function parse<T>(response: Response, fallback: string): Promise<T> {
  if (!response.ok) {
    throw new Error(await readApiError(response, fallback));
  }
  return readApiData<T>(response);
}

export async function getAgmDashboard(schemeId: string): Promise<AgmDashboard> {
  return parse(
    await apiFetch(`/api/v1/agm/${schemeId}`),
    "Failed to load AGM dashboard",
  );
}

export async function scheduleAgmMeeting(
  schemeId: string,
  input: {
    date: string;
    quorum_required: number;
    resolutions: Array<{ title: string; description: string }>;
  },
): Promise<AgmMeetingInfo> {
  return parse(
    await apiFetch(`/api/v1/agm/${schemeId}/meetings`, {
      method: "POST",
      body: JSON.stringify(input),
    }),
    "Failed to schedule AGM",
  );
}

export async function castAgmVote(
  schemeId: string,
  resolutionId: string,
  choice: AgmVoteChoice,
): Promise<AgmResolutionInfo> {
  return parse(
    await apiFetch(`/api/v1/agm/${schemeId}/resolutions/${resolutionId}/vote`, {
      method: "POST",
      body: JSON.stringify({ choice }),
    }),
    "Failed to cast vote",
  );
}

export async function assignAgmProxy(
  schemeId: string,
  meetingId: string,
  granteeUserId: string,
): Promise<void> {
  const response = await apiFetch(`/api/v1/agm/${schemeId}/meetings/${meetingId}/proxy`, {
    method: "POST",
    body: JSON.stringify({ grantee_user_id: granteeUserId }),
  });

  if (!response.ok) {
    throw new Error(await readApiError(response, "Failed to assign proxy"));
  }
}
