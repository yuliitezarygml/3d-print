export const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

type Session = { accessToken: string; refreshToken: string; expiresIn: number; user: { id: string; name: string; role: string } };

export const session = {
  get(): Session | null {
    if (typeof window === "undefined") return null;
    const raw = localStorage.getItem("printforge.session");
    if (!raw) return null;
    try { return JSON.parse(raw) as Session; } catch { return null; }
  },
  set(value: Session) { localStorage.setItem("printforge.session", JSON.stringify(value)); },
  clear() { localStorage.removeItem("printforge.session"); },
};

export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const active = session.get();
  const headers = new Headers(init.headers);
  if (!(init.body instanceof FormData)) headers.set("Content-Type", "application/json");
  if (active?.accessToken) headers.set("Authorization", `Bearer ${active.accessToken}`);
  let response = await fetch(`${API_URL}${path}`, { ...init, headers });
  if (response.status === 401 && active?.refreshToken && path !== "/api/auth/refresh") {
    const refreshed = await fetch(`${API_URL}/api/auth/refresh`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ refreshToken: active.refreshToken }) });
    if (refreshed.ok) {
      const next = await refreshed.json() as Session;
      session.set(next);
      headers.set("Authorization", `Bearer ${next.accessToken}`);
      response = await fetch(`${API_URL}${path}`, { ...init, headers });
    } else session.clear();
  }
  if (!response.ok) {
    const body = await response.json().catch(() => ({ error: `Request failed (${response.status})` })) as { error?: string };
    throw new Error(body.error ?? `Request failed (${response.status})`);
  }
  if (response.status === 204) return undefined as T;
  return response.json() as Promise<T>;
}

export async function downloadAuthenticated(path: string, filename: string) {
  const active = session.get();
  const response = await fetch(`${API_URL}${path}`, { headers: active?.accessToken ? { Authorization: `Bearer ${active.accessToken}` } : {} });
  if (!response.ok) throw new Error("Не удалось скачать файл");
  const url = URL.createObjectURL(await response.blob());
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}

export function money(value: number, currency = "MDL") {
  return `${new Intl.NumberFormat("ru-MD", { minimumFractionDigits: 2, maximumFractionDigits: 2 }).format(value)} ${currency}`;
}

export function duration(minutes: number) {
  const h = Math.floor(minutes / 60);
  const m = minutes % 60;
  return h ? `${h} ч ${m} мин` : `${m} мин`;
}
