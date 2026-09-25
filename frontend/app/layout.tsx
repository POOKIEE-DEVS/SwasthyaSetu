import type { Metadata, Viewport } from "next";
import { Geist, Noto_Sans_Devanagari } from "next/font/google";

import { ConnectionStatus } from "@/components/connection-status";

import "./globals.css";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

// Geist has no Devanagari glyphs. This keeps Nepali text consistent across
// devices instead of falling back to whatever the OS has.
const notoDevanagari = Noto_Sans_Devanagari({
  variable: "--font-devanagari",
  subsets: ["devanagari"],
  weight: ["400", "500", "600"],
});

export const metadata: Metadata = {
  title: {
    default: "SwasthyaSetu",
    template: "%s · SwasthyaSetu",
  },
  description:
    "First-aid guidance from an AI assistant and a live video call with a volunteer doctor.",
  applicationName: "SwasthyaSetu",
  manifest: "/manifest.webmanifest",
  icons: {
    icon: [
      { url: "/icon.svg", type: "image/svg+xml" },
      { url: "/icons/icon-192.png", sizes: "192x192", type: "image/png" },
    ],
    apple: "/apple-touch-icon.png",
  },
};

export const viewport: Viewport = {
  viewportFit: "cover",
  themeColor: "#0d7f8c",
};

export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en" className={`${geistSans.variable} ${notoDevanagari.variable}`}>
      <body className="min-h-dvh font-sans">
        <ConnectionStatus />
        {children}
      </body>
    </html>
  );
}
