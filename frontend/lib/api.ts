/**
 * Backend client.
 *
 * In production the frontend is served by the backend itself, so every
 * request is same-origin and NEXT_PUBLIC_API_URL is left unset. It is only
 * set for local development, when `next dev` runs on :3000 and the API on
 * :8000. It must be read as a literal `process.env.NEXT_PUBLIC_API_URL`,
 * because Next.js inlines it at build time.
 *
 * Sign-in is a session cookie set by the backend (HttpOnly, so scripts on
 * the page can never read it). `credentials: "include"` sends it on the
 * cross-origin development setup too.
 */

export const API_BASE = (process.env.NEXT_PUBLIC_API_URL ?? "").replace(/\/$/, "");

export type ChatMessage = { role: "user" | "assistant"; content: string };
export type ChatResponse = { reply: string; urgent: boolean; chat_id: string | null };

export type IceServer = {
  urls: string[];
  username?: string | null;
  credential?: string | null;
};

export type ProfessionalRole = "doctor" | "pharmacist" | "student";
export type Role = "patient" | ProfessionalRole;

export type ProfessionalBadge = { name: string; role: ProfessionalRole };

export type Consultation = {
  id: string;
  patient_name: string;
  summary: string | null;
  status: "waiting" | "active" | "ended";
  created_at: number;
  professional?: ProfessionalBadge | null;
};

export type CallTicket = {
  consultation: Consultation;
  role: "patient" | "doctor";
  token: string;
  ice_servers: IceServer[];
};

export type VerificationStatus = "pending" | "approved" | "rejected";

export type VerificationSummary = {
  role: ProfessionalRole;
  status: VerificationStatus;
  rejection_reason: string | null;
  full_name: string;
};

export type User = {
  id: number;
  email: string;
  name: string;
  picture_url: string | null;
  role: Role | null;
  is_admin: boolean;
  verification: VerificationSummary | null;
};

export type Me = { user: User | null; google_enabled: boolean; dev_login: boolean };

export type DocumentKind =
  | "citizenship_front"
  | "citizenship_back"
  | "council_certificate"
  | "recommendation_letter"
  | "selfie";

export type DocumentInfo = { id: number; kind: DocumentKind; content_type: string; size: number };

export type Application = {
  id: number;
  role: ProfessionalRole;
  full_name: string;
  phone: string;
  citizenship_number: string;
  citizenship_district: string;
  council_number: string | null;
  institution: string | null;
  recommender_name: string | null;
  recommender_nmc: string | null;
  status: VerificationStatus;
  rejection_reason: string | null;
  submitted_at: number;
  reviewed_at: number | null;
  documents: DocumentInfo[];
};

export type AuditEvent = {
  action: "submitted" | "resubmitted" | "approved" | "rejected";
  actor_email: string;
  reason: string | null;
  at: number;
};

export type AdminApplication = Application & {
  user_email: string;
  user_name: string;
  user_picture_url: string | null;
  history: AuditEvent[];
};

export type ChatSummary = { id: string; title: string; updated_at: number; message_count: number };
export type ChatDetail = { id: string; title: string; updated_at: number; messages: ChatMessage[] };

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  // JSON bodies get a JSON content type. FormData (file uploads) must not:
  // the browser sets the multipart boundary itself.
  const headers =
    typeof init?.body === "string"
      ? { "Content-Type": "application/json", ...init.headers }
      : init?.headers;

  let response: Response;
  try {
    response = await fetch(`${API_BASE}${path}`, {
      ...init,
      headers,
      credentials: "include",
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

const post = (body?: unknown): RequestInit => ({
  method: "POST",
  body: body === undefined ? undefined : JSON.stringify(body),
});

export const api = {
  chat: (messages: ChatMessage[], chatId: string | null) =>
    request<ChatResponse>("/api/v1/chat", post({ messages, chat_id: chatId })),

  requestDoctor: (patientName: string, summary: string | null) =>
    request<CallTicket>(
      "/api/v1/consultations",
      post({ patient_name: patientName, summary }),
    ),

  accept: (consultationId: string) =>
    request<CallTicket>(`/api/v1/consultations/${consultationId}/accept`, post()),

  end: (consultationId: string, token: string) =>
    request<void>(`/api/v1/consultations/${consultationId}/end`, {
      ...post({ token }),
      // Lets the request finish even if the tab is closing.
      keepalive: true,
    }),

  auth: {
    me: () => request<Me>("/api/v1/auth/me"),
    logout: () => request<void>("/api/v1/auth/logout", post()),
    chooseRole: (role: Role) => request<User>("/api/v1/auth/role", post({ role })),
    devLogin: (email: string, name: string) =>
      request<User>("/api/v1/auth/dev-login", post({ email, name })),
    /** A full-page navigation, not fetch: Google's consent screen takes over. */
    googleLoginUrl: (next: string) =>
      `${API_BASE}/api/v1/auth/google/login?${new URLSearchParams({ next })}`,
  },

  chats: {
    list: () => request<ChatSummary[]>("/api/v1/chats"),
    get: (id: string) => request<ChatDetail>(`/api/v1/chats/${encodeURIComponent(id)}`),
    save: (messages: ChatMessage[]) => request<ChatSummary>("/api/v1/chats", post({ messages })),
    remove: (id: string) =>
      request<void>(`/api/v1/chats/${encodeURIComponent(id)}`, { method: "DELETE" }),
  },

  applications: {
    mine: () => request<Application | null>("/api/v1/applications/me"),
    submit: (form: FormData) =>
      request<Application>("/api/v1/applications", { method: "POST", body: form }),
  },

  admin: {
    list: (status: VerificationStatus | "all") =>
      request<AdminApplication[]>(`/api/v1/admin/applications?status=${status}`),
    approve: (id: number) =>
      request<AdminApplication>(`/api/v1/admin/applications/${id}/approve`, post()),
    reject: (id: number, reason: string) =>
      request<AdminApplication>(`/api/v1/admin/applications/${id}/reject`, post({ reason })),
    documentUrl: (id: number) => `${API_BASE}/api/v1/admin/documents/${id}`,
  },
};

/** ws:// or wss:// URL for a backend WebSocket path, matching the page's scheme. */
export function wsUrl(path: string): string {
  const url = new URL(path, API_BASE || window.location.origin);
  url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
  return url.toString();
}

export const ROLE_LABEL: Record<Role, string> = {
  patient: "Patient",
  doctor: "Doctor",
  pharmacist: "Pharmacist",
  student: "MBBS Student",
};
