// frontend/lib/api.ts – simple fetch wrapper.
// As chamadas usam caminho relativo /api/* (same-origin). Em producao o
// proxy.ts encaminha /api/* para o backend, permitindo que o cookie de
// sessao (HttpOnly) seja gravado no dominio do frontend. Isso e necessario
// porque o backend roda em outro hostname e o cookie nao cruzaria dominios.
const BASE_URL = "";

export async function apiGet<T>(url: string): Promise<T> {
  const res = await fetch(`${BASE_URL}${url}`, {
    method: "GET",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
  });
  if (!res.ok) {
    const err = await res.text();
    throw new Error(err || "GET request failed");
  }
  return (await res.json()) as T;
}

export async function apiPost<T>(url: string, body: unknown): Promise<T> {
  const res = await fetch(`${BASE_URL}${url}`, {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    const err = await res.text();
    throw new Error(err || "POST request failed");
  }
  return (await res.json()) as T;
}

export async function apiPut<T>(url: string, body: unknown): Promise<T> {
  const res = await fetch(`${BASE_URL}${url}`, {
    method: "PUT",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    const err = await res.text();
    throw new Error(err || "PUT request failed");
  }
  return (await res.json()) as T;
}

export async function apiDelete<T>(url: string): Promise<T> {
  const res = await fetch(`${BASE_URL}${url}`, {
    method: "DELETE",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
  });
  if (!res.ok) {
    const err = await res.text();
    throw new Error(err || "DELETE request failed");
  }
  return (await res.json()) as T;
}

// Extrai a mensagem de erro do envelope da API ({"success":false,"error":"..."})
// ou devolve uma mensagem amigavel para falhas de rede/CORS.
export function apiErrorMessage(error: unknown, fallback: string): string {
  const raw = error instanceof Error ? error.message : "";
  try {
    const parsed = JSON.parse(raw) as { error?: string; message?: string };
    if (parsed?.error) return parsed.error;
    if (parsed?.message) return parsed.message;
  } catch {
    // Nao e JSON: provavelmente falha de rede/CORS do fetch.
  }
  if (/fetch|network|failed to load/i.test(raw)) {
    const base =
      typeof window !== "undefined" ? window.location.origin : "/api";
    return `Não foi possível conectar ao servidor (${base}). Tente novamente.`;
  }
  return fallback;
}
