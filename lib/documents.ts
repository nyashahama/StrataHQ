export type DocumentFileType = "pdf" | "docx" | "xlsx" | "jpg" | "png";
export type DocumentCategory =
  | "rules"
  | "minutes"
  | "insurance"
  | "financial"
  | "other";

export interface SchemeDocumentInfo {
  uploaded_by_name?: string | null;
  id: string;
  scheme_id: string;
  name: string;
  storage_key: string;
  file_type: DocumentFileType;
  category: DocumentCategory;
  size_bytes: number;
  uploaded_at: string;
}

export interface DocumentsDashboard {
  documents: SchemeDocumentInfo[];
  role: string;
  total: number;
}
