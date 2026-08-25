import type { Metadata } from "next";
import { Inter, Manrope } from "next/font/google";
import "./globals.css";
import "./desktop-extra.css";

const inter = Inter({ subsets:["latin","cyrillic"], variable:"--font-body" });
const manrope = Manrope({ subsets:["latin","cyrillic"], variable:"--font-display" });

export const metadata: Metadata = {
  title: "PrintForge Desktop",
  description: "Offline-first 3D printing workshop management",
};

export default function RootLayout({ children }:{ children:React.ReactNode }) {
  return <html lang="ru"><body className={`${inter.variable} ${manrope.variable}`}>{children}</body></html>;
}
