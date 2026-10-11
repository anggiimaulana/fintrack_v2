"use client";

import { Skeleton } from "@/components/ui/skeleton";
import { ArrowRight, Info } from "lucide-react";
import Link from "next/link";

interface SpkWidgetProps {
  score: number;
  status: "Sangat Baik" | "Baik" | "Cukup" | "Boros" | "Kritis" | "Belum Ada Data";
  insights: string[];
  isLoading?: boolean;
  hasData?: boolean;
}

export function SpkWidgetSkeleton() {
  return (
    <div className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-5 flex flex-col gap-4">
      <div className="flex justify-between items-center">
        <Skeleton className="h-5 w-32 border-2 border-foreground" />
        <Skeleton className="h-5 w-5 border-2 border-foreground" />
      </div>
      <div className="flex items-center gap-6 my-4">
        <Skeleton className="w-24 h-24 border-2 border-foreground" />
        <div className="space-y-2 flex-1">
          <Skeleton className="h-6 w-24 border-2 border-foreground" />
          <Skeleton className="h-4 w-full border-2 border-foreground" />
          <Skeleton className="h-4 w-4/5 border-2 border-foreground" />
        </div>
      </div>
      <div className="space-y-3">
        {[1, 2, 3].map((i) => (
          <div key={i} className="flex gap-2">
            <Skeleton className="w-1.5 h-1.5 mt-1.5 shrink-0 border-2 border-foreground" />
            <Skeleton className="h-4 w-full border-2 border-foreground" />
          </div>
        ))}
      </div>
    </div>
  );
}

export function SpkWidget({ score, status, insights, isLoading, hasData }: SpkWidgetProps) {
  if (isLoading) {
    return <SpkWidgetSkeleton />;
  }

  const getStatusStyle = () => {
    switch (status) {
      case "Sangat Baik": return "bg-green-100 text-green-700 border-green-300";
      case "Baik": return "bg-primary/10 text-primary-strong border-primary/30";
      case "Cukup": return "bg-amber-100 text-amber-700 border-amber-300";
      case "Boros": return "bg-orange-100 text-orange-700 border-orange-300";
      case "Kritis": return "bg-red-100 text-red-700 border-red-300";
      case "Belum Ada Data": return "bg-neutral-100 text-neutral-500 border-neutral-300";
      default: return "bg-neutral-100 text-neutral-700 border-neutral-300";
    }
  };

  const getProgressColor = () => {
    switch (status) {
      case "Sangat Baik": return "stroke-green-500";
      case "Baik": return "stroke-primary";
      case "Cukup": return "stroke-amber-500";
      case "Boros": return "stroke-orange-500";
      case "Kritis": return "stroke-red-600";
      case "Belum Ada Data": return "stroke-neutral-300";
      default: return "stroke-neutral-500";
    }
  };

  const colors = getStatusStyle();
  const progressColor = getProgressColor();

  return (
    <div className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-5 flex flex-col h-full">
      <div className="flex justify-between items-center mb-5">
        <h3 className="text-xs font-black uppercase tracking-wide text-foreground">Kondisi Keuangan</h3>
        <button className="text-muted-foreground hover:text-foreground cursor-pointer" title="Sistem Pendukung Keputusan">
          <Info className="w-4 h-4" />
        </button>
      </div>

      <div className="flex flex-col md:flex-row items-center gap-6 mb-6">
        <div className="relative w-28 h-28 shrink-0 flex items-center justify-center">
          <svg className="absolute inset-0 w-full h-full -scale-x-100" viewBox="0 0 100 100">
            <circle cx="50" cy="50" r="46" fill="transparent" stroke="#e5e7eb" strokeWidth="8" />
            <circle 
              cx="50" cy="50" r="46" 
              fill="transparent" 
              strokeWidth="8" 
              strokeDasharray="289" 
              strokeDashoffset={289 - (289 * score) / 100} 
              strokeLinecap="square"
              className={progressColor}
            />
          </svg>
          <div className="flex flex-col items-center">
            <span className="text-3xl font-black font-mono text-foreground">{score}</span>
          </div>
        </div>
        
        <div className="flex-1 text-center md:text-left">
          <div className={`inline-block border-2 border-foreground px-3 py-1 text-sm font-black uppercase tracking-wide ${colors}`}>
            {status}
          </div>
          <p className="text-xs text-muted-foreground font-medium mt-2">
            {hasData ? "Berdasarkan analisis SPK pada pola transaksi Anda 30 hari terakhir." : "Mulai catat transaksi untuk melihat analisis keuangan."}
          </p>
        </div>
      </div>

      <div className="flex-1 space-y-3">
        <h4 className="text-[10px] font-black uppercase tracking-widest text-muted-foreground">Insights Utama</h4>
        {insights.map((insight, i) => (
          <div key={i} className="flex gap-2 items-start text-sm font-medium text-muted-foreground">
            <div className="w-2 h-2 bg-foreground mt-1.5 shrink-0"></div>
            <p>{insight}</p>
          </div>
        ))}
      </div>

      <Link href="/user/reports" className="mt-5 flex items-center justify-center gap-2 w-full py-2.5 text-sm font-black text-primary bg-primary/10 border-2 border-foreground shadow-[3px_3px_0px_0px] shadow-foreground hover:shadow-[2px_2px_0px_0px] hover:shadow-foreground active:shadow-none active:translate-x-[3px] active:translate-y-[3px] transition-all cursor-pointer uppercase tracking-wide">
        Lihat Detail Laporan
        <ArrowRight className="w-4 h-4" />
      </Link>
    </div>
  );
}
