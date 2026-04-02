export type AgmVoteChoice = "for" | "against";
export type AgmResolutionStatus = "open" | "passed" | "failed";
export type AgmMeetingStatus = "upcoming" | "in_progress" | "closed";

export interface AgmResolutionInfo {
  user_vote?: AgmVoteChoice | null;
  id: string;
  meeting_id: string;
  title: string;
  description: string;
  votes_for: number;
  votes_against: number;
  total_eligible: number;
  status: AgmResolutionStatus;
  created_at: string;
}

export interface AgmMeetingInfo {
  user_proxy_grantee_id?: string | null;
  resolutions: AgmResolutionInfo[];
  id: string;
  scheme_id: string;
  date: string;
  status: AgmMeetingStatus;
  quorum_required: number;
  quorum_present: number;
}

export interface AgmDashboard {
  latest?: AgmMeetingInfo | null;
  upcoming?: AgmMeetingInfo | null;
  role: string;
}
