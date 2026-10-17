import type { Metadata, Viewport } from "next";
import { Inter, JetBrains_Mono, Funnel_Display } from "next/font/google";
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

// Display font for marketing headings (spec doc 03 §A.4). Swap the single
// import below to change it (alts: Lexend_Mega, Baloo_2).
const display = Funnel_Display({
  variable: "--font-display",
  subsets: ["latin"],
});

const siteUrl = process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3001";

export const metadata: Metadata = {
  metadataBase: new URL(siteUrl),
  title: {
    default: "FinTrack — Catat Keuangan Otomatis dengan AI",
    template: "%s · FinTrack",
  },
  description:
    "FinTrack mencatat transaksi lewat chat, suara, atau foto struk. AI berbahasa Indonesia yang menyimpan rapi dan menganalisis kondisi keuangan Anda.",
  applicationName: "FinTrack",
  keywords: [
    "aplikasi pencatat keuangan",
    "catat keuangan otomatis",
    "catat pengeluaran",
    "scan struk otomatis",
    "analisis keuangan pribadi",
    "aplikasi keuangan AI",
  ],
  openGraph: {
    type: "website",
    locale: "id_ID",
    siteName: "FinTrack",
    url: siteUrl,
    title: "FinTrack — Catat Keuangan Otomatis dengan AI",
    description:
      "Catat transaksi lewat chat, suara, atau foto struk. AI berbahasa Indonesia.",
    images: [
      {
        url: `${siteUrl}/opengraph-image`,
        width: 1200,
        height: 630,
        alt: "FinTrack — Catat Keuangan Otomatis dengan AI",
      },
    ],
  },
  twitter: {
    card: "summary_large_image",
    title: "FinTrack — Catat Keuangan Otomatis dengan AI",
    description:
      "Catat transaksi lewat chat, suara, atau foto struk. AI berbahasa Indonesia.",
    images: [`${siteUrl}/opengraph-image`],
  },
  alternates: { canonical: "/" },
  robots: { index: true, follow: true },
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
      className={`${inter.variable} ${jetbrainsMono.variable} ${display.variable} h-full antialiased`}
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
