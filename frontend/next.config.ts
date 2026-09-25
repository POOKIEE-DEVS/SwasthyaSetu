import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // A static export: plain HTML/JS/CSS in `out/`, served by the FastAPI
  // backend from the same origin. Every page here is client-side, so no
  // Node server is needed in production.
  output: "export",
  // Emits patient/index.html rather than patient.html, which is what the
  // backend's static file server resolves for /patient/.
  trailingSlash: true,
  images: { unoptimized: true },
};

export default nextConfig;
