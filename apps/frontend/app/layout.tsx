import type { Metadata } from "next";
import { Inter, Manrope } from "next/font/google";
import { Providers } from "@/components/providers";
import "./globals.css";
import "./theme.css";

const inter = Inter({ subsets: ["latin", "cyrillic"], variable: "--font-body" });
const manrope = Manrope({ subsets: ["latin", "cyrillic"], variable: "--font-display" });

export const metadata: Metadata = { title: "PrintForge — управление 3D-печатью", description: "Заказы, принтеры, материалы и честная себестоимость в одной системе." };

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return <html lang="ru" suppressHydrationWarning><body className={`${inter.variable} ${manrope.variable}`}><Providers>{children}</Providers></body></html>;
}
