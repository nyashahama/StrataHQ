export interface ApiMeta {
  page: number;
  per_page: number;
  total: number;
}

export interface ApiSuccess<T> {
  data: T;
  meta?: ApiMeta;
}

export interface ApiErrorBody {
  code: string;
  message: string;
}

export interface ApiErrorResponse {
  error: ApiErrorBody;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

export async function readApiBody(response: Response): Promise<unknown> {
  const text = await response.text();
  if (!text) return null;

  try {
    return JSON.parse(text);
  } catch {
    return text;
  }
}

export function unwrapApiData<T>(body: unknown): T {
  if (isRecord(body) && "data" in body) {
    return body.data as T;
  }

  return body as T;
}

export function getApiErrorMessage(body: unknown, fallback: string): string {
  if (isRecord(body) && "error" in body && isRecord(body.error)) {
    const message = body.error.message;
    if (typeof message === "string" && message.trim() !== "") {
      return message;
    }
  }

  if (typeof body === "string" && body.trim() !== "") {
    return body;
  }

  return fallback;
}

export async function readApiData<T>(response: Response): Promise<T> {
  const body = await readApiBody(response);
  return unwrapApiData<T>(body);
}

export async function readApiError(
  response: Response,
  fallback: string,
): Promise<string> {
  const body = await readApiBody(response);
  return getApiErrorMessage(body, fallback);
}
