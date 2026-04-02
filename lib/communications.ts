export type NoticeType = "general" | "urgent" | "agm" | "levy";

export interface NoticeInfo {
  sent_by_name?: string | null;
  id: string;
  scheme_id: string;
  title: string;
  body: string;
  type: NoticeType;
  sent_at: string;
}

export interface CommunicationsDashboard {
  notices: NoticeInfo[];
  role: string;
  total: number;
}
