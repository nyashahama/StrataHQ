export type WhatsAppSender = "resident" | "bot" | "operator";
export type WhatsAppBroadcastType = "general" | "agm" | "levy" | "maintenance";

export interface WhatsAppMedia {
  id: string;
  url: string;
  content_type: string;
}

export interface WhatsAppMaintenanceIntake {
  maintenance_request_id?: string | null;
  created_at: string;
  id: string;
  scheme_id: string;
  thread_id: string;
  message_id: string;
  unit_id: string;
  unit_identifier: string;
  owner_name: string;
  status: "candidate" | "ticket_created" | "dismissed";
  category: "plumbing" | "electrical" | "structural" | "garden" | "pool" | "other";
  title: string;
  description: string;
  media_count: number;
}

export interface WhatsAppMessage {
  maintenance_request_id?: string | null;
  media: WhatsAppMedia[];
  id: string;
  from: WhatsAppSender;
  text: string;
  sent_at: string;
}

export interface WhatsAppThread {
  phone_number?: string | null;
  messages: WhatsAppMessage[];
  id: string;
  unit_id: string;
  unit_identifier: string;
  owner_name: string;
  connected: boolean;
  last_active: string;
  unread: number;
}

export interface WhatsAppBroadcast {
  sent_by_name?: string | null;
  id: string;
  scheme_id: string;
  message: string;
  type: WhatsAppBroadcastType;
  sent_at: string;
  recipient_count: number;
  delivered_recipient_count?: number;
  failed_recipient_count?: number;
}

export interface WhatsAppDashboard {
  resident_thread?: WhatsAppThread | null;
  threads: WhatsAppThread[];
  broadcasts: WhatsAppBroadcast[];
  maintenance_intakes: WhatsAppMaintenanceIntake[];
  role: string;
  phone_number: string;
  total_residents: number;
  connected_count: number;
  unread_count: number;
}
