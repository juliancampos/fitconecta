import type { ApiEnvelope } from "@/lib/admin/types";

const BASE_URL = "";

async function request<T>(url: string, options: RequestInit): Promise<T> {
  const res = await fetch(`${BASE_URL}${url}`, {
    credentials: "include",
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...(options.headers || {}),
    },
  });

  const text = await res.text();
  let payload: ApiEnvelope<T> | null = null;
  try {
    payload = text ? (JSON.parse(text) as ApiEnvelope<T>) : null;
  } catch {
    payload = null;
  }

  if (!res.ok) {
    if (
      res.status === 401 &&
      typeof window !== "undefined" &&
      !url.endsWith("/auth/login")
    ) {
      window.location.replace("/admin/login");
    }
    throw new Error(text || `Request failed (${res.status})`);
  }

  return (payload?.data as T) ?? ({} as T);
}

export function apiGet<T>(url: string): Promise<T> {
  return request<T>(url, { method: "GET" });
}

export function apiPost<T>(url: string, body: unknown = {}): Promise<T> {
  return request<T>(url, { method: "POST", body: JSON.stringify(body) });
}

export function apiPatch<T>(url: string, body: unknown): Promise<T> {
  return request<T>(url, { method: "PATCH", body: JSON.stringify(body) });
}

export function apiErrorMessage(error: unknown, fallback: string): string {
  const raw = error instanceof Error ? error.message : "";
  try {
    const parsed = JSON.parse(raw) as { error?: string; message?: string };
    if (parsed?.error) return parsed.error;
    if (parsed?.message) return parsed.message;
  } catch {
    // Resposta nao-JSON: provavel falha de rede ou proxy.
  }
  if (/fetch|network|failed to load/i.test(raw)) {
    return "Nao foi possivel conectar ao servidor. Tente novamente.";
  }
  return fallback;
}
