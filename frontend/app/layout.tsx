import type { Metadata, Viewport } from "next";
import { Inter, Manrope, Noto_Sans_Devanagari } from "next/font/google";

import { ConnectionStatus } from "@/components/connection-status";

import "./globals.css";

// Headings: Manrope. Text and UI: Inter, highly legible at small sizes.
const manrope = Manrope({
  variable: "--font-manrope",
  subsets: ["latin"],
  weight: ["500", "600", "700", "800"],
});

const inter = Inter({
  variable: "--font-inter",
  subsets: ["latin"],
});

// Neither has Devanagari glyphs. This keeps Nepali text consistent across
// devices instead of falling back to whatever the OS has.
const notoDevanagari = Noto_Sans_Devanagari({
  variable: "--font-devanagari",
  subsets: ["devanagari"],
  weight: ["400", "500", "600", "700"],
});

export const metadata: Metadata = {
  title: {
    default: "SwasthyaSetu: Bridging Health. Delivering Hope.",
    template: "%s · SwasthyaSetu",
  },
  description:
    "First-aid guidance in English and Nepali, then a live video call with a verified doctor.",
  applicationName: "SwasthyaSetu",
  manifest: "/manifest.webmanifest",
  icons: {
    icon: [
      { url: "/favicon.ico", sizes: "any" },
      { url: "/icons/icon-192.png", sizes: "192x192", type: "image/png" },
    ],
    apple: "/apple-touch-icon.png",
  },
};

export const viewport: Viewport = {
  viewportFit: "cover",
  themeColor: [
    { media: "(prefers-color-scheme: light)", color: "#f7faf8" },
    { media: "(prefers-color-scheme: dark)", color: "#0e1715" },
  ],
};

export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en" className={`${manrope.variable} ${inter.variable} ${notoDevanagari.variable}`}>
      <body className="min-h-dvh font-sans">
        <a
          href="#main"
          className="sr-only focus:not-sr-only focus:fixed focus:left-4 focus:top-3 focus:z-50 focus:rounded-full focus:bg-primary focus:px-4 focus:py-2 focus:text-sm focus:text-primary-foreground"
        >
          Skip to content
        </a>
        <ConnectionStatus />
        {children}
      </body>
    </html>
  );
}
