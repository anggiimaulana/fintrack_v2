"use client";

import { Suspense, useState, useMemo, useCallback, useEffect } from "react";
import { useSearchParams, useRouter } from "next/navigation";
import { useQueryClient } from "@tanstack/react-query";
import { TrendingUp, TrendingDown, PiggyBank, Wallet } from "lucide-react";
import { StatCard, StatCardSkeleton } from "@/components/dashboard/StatCard";
import { ChartContainerSkeleton } from "@/components/charts/ChartContainer";
import { ChartContainer } from "@/components/charts/ChartContainer";
import { SpkWidgetSkeleton } from "@/components/dashboard/SpkWidget";
import { SpkWidget } from "@/components/dashboard/SpkWidget";
import { TransactionTable } from "@/components/transactions/TransactionTable";
import { useDashboardOverview, useDashboardCharts } from "@/hooks/useDashboard";
import { useTransactions } from "@/hooks/useTransactions";
import { useSpkLatest } from "@/hooks/useSpk";
import { DateRangeFilter, type DateRangeValue, type PresetKey, computePresetDates } from "@/components/shared/DateRangeFilter";
import { mapOverview } from "@/hooks/useDashboard";

function parseFilterFromParams(sp: URLSearchParams, now: Date): DateRangeValue {
  const preset = (sp.get("preset") ?? "this_month") as PresetKey;
  const startDate = sp.get("start") || "";
  const endDate = sp.get("end") || "";
  const compareStart = sp.get("cs") || "";
  const compareEnd = sp.get("ce") || "";
  const enableCompare = sp.get("cmp") === "1";

  if (startDate && endDate) {
    return { preset: "custom", startDate, endDate, compareStart, compareEnd, enableCompare };
  }
  const fallback = computePresetDates(preset, now);
  return { preset, startDate: fallback.start, endDate: fallback.end, compareStart, compareEnd, enableCompare };
}

function filterToParams(f: DateRangeValue): Record<string, string> {
  const p: Record<string, string> = {};
  if (f.preset !== "custom") p.preset = f.preset;
  if (f.preset === "custom" || f.startDate) {
    p.start = f.startDate;
    p.end = f.endDate;
  }
  if (f.enableCompare) {
    p.cmp = "1";
    if (f.compareStart) p.cs = f.compareStart;
    if (f.compareEnd) p.ce = f.compareEnd;
  }
  return p;
}

function UserDashboardContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const queryClient = useQueryClient();
  const now = useMemo(() => new Date(), []);
  const today = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}-${String(now.getDate()).padStart(2, "0")}`;

  const [dateFilter, setDateFilterState] = useState<DateRangeValue>(() => parseFilterFromParams(searchParams, now));

  const setDateFilter = useCallback((val: DateRangeValue) => {
    setDateFilterState(val);
    const params = filterToParams(val);
    const qs = new URLSearchParams(params).toString();
    router.replace(`/user?${qs}`, { scroll: false });
  }, [router]);

  const overviewParams = dateFilter.startDate && dateFilter.endDate
    ? {
        preset: dateFilter.preset !== "custom" ? dateFilter.preset : undefined,
        startDate: dateFilter.startDate,
        endDate: dateFilter.endDate,
        compareStart: dateFilter.enableCompare && dateFilter.compareStart ? dateFilter.compareStart : undefined,
        compareEnd: dateFilter.enableCompare && dateFilter.compareEnd ? dateFilter.compareEnd : undefined,
      }
    : undefined;

  const { data: overview, isLoading: overviewLoading } = useDashboardOverview(overviewParams);
  const [txFilterType, setTxFilterType] = useState<string>("all");
  const { data: txData, isLoading: txLoading } = useTransactions({ perPage: 5, type: txFilterType as any });
  const { data: spk, isLoading: spkLoading } = useSpkLatest();

  const chartDateRange = dateFilter.startDate && dateFilter.endDate
    ? { startDate: dateFilter.startDate, endDate: dateFilter.endDate }
    : undefined;

  // Prefetch common periods so switching is instant
  useEffect(() => {
    const common: PresetKey[] = ["this_month", "last_month", "3months", "this_year", "today"];
    const now2 = new Date();
    for (const preset of common) {
      const d = computePresetDates(preset, now2);
      if (d.start && d.end) {
        queryClient.prefetchQuery({
          queryKey: ["dashboard", "overview", "default", preset],
          queryFn: async () => {
            const mod = await import("@/lib/api");
            const res = await mod.default.get(`/dashboard/overview?start_date=${d.start}&end_date=${d.end}`);
            return mapOverview(res.data?.data);
          },
          staleTime: 120_000,
        });
      }
    }
  }, [queryClient]);

  return (
    <div className="flex flex-col gap-6 w-full max-w-7xl mx-auto">
      <div className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-4">
        <DateRangeFilter value={dateFilter} onChange={setDateFilter} />
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 lg:gap-6">
        {overviewLoading ? (
          <>
            <StatCardSkeleton />
            <StatCardSkeleton />
            <StatCardSkeleton />
            <StatCardSkeleton />
          </>
        ) : (
          <>
            <StatCard 
              title="Total Pemasukan" 
              amount={overview?.totalIncome || 0} 
              trend={overview?.incomeTrend} 
              icon={TrendingUp} 
              variant="success" 
            />
            <StatCard 
              title="Total Pengeluaran" 
              amount={overview?.totalExpense || 0} 
              trend={overview?.expenseTrend} 
              icon={TrendingDown} 
              variant="danger" 
            />
            <StatCard 
              title="Saldo Bersih" 
              amount={overview?.netBalance || 0} 
              icon={Wallet} 
              variant="primary" 
            />
            <StatCard 
              title="Tabungan" 
              amount={overview?.savings || 0} 
              trend={overview?.savingsTrend} 
              icon={PiggyBank} 
              variant="primary" 
            />
          </>
        )}
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-3 gap-6">
        <div className="xl:col-span-2">
          {overviewLoading ? <ChartContainerSkeleton /> : <ChartContainer dateRange={chartDateRange} />}
        </div>
        <div className="xl:col-span-1">
          {spkLoading ? (
            <SpkWidgetSkeleton />
          ) : (
            <SpkWidget 
              score={spk?.score ?? 0} 
              status={spk?.status || "Belum Ada Data"} 
              insights={spk?.insights && spk.insights.length > 0 ? spk.insights : ["Belum ada data transaksi. Mulai catat pemasukan dan pengeluaran Anda untuk melihat analisis keuangan."]} 
              hasData={!!spk && (spk?.score ?? 0) > 0}
            />
          )}
        </div>
      </div>

      <div className="mb-8">
        <div className="mb-3 flex border-2 border-foreground p-1 bg-background w-full md:w-auto">
          {[{ key: "all", label: "Semua" }, { key: "income", label: "Pemasukan" }, { key: "expense", label: "Pengeluaran" }, { key: "savings", label: "Tabungan" }].map((f) => (
            <button
              key={f.key}
              onClick={() => setTxFilterType(f.key)}
              className={`flex-1 md:flex-none px-4 py-1.5 text-xs font-bold cursor-pointer transition-colors ${
                txFilterType === f.key
                  ? "bg-card text-foreground shadow-[2px_2px_0px_0px] shadow-foreground border-x-2 border-foreground"
                  : "text-muted-foreground hover:text-foreground"
              }`}
            >
              {f.label}
            </button>
          ))}
        </div>
        <TransactionTable transactions={txData?.data || []} isLoading={txLoading} />
      </div>
    </div>
  );
}

export default function UserDashboardPage() {
  return (
    <Suspense fallback={<div className="h-40 animate-pulse bg-muted border-2 border-foreground" />}>
      <UserDashboardContent />
    </Suspense>
  );
}
