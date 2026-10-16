import type { Metadata, Viewport } from "next";
import { Inter, JetBrains_Mono } from "next/font/google";
import "./globals.css";
import { Providers } from "./providers";
import { THEME_INIT_SCRIPT } from "@/lib/theme";

const inter = Inter({
  variable: "--font-inter",
  subsets: ["latin"],
});

const jetbrainsMono = JetBrains_Mono({
  variable: "--font-jetbrains-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "FinTrack - Kelola Keuangan Cerdas",
  description: "Catat transaksi via teks, suara, atau foto. AI FinTrack memahami bahasa Indonesia dan langsung menyimpan ke database Anda.",
};

// `viewportFit: "cover"` lets the mobile bottom navigation extend into the
// iPhone home-indicator area; the nav itself pads with `env(safe-area-inset-*)`
// so nothing interactive is hidden behind the indicator.
export const viewport: Viewport = {
  width: "device-width",
  initialScale: 1,
  viewportFit: "cover",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html
      lang="id"
      suppressHydrationWarning
      className={`${inter.variable} ${jetbrainsMono.variable} h-full antialiased`}
    >
      <head>
        {/* Anti-flash: apply the stored theme before first paint. This lives in
            the server layout's <head> (where a <script> is legal) instead of
            inside a client component, which React 19 would refuse to execute. */}
        <script dangerouslySetInnerHTML={{ __html: THEME_INIT_SCRIPT }} />
      </head>
      <body className="min-h-full flex flex-col">
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
