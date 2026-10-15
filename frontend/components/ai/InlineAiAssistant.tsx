"use client";

import { useState, useRef, useEffect, useCallback } from "react";
import type { TransactionData } from "@/components/transactions/TransactionRow";
import { Sparkles, Loader2, CheckCircle2, Send, X, ChevronDown, ChevronUp, Plus, Camera, MessageSquare, Square } from "lucide-react";
import { Button } from "@/components/ui/button";
import { ChatMessage } from "./ChatMessage";
import { useAiChat } from "@/hooks/useAiChat";
import { VoiceInput } from "./VoiceInput";
import { BarChartView } from "@/components/charts/BarChartView";
import { LineChartView } from "@/components/charts/LineChartView";
import { PieChartView } from "@/components/charts/PieChartView";
import { toast } from "sonner";
import { notifyEntitlementError } from "@/lib/entitlement-errors";
import api, { aiRequestTimeout } from "@/lib/api";
import { buildReceiptTransaction, formatRupiahIDR, normalizeReceiptType, receiptTypeLabel, type ReceiptTransactionPayload } from "@/lib/receipt";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useAdminCategories } from "@/hooks/useAdminCategories";
import { useCreateTransaction } from "@/hooks/useTransactions";
import { useInvalidateQueries } from "@/lib/cache-invalidation";

const toolLabels: Record<string, string> = {
  analyzing: "Menganalisis perintah...",
  create_transaction: "Mencatat transaksi...",
  get_spending_summary: "Mengambil ringkasan pengeluaran...",
  get_category_expenses: "Menganalisis pengeluaran per kategori...",
  get_transactions: "Mencari transaksi...",
  get_categories: "Memuat daftar kategori...",
  get_finance_insight: "Menganalisis data keuangan...",
  generate_chart: "Membuat visualisasi grafik...",
};

export function InlineAiAssistant() {
  const { sessions, sessionsLoading, createSession, deleteSession, streamChat, cancelStream, isStreaming, streamingContent, toolSteps, lastToolSteps, cooldownSeconds } = useAiChat("transactions");
  const queryClient = useQueryClient();
  const [isOpen, setIsOpen] = useState(false);
  const [messages, setMessages] = useState<Array<{ role: string; content: string }>>([]);
  const [inputText, setInputText] = useState("");
  const [sessionId, setSessionId] = useState<string | null>(null);
  const [hasClickedNew, setHasClickedNew] = useState(false);
  const [showHistory, setShowHistory] = useState(false);
  // Store chart data per assistant message index so they persist across new messages
  const [chartDataMap, setChartDataMap] = useState<Record<number, unknown[]>>({});
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const shouldAutoScroll = useRef(true);

  const { categories: incomeCategories } = useAdminCategories("income");
  const { categories: expenseCategories } = useAdminCategories("expense");
  const { categories: savingsCategories } = useAdminCategories("savings");
  const { invalidateTransactions } = useInvalidateQueries();
  const { mutateAsync: createTransaction } = useCreateTransaction();

  const { data: sessionMessages } = useQuery({
    queryKey: ["inline-chat-messages", sessionId],
    queryFn: async () => {
      if (!sessionId) return [];
      try { const res = await api.get(`/ai/sessions/${sessionId}/messages`); return res.data?.data || []; }
      catch { return []; }
    },
    enabled: !!sessionId && isOpen,
    placeholderData: (prev: unknown) => prev as never,
  });

  useEffect(() => {
    if (sessionMessages && !isStreaming) {
      if (sessionMessages.length >= messages.length && sessionMessages.length > 0) {
        // eslint-disable-next-line react-hooks/set-state-in-effect
        setMessages(sessionMessages.map((m: { role: "user" | "assistant", content: string }) => ({ role: m.role, content: m.content })));
      }
    }
  }, [sessionMessages, isStreaming, messages.length]);

  // Track last assistant index for chart persistence
  const lastAssistantIndex = messages.map((m, i) => m.role === "assistant" ? i : -1).filter(i => i >= 0).pop() ?? -1;

  // Persist tool steps (chart data) per assistant message index when streaming ends
  useEffect(() => {
    if (!isStreaming && lastToolSteps.length > 0 && lastAssistantIndex >= 0) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setChartDataMap((prev) => ({ ...prev, [lastAssistantIndex]: lastToolSteps }));
    }
  }, [isStreaming, lastToolSteps, lastAssistantIndex]);

  // Smart scroll: only auto-scroll if user is near the bottom
  const handleScroll = useCallback(() => {
    const container = scrollContainerRef.current;
    if (!container) return;
    const threshold = 80;
    const isNearBottom = container.scrollHeight - container.scrollTop - container.clientHeight < threshold;
    shouldAutoScroll.current = isNearBottom;
  }, []);

  // Only auto-scroll within the chat container — never scroll the page
  useEffect(() => {
    if (shouldAutoScroll.current && scrollContainerRef.current) {
      const container = scrollContainerRef.current;
      container.scrollTo({ top: container.scrollHeight, behavior: "smooth" });
    }
  }, [messages.length, streamingContent]);

  const handleCreateSession = useCallback(async () => {
    setHasClickedNew(true);
    setShowHistory(false);
    setChartDataMap({});
    const ns = await createSession.mutateAsync("Percakapan baru");
    if (ns) {
      setSessionId(ns.id);
      setMessages([]);
    }
  }, [createSession]);

  const handleSwitchSession = useCallback(async (id: string) => {
    setSessionId(id);
    setMessages([]);
    setChartDataMap({});
    setHasClickedNew(true);
    setShowHistory(false);
    try {
      const res = await api.get(`/ai/sessions/${id}/messages`);
      const msgs = res.data?.data || [];
      setMessages(msgs.map((m: { role: "user" | "assistant", content: string }) => ({ role: m.role, content: m.content })));
    } catch {}
  }, []);

  const handleDeleteSession = useCallback(async (id: string) => {
    await deleteSession.mutateAsync(id);
    if (sessionId === id) {
      setSessionId(null);
      setMessages([]);
      setChartDataMap({});
      setHasClickedNew(false);
    }
  }, [deleteSession, sessionId]);

  const handleSend = useCallback(async (text: string) => {
    if (!text.trim() || isStreaming) return;
    const sid = sessionId;
    if (!sid) return;
    setMessages((prev) => [...prev, { role: "user", content: text }]);
    setInputText("");
    // Don't clear chartDataMap — previous charts should persist
    shouldAutoScroll.current = true;
    try {
      const result = await streamChat(text, sid, () => {}, "fast");
      // Add the assistant response to messages after stream completes
      if (result) {
        setMessages((prev) => {
          const next = [...prev];
          const lastIdx = next.length - 1;
          const patch = { content: result.content, mode: (result.mode as "fast" | "reasoning" | undefined) ?? "fast", modelUsed: result.model ?? null };
          if (lastIdx >= 0 && next[lastIdx].role === "assistant") {
            next[lastIdx] = { ...next[lastIdx], ...patch };
          } else {
            next.push({ role: "assistant", ...patch });
          }
          return next;
        });
      }
      // Refresh messages from server to get the persisted version
      queryClient.invalidateQueries({ queryKey: ["inline-chat-messages", sid] });
    } catch (err) {
      // A silent catch left the user with no feedback at all. Plan/quota gates
      // get the upgrade toast; anything else gets a short retry message.
      const code = err instanceof Error ? err.message : String(err);
      if (!notifyEntitlementError({ response: { status: 402, data: { error: code } } })) {
        toast.error("Maaf, terjadi kesalahan. Silakan coba lagi.");
      }
    }
  }, [sessionId, streamChat, isStreaming, queryClient]);

  const handleVoiceTranscript = useCallback((text: string) => {
    handleSend(text);
  }, [handleSend]);

  // Receipt analysis → auto-save to DB (same as ReceiptScanner)
  const analyzePhotoMutation = useMutation({
    mutationFn: async (file: File) => {
      const formData = new FormData();
      formData.append("image", file);
      const res = await api.post("/ai/receipt", formData, { timeout: aiRequestTimeout.ms });
      return res.data?.data;
    },
    onSuccess: async (data) => {
      if (!data?.success) {
        toast.error(data?.message || "Gagal menganalisis foto.");
        return;
      }

      const formatted = formatRupiahIDR(data.total || 0);

      // Try to auto-save to DB using the shared matcher (same as ReceiptScanner).
      let payload: ReceiptTransactionPayload;
      try {
        ({ payload } = buildReceiptTransaction(data, {
          income: incomeCategories,
          expense: expenseCategories,
          savings: savingsCategories,
        }));
      } catch (err: unknown) {
        // Can't build a valid transaction (no amount / no matching category):
        // show the analysis so the user can record it manually.
        const typeLabel = receiptTypeLabel(normalizeReceiptType(data.type));
        const msg = `📋 Hasil scan foto\n\n**Deskripsi:** ${data.description || data.merchant || "-"}\n**Tanggal:** ${data.date || "-"}\n**Kategori:** ${data.category || "-"} (${typeLabel})\n**Total:** ${formatted}\n\n*${(err as Error).message}*`;
        handleSend(msg);
        return;
      }

      try {
        await createTransaction(payload as unknown as Parameters<typeof createTransaction>[0]);
        invalidateTransactions();
        toast.success("Struk berhasil dicatat sebagai transaksi!");

        const typeLabel = receiptTypeLabel(normalizeReceiptType(data.type));
        const msg = `✅ Struk berhasil dicatat sebagai transaksi **${typeLabel}** sebesar ${formatted}\n\n**Deskripsi:** ${data.description || data.merchant || "-"}\n**Kategori:** ${data.category || "-"}\n**Tanggal:** ${data.date || "-"}`;
        handleSend(msg);
      } catch (err: unknown) {
        toast.error("Gagal menyimpan: " + ((err as Error).message || "Error"));
        const msg = `⚠️ Gagal menyimpan struk: ${(err as Error).message}\n\nHasil analisis: **${data.description || data.merchant || "-"}** - ${formatted}`;
        handleSend(msg);
      }
    },
    onError: (err: unknown) => {
      const error = err as { response?: { data?: { message?: string } }, message?: string };
      toast.error(error?.response?.data?.message || error?.message || "Gagal memproses foto.");
    },
  });

  const handlePhotoSelect = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    if (file.size > 10 * 1024 * 1024) {
      toast.error("Ukuran file maksimal 10MB.");
      return;
    }
    analyzePhotoMutation.mutate(file);
    if (fileInputRef.current) fileInputRef.current.value = "";
  }, [analyzePhotoMutation]);

  const renderChartResult = (result: Record<string, unknown>): React.ReactNode => {
    const labels: string[] = (result.labels as string[]) || [];
    const income: number[] = (result.income as number[]) || [];
    const expense: number[] = (result.expense as number[]) || [];
    const savings: number[] = (result.savings as number[]) || [];
    const chartType: string = (result.chart_type as string) || "comparison";
    if (labels.length === 0) return null;

    const chartData = labels.map((name: string, i: number) => ({
      name,
      income: income[i] || 0,
      expense: expense[i] || 0,
      savings: savings[i] || 0,
    }));

    const totalIncome = income.reduce((a: number, b: number) => a + b, 0);
    const totalExpense = expense.reduce((a: number, b: number) => a + b, 0);
    const totalSavings = savings.reduce((a: number, b: number) => a + b, 0);
    const pieData = totalIncome > 0 || totalExpense > 0 || totalSavings > 0
      ? [{ name: "Pemasukan", value: totalIncome }, { name: "Pengeluaran", value: totalExpense }, { name: "Tabungan", value: totalSavings }]
      : undefined;

    // Default: pie chart — specific types shown when chart_type is specified
    return (
      <div className="space-y-2 mt-1">
        {pieData && (
          <div className="h-36 border-2 rounded-md border-foreground p-2 bg-card">
            <p className="text-[10px] font-black uppercase tracking-wide text-muted-foreground mb-1">Distribusi Keuangan</p>
            <div className="h-[calc(100%-40px)]">
              <PieChartView data={pieData} compact />
            </div>
            <div className="flex justify-around text-[10px] font-bold">
              <span className="text-green-700">Pemasukan: {new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(totalIncome)}</span>
              <span className="text-red-700">Pengeluaran: {new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(totalExpense)}</span>
              <span className="text-blue-700">Tabungan: {new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(totalSavings)}</span>
            </div>
          </div>
        )}
        {chartType !== "pie" && labels.length > 1 && (
          <div className="grid grid-cols-2 gap-2">
            <div className="h-32 border-2 rounded-md border-foreground p-2 bg-card">
              <p className="text-[10px] font-black uppercase tracking-wide text-muted-foreground mb-1">Grafik Batang</p>
              <div className="h-[calc(100%-20px)]">
                <BarChartView data={chartData} />
              </div>
            </div>
            <div className="h-32 border-2 rounded-md border-foreground p-2 bg-card">
              <p className="text-[10px] font-black uppercase tracking-wide text-muted-foreground mb-1">Grafik Garis</p>
              <div className="h-[calc(100%-20px)]">
                <LineChartView data={chartData} />
              </div>
            </div>
          </div>
        )}
      </div>
    );
  };

  const renderSessionHistory = () => {
    if (sessionsLoading) return <p className="text-xs text-muted-foreground py-2">Memuat sesi...</p>;
    if (!sessions || sessions.length === 0) return null;
    return (
      <div className="mt-2 space-y-1 text-left max-h-32 overflow-y-auto">
        {sessions.slice(0, 10).map((s: { id: string; title: string }) => (
          <div key={s.id} className="flex items-center gap-1 group">
            <button
              onClick={() => handleSwitchSession(s.id)}
              className="flex-1 text-left px-2 py-1.5 text-xs font-bold text-muted-foreground hover:text-foreground hover:bg-card border-2 rounded-md border-transparent hover:border-foreground truncate cursor-pointer"
            >
              <MessageSquare className="w-3 h-3 inline-block mr-1.5 shrink-0" />
              {s.title || "Percakapan"}
            </button>
            <button
              onClick={() => handleDeleteSession(s.id)}
              className="opacity-0 group-hover:opacity-100 p-1 text-muted-foreground hover:text-red-500 cursor-pointer"
            >
              <X className="w-3 h-3" />
            </button>
          </div>
        ))}
      </div>
    );
  };

  // Determine which tool steps to display: streaming ones take priority
  const activeToolSteps = isStreaming ? toolSteps : [];

  const renderToolSteps = () => {
    if (activeToolSteps.length === 0) return null;
    const chartStep = activeToolSteps.find((s: { tool: string; status: string; result?: unknown }) => s.tool === "generate_chart" && s.status === "success" && s.result);
    return (
      <div className="border-2 rounded-md border-foreground bg-muted p-2 space-y-1 mb-1">
        <p className="text-[10px] font-black uppercase tracking-wide text-muted-foreground">AI Agent</p>
        {activeToolSteps.map((step: { status: string; tool: string }, i: number) => (
          <div key={i} className="flex items-center gap-1.5 text-[11px] font-bold">
            {step.status === "running" ? (
              <Loader2 className="w-3 h-3 text-blue-500 animate-spin" />
            ) : step.status === "success" ? (
              <CheckCircle2 className="w-3 h-3 text-green-600" />
            ) : (
              <span className="w-3 h-3 text-red-500 text-center leading-none">!</span>
            )}
            <span className={step.status === "running" ? "text-blue-500" : step.status === "success" ? "text-green-700" : "text-red-500"}>
              {toolLabels[step.tool] || step.tool}
              {step.status === "success" && " ✓"}
            </span>
          </div>
        ))}
        {chartStep && chartStep.result ? renderChartResult(chartStep.result as Record<string, unknown>) : null}
      </div>
    );
  };

  const renderMessages = () => {
    return messages.map((msg, i) => {
      const msgChartData = chartDataMap[i];
      const hasChart = msgChartData && msgChartData.length > 0 && msg.role === "assistant";
      return (
        <div key={i}>
          <ChatMessage author={msg.role} content={msg.content} />
          {hasChart && !isStreaming && renderPersistedCharts(msgChartData)}
        </div>
      );
    });
  };

  const renderPersistedCharts = (steps: unknown[]) => {
    const chartStep = (steps as { tool: string; status: string; result?: unknown }[]).find(s => s.tool === "generate_chart" && s.status === "success" && s.result);
    if (!chartStep) return null;
    return (
      <div className="border-2 rounded-md border-foreground bg-muted p-2 space-y-1 mb-1 mt-1">
        <p className="text-[10px] font-black uppercase tracking-wide text-muted-foreground">Visualisasi</p>
        {chartStep.result ? renderChartResult(chartStep.result as Record<string, unknown>) : null}
      </div>
    );
  };

  return (
    <div className="border-2 rounded-md border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground">
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="w-full flex items-center justify-between px-4 py-3 cursor-pointer hover:bg-muted/50 transition-colors"
      >
        <div className="flex items-center gap-2">
          <Sparkles className="w-4 h-4 text-primary" />
          <span className="text-xs font-black uppercase tracking-wide text-foreground">AI Assistant</span>
          {isStreaming && (
            <span className="text-[10px] font-bold text-blue-500 animate-pulse">Sedang mengetik...</span>
          )}
        </div>
        {isOpen ? <ChevronUp className="w-4 h-4 text-muted-foreground" /> : <ChevronDown className="w-4 h-4 text-muted-foreground" />}
      </button>

      {isOpen && (
        <div className="border-t-2 border-foreground">
          <div ref={scrollContainerRef} onScroll={handleScroll} className="max-h-80 overflow-y-auto p-3 space-y-2 bg-background">
            {!sessionId && !hasClickedNew && (
              <div className="text-center py-6">
                <div className="w-12 h-12 border-2 rounded-md border-foreground bg-primary/10 flex items-center justify-center mx-auto mb-3">
                  <MessageSquare className="w-5 h-5 text-primary" />
                </div>
                <p className="text-sm font-black uppercase tracking-wide text-foreground mb-1">Belum Ada Sesi</p>
                <p className="text-xs font-bold text-muted-foreground mb-4">
                  Buat sesi baru untuk bertanya atau scan struk pakai AI.
                </p>
                <Button
                  size="sm"
                  onClick={handleCreateSession}
                  disabled={createSession.isPending}
                  className="shadow-[2px_2px_0px_0px] shadow-foreground"
                >
                  {createSession.isPending ? "Membuat..." : "Buat Sesi Baru"}
                </Button>
                {sessions && sessions.length > 0 && (
                  <div className="mt-3">
                    <button
                      onClick={() => setShowHistory(!showHistory)}
                      className="text-xs font-bold text-muted-foreground hover:text-foreground underline underline-offset-2 cursor-pointer"
                    >
                      {showHistory ? "Tutup" : `Lihat ${sessions.length} sesi sebelumnya`}
                    </button>
                  </div>
                )}
                {showHistory && renderSessionHistory()}
              </div>
            )}

            {sessionId && !isStreaming && (
              <div className="flex items-center justify-between border-b-2 border-foreground pb-1 mb-2">
                <span className="text-[10px] font-black uppercase tracking-wide text-muted-foreground truncate">
                  {sessions?.find((s: { id: string }) => s.id === sessionId)?.title || "Percakapan"}
                </span>
                <div className="flex items-center gap-1 shrink-0 ml-2">
                  <button
                    onClick={() => {
                      setShowHistory(false);
                      setSessionId(null);
                      setMessages([]);
                      setChartDataMap({});
                      setHasClickedNew(false);
                    }}
                    className="text-[10px] font-bold text-muted-foreground hover:text-foreground cursor-pointer"
                    title="Sesi Baru"
                  >
                    <Plus className="w-3 h-3" />
                  </button>
                  <button
                    onClick={() => setShowHistory(!showHistory)}
                    className="text-[10px] font-bold text-muted-foreground hover:text-foreground cursor-pointer"
                  >
                    {showHistory ? "Tutup" : "Sesi"}
                  </button>
                </div>
              </div>
            )}
            {showHistory && sessionId && renderSessionHistory()}

            {sessionId && messages.length === 0 && !isStreaming && !analyzePhotoMutation.isPending && (
              <p className="text-xs font-bold text-muted-foreground text-center py-4">
                Tanyakan tentang transaksi atau minta dibuatkan visualisasi.
              </p>
            )}

            {analyzePhotoMutation.isPending && (
              <div className="flex items-center gap-2 text-xs font-bold text-muted-foreground py-2">
                <Loader2 className="w-3 h-3 animate-spin" /> Menganalisis foto...
              </div>
            )}

            {renderMessages()}
            
            {/* Show tool steps only during active streaming */}
            {isStreaming && activeToolSteps.length > 0 && renderToolSteps()}
            
            {isStreaming && streamingContent && (
              <ChatMessage author="assistant" content={streamingContent} />
            )}
            {isStreaming && !streamingContent && toolSteps.length === 0 && (
              <div className="flex items-center gap-2 text-xs font-bold text-muted-foreground">
                <Loader2 className="w-3 h-3 animate-spin" /> Memproses...
              </div>
            )}
            <div ref={messagesEndRef} />
          </div>

          {sessionId && (
            <div className="border-t-2 border-foreground p-2">
              {cooldownSeconds > 0 ? (
                <div className="flex items-center gap-2 px-3 py-2 bg-amber-50 border-2 rounded-md border-amber-500 text-xs font-bold text-amber-700">
                  <Loader2 className="w-3.5 h-3.5 animate-spin shrink-0" />
                  Cooldown {cooldownSeconds} detik. Tunggu sebelum mengirim pesan lagi.
                </div>
              ) : (
                <div className="flex items-center gap-2">
                  <input
                    type="text"
                    value={inputText}
                    onChange={(e) => setInputText(e.target.value)}
                    onKeyDown={(e) => { if (e.key === "Enter") handleSend(inputText); }}
                    placeholder="Tanya AI Assistant..."
                    disabled={isStreaming}
                    className="flex-1 border-2 rounded-md border-foreground bg-background px-3 py-1.5 text-sm font-bold focus:outline-none focus:shadow-[2px_2px_0px_0px] focus:shadow-foreground placeholder:text-muted-foreground/70"
                  />
                  <VoiceInput onTranscript={handleVoiceTranscript} disabled={isStreaming} />
                  <button
                    onClick={() => fileInputRef.current?.click()}
                    disabled={isStreaming || analyzePhotoMutation.isPending}
                    className="p-1.5 border-2 rounded-md border-foreground bg-muted text-muted-foreground hover:text-foreground hover:bg-card disabled:opacity-30 cursor-pointer disabled:cursor-not-allowed"
                    title="Scan struk dari foto"
                  >
                    <Camera className="w-3.5 h-3.5" />
                  </button>
                  <input
                    type="file"
                    accept="image/jpeg,image/png,image/webp,image/heic"
                    className="hidden"
                    ref={fileInputRef}
                    onChange={handlePhotoSelect}
                  />
                  {isStreaming ? (
                    <button
                      type="button"
                      onClick={cancelStream}
                      className="p-1.5 border-2 rounded-md border-foreground bg-red-600 text-white hover:bg-red-700 shadow-[2px_2px_0px_0px] shadow-foreground active:translate-x-[1px] active:translate-y-[1px] cursor-pointer"
                      title="Hentikan respons"
                      aria-label="Hentikan respons"
                    >
                      <Square className="w-3.5 h-3.5 fill-current" />
                    </button>
                  ) : (
                    <button
                      type="button"
                      onClick={() => handleSend(inputText)}
                      disabled={!inputText.trim() || isStreaming}
                      className="p-1.5 border-2 rounded-md border-foreground bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-30 cursor-pointer disabled:cursor-not-allowed"
                      title="Kirim pesan"
                      aria-label="Kirim pesan"
                    >
                      <Send className="w-3.5 h-3.5" />
                    </button>
                  )}
                </div>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
