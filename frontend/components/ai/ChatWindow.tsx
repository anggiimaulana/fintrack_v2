"use client";

import { useState, useRef, useEffect, useCallback } from "react";
import { useAiChat } from "@/hooks/useAiChat";
import { ChatMessage } from "@/components/ai/ChatMessage";
import { ChatInput } from "@/components/ai/ChatInput";
import { TypingIndicator } from "@/components/ai/TypingIndicator";
import { Button } from "@/components/ui/button";
import { toast } from "sonner";
import { Trash2, Plus, MessageSquare, Loader2, CheckCircle2, Search, PenLine, BarChart3, X, Zap, Brain, PanelLeftClose, PanelLeft } from "lucide-react";
import { ConfirmDialog } from "@/components/shared/ConfirmDialog";
import { useQuery, useQueryClient, useMutation } from "@tanstack/react-query";
import api from "@/lib/api";
import { useAdminCategories } from "@/hooks/useAdminCategories";

import { BarChartView } from "@/components/charts/BarChartView";
import { LineChartView } from "@/components/charts/LineChartView";
import { PieChartView } from "@/components/charts/PieChartView";
import { useMediaQuery } from "@/hooks/useMediaQuery";

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

function updateUrlSession(sessionId: string | null) {
  if (typeof window === "undefined") return;
  const url = new URL(window.location.href);
  if (sessionId) {
    url.searchParams.set("session", sessionId);
  } else {
    url.searchParams.delete("session");
  }
  window.history.replaceState({}, "", url.toString());
}

export default function ChatWindow() {
  const { sessions, sessionsLoading, createSession, deleteSession, updateSession, streamChat, cancelStream, isStreaming, streamingContent, toolSteps, lastToolSteps, cooldownSeconds } = useAiChat("chatbot");
  const queryClient = useQueryClient();

  const [messages, setMessages] = useState<Array<{ role: string; content: string }>>([]);
  const [currentSessionId, setCurrentSessionId] = useState<string | null>(null);
  const [activeSessionId, setActiveSessionId] = useState<string | null>(null);
  const [editingSessionId, setEditingSessionId] = useState<string | null>(null);
  const [editTitle, setEditTitle] = useState("");
  const [aiMode, setAiMode] = useState<"fast" | "reasoning">("fast");
  const { categories: incomeCategories } = useAdminCategories("income");
  const { categories: expenseCategories } = useAdminCategories("expense");
  const isMobile = useMediaQuery("(max-width: 767px)");
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const [pendingAutoSend, setPendingAutoSend] = useState<string | null>(null);
  const [confirmDeleteSessionId, setConfirmDeleteSessionId] = useState<string | null>(null);
  // Store chart data per assistant message index so they persist across new messages
  const [chartDataMap, setChartDataMap] = useState<Record<number, any[]>>({});
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const shouldAutoScroll = useRef(true);

  const { data: sessionMessages, isLoading: sessionMessagesLoading } = useQuery({
    queryKey: ["chat-messages", currentSessionId],
    queryFn: async () => {
      if (!currentSessionId) return [];
      try { const res = await api.get(`/ai/sessions/${currentSessionId}/messages`); return res.data?.data || []; }
      catch { return []; }
    },
    enabled: !!currentSessionId,
    placeholderData: (prev: any) => prev,
  });

  const createFromReceiptMutation = useMutation({
    mutationFn: async (result: any) => {
      const detectedType = result.type === "income" ? "income" : "expense";
      const catList = detectedType === "income" ? (incomeCategories || []) : (expenseCategories || []);
      let matched = catList.find((c: any) =>
        c.name?.toLowerCase() === (result.category || "").toLowerCase()
      );
      if (!matched) {
        matched = catList.find((c: any) =>
          c.name?.toLowerCase().includes((result.category || "").toLowerCase()) ||
          (result.category || "").toLowerCase().includes(c.name?.toLowerCase())
        );
      }
      if (!matched) {
        const fallbackNames = detectedType === "expense" ? ["lain-lain", "lainnya"] : ["lainnya"];
        matched = catList.find((c: any) => fallbackNames.some(f => c.name?.toLowerCase() === f));
      }
      if (!matched) throw new Error("Kategori tidak ditemukan: " + (result.category || "Unknown"));
      const payload = {
        type: detectedType,
        amount: result.total,
        expense_category_id: detectedType === "expense" ? matched.id : undefined,
        income_category_id: detectedType === "income" ? matched.id : undefined,
        description: result.merchant || "",
        transaction_date: result.date || new Date().toISOString().split("T")[0],
        source: "receipt",
      };
      const res = await api.post("/ai/receipt/transaction", payload);
      return res.data?.data;
    },
    onSuccess: (tx, result) => {
      const formatted = new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(result.total || 0);
      const msg = `\u2705 Struk berhasil dicatat sebagai transaksi **${result.type === "income" ? "Pemasukan" : "Pengeluaran"}** sebesar ${formatted}\n\n**Merchant:** ${result.merchant || "-"}\n**Kategori:** ${result.category || "-"}\n**Tanggal:** ${result.date || "-"}\n**ID Transaksi:** ${tx?.id || "-"}`;
      handleSend(msg);
    },
    onError: (err: any, result) => {
      const formatted = new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(result.total || 0);
      handleSend(`\u26A0\uFE0F Gagal menyimpan struk: ${err.message}\n\nHasil analisis: ${result.merchant || "-"} - ${formatted}`);
    },
  });

  const handleReceiptResult = (result: any) => {
    if (result?.success && result?.total) {
      if (result.category) {
        createFromReceiptMutation.mutate(result);
      } else {
        const formatted = new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(result.total || 0);
        handleSend(`\uD83D\uDCCB Hasil scan struk: **${result.merchant || "-"}** - ${formatted}\n\n*Tidak dapat menentukan kategori otomatis. Silakan catat manual di menu Transaksi.*`);
      }
    }
  };

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const session = params.get("session");
    if (session) setCurrentSessionId(session);
    const q = params.get("q");
    if (q) {
      const url = new URL(window.location.href);
      url.searchParams.delete("q");
      window.history.replaceState({}, "", url.toString());
      setPendingAutoSend(q);
    }
  }, []);

  useEffect(() => {
    if (sessionMessages && !isStreaming && activeSessionId !== currentSessionId) {
      setMessages(sessionMessages.map((m: any) => ({ role: m.role, content: m.content })));
      setActiveSessionId(currentSessionId);
    }
  }, [sessionMessages, isStreaming, currentSessionId, activeSessionId]);

  useEffect(() => {
    const handlePopState = () => {
      const params = new URLSearchParams(window.location.search);
      const session = params.get("session");
      if (session !== currentSessionId) {
        setCurrentSessionId(session);
        setMessages([]);
      }
    };
    window.addEventListener("popstate", handlePopState);
    return () => window.removeEventListener("popstate", handlePopState);
  }, [currentSessionId]);

  const lastAssistantIndex = messages.map((m, i) => m.role === "assistant" ? i : -1).filter(i => i >= 0).pop() ?? -1;

  // Persist tool steps (chart data) per assistant message index when streaming ends
  useEffect(() => {
    if (!isStreaming && lastToolSteps.length > 0 && lastAssistantIndex >= 0) {
      setChartDataMap((prev) => ({ ...prev, [lastAssistantIndex]: lastToolSteps }));
    }
  }, [isStreaming, lastToolSteps, lastAssistantIndex]);

  // Smart scroll: only auto-scroll if user is near the bottom
  const handleScroll = useCallback(() => {
    const container = scrollContainerRef.current;
    if (!container) return;
    const threshold = 100;
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

  useEffect(() => {
    if (!pendingAutoSend || isStreaming) return;
    const msg = pendingAutoSend;
    setPendingAutoSend(null);
    createSession.mutateAsync(msg.substring(0, 50)).then((newSession) => {
      if (newSession) {
        setCurrentSessionId(newSession.id);
        setActiveSessionId(newSession.id);
        updateUrlSession(newSession.id);
        setTimeout(() => handleSend(msg, newSession.id), 300);
      }
    });
  }, [pendingAutoSend]);

  const handleSend = async (message: string, overrideSessionId?: string) => {
    let sessionId: string | null = overrideSessionId || currentSessionId;

    if (!sessionId && sessions && sessions.length > 0) {
      sessionId = sessions[0].id;
      setCurrentSessionId(sessionId);
      updateUrlSession(sessionId);
    }

    if (!sessionId) {
      const newSession = await createSession.mutateAsync(message.substring(0, 50));
      if (newSession) {
        sessionId = newSession.id;
        setCurrentSessionId(sessionId);
        updateUrlSession(sessionId);
      }
    }

    const userMsg = { role: "user" as const, content: message };
    setMessages((prev) => [...prev, userMsg]);
    // Don't clear chartDataMap — previous charts should persist
    shouldAutoScroll.current = true;

    if (!sessionId) return;

    try {
      const resultContent = await streamChat(
        message,
        sessionId,
        (msg) => {
          if (msg.role === "assistant" && !msg.content) {
            setMessages((prev) => [...prev, { role: "assistant", content: "" }]);
          }
        },
        aiMode,
      );

      if (resultContent) {
        setMessages((prev) => {
          const next = [...prev];
          const lastIdx = next.length - 1;
          if (lastIdx >= 0 && next[lastIdx].role === "assistant") {
            next[lastIdx] = { ...next[lastIdx], content: resultContent };
          } else {
            next.push({ role: "assistant", content: resultContent });
          }
          return next;
        });
      }
      // Refresh from server
      queryClient.invalidateQueries({ queryKey: ["chat-messages", sessionId] });
    } catch {
      setMessages((prev) => [...prev, { role: "assistant", content: "Maaf, terjadi kesalahan. Silakan coba lagi." }]);
    }
  };

  const switchSession = async (id: string) => {
    if (isStreaming) cancelStream();
    setMessages([]);
    setActiveSessionId(id);
    setCurrentSessionId(id);
    updateUrlSession(id);
    try {
      const res = await api.get(`/ai/sessions/${id}/messages`);
      const msgs = res.data?.data || [];
      setMessages(msgs.map((m: any) => ({ role: m.role, content: m.content })));
    } catch {
      setMessages([]);
    }
  };

  const startEdit = (id: string, currentTitle: string) => {
    setEditingSessionId(id);
    setEditTitle(currentTitle);
  };

  const saveEdit = async (id: string) => {
    if (editTitle.trim()) {
      await updateSession.mutateAsync({ id, title: editTitle.trim() });
      toast.success("Nama sesi berhasil diubah.");
    }
    setEditingSessionId(null);
  };

  const handleNewSession = async () => {
    if (isStreaming) cancelStream();
    const newSession = await createSession.mutateAsync("Percakapan baru");
    if (newSession) {
      setMessages([]);
      setCurrentSessionId(newSession.id);
      setActiveSessionId(newSession.id);
      updateUrlSession(newSession.id);
    }
  };

  // Use streaming tool steps during stream, persisted ones (from chartDataMap) after
  const activeToolSteps = isStreaming ? toolSteps : (chartDataMap[lastAssistantIndex] || []);

  const renderToolSteps = () => {
    if (activeToolSteps.length === 0) return null;

    const chartStep = activeToolSteps.find((s: any) => s.tool === "generate_chart" && s.status === "success" && s.result);

    return (
      <div className="border-2 border-foreground bg-muted p-3 space-y-2 mb-2">
        <p className="text-[10px] font-black uppercase tracking-wide text-muted-foreground">AI Agent</p>
        {activeToolSteps.map((step: any, i: number) => (
          <div key={`tool-${i}`} className="flex items-center gap-2 text-xs font-bold">
            {step.status === "running" ? (
              <Loader2 className="w-3.5 h-3.5 text-blue-500 animate-spin" />
            ) : step.status === "success" ? (
              <CheckCircle2 className="w-3.5 h-3.5 text-green-600" />
            ) : (
              <span className="w-3.5 h-3.5 text-red-500 text-center leading-none">!</span>
            )}
            <span className={
              step.status === "running" ? "text-blue-500" :
              step.status === "success" ? "text-green-700" : "text-red-500"
            }>
              {toolLabels[step.tool] || `Menjalankan ${step.tool}...`}
              {step.status === "success" && " \u2713"}
              {step.status === "error" && ` Gagal: ${step.error || ""}`}
            </span>
          </div>
        ))}
        {chartStep && chartStep.result && renderChartResult(chartStep.result)}
      </div>
    );
  };

  const renderChartResult = (result: any) => {
    const labels: string[] = result.labels || [];
    const income: number[] = result.income || [];
    const expense: number[] = result.expense || [];
    const savings: number[] = result.savings || [];
    const chartType: string = result.chart_type || "comparison";

    if (labels.length === 0) return null;

    const chartData = labels.map((name, i) => ({
      name,
      income: income[i] || 0,
      expense: expense[i] || 0,
      savings: savings[i] || 0,
    }));

    const totalIncome = income.reduce((a: number, b: number) => a + b, 0);
    const totalExpense = expense.reduce((a: number, b: number) => a + b, 0);
    const totalSavings = savings.reduce((a: number, b: number) => a + b, 0);
    const pieData = totalIncome > 0 || totalExpense > 0 || totalSavings > 0
      ? [
          { name: "Pemasukan", value: totalIncome },
          { name: "Pengeluaran", value: totalExpense },
          { name: "Tabungan", value: totalSavings },
        ]
      : undefined;

    // Default: show pie chart (distribusi) — specific types shown when chart_type is specified
    return (
      <div className="space-y-3 mt-2">
        {pieData && (
          <div className="h-44 border-2 border-foreground p-2 bg-card">
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
            <div className="h-36 border-2 border-foreground p-2 bg-card">
              <p className="text-[10px] font-black uppercase tracking-wide text-muted-foreground mb-1">Grafik Batang</p>
              <div className="h-[calc(100%-20px)]">
                <BarChartView data={chartData} />
              </div>
            </div>
            <div className="h-36 border-2 border-foreground p-2 bg-card">
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

  const renderMessages = () => {
    return messages.map((msg, index) => {
      // Show persisted chart data for each assistant message that had charts
      const msgChartData = chartDataMap[index];
      const hasChart = msgChartData && msgChartData.length > 0 && msg.role === "assistant";

      // For the currently streaming assistant message, show streaming content
      const isCurrentStreaming = index === lastAssistantIndex && isStreaming && streamingContent;
      const displayContent = isCurrentStreaming ? streamingContent : msg.content;

      return (
        <div key={index}>
          <ChatMessage role={msg.role} content={displayContent} />
          {hasChart && !isStreaming && renderPersistedToolSteps(msgChartData)}
        </div>
      );
    });
  };

  const renderPersistedToolSteps = (steps: any[]) => {
    const chartStep = steps.find((s: any) => s.tool === "generate_chart" && s.status === "success" && s.result);
    if (!chartStep) return null;
    return (
      <div className="border-2 border-foreground bg-muted p-3 space-y-2 mb-2 mt-1">
        <p className="text-[10px] font-black uppercase tracking-wide text-muted-foreground">Visualisasi</p>
        {chartStep.result && renderChartResult(chartStep.result)}
      </div>
    );
  };

  return (
    <div className="flex h-[calc(100vh-120px)]">
      {isMobile && sidebarOpen && (
        <div className="fixed inset-0 z-50 flex">
          <div className="absolute inset-0 bg-black/30" onClick={() => setSidebarOpen(false)} />
          <div className="relative w-64 bg-muted border-r-2 border-foreground flex flex-col overflow-hidden animate-in slide-in-from-left duration-200">
            <div className="p-3 border-b-2 border-foreground">
              <Button
                size="sm"
                className="w-full text-xs"
                onClick={handleNewSession}
                disabled={createSession.isPending}
              >
                <Plus className="w-3.5 h-3.5 mr-2" /> Sesi Baru
              </Button>
            </div>
            <div className="flex-1 overflow-y-auto p-2 space-y-1">
              {(sessions || []).map((s: any) => (
                <div key={s.id} className="flex items-center group">
                  {editingSessionId === s.id ? (
                    <input
                      autoFocus
                      value={editTitle}
                      onChange={(e) => setEditTitle(e.target.value)}
                      onBlur={() => saveEdit(s.id)}
                      onKeyDown={(e) => { if (e.key === "Enter") saveEdit(s.id); if (e.key === "Escape") setEditingSessionId(null); }}
                      className="flex-1 px-3 py-2 text-xs font-bold border-2 border-foreground bg-background outline-none"
                    />
                  ) : (
                    <div className="flex items-center gap-1 flex-1 min-w-0">
                      <button
                        onClick={() => { switchSession(s.id); if (isMobile) setSidebarOpen(false); }}
                        className={`flex-1 text-left px-3 py-2 text-xs font-bold transition-all cursor-pointer flex items-center gap-2 ${
                          currentSessionId === s.id
                            ? "bg-primary text-primary-foreground border-2 border-foreground shadow-[2px_2px_0px_0px] shadow-foreground"
                            : "text-muted-foreground hover:text-foreground hover:bg-card border-2 border-transparent"
                        }`}
                      >
                        <MessageSquare className="w-3.5 h-3.5 shrink-0" />
                        <div className="flex-1 min-w-0">
                          <span className="block truncate">{s.title || "Percakapan"}</span>
                          <span className="block text-[10px] text-muted-foreground/60 font-normal">{new Date(s.created_at).toLocaleDateString("id-ID", { month: "short", day: "numeric" })}</span>
                        </div>
                      </button>
                      <button
                        onClick={() => startEdit(s.id, s.title || "")}
                        className="p-1.5 text-muted-foreground hover:text-primary transition-all shrink-0"
                        title="Edit nama"
                      >
                        <PenLine className="w-3 h-3" />
                      </button>
                    </div>
                  )}
                  <button
                    onClick={() => setConfirmDeleteSessionId(s.id)}
                    className="opacity-0 group-hover:opacity-100 p-1 text-muted-foreground hover:text-red-500 transition-all"
                  >
                    <X className="w-3 h-3" />
                  </button>
                </div>
              ))}
              {(!sessions || sessions.length === 0) && !sessionsLoading && (
                <p className="text-xs text-muted-foreground text-center py-4 font-bold">Belum ada sesi</p>
              )}
              {sessionsLoading && (
                <div className="space-y-2 p-2">
                  {[1,2,3].map((i) => (
                    <div key={i} className="h-8 bg-muted-foreground/10 border-2 border-foreground animate-pulse" />
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>
      )}
      {!isMobile && (
        <div className={`${sidebarOpen ? "w-60" : "w-0"} border-r-2 border-foreground bg-muted flex flex-col overflow-hidden transition-all duration-200`}>
          <div className="p-3 border-b-2 border-foreground">
            <Button
              size="sm"
              className="w-full text-xs"
              onClick={handleNewSession}
              disabled={createSession.isPending}
            >
              <Plus className="w-3.5 h-3.5 mr-2" /> Sesi Baru
            </Button>
          </div>
          <div className="flex-1 overflow-y-auto p-2 space-y-1">
            {(sessions || []).map((s: any) => (
              <div key={s.id} className="flex items-center group">
                {editingSessionId === s.id ? (
                  <input
                    autoFocus
                    value={editTitle}
                    onChange={(e) => setEditTitle(e.target.value)}
                    onBlur={() => saveEdit(s.id)}
                    onKeyDown={(e) => { if (e.key === "Enter") saveEdit(s.id); if (e.key === "Escape") setEditingSessionId(null); }}
                    className="flex-1 px-3 py-2 text-xs font-bold border-2 border-foreground bg-background outline-none"
                  />
                ) : (
                  <div className="flex items-center gap-1 flex-1 min-w-0">
                    <button
                      onClick={() => switchSession(s.id)}
                      className={`flex-1 text-left px-3 py-2 text-xs font-bold transition-all cursor-pointer flex items-center gap-2 ${
                        currentSessionId === s.id
                          ? "bg-primary text-primary-foreground border-2 border-foreground shadow-[2px_2px_0px_0px] shadow-foreground"
                          : "text-muted-foreground hover:text-foreground hover:bg-card border-2 border-transparent"
                      }`}
                    >
                      <MessageSquare className="w-3.5 h-3.5 shrink-0" />
                      <div className="flex-1 min-w-0">
                        <span className="block truncate">{s.title || "Percakapan"}</span>
                        <span className="block text-[10px] text-muted-foreground/60 font-normal">{new Date(s.created_at).toLocaleDateString("id-ID", { month: "short", day: "numeric" })}</span>
                      </div>
                    </button>
                    <button
                      onClick={() => startEdit(s.id, s.title || "")}
                      className="p-1.5 text-muted-foreground hover:text-primary transition-all shrink-0"
                      title="Edit nama"
                    >
                      <PenLine className="w-3 h-3" />
                    </button>
                  </div>
                )}
                <button
                  onClick={() => setConfirmDeleteSessionId(s.id)}
                  className="opacity-0 group-hover:opacity-100 p-1 text-muted-foreground hover:text-red-500 transition-all"
                >
                  <X className="w-3 h-3" />
                </button>
              </div>
            ))}
            {(!sessions || sessions.length === 0) && !sessionsLoading && (
              <p className="text-xs text-muted-foreground text-center py-4 font-bold">Belum ada sesi</p>
            )}
            {sessionsLoading && (
              <div className="space-y-2 p-2">
                {[1,2,3].map((i) => (
                  <div key={i} className="h-8 bg-muted-foreground/10 border-2 border-foreground animate-pulse" />
                ))}
              </div>
            )}
          </div>
        </div>
      )}

      <div className="flex-1 flex flex-col">
        <div className="flex items-center justify-between p-3 border-b-2 border-foreground bg-card">
          <div className="flex items-center gap-2">
            <button
              onClick={() => setSidebarOpen(!sidebarOpen)}
              className="p-1 text-muted-foreground hover:text-foreground cursor-pointer"
              title={sidebarOpen ? "Tutup daftar sesi" : "Buka daftar sesi"}
            >
              {sidebarOpen ? <PanelLeftClose className="w-4 h-4" /> : <PanelLeft className="w-4 h-4" />}
            </button>
            <MessageSquare className="w-4 h-4 text-primary" />
            <h2 className="text-xs font-black uppercase tracking-wide text-foreground">AI Assistant</h2>
          </div>
          <div className="flex items-center gap-2">
            <div className="flex border-2 border-foreground p-0.5 bg-muted">
              <button
                onClick={() => setAiMode("fast")}
                className={`px-2 py-1 text-[10px] font-black cursor-pointer flex items-center gap-1 ${
                  aiMode === "fast" ? "bg-primary text-primary-foreground border-2 border-foreground shadow-[1px_1px_0px_0px] shadow-foreground" : "text-muted-foreground hover:text-foreground"
                }`}
              >
                <Zap className="w-3 h-3" /> CEPAT
              </button>
              <button
                onClick={() => setAiMode("reasoning")}
                className={`px-2 py-1 text-[10px] font-black cursor-pointer flex items-center gap-1 ${
                  aiMode === "reasoning" ? "bg-primary text-primary-foreground border-2 border-foreground shadow-[1px_1px_0px_0px] shadow-foreground" : "text-muted-foreground hover:text-foreground"
                }`}
              >
                <Brain className="w-3 h-3" /> PENALARAN
              </button>
            </div>
            {isStreaming && (
              <Button variant="ghost" size="sm" className="text-xs text-red-500" onClick={cancelStream}>
                <X className="w-3.5 h-3.5 mr-1" /> Stop
              </Button>
            )}
            {(messages.length > 0 || currentSessionId) && (
              <Button variant="ghost" size="sm" className="text-red-500 text-xs" onClick={() => { setMessages([]); }}>
                <Trash2 className="w-3.5 h-3.5 mr-1" /> Hapus
              </Button>
            )}
          </div>
        </div>

        <div ref={scrollContainerRef} onScroll={handleScroll} className="flex-1 overflow-y-auto p-4 space-y-3 bg-background">
          {sessionMessagesLoading && currentSessionId && messages.length === 0 && !isStreaming && (
            <div className="space-y-4 p-4">
              {[1,2,3].map((i) => (
                <div key={i} className="space-y-2">
                  <div className="h-4 bg-muted-foreground/10 border-2 border-foreground animate-pulse w-3/4" />
                  <div className="h-4 bg-muted-foreground/10 border-2 border-foreground animate-pulse w-1/2" />
                </div>
              ))}
            </div>
          )}

          {!currentSessionId && messages.length === 0 && !isStreaming && !pendingAutoSend && (
            <div className="flex items-center justify-center h-full">
              <div className="text-center max-w-sm">
                <div className="w-16 h-16 border-2 border-foreground bg-primary/10 flex items-center justify-center mx-auto mb-5">
                  <MessageSquare className="w-7 h-7 text-primary" />
                </div>
                <p className="text-base font-black uppercase tracking-wide text-foreground mb-2">Belum Ada Sesi</p>
                <p className="text-sm font-bold text-muted-foreground mb-6">
                  Mulai percakapan baru untuk bertanya atau mencatat transaksi menggunakan AI Assistant.
                </p>
                <Button
                  size="lg"
                  onClick={handleNewSession}
                  disabled={createSession.isPending}
                  className="shadow-[3px_3px_0px_0px] shadow-foreground"
                >
                  <Plus className="w-4 h-4 mr-2" />
                  {createSession.isPending ? "Membuat..." : "Buat Pesan Baru"}
                </Button>
              </div>
            </div>
          )}

          {currentSessionId && messages.length === 0 && !isStreaming && !(sessionMessagesLoading && currentSessionId) && (
            <div className="flex items-center justify-center h-full">
              <div className="text-center max-w-sm">
                <MessageSquare className="w-8 h-8 text-muted-foreground mx-auto mb-3" />
                <p className="text-sm font-bold text-muted-foreground mb-4">
                  Tanyakan apapun tentang keuangan Anda
                </p>
                <div className="space-y-2">
                  {[
                    { icon: Search, text: "Berapa total pengeluaran bulan ini?" },
                    { icon: PenLine, text: "Catat pengeluaran kopi Rp 25.000" },
                    { icon: BarChart3, text: "Buat grafik pengeluaran per kategori" },
                  ].map((item, i) => (
                    <button
                      key={i}
                      onClick={() => handleSend(item.text)}
                      className="w-full text-left border-2 border-foreground px-3 py-2 text-xs font-bold text-muted-foreground hover:text-foreground hover:bg-card transition-all cursor-pointer flex items-center gap-2"
                    >
                      <item.icon className="w-3.5 h-3.5 text-primary shrink-0" />
                      {item.text}
                    </button>
                  ))}
                </div>
              </div>
            </div>
          )}

          {renderMessages()}

          {/* Show tool steps only during active streaming */}
          {isStreaming && activeToolSteps.length > 0 && renderToolSteps()}

          {isStreaming && !streamingContent && toolSteps.length === 0 && <TypingIndicator />}
          <div ref={messagesEndRef} />
        </div>

        {currentSessionId && (
          <>
            {cooldownSeconds > 0 && (
              <div className="flex items-center gap-2 px-4 py-2 bg-amber-50 border-b-2 border-amber-500 text-xs font-bold text-amber-700">
                <Loader2 className="w-3.5 h-3.5 animate-spin shrink-0" />
                Cooldown {cooldownSeconds} detik. Tunggu sebelum mengirim pesan lagi.
              </div>
            )}
            <ChatInput onSend={handleSend} isSending={isStreaming || cooldownSeconds > 0} onReceiptResult={handleReceiptResult} />
          </>
        )}
      </div>

      <ConfirmDialog
        open={!!confirmDeleteSessionId}
        onOpenChange={(o) => { if (!o) setConfirmDeleteSessionId(null); }}
        onConfirm={() => {
          if (confirmDeleteSessionId) {
            if (currentSessionId === confirmDeleteSessionId) {
              setMessages([]);
              setCurrentSessionId(null);
              updateUrlSession(null);
            }
            deleteSession.mutate(confirmDeleteSessionId);
            toast.success("Sesi berhasil dihapus.");
            setConfirmDeleteSessionId(null);
          }
        }}
        title="Hapus Sesi"
        message="Yakin ingin menghapus sesi percakapan ini?"
        confirmText="Ya, Hapus"
      />
    </div>
  );
}
