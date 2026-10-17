import { ImageResponse } from "next/og";

/**
 * Dynamic Open Graph / Twitter image (spec doc 07 §3).
 *
 * Generated on demand with `next/og` (no external sharp/canvas dependency).
 * Neobrutalist design: dark surface, thick border, royal brand blue accent, Funnel
 * Display inspired heavy typography.
 */
export const size = { width: 1200, height: 630 };
export const contentType = "image/png";
export const alt = "FinTrack — Catat Keuangan Otomatis dengan AI";

export default function OpengraphImage() {
  return new ImageResponse(
    (
      <div
        style={{
          width: "100%",
          height: "100%",
          display: "flex",
          flexDirection: "column",
          justifyContent: "space-between",
          padding: 64,
          backgroundColor: "#121212",
          color: "#ffffff",
          fontFamily: "system-ui, -apple-system, sans-serif",
          border: "8px solid #2563EB",
        }}
      >
        <div style={{ display: "flex", alignItems: "center", gap: 16 }}>
          <div
            style={{
              width: 56,
              height: 56,
              backgroundColor: "#2563EB",
              color: "#ffffff",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              fontSize: 32,
              fontWeight: 900,
              borderRadius: 8,
              border: "3px solid #000000",
            }}
          >
            F
          </div>
          <span
            style={{
              fontSize: 28,
              fontWeight: 900,
              letterSpacing: "-0.02em",
              color: "#ffffff",
            }}
          >
            FinTrack
          </span>
          <span
            style={{
              marginLeft: 16,
              padding: "4px 12px",
              backgroundColor: "#1E1E1E",
              borderRadius: 6,
              border: "2px solid #333333",
              fontSize: 14,
              fontWeight: 700,
              color: "#2563EB",
              letterSpacing: "0.08em",
              textTransform: "uppercase",
            }}
          >
            AI Finansial
          </span>
        </div>

        <div style={{ display: "flex", flexDirection: "column", gap: 16, maxWidth: 960 }}>
          <div
            style={{
              fontSize: 56,
              fontWeight: 900,
              lineHeight: 1.1,
              letterSpacing: "-0.02em",
              color: "#ffffff",
            }}
          >
            Catat Keuangan Otomatis dengan AI
          </div>
          <div
            style={{
              fontSize: 24,
              lineHeight: 1.4,
              color: "#a3a3a3",
            }}
          >
            Tulis lewat chat, ucapkan lewat suara, atau foto struk belanja Anda.
            Asisten AI berbahasa Indonesia untuk kelola keuangan tanpa spreadsheet.
          </div>
        </div>

        <div
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            borderTop: "3px solid #262626",
            paddingTop: 24,
          }}
        >
          <div style={{ display: "flex", gap: 32, fontSize: 16, color: "#e5e5e5", fontWeight: 700 }}>
            <span>* Chat &amp; Suara</span>
            <span>* Scan Struk (OCR)</span>
            <span>* Skor Finansial</span>
            <span>* Mulai Gratis</span>
          </div>
          <div
            style={{
              padding: "10px 24px",
              backgroundColor: "#2563EB",
              color: "#ffffff",
              fontSize: 18,
              fontWeight: 900,
              borderRadius: 6,
              border: "3px solid #000000",
            }}
          >
            fintrack.id
          </div>
        </div>
      </div>
    ),
    { ...size },
  );
}
