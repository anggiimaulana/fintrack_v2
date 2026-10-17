"use client";

import * as React from "react";
import { Camera, Check, MessageSquare, Mic } from "lucide-react";
import { MICROCOPY } from "@/lib/marketing/copy";
import { cn } from "@/lib/utils";

/**
 * Interactive product demo (spec doc 03 §B.5): a static, backend-free
 * illustration of the three ways to record a transaction. All three panels live
 * in the DOM (`hidden` when inactive) so the content is readable without any
 * interaction; the tabs implement the ARIA tabs pattern with roving tabindex.
 */

const TABS = [
  { id: "chat", label: "Chat", icon: MessageSquare },
  { id: "voice", label: "Suara", icon: Mic },
  { id: "receipt", label: "Struk", icon: Camera },
] as const;

type TabId = (typeof TABS)[number]["id"];

const RECEIPT_ITEMS = [
  { label: "Beras 5 kg", amount: "Rp 75.000" },
  { label: "Minyak 2 L", amount: "Rp 40.000" },
  { label: "Telur 1 kg", amount: "Rp 32.000" },
  { label: "Sabun mandi", amount: "Rp 25.500" },
  { label: "Susu kotak 2×", amount: "Rp 40.000" },
  { label: "Camilan", amount: "Rp 100.000" },
];

function ResultBubble({ children }: { children: React.ReactNode }) {
  return (
    <p className="flex items-start gap-2 rounded-md border-2 border-foreground bg-card p-3 text-sm font-medium">
      <Check aria-hidden className="mt-0.5 size-4 shrink-0" />
      <span>{children}</span>
    </p>
  );
}

export function InteractiveDemo() {
  const [active, setActive] = React.useState<TabId>("chat");
  const tabRefs = React.useRef<Array<HTMLButtonElement | null>>([]);

  const onKeyDown = (event: React.KeyboardEvent<HTMLButtonElement>, index: number) => {
    let next: number | null = null;
    switch (event.key) {
      case "ArrowRight":
        next = (index + 1) % TABS.length;
        break;
      case "ArrowLeft":
        next = (index - 1 + TABS.length) % TABS.length;
        break;
      case "Home":
        next = 0;
        break;
      case "End":
        next = TABS.length - 1;
        break;
      default:
        return;
    }
    event.preventDefault();
    setActive(TABS[next].id);
    tabRefs.current[next]?.focus();
  };

  return (
    <div className="w-full">
      <div
        role="tablist"
        aria-label="Contoh cara mencatat transaksi"
        className="flex flex-wrap gap-2"
      >
        {TABS.map((tab, index) => {
          const selected = active === tab.id;
          const Icon = tab.icon;
          return (
            <button
              key={tab.id}
              ref={(el) => {
                tabRefs.current[index] = el;
              }}
              type="button"
              role="tab"
              id={`demo-tab-${tab.id}`}
              aria-selected={selected}
              aria-controls={`demo-panel-${tab.id}`}
              tabIndex={selected ? 0 : -1}
              onClick={() => setActive(tab.id)}
              onKeyDown={(event) => onKeyDown(event, index)}
              className={cn(
                "inline-flex items-center gap-2 rounded-md border-2 border-foreground px-3 py-1.5 text-sm font-bold",
                "shadow-[3px_3px_0px_0px] shadow-foreground transition-colors",
                selected
                  ? "bg-accent-acid text-accent-acid-foreground"
                  : "bg-card text-foreground hover:bg-muted",
              )}
            >
              <Icon aria-hidden className="size-4" />
              {tab.label}
            </button>
          );
        })}
      </div>

      <div className="mt-3 rounded-md border-[3px] border-foreground bg-background p-4 shadow-[6px_6px_0px_0px] shadow-foreground sm:p-6">
        <div
          role="tabpanel"
          id="demo-panel-chat"
          aria-labelledby="demo-tab-chat"
          hidden={active !== "chat"}
          tabIndex={0}
          className="flex flex-col gap-3"
        >
          <p className="self-end rounded-md border-2 border-foreground bg-primary p-3 text-sm font-medium text-primary-foreground">
            beli kopi 25 ribu tadi siang
          </p>
          <ResultBubble>Pengeluaran Rp 25.000 — Kopi. Tercatat.</ResultBubble>
        </div>

        <div
          role="tabpanel"
          id="demo-panel-voice"
          aria-labelledby="demo-tab-voice"
          hidden={active !== "voice"}
          tabIndex={0}
          className="flex flex-col gap-3"
        >
          <div aria-hidden className="flex items-end justify-center gap-1 py-4">
            {[9, 20, 32, 16, 40, 24, 12, 30, 18, 26, 14, 22].map((height, index) => (
              <span
                key={index}
                className="w-1.5 rounded-full bg-accent-acid-strong"
                style={{ height: `${height}px` }}
              />
            ))}
          </div>
          <p className="text-sm font-medium text-muted-foreground">“makan siang empat puluh ribu”</p>
          <ResultBubble>Pengeluaran Rp 40.000 — Makan. Tercatat.</ResultBubble>
        </div>

        <div
          role="tabpanel"
          id="demo-panel-receipt"
          aria-labelledby="demo-tab-receipt"
          hidden={active !== "receipt"}
          tabIndex={0}
          className="flex flex-col gap-3"
        >
          <div className="rounded-md border-2 border-dashed border-foreground bg-muted p-4 font-mono text-xs">
            <p className="mb-2 font-bold uppercase tracking-widest">Toko Sembako Sejahtera</p>
            <ul className="flex flex-col gap-1">
              {RECEIPT_ITEMS.map((item) => (
                <li key={item.label} className="flex justify-between gap-4">
                  <span className="truncate">{item.label}</span>
                  <span className="shrink-0 tabular-nums">{item.amount}</span>
                </li>
              ))}
            </ul>
            <p className="mt-2 flex justify-between gap-4 border-t-2 border-foreground pt-2 font-bold">
              <span>TOTAL</span>
              <span className="tabular-nums">Rp 312.500</span>
            </p>
          </div>
          <ResultBubble>Belanja bulanan Rp 312.500 tercatat, 6 item.</ResultBubble>
        </div>

        <p className="mt-4 font-mono text-xs uppercase tracking-[0.08em] text-muted-foreground">
          {MICROCOPY.demoLabel}
        </p>
      </div>
    </div>
  );
}
