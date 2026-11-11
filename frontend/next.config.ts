import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  experimental: {
    // Necessario para o nudge de seguranca do Next (upgrade 16.4.0 -> 16.4.3).
    agentUpgrade: "security",
  },
  /* config options here */
  // "standalone" gera um servidor minimo para imagem Docker (ver Dockerfile).
  output: "standalone",
  cacheComponents: true,
  partialPrefetching: true,
  images: {
    // As imagens de perfil ficam no storage S3-compativel. Em producao (R2)
    // e um dominio https; em dev o storage roda em localhost:9000 (IP local),
    // o que exige liberar IPs locais no otimizador do next/image.
    dangerouslyAllowLocalIP: true,
    remotePatterns: [
      { protocol: "http", hostname: "localhost", port: "9000", pathname: "/**" },
      { protocol: "http", hostname: "127.0.0.1", port: "9000", pathname: "/**" },
      // No docker-compose o storage atende tanto o browser quanto a API em
      // minio.localhost:9000 (alias de rede + resolucao loopback do browser).
      { protocol: "http", hostname: "minio.localhost", port: "9000", pathname: "/**" },
      { protocol: "https", hostname: "**" },
    ],
  },
  async redirects() {
    return [
      // O menu "Usuarios" virou "Profissionais" e "Clientes".
      { source: "/admin/usuarios", destination: "/admin/profissionais", permanent: false },
    ];
  },
  async headers() {
    return [
      {
        source: "/admin/:path*",
        headers: [
          { key: "X-Robots-Tag", value: "noindex, nofollow" },
          { key: "X-Frame-Options", value: "DENY" },
        ],
      },
    ];
  },
  turbopack: {
    root: import.meta.dirname,
    rules: {
      "*.css": {
        loaders: ["@tailwindcss/turbopack"],
        as: "*.css",
      },
    },
  },
};

export default nextConfig;
