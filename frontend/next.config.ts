import type { NextConfig } from "next";

const backendOrigin = process.env.BACKEND_ORIGIN || "http://localhost:8080";

const nextConfig: NextConfig = {
  async redirects() {
    return [
      {
        source: "/settings/ai-keys",
        destination: "/user/settings/subscription",
        permanent: true,
      },
      {
        source: "/user/api-keys",
        destination: "/user/settings/subscription",
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
