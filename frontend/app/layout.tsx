import type { Metadata, Viewport } from "next";
import { Geist, Geist_Mono } from "next/font/google";

import { ConnectionStatus } from "@/components/connection-status";
import { ServiceWorkerRegistrar } from "@/components/service-worker-registrar";

import "./globals.css";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: {
    default: "SwasthyaSetu",
    template: "%s · SwasthyaSetu",
  },
  description:
    "The bridge to health — emergency consultations with volunteer doctors and AI-assisted first-aid triage, available offline.",
  applicationName: "SwasthyaSetu",
  manifest: "/manifest.webmanifest",
  appleWebApp: {
    capable: true,
    title: "SwasthyaSetu",
    statusBarStyle: "default",
  },
  icons: {
    icon: [
      { url: "/icon.svg", type: "image/svg+xml" },
      { url: "/icons/icon-192.png", sizes: "192x192", type: "image/png" },
    ],
    apple: "/apple-touch-icon.png",
  },
};

export const viewport: Viewport = {
  // Fills the notch area on an installed PWA.
  viewportFit: "cover",
  themeColor: [
    { media: "(prefers-color-scheme: light)", color: "#0d7f8c" },
    { media: "(prefers-color-scheme: dark)", color: "#0b2027" },
  ],
};

export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en" className={`${geistSans.variable} ${geistMono.variable}`}>
      <body className="min-h-dvh font-sans">
        <ConnectionStatus />
        {children}
        <ServiceWorkerRegistrar />
      </body>
    </html>
  );
}
