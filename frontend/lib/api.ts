/**
 * Backend client.
 *
 * In production the frontend is served by the backend itself, so every
 * request is same-origin and NEXT_PUBLIC_API_URL is left unset. It is only
 * set for local development, when `next dev` runs on :3000 and the API on
 * :8000. It must be read as a literal `process.env.NEXT_PUBLIC_API_URL`,
 * because Next.js inlines it at build time.
 */

export const API_BASE = (process.env.NEXT_PUBLIC_API_URL ?? "").replace(/\/$/, "");

export type ChatMessage = { role: "user" | "assistant"; content: string };
export type ChatResponse = { reply: string; urgent: boolean };

export type IceServer = {
  urls: string[];
  username?: string | null;
  credential?: string | null;
};

export type Consultation = {
  id: string;
  patient_name: string;
  summary: string | null;
  status: "waiting" | "active" | "ended";
  created_at: number;
};

export type CallTicket = {
  consultation: Consultation;
  role: "patient" | "doctor";
  token: string;
  ice_servers: IceServer[];
};

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  let response: Response;
  try {
    response = await fetch(`${API_BASE}${path}`, {
      ...init,
      headers: { "Content-Type": "application/json", ...init?.headers },
    });
  } catch {
    throw new ApiError(0, "Can't reach the server. Check your internet connection.");
  }

  if (!response.ok) {
    let detail = `Request failed (${response.status}).`;
    try {
      const body = await response.json();
      if (typeof body.detail === "string") detail = body.detail;
    } catch {
      // Non-JSON error body; keep the generic message.
    }
    throw new ApiError(response.status, detail);
  }

  return response.status === 204 ? (undefined as T) : ((await response.json()) as T);
}

export const api = {
  chat: (messages: ChatMessage[]) =>
    request<ChatResponse>("/api/v1/chat", {
      method: "POST",
      body: JSON.stringify({ messages }),
    }),

  requestDoctor: (patientName: string, summary: string | null) =>
    request<CallTicket>("/api/v1/consultations", {
      method: "POST",
      body: JSON.stringify({ patient_name: patientName, summary }),
    }),

  accept: (consultationId: string) =>
    request<CallTicket>(`/api/v1/consultations/${consultationId}/accept`, {
      method: "POST",
    }),

  end: (consultationId: string, token: string) =>
    request<void>(`/api/v1/consultations/${consultationId}/end`, {
      method: "POST",
      body: JSON.stringify({ token }),
      // Lets the request finish even if the tab is closing.
      keepalive: true,
    }),
};

/** ws:// or wss:// URL for a backend WebSocket path, matching the page's scheme. */
export function wsUrl(path: string): string {
  const url = new URL(path, API_BASE || window.location.origin);
  url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
  return url.toString();
}
