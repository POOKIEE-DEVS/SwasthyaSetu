import type { Metadata, Viewport } from "next";
import { Inter, Newsreader, Noto_Sans_Devanagari, Noto_Serif_Devanagari } from "next/font/google";

import { ConnectionStatus } from "@/components/connection-status";

import "./globals.css";

// Text, UI and UI headings: Inter, highly legible at small sizes.
const inter = Inter({
  variable: "--font-inter",
  subsets: ["latin"],
});

// The large marketing headlines only: an editorial serif, with optical
// sizes so it tightens up at display sizes.
const newsreader = Newsreader({
  variable: "--font-newsreader",
  subsets: ["latin"],
  axes: ["opsz"],
});

// Neither has Devanagari glyphs. These keep Nepali consistent across
// devices instead of falling back to whatever the OS has: sans for text,
// serif for the Nepali headlines (only loaded when Nepali is shown).
const notoDevanagari = Noto_Sans_Devanagari({
  variable: "--font-devanagari",
  subsets: ["devanagari"],
});

const notoSerifDevanagari = Noto_Serif_Devanagari({
  variable: "--font-serif-devanagari",
  subsets: ["devanagari"],
  preload: false,
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
    { media: "(prefers-color-scheme: light)", color: "#faf9f5" },
    { media: "(prefers-color-scheme: dark)", color: "#121614" },
  ],
};

export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html
      lang="en"
      className={`${inter.variable} ${newsreader.variable} ${notoDevanagari.variable} ${notoSerifDevanagari.variable}`}
    >
      <body className="min-h-dvh font-sans">
        {/* Without JavaScript the landing page's entrance can't happen, so
            show the hero straight away rather than after its fallback. */}
        <noscript>
          <style>{".hero-line,.hero-after,.hero .setu-bridge,.hero-intro-header{animation:none!important}"}</style>
        </noscript>
        <a
          href="#main"
          className="sr-only focus:not-sr-only focus:fixed focus:left-4 focus:top-3 focus:z-50 focus:rounded-[8px] focus:bg-primary focus:px-4 focus:py-2 focus:text-sm focus:text-primary-foreground"
        >
          Skip to content
        </a>
        <ConnectionStatus />
        {children}
      </body>
    </html>
  );
}
