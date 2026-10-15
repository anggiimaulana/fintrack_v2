import type { NextConfig } from "next";

const backendOrigin = process.env.BACKEND_ORIGIN || "http://localhost:8080";

// Next's rewrite proxy aborts a request after 30s by default — the exact
// symptom users hit on receipt scanning. Raise it well above the slowest AI
// call (vision/voice can take a minute+) so the proxy never cuts a healthy
// answer short. Override with API_PROXY_TIMEOUT_MS when a deployment needs a
// different ceiling. Note: the value must be a positive number; 0 falls back to
// Next's 30s default.
const apiProxyTimeoutMs = Number(process.env.API_PROXY_TIMEOUT_MS) || 180_000;

const nextConfig: NextConfig = {
  experimental: {
    proxyTimeout: apiProxyTimeoutMs,
  },
  async redirects() {
    return [
      {
        source: "/settings/ai-keys",
        destination: "/user/subscription",
        permanent: true,
      },
      {
        source: "/user/api-keys",
        destination: "/user/subscription",
        permanent: true,
      },
      {
        // Langganan moved out of Settings into its own top-level route (the old
        // path keeps working for bookmarks, notifications, and search).
        source: "/user/settings/subscription",
        destination: "/user/subscription",
        permanent: true,
      },
      {
        // Profile and settings merged into one page (accordion groups); the old
        // settings route keeps working for bookmarks, notifications, and search.
        source: "/user/settings",
        destination: "/user/profile",
        permanent: true,
      },
    ];
  },
  async rewrites() {
    return [
      {
        source: "/api/:path*",
        destination: `${backendOrigin}/api/:path*`,
      },
    ];
  },
};

export default nextConfig;
