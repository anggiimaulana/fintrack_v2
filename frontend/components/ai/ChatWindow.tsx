"use client";

import { useState, useRef, useEffect, useCallback } from "react";
import { useAiChat } from "@/hooks/useAiChat";
import { ChatMessage } from "@/components/ai/ChatMessage";
import { ChatInput } from "@/components/ai/ChatInput";
import { VoiceHistoryPanel } from "@/components/ai/VoiceHistoryPanel";
import { AiQuotaDialog } from "@/components/ai/AiQuotaDialog";
import { TypingIndicator } from "@/components/ai/TypingIndicator";
import { Button } from "@/components/ui/button";
import { toast } from "sonner";
import { Trash2, Plus, MessageSquare, Loader2, CheckCircle2, Search, PenLine, BarChart3, X, Zap, Brain, PanelLeftClose, PanelLeft, Gauge, AlertTriangle } from "lucide-react";
import { ConfirmDialog } from "@/components/shared/ConfirmDialog";
import { useQuery, useQueryClient, useMutation } from "@tanstack/react-query";

import api from "@/lib/api";
import { notifyEntitlementError } from "@/lib/entitlement-errors";
import { useAdminCategories } from "@/hooks/useAdminCategories";
import { useEntitlements } from "@/hooks/use-entitlements";
import type { ReceiptResult } from "@/lib/receipt";
import { buildReceiptTransaction, formatRupiahIDR, normalizeReceiptType, receiptTypeLabel } from "@/lib/receipt";

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
  // Quota mirror (UX only — the backend is the gate). Used to warn the user
  // before a token-exhausted turn surprises them with a limit error.
  const { usage: entitlementUsage } = useEntitlements();

  const [messages, setMessages] = useState<Array<{ role: string; content: string; mode?: "fast" | "reasoning" | null; modelUsed?: string | null }>>([]);
  // Initialised to null, NOT from `location.search`: the server has no window,
  // so reading the URL during the first render made the client tree differ from
  // the server one (the "Hapus" button rendered client-only) and triggered a
  // hydration mismatch. The URL is read after mount instead (effect below).
  const [currentSessionId, setCurrentSessionId] = useState<string | null>(null);
  const lastLoadedSessionIdRef = useRef<string | null>(null);
  const [editingSessionId, setEditingSessionId] = useState<string | null>(null);
  // Focus the rename field when it appears. Programmatic focus rather than the
  // `autoFocus` prop: same UX, and it does not steal focus on first render.
  const renameInputRef = useRef<HTMLInputElement>(null);
  useEffect(() => {
    if (editingSessionId) renameInputRef.current?.focus();
  }, [editingSessionId]);
  const [editTitle, setEditTitle] = useState("");
  const [aiMode, setAiMode] = useState<"fast" | "reasoning">("fast");
  const { categories: incomeCategories } = useAdminCategories("income");
  const { categories: expenseCategories } = useAdminCategories("expense");
  const { categories: savingsCategories } = useAdminCategories("savings");
  const isMobile = useMediaQuery("(max-width: 767px)");
  // Two independent states rather than one shared flag. The session list is a
  // docked rail on desktop (open by default) and a full-screen overlay on
  // mobile (closed by default). Keeping them separate means the mobile overlay
  // starts closed without touching the desktop rail's initial width — a single
  // flag defaulting to "closed" would flip the desktop rail to w-0 during
  // hydration, and defaulting to "open" would flash the overlay on phones.
  const [desktopSidebarOpen, setDesktopSidebarOpen] = useState(true);
  const [mobileSidebarOpen, setMobileSidebarOpen] = useState(false);
  const sidebarOpen = isMobile ? mobileSidebarOpen : desktopSidebarOpen;
  const toggleSidebar = useCallback(() => {
    if (isMobile) setMobileSidebarOpen((v) => !v);
    else setDesktopSidebarOpen((v) => !v);
  }, [isMobile]);
  // Also null on the server: reading `?q=` during the first render would make
  // the client tree differ from the server one. Resolved in the mount effect.
  const [pendingAutoSend, setPendingAutoSend] = useState<string | null>(null);
  const [confirmDeleteSessionId, setConfirmDeleteSessionId] = useState<string | null>(null);
  const [quotaModalOpen, setQuotaModalOpen] = useState(false);
  // Store chart data per assistant message index so they persist across new messages
  const [chartDataMap, setChartDataMap] = useState<Record<number, { tool: string; status: string; result?: unknown; error?: string }[]>>({});
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const shouldAutoScroll = useRef(true);
  // Holds the in-flight session creation so concurrent sends reuse one session
  // instead of each creating their own (the 1-message → 2-sessions bug).
  const sessionCreateRef = useRef<Promise<{ id: string } | null> | null>(null);

  const { data: sessionMessages, isLoading: sessionMessagesLoading } = useQuery({
    queryKey: ["chat-messages", currentSessionId],
    queryFn: async () => {
      if (!currentSessionId) return [];
      try { const res = await api.get(`/ai/sessions/${currentSessionId}/messages`); return res.data?.data || []; }
      catch { return []; }
    },
    enabled: !!currentSessionId,
  });

  // Hydrate the active session and any pending auto-send from the URL AFTER
  // mount. Doing this in the initialiser read `location.search` during the
  // first render, which the server cannot do — the resulting mismatch made
  // React throw "Hydration failed" and regenerate the whole tree on the client.
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const session = params.get("session");
    if (session) {
      lastLoadedSessionIdRef.current = session;
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setCurrentSessionId(session);
    }
    const q = params.get("q");
    if (q) {
      const url = new URL(window.location.href);
      url.searchParams.delete("q");
      window.history.replaceState({}, "", url.toString());
      setPendingAutoSend(q);
    }
  }, []);

  useEffect(() => {
    if (!currentSessionId) {
      lastLoadedSessionIdRef.current = null;
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setMessages([]);
      return;
    }

    if (currentSessionId !== lastLoadedSessionIdRef.current) {
      lastLoadedSessionIdRef.current = currentSessionId;
      if (sessionMessages && sessionMessages.length > 0) {
        setMessages(sessionMessages.map((m: { role: "user" | "assistant", content: string, mode?: "fast" | "reasoning" | null, model_used?: string | null }) => ({
          role: m.role,
          content: m.content,
          mode: m.mode ?? null,
          modelUsed: m.model_used ?? null,
        })));
      } else {
        setMessages([]);
      }
      return;
    }

    if (sessionMessages && !isStreaming) {
      if (sessionMessages.length >= messages.length && sessionMessages.length > 0) {
        setMessages(sessionMessages.map((m: { role: "user" | "assistant", content: string, mode?: "fast" | "reasoning" | null, model_used?: string | null }) => ({
          role: m.role,
          content: m.content,
          mode: m.mode ?? null,
          modelUsed: m.model_used ?? null,
        })));
      }
    }
  }, [sessionMessages, isStreaming, currentSessionId, messages.length]);

  useEffect(() => {
    const handlePopState = () => {
      const params = new URLSearchParams(window.location.search);
      const session = params.get("session");
      if (session !== currentSessionId) {
        lastLoadedSessionIdRef.current = session;
        setCurrentSessionId(session);
        setMessages([]);
      }
    };
    window.addEventListener("popstate", handlePopState);
    return () => window.removeEventListener("popstate", handlePopState);
  }, [currentSessionId]);

  const lastAssistantIndex = messages.map((m, i) => m.role === "assistant" ? i : -1).filter(i => i >= 0).pop() ?? -1;

  // Token quota mirror for the header. `low` warns at 80% consumed so the user
  // is told the limit is near *before* the next turn is refused.
  const tokenUsage = entitlementUsage?.["ai_tokens_per_cycle"];
  const tokenQuota = tokenUsage
    ? {
        unlimited: tokenUsage.unlimited,
        remaining: Math.max(0, tokenUsage.remaining),
        limit: tokenUsage.limit,
        exhausted: !tokenUsage.unlimited && tokenUsage.remaining <= 0,
        low: !tokenUsage.unlimited && tokenUsage.limit > 0 && tokenUsage.remaining <= tokenUsage.limit * 0.2,
      }
    : null;

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

  // ensureSession returns the active session, creating exactly one when none
  // exists. Concurrent callers await the same in-flight promise, so a rapid
  // double-send (or the ?q= auto-send racing a manual send) cannot produce two
  // sessions for one message.
  const ensureSession = useCallback(
    async (title: string): Promise<{ id: string } | null> => {
      if (currentSessionId) return { id: currentSessionId };
      if (sessionCreateRef.current) return sessionCreateRef.current;

      const promise = createSession
        .mutateAsync(title)
        .then((newSession: { id: string } | null) => {
          if (newSession) {
            lastLoadedSessionIdRef.current = newSession.id;
            setCurrentSessionId(newSession.id);
            updateUrlSession(newSession.id);
          }
          return newSession ?? null;
        })
        .finally(() => {
          sessionCreateRef.current = null;
        });

      sessionCreateRef.current = promise;
      return promise;
    },
    [currentSessionId, createSession],
  );

  const handleSend = async (message: string, overrideSessionId?: string) => {
    let sessionId: string | null = overrideSessionId || currentSessionId;

    if (!sessionId && sessions && sessions.length > 0) {
      sessionId = sessions[0].id;
      setCurrentSessionId(sessionId);
      updateUrlSession(sessionId);
    }

    if (!sessionId) {
      // A single message must never create two sessions. Concurrent callers
      // (a fast double-click, or the ?q= auto-send path racing the send) share
      // one in-flight creation instead of each issuing its own POST.
      const created = await ensureSession(message.substring(0, 50));
      sessionId = created?.id ?? null;
    }

    const userMsg = { role: "user" as const, content: message };
    setMessages((prev) => [...prev, userMsg]);
    // Don't clear chartDataMap — previous charts should persist
    shouldAutoScroll.current = true;

    if (!sessionId) return;

    try {
      const result = await streamChat(
        message,
        sessionId,
        (msg) => {
          if (msg.role === "assistant" && !msg.content) {
            setMessages((prev) => [...prev, { role: "assistant", content: "", mode: aiMode }]);
          }
        },
        aiMode,
      );

      if (result) {
        const resultMode = (result.mode as "fast" | "reasoning" | undefined) ?? aiMode;
        setMessages((prev) => {
          const next = [...prev];
          const lastIdx = next.length - 1;
          const patch = { content: result.content, mode: resultMode, modelUsed: result.model ?? null };
          if (lastIdx >= 0 && next[lastIdx].role === "assistant") {
            next[lastIdx] = { ...next[lastIdx], ...patch };
          } else {
            next.push({ role: "assistant", ...patch });
          }
          return next;
        });
      } else {
        setMessages((prev) => {
          const last = prev[prev.length - 1];
          if (last && last.role === "assistant" && !last.content) {
            return prev.slice(0, -1);
          }
          return prev;
        });
      }
      // Refresh from server
      queryClient.invalidateQueries({ queryKey: ["chat-messages", sessionId] });
    } catch (err) {
      if (err instanceof Error && err.name === "AbortError") {
        setMessages((prev) => {
          const last = prev[prev.length - 1];
          if (last && last.role === "assistant" && !last.content) {
            return prev.slice(0, -1);
          }
          return prev;
        });
      } else {
        // Any plan/quota gate (402 limit_exceeded, 402 limit_not_in_plan,
        // 403 feature_not_in_plan, 429 insufficient_quota) is a plan issue, not
        // a bug: show the upgrade CTA instead of a bubble with a raw error code.
        const code = err instanceof Error ? err.message : String(err);
        const isGate = notifyEntitlementError({ response: { status: 402, data: { error: code } } });
        setMessages((prev) => {
          const last = prev[prev.length - 1];
          if (last && last.role === "assistant" && !last.content) {
            return prev.slice(0, -1);
          }
          return prev;
        });
        if (!isGate) {
          setMessages((prev) => [...prev, { role: "assistant", content: "Maaf, terjadi kesalahan. Silakan coba lagi." }]);
        }
      }
    }
  };

  useEffect(() => {
    if (!pendingAutoSend || isStreaming) return;
    const msg = pendingAutoSend;
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setPendingAutoSend(null);
    // Send directly: handleSend creates the session when needed (through the
    // shared ensureSession guard). The old code pre-created a session and then
    // re-sent after a 300 ms timer, which opened a race window that produced a
    // second session for the same message.
    void handleSend(msg);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pendingAutoSend]);

  

  const switchSession = async (id: string) => {
    if (isStreaming) cancelStream();
    lastLoadedSessionIdRef.current = id;
    setMessages([]);
    setCurrentSessionId(id);
    updateUrlSession(id);
    try {
      const res = await api.get(`/ai/sessions/${id}/messages`);
      const msgs = res.data?.data || [];
      setMessages(msgs.map((m: { role: "user" | "assistant", content: string, mode?: "fast" | "reasoning" | null, model_used?: string | null }) => ({
        role: m.role,
        content: m.content,
        mode: m.mode ?? null,
        modelUsed: m.model_used ?? null,
      })));
    } catch {
      setMessages([]);
    }
  };

  const createFromReceiptMutation = useMutation({
    mutationFn: async ({ result, sessionId }: { result: ReceiptResult; sessionId: string | null }) => {
      const { payload } = buildReceiptTransaction(result, {
        income: incomeCategories,
        expense: expenseCategories,
        savings: savingsCategories,
      });
      const res = await api.post("/ai/receipt/transaction", payload);
      return { tx: res.data?.data, result, sessionId };
    },
    onSuccess: ({ tx, result, sessionId }) => {
      const formatted = formatRupiahIDR(result.total || 0);
      const typeLabel = receiptTypeLabel(normalizeReceiptType(result.type));
      let assistantMsg = `✅ **Struk berhasil dicatat sebagai transaksi ${typeLabel}!**\n\n- **Deskripsi:** ${result.description || result.merchant || "-"}\n- **Kategori:** ${result.category || "-"}\n- **Total:** ${formatted}\n- **Tanggal:** ${result.date || "-"}\n- **ID Transaksi:** \`${tx?.id || "-"}\``;
      if (result.items && result.items.length > 0) {
        assistantMsg += `\n\n**Rincian Item:**\n` + result.items.map((it, idx) => `${idx + 1}. ${it.name} (${it.quantity || 1}x) - ${formatRupiahIDR(it.total || it.unit_price || 0)}`).join("\n");
      }
      setMessages((prev) => [...prev, { role: "assistant", content: assistantMsg, mode: aiMode }]);
      shouldAutoScroll.current = true;
      if (sessionId) {
        api.post(`/ai/sessions/${sessionId}/messages`, {
          role: "assistant",
          content: assistantMsg,
          mode: aiMode,
        }).catch(() => {});
      }
    },
    onError: (err: unknown, { result, sessionId }) => {
      const formatted = formatRupiahIDR(result.total || 0);
      const errorMsg = `⚠️ Gagal menyimpan struk: ${(err instanceof Error ? err.message : String(err))}\n\nHasil analisis: ${result.description || result.merchant || "-"} - ${formatted}`;
      setMessages((prev) => [...prev, { role: "assistant", content: errorMsg, mode: aiMode }]);
      shouldAutoScroll.current = true;
      if (sessionId) {
        api.post(`/ai/sessions/${sessionId}/messages`, {
          role: "assistant",
          content: errorMsg,
          mode: aiMode,
        }).catch(() => {});
      }
    },
  });

  const handleReceiptResult = async (result: ReceiptResult, previewUrl?: string) => {
    if (!result?.success || !result?.total) return;

    let sessionId: string | null = currentSessionId;
    if (!sessionId && sessions && sessions.length > 0) {
      sessionId = sessions[0].id;
      setCurrentSessionId(sessionId);
      updateUrlSession(sessionId);
    }
    if (!sessionId) {
      const created = await ensureSession(result.merchant ? `Struk: ${result.merchant}` : "Struk Belanja");
      sessionId = created?.id ?? null;
    }

    const userMsgContent = previewUrl
      ? `![Foto Struk](${previewUrl})\n\n📷 *Mengunggah foto struk belanja: ${result.merchant || result.description || "Struk"}*`
      : `📷 *Mengunggah foto struk belanja: ${result.merchant || result.description || "Struk"}*`;

    setMessages((prev) => [...prev, { role: "user", content: userMsgContent }]);
    shouldAutoScroll.current = true;

    if (sessionId) {
      api.post(`/ai/sessions/${sessionId}/messages`, {
        role: "user",
        content: userMsgContent,
      }).catch(() => {});
    }

    if (result.category) {
      createFromReceiptMutation.mutate({ result, sessionId });
    } else {
      const formatted = formatRupiahIDR(result.total || 0);
      const assistantMsg = `📋 **Hasil scan struk: ${result.merchant || "-"}** - ${formatted}\n\n*Tidak dapat menentukan kategori otomatis. Silakan catat manual di menu Transaksi.*`;
      setMessages((prev) => [...prev, { role: "assistant", content: assistantMsg, mode: aiMode }]);
      shouldAutoScroll.current = true;
      if (sessionId) {
        api.post(`/ai/sessions/${sessionId}/messages`, {
          role: "assistant",
          content: assistantMsg,
          mode: aiMode,
        }).catch(() => {});
      }
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
    // A rapid double-click must not create two sessions; the button's
    // `disabled` only flips after React re-renders.
    if (createSession.isPending) return;
    const newSession = await createSession.mutateAsync("Percakapan baru");
    if (newSession) {
      lastLoadedSessionIdRef.current = newSession.id;
      setMessages([]);
      setCurrentSessionId(newSession.id);
      updateUrlSession(newSession.id);
    }
  };

  // Use streaming tool steps during stream, persisted ones (from chartDataMap) after
  const activeToolSteps = isStreaming ? toolSteps : (chartDataMap[lastAssistantIndex] || []);

  const renderToolSteps = () => {
    if (activeToolSteps.length === 0) return null;

    const chartStep = (activeToolSteps as { tool: string; status: string; result?: unknown }[]).find((s) => s.tool === "generate_chart" && s.status === "success" && s.result);

    return (
      <div className="border-2 rounded-md border-foreground bg-muted p-3 space-y-2 mb-2">
        <p className="text-[10px] font-black uppercase tracking-wide text-muted-foreground">AI Agent</p>
        {(activeToolSteps as { tool: string; status: string; error?: string; result?: unknown }[]).map((step, i: number) => (
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
        {chartStep && chartStep.result ? renderChartResult(chartStep.result as Record<string, unknown>) : null}
      </div>
    );
  };

  const renderChartResult = (result: Record<string, unknown>) => {
    const labels: string[] = (result.labels as string[]) || [];
    const income: number[] = (result.income as number[]) || [];
    const expense: number[] = (result.expense as number[]) || [];
    const savings: number[] = (result.savings as number[]) || [];
    const chartType: string = (result.chart_type as string) || "comparison";

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
          <div className="h-44 border-2 rounded-md border-foreground p-2 bg-card">
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
            <div className="h-36 border-2 rounded-md border-foreground p-2 bg-card">
              <p className="text-[10px] font-black uppercase tracking-wide text-muted-foreground mb-1">Grafik Batang</p>
              <div className="h-[calc(100%-20px)]">
                <BarChartView data={chartData} />
              </div>
            </div>
            <div className="h-36 border-2 rounded-md border-foreground p-2 bg-card">
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
          <ChatMessage author={msg.role} content={displayContent} mode={msg.mode} modelUsed={msg.modelUsed} />
          {hasChart && !isStreaming && renderPersistedToolSteps(msgChartData)}
        </div>
      );
    });
  };

  const renderPersistedToolSteps = (steps: unknown[]) => {
    const chartStep = (steps as { tool: string; status: string; result?: unknown }[]).find(s => s.tool === "generate_chart" && s.status === "success" && s.result);
    if (!chartStep) return null;
    return (
      <div className="border-2 rounded-md border-foreground bg-muted p-3 space-y-2 mb-2 mt-1">
        <p className="text-[10px] font-black uppercase tracking-wide text-muted-foreground">Visualisasi</p>
        {chartStep.result ? renderChartResult(chartStep.result as Record<string, unknown>) : null}
      </div>
    );
  };

  return (
    <div className="flex h-full min-h-0 w-full">
      {isMobile && sidebarOpen && (
        <div className="fixed inset-0 z-50 flex">
          <div role="presentation" className="absolute inset-0 bg-black/30" onClick={() => setMobileSidebarOpen(false)} />
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
            <div className="flex-1 overflow-y-auto overflow-x-hidden p-2 space-y-1">
              {(sessions || []).map((s: { id: string; title: string; created_at?: string }) => (
                <div key={s.id} className="flex items-center group min-w-0">
                  {editingSessionId === s.id ? (
                    <input
                      ref={renameInputRef}
                      value={editTitle}
                      onChange={(e) => setEditTitle(e.target.value)}
                      onBlur={() => saveEdit(s.id)}
                      onKeyDown={(e) => { if (e.key === "Enter") saveEdit(s.id); if (e.key === "Escape") setEditingSessionId(null); }}
                      className="flex-1 px-3 py-2 text-xs font-bold border-2 rounded-md border-foreground bg-background outline-none"
                    />
                  ) : (
                    <div className="flex items-center gap-1 flex-1 min-w-0">
                      <button
                        onClick={() => { switchSession(s.id); if (isMobile) setMobileSidebarOpen(false); }}
                        className={`flex-1 min-w-0 text-left px-3 py-2 text-xs font-bold transition-all cursor-pointer flex items-center gap-2 ${
                          currentSessionId === s.id
                            ? "bg-primary text-primary-foreground border-2 rounded-md border-foreground shadow-[2px_2px_0px_0px] shadow-foreground"
                            : "text-muted-foreground hover:text-foreground hover:bg-card border-2 border-transparent"
                        }`}
                      >
                        <MessageSquare className="w-3.5 h-3.5 shrink-0" />
                        <div className="flex-1 min-w-0">
                          <span className="block truncate">{s.title || "Percakapan"}</span>
                          <span className="block text-[10px] text-muted-foreground/60 font-normal">{new Date(s.created_at ? s.created_at : new Date().toISOString()).toLocaleDateString("id-ID", { month: "short", day: "numeric" })}</span>
                        </div>
                      </button>
                      <button
                        onClick={() => startEdit(s.id, s.title || "")}
                        className="p-1.5 text-muted-foreground hover:text-primary transition-all shrink-0 cursor-pointer"
                        title="Edit nama"
                      >
                        <PenLine className="w-3 h-3" />
                      </button>
                    </div>
                  )}
                  <button
                    onClick={() => setConfirmDeleteSessionId(s.id)}
                    aria-label="Hapus sesi"
                    className="opacity-0 group-hover:opacity-100 p-1 text-muted-foreground hover:text-red-500 transition-all cursor-pointer"
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
                    <div key={i} className="h-8 bg-muted-foreground/10 border-2 rounded-md border-foreground animate-pulse" />
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>
      )}
      {/* Docked session rail. Visibility is CSS-driven (`hidden md:flex`) rather
          than JS-gated so the server-rendered markup never flashes this rail on
          a phone before hydration reports a mobile viewport. */}
      <div
        className={`hidden md:flex ${desktopSidebarOpen ? "md:w-64" : "md:w-0"} min-w-0 border-r-2 border-foreground bg-muted flex-col overflow-hidden transition-all duration-200`}
      >
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
          <div className="flex-1 overflow-y-auto overflow-x-hidden p-2 space-y-1">
            {(sessions || []).map((s: { id: string; title: string; created_at?: string }) => (
              <div key={s.id} className="flex items-center group min-w-0">
                {editingSessionId === s.id ? (
                  <input
                    ref={renameInputRef}
                    value={editTitle}
                    onChange={(e) => setEditTitle(e.target.value)}
                    onBlur={() => saveEdit(s.id)}
                    onKeyDown={(e) => { if (e.key === "Enter") saveEdit(s.id); if (e.key === "Escape") setEditingSessionId(null); }}
                    className="flex-1 min-w-0 px-3 py-2 text-xs font-bold border-2 rounded-md border-foreground bg-background outline-none"
                  />
                ) : (
                  <div className="flex items-center gap-1 flex-1 min-w-0">
                    <button
                      onClick={() => switchSession(s.id)}
                      className={`flex-1 min-w-0 text-left px-3 py-2 text-xs font-bold transition-all cursor-pointer flex items-center gap-2 ${
                        currentSessionId === s.id
                          ? "bg-primary text-primary-foreground border-2 rounded-md border-foreground shadow-[2px_2px_0px_0px] shadow-foreground"
                          : "text-muted-foreground hover:text-foreground hover:bg-card border-2 border-transparent"
                      }`}
                    >
                      <MessageSquare className="w-3.5 h-3.5 shrink-0" />
                      <div className="flex-1 min-w-0">
                        <span className="block truncate">{s.title || "Percakapan"}</span>
                        <span className="block text-[10px] text-muted-foreground/60 font-normal">{new Date(s.created_at ? s.created_at : new Date().toISOString()).toLocaleDateString("id-ID", { month: "short", day: "numeric" })}</span>
                      </div>
                    </button>
                    <button
                      onClick={() => startEdit(s.id, s.title || "")}
                      className="p-1.5 text-muted-foreground hover:text-primary transition-all shrink-0 cursor-pointer"
                      title="Edit nama"
                    >
                      <PenLine className="w-3 h-3" />
                    </button>
                  </div>
                )}
                <button
                  onClick={() => setConfirmDeleteSessionId(s.id)}
                  aria-label="Hapus sesi"
                  className="opacity-0 group-hover:opacity-100 p-1 text-muted-foreground hover:text-red-500 transition-all cursor-pointer"
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
                  <div key={i} className="h-8 bg-muted-foreground/10 border-2 rounded-md border-foreground animate-pulse" />
                ))}
              </div>
            )}
          </div>
      </div>

      <div className="flex-1 flex flex-col min-w-0">
        <div className="flex items-center justify-between gap-1.5 sm:gap-2 p-2 sm:p-3 border-b-2 border-foreground bg-card min-w-0">
          <div className="flex items-center gap-1.5 sm:gap-2 min-w-0 shrink-0">
            <button
              onClick={toggleSidebar}
              className="p-1 text-muted-foreground hover:text-foreground cursor-pointer shrink-0 border-2 rounded-md border-transparent hover:border-foreground transition-colors"
              aria-label={sidebarOpen ? "Tutup daftar sesi" : "Buka daftar sesi"}
              title={sidebarOpen ? "Tutup daftar sesi" : "Buka daftar sesi"}
            >
              {sidebarOpen ? <PanelLeftClose className="w-4 h-4" /> : <PanelLeft className="w-4 h-4" />}
            </button>
            <MessageSquare className="w-4 h-4 text-primary shrink-0" />
            {/* The title yields to actions when sidebar is open or screen is narrow */}
            <h2 className={`text-xs font-black uppercase tracking-wide text-foreground truncate ${desktopSidebarOpen ? "hidden 2xl:block" : "hidden md:block"}`}>
              AI Assistant
            </h2>
          </div>
          <div className="flex items-center gap-1 sm:gap-1.5 shrink-0">
            <VoiceHistoryPanel />
            <div className="flex border-2 rounded-md border-foreground p-0.5 bg-muted">
              <button
                onClick={() => setAiMode("fast")}
                aria-label="Mode cepat"
                title="Mode Cepat (Respons kilat, hemat kuota)"
                aria-pressed={aiMode === "fast"}
                className={`px-1.5 sm:px-2 py-1 text-[10px] font-black cursor-pointer flex items-center gap-1 transition-all ${
                  aiMode === "fast"
                    ? "bg-primary text-primary-foreground border-2 rounded-md border-foreground shadow-[1px_1px_0px_0px] shadow-foreground"
                    : "text-muted-foreground hover:text-foreground"
                }`}
              >
                <Zap className="w-3.5 h-3.5 shrink-0" />
                <span className={desktopSidebarOpen ? "hidden xl:inline" : "hidden sm:inline"}>CEPAT</span>
              </button>
              <button
                onClick={() => setAiMode("reasoning")}
                aria-label="Mode penalaran"
                title="Mode Penalaran (Analisis mendalam)"
                aria-pressed={aiMode === "reasoning"}
                className={`px-1.5 sm:px-2 py-1 text-[10px] font-black cursor-pointer flex items-center gap-1 transition-all ${
                  aiMode === "reasoning"
                    ? "bg-primary text-primary-foreground border-2 rounded-md border-foreground shadow-[1px_1px_0px_0px] shadow-foreground"
                    : "text-muted-foreground hover:text-foreground"
                }`}
              >
                <Brain className="w-3.5 h-3.5 shrink-0" />
                <span className={desktopSidebarOpen ? "hidden xl:inline" : "hidden sm:inline"}>PENALARAN</span>
              </button>
            </div>
            {tokenQuota && (
              <button
                type="button"
                onClick={() => setQuotaModalOpen(true)}
                title={`Sisa kuota AI: ${tokenQuota.unlimited ? "Unlimited" : `${tokenQuota.remaining.toLocaleString("id-ID")} / ${tokenQuota.limit.toLocaleString("id-ID")}`} (Klik untuk detail)`}
                aria-label="Lihat informasi kuota AI"
                className={`inline-flex items-center gap-1 px-2 py-1 text-[10px] font-black uppercase tracking-wide border-2 rounded-md border-foreground cursor-pointer transition-transform hover:scale-105 active:scale-95 shadow-[1px_1px_0px_0px] shadow-foreground ${
                  tokenQuota.exhausted
                    ? "bg-destructive text-primary-foreground"
                    : tokenQuota.low
                      ? "bg-amber-100 text-amber-800 dark:bg-amber-950/40 dark:text-amber-300"
                      : "bg-muted text-foreground hover:bg-muted/80"
                }`}
              >
                <Gauge className="w-3.5 h-3.5 shrink-0" />
                <span className={desktopSidebarOpen ? "hidden 2xl:inline" : "hidden xl:inline"}>
                  {tokenQuota.exhausted
                    ? "Kuota habis"
                    : tokenQuota.unlimited
                      ? "Kuota Unlimited"
                      : `Sisa kuota ${tokenQuota.remaining.toLocaleString("id-ID")}`}
                </span>
              </button>
            )}
            {(messages.length > 0 || currentSessionId) && (
              <Button
                variant="ghost"
                size="sm"
                className="text-red-500 text-xs px-2 cursor-pointer hover:bg-red-50 dark:hover:bg-red-950/20"
                onClick={() => { setMessages([]); }}
                aria-label="Bersihkan pesan"
                title="Bersihkan pesan"
              >
                <Trash2 className="w-3.5 h-3.5" />
                <span className={desktopSidebarOpen ? "hidden xl:inline ml-1" : "hidden sm:inline ml-1"}>Hapus</span>
              </Button>
            )}
          </div>
        </div>

        <div ref={scrollContainerRef} onScroll={handleScroll} className="flex-1 overflow-y-auto p-4 space-y-3 bg-background">
          {sessionMessagesLoading && currentSessionId && messages.length === 0 && !isStreaming && (
            <div className="space-y-4 p-4">
              {[1,2,3].map((i) => (
                <div key={i} className="space-y-2">
                  <div className="h-4 bg-muted-foreground/10 border-2 rounded-md border-foreground animate-pulse w-3/4" />
                  <div className="h-4 bg-muted-foreground/10 border-2 rounded-md border-foreground animate-pulse w-1/2" />
                </div>
              ))}
            </div>
          )}

          {!currentSessionId && messages.length === 0 && !isStreaming && !pendingAutoSend && (
            <div className="flex items-center justify-center h-full px-2">
              <div className="text-center w-full max-w-sm">
                <div className="w-16 h-16 border-2 border-foreground bg-primary/10 flex items-center justify-center mx-auto mb-5">
                  <MessageSquare className="w-7 h-7 text-primary" />
                </div>
                <p className="text-base font-black uppercase tracking-wide text-foreground mb-2">Belum Ada Sesi</p>
                <p className="text-sm font-bold text-muted-foreground mb-6 text-pretty">
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
            <div className="flex items-center justify-center h-full px-2">
              <div className="text-center w-full max-w-sm">
                <MessageSquare className="w-8 h-8 text-muted-foreground mx-auto mb-3" />
                <p className="text-sm font-bold text-muted-foreground mb-4 text-pretty">
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
                      disabled={isStreaming}
                      className="w-full text-left border-2 rounded-md border-foreground px-3 py-2 text-xs font-bold text-muted-foreground hover:text-foreground hover:bg-card transition-all cursor-pointer disabled:cursor-not-allowed disabled:opacity-50 flex items-center gap-2"
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
            {tokenQuota?.exhausted && (
              <div className="flex items-center gap-2 px-4 py-2 bg-destructive/10 border-b-2 border-destructive text-xs font-bold text-destructive">
                <AlertTriangle className="w-3.5 h-3.5 shrink-0" />
                <span>
                  Kuota token AI siklus ini sudah habis.{" "}
                  <a href="/user/subscription" className="underline font-black">
                    Tingkatkan paket
                  </a>{" "}
                  untuk melanjutkan.
                </span>
              </div>
            )}
            {cooldownSeconds > 0 && (
              <div className="flex items-center gap-2 px-4 py-2 bg-amber-50 border-b-2 border-amber-500 text-xs font-bold text-amber-700">
                <Loader2 className="w-3.5 h-3.5 animate-spin shrink-0" />
                Cooldown {cooldownSeconds} detik. Tunggu sebelum mengirim pesan lagi.
              </div>
            )}
            <ChatInput onSend={handleSend} onStop={cancelStream} isSending={cooldownSeconds > 0} isStreaming={isStreaming} onReceiptResult={handleReceiptResult} />
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
      <AiQuotaDialog open={quotaModalOpen} onOpenChange={setQuotaModalOpen} />
    </div>
  );
}
