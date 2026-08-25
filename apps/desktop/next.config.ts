import type { NextConfig } from "next";
import path from "node:path";

const internalHost = process.env.TAURI_DEV_HOST ?? "localhost";

const nextConfig: NextConfig = {
  output: "export",
  images: { unoptimized: true },
  assetPrefix: process.env.NODE_ENV === "development" ? `http://${internalHost}:3000` : undefined,
  trailingSlash: true,
  turbopack: { root: path.join(__dirname,"../..") },
};

export default nextConfig;
