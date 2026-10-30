// proxy.ts – protege as rotas dentro de /app/* e /admin/* e faz o proxy
// same-origin da API. No Next.js 16, "middleware" passou a se chamar "proxy".
import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

const JWT_COOKIE_NAME = "fitconecta_jwt";
const ADMIN_COOKIE_NAME = "fitconecta_admin_jwt";

// Em producao (Fly) o backend roda em outro hostname (backend-*.fly.dev).
// Um cookie gravado por esse dominio NAO e visivel para o frontend
// (fitconecta.fly.dev e "public suffix", logo sao sites distintos), e o
// SameSite=Strict impede o envio em requisicoes cross-site. Por isso
// requisicoes /api sao encaminhadas aqui, de forma same-origin: o cookie
// passa a ser gravado no dominio do frontend e o proxy de auth funciona.
// LOCAL: sem BACKEND_PROXY_URL, cai no comportamento direto (mesmo host).
const BACKEND_PROXY_URL = process.env.BACKEND_PROXY_URL || "";

export function proxy(request: NextRequest) {
  const url = request.nextUrl.clone();

  // 1) Proxy same-origin da API para o backend.
  if (url.pathname.startsWith("/api/")) {
    if (BACKEND_PROXY_URL) {
      const target = new URL(url.pathname + url.search, BACKEND_PROXY_URL);
      return NextResponse.rewrite(target);
    }
    return NextResponse.next();
  }

  // 2) Painel administrativo: mesma URL, cookie e login proprios.
  if (url.pathname === "/admin" || url.pathname.startsWith("/admin/")) {
    if (url.pathname === "/admin/login") {
      return NextResponse.next();
    }

    const adminToken = request.cookies.get(ADMIN_COOKIE_NAME)?.value;
    if (!adminToken) {
      url.pathname = "/admin/login";
      url.search = "";
      return NextResponse.redirect(url);
    }

    return NextResponse.next();
  }

  // 3) Guarda de autenticacao das rotas logadas.
  const token = request.cookies.get(JWT_COOKIE_NAME)?.value;

  const publicPaths = ["/sobre", "/contato", "/auth"];
  const isPublic =
    url.pathname === "/" ||
    publicPaths.some((path) => url.pathname.startsWith(path));

  if (isPublic) {
    return NextResponse.next();
  }

  if (!token) {
    url.pathname = "/auth/login";
    return NextResponse.redirect(url);
  }

  return NextResponse.next();
}

export const config = {
  matcher: ["/app/:path*", "/admin/:path*", "/api/:path*"],
};
