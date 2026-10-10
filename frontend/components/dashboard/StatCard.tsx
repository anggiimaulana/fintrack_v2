"use client";

import { Skeleton } from "@/components/ui/skeleton";
import { LucideIcon, TrendingUp, TrendingDown } from "lucide-react";

interface StatCardProps {
  title: string;
  amount: number;
  trend?: number;
  icon: LucideIcon;
  variant: "success" | "danger" | "primary" | "neutral";
}

export function StatCard({ title, amount, trend, icon: Icon, variant }: StatCardProps) {
  const isPositiveTrend = trend !== undefined && trend >= 0;
  
  const variantStyles = {
    success: "bg-green-100 text-green-700",
    danger: "bg-red-100 text-red-700",
    primary: "bg-primary/10 text-primary",
    neutral: "bg-neutral-100 text-neutral-700",
  };

  const formattedAmount = new Intl.NumberFormat("id-ID", {
    style: "currency",
    currency: "IDR",
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(amount);

  return (
    <div className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-5 flex flex-col gap-3">
      <div className="flex items-center gap-3">
        <div className={`w-10 h-10 border-2 border-foreground flex items-center justify-center ${variantStyles[variant]}`}>
          <Icon className="w-5 h-5" />
        </div>
        <h3 className="text-xs font-black uppercase tracking-wide text-muted-foreground">{title}</h3>
      </div>
      <div className="flex items-baseline justify-between">
        <div className="text-xl font-black font-mono text-foreground">{formattedAmount}</div>
        {trend !== undefined && (
          <div className={`flex items-center gap-1 text-xs font-bold px-2 py-1 border-2 border-foreground ${
            isPositiveTrend ? "bg-green-100 text-green-700" : "bg-red-100 text-red-700"
          }`}>
            {isPositiveTrend ? <TrendingUp className="w-3 h-3" /> : <TrendingDown className="w-3 h-3" />}
            {Math.abs(trend)}%
          </div>
        )}
      </div>
    </div>
  );
}

export function StatCardSkeleton() {
  return (
    <div className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-5 flex flex-col gap-3">
      <div className="flex items-center gap-3">
        <Skeleton className="w-10 h-10 border-2 border-foreground" />
        <Skeleton className="h-3 w-24" />
      </div>
      <div className="flex items-baseline justify-between">
        <Skeleton className="h-7 w-36" />
        <Skeleton className="h-5 w-14 border-2 border-foreground" />
      </div>
    </div>
  );
}
