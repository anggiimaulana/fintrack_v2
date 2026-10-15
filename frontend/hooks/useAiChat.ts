"use client";

import { keepPreviousData, useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useState, useCallback, useRef, useEffect, useMemo } from "react";
import api, { refreshSession } from "@/lib/api";
import { queryKeys } from "@/lib/cache-keys";

interface ChatMessage {
  role: "user" | "assistant";
  content: string;
  /** Assistant mode that produced the message, when known. */
  mode?: "fast" | "reasoning" | null;
  /** Concrete upstream model that served the reply, when known. */
  modelUsed?: string | null;
}

interface StreamEvent {
  type: "content" | "session_created" | "agent_step" | "tool_call" | "tool_result" | "title_update" | "done" | "error" | "cooldown";
  content?: string;
  step?: string;
  step_label?: string;
  step_status?: string;
  mode?: string;
  model?: string;
  tool_call?: { tool: string; parameters: Record<string, unknown> };
  tool_result?: { tool: string; data?: unknown; error?: string };
}

interface ToolExecStep {
  tool: string;
  status: "running" | "success" | "error";
  result?: unknown;
  error?: string;
}

export function useAiChat(source?: string) {
  const queryClient = useQueryClient();
  const [streamingContent, setStreamingContent] = useState("");
  const [isStreaming, setIsStreaming] = useState(false);
  const [toolSteps, setToolSteps] = useState<ToolExecStep[]>([]);
  const [lastToolSteps, setLastToolSteps] = useState<ToolExecStep[]>([]);
  const [cooldownSeconds, setCooldownSeconds] = useState(0);
  const abortRef = useRef<AbortController | null>(null);
  const cooldownRef = useRef<ReturnType<typeof setInterval> | null>(null);
  // Ref mirrors toolSteps state to avoid stale closure in streamChat
  const toolStepsRef = useRef<ToolExecStep[]>([]);

  useEffect(() => {
    if (cooldownSeconds > 0) {
      cooldownRef.current = setInterval(() => {
        setCooldownSeconds((prev) => {
          if (prev <= 1) {
            if (cooldownRef.current) clearInterval(cooldownRef.current);
            return 0;
          }
          return prev - 1;
        });
      }, 1000);
    }
    return () => {
      if (cooldownRef.current) clearInterval(cooldownRef.current);
    };
  }, [cooldownSeconds]);

  const queryKey = useMemo(() => source ? ["chat-sessions", source] : ["chat-sessions"], [source]);

  const { data: sessions, isLoading: sessionsLoading } = useQuery({
    queryKey,
    queryFn: async () => {
      try {
        const params = source ? `?source=${source}` : "";
        const res = await api.get(`/ai/sessions${params}`);
        return res.data?.data || [];
      } catch {
        return [];
      }
    },
    staleTime: 30_000,
    placeholderData: keepPreviousData,
  });

  const createSession = useMutation({
    mutationFn: async (title: string) => {
      const res = await api.post("/ai/sessions", { title, source });
      return res.data?.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey });
    },
  });

  const deleteSession = useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/ai/sessions/${id}`);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey });
    },
  });

  const updateSession = useMutation({
    mutationFn: async ({ id, title }: { id: string; title: string }) => {
      const res = await api.put(`/ai/sessions/${id}`, { title });
      return res.data?.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey });
    },
  });

  const streamChat = useCallback(
    async (
      message: string,
      sessionId: string | null,
      onMessage: (msg: ChatMessage) => void,
      mode: "fast" | "reasoning" = "fast",
    ): Promise<{ content: string; mode?: string; model?: string } | null> => {
      setIsStreaming(true);
      setStreamingContent("");
      setToolSteps([]);
      toolStepsRef.current = [];
      setLastToolSteps([]);
      setCooldownSeconds(0);

      abortRef.current = new AbortController();

      let lastContent = "";
      let doneMode: string | undefined;
      let doneModel: string | undefined;
      try {
        let url = `/api/v1/ai/chat?source=${source || "chatbot"}`;
        if (sessionId) url += `&session_id=${sessionId}`;

        let response = await fetch(url, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            "X-Requested-With": "XMLHttpRequest",
          },
          body: JSON.stringify({ message, mode }),
          credentials: "same-origin",
          signal: abortRef.current.signal,
        });

        if (response.status === 401) {
          try {
            await refreshSession();
            response = await fetch(url, {
              method: "POST",
              headers: {
                "Content-Type": "application/json",
                "X-Requested-With": "XMLHttpRequest",
              },
              body: JSON.stringify({ message, mode }),
              credentials: "same-origin",
              signal: abortRef.current.signal,
            });
          } catch {
            if (typeof window !== "undefined") {
              window.location.href = "/login";
            }
            throw new Error("HTTP 401");
          }
        }
        if (!response.ok) {
          // Preserve the server's error code so quota gates (429
          // insufficient_quota) can be surfaced as an upsell instead of a
          // generic failure. The body is JSON for non-streaming errors.
          let code = `HTTP ${response.status}`;
          try {
            const body = await response.json();
            if (body && typeof body.error === "string") code = body.error;
          } catch {
            // Non-JSON body; keep the HTTP status code.
          }
          throw new Error(code);
        }

		const reader = response.body?.getReader();
		if (!reader) throw new Error("No response body");

		const decoder = new TextDecoder();
		let buffer = "";

		while (true) {
          const { done, value } = await reader.read();
          if (done) break;

          buffer += decoder.decode(value, { stream: true });
          const lines = buffer.split("\n");
          buffer = lines.pop() || "";

          for (const line of lines) {
            if (!line.startsWith("data: ")) continue;
            const jsonStr = line.slice(6).trim();
            if (!jsonStr) continue;

            try {
              const event: StreamEvent = JSON.parse(jsonStr);

  			switch (event.type) {
				case "session_created":
				  onMessage({ role: "assistant", content: "" });
				  break;
				case "content":
                  if (event.content) {
                    lastContent += event.content;
                    setStreamingContent(lastContent);
                  }
                  break;
                case "agent_step": {
                  const label = event.step_label || "";
                  const rawStatus = event.step_status || "running";
                  const stepTool = label.toLowerCase().replace(/[^a-z]/g, "_").replace(/_+/g, "_").replace(/^_|_$/g, "") || "analyzing";
                  const mappedStatus = rawStatus === "running" ? "running" : "success";
                  setToolSteps((prev) => {
                    const existing = prev.findIndex(s => s.tool === stepTool);
                    let next: ToolExecStep[];
                    if (existing >= 0) {
                      next = [...prev];
                      next[existing] = { ...next[existing], status: mappedStatus };
                    } else {
                      next = [...prev, { tool: stepTool, status: mappedStatus }];
                    }
                    toolStepsRef.current = next;
                    return next;
                  });
                  break;
                }
                case "tool_call": {
                  const info = event.tool_call;
                  if (info) {
                    setToolSteps((prev) => {
                      const next = [...prev, { tool: info.tool, status: "running" as const }];
                      toolStepsRef.current = next;
                      return next;
                    });
                  }
                  break;
                }
                case "tool_result": {
                  const info = event.tool_result;
                  if (info) {
                    setToolSteps((prev) => {
                      const next = [...prev];
                      let idx = -1;
                      for (let i = next.length - 1; i >= 0; i--) {
                        if (next[i].tool === info.tool && next[i].status === "running") {
                          idx = i;
                          break;
                        }
                      }
                      if (idx >= 0) {
                        next[idx] = {
                          tool: info.tool,
                          status: info.error ? "error" : "success",
                          result: info.data,
                          error: info.error,
                        };
                      }
                      toolStepsRef.current = next;
                      return next;
                    });
                    if (info.tool === "create_transaction" && !info.error) {
                      queryClient.invalidateQueries({ queryKey: ["transactions"] });
                      queryClient.invalidateQueries({ queryKey: ["dashboard"] });
                      queryClient.invalidateQueries({ queryKey: ["stats"] });
                    }
                  }
                  break;
                }
                case "title_update":
                  if (event.content) {
                    queryClient.invalidateQueries({ queryKey });
                  }
                  break;
                case "cooldown":
                  if (event.content) {
                    setCooldownSeconds(parseInt(event.content, 10) || 5);
                  }
                  break;
                case "done":
                  if (event.mode) doneMode = event.mode;
                  if (event.model) doneModel = event.model;
                  break;
                case "error":
                  if (event.content) lastContent += event.content;
                  break;
              }
            } catch {
              // skip malformed events
            }
          }
        }

		setStreamingContent("");
		setLastToolSteps([...toolStepsRef.current]);
		setIsStreaming(false);
		if (!lastContent) return null;
		return { content: lastContent, mode: doneMode, model: doneModel };
      } catch (err: unknown) {
        if (err instanceof Error && err.name === "AbortError") {
          setLastToolSteps([...toolStepsRef.current]);
          setIsStreaming(false);
          if (!lastContent) return null;
          return { content: lastContent, mode: doneMode, model: doneModel };
        }
        setLastToolSteps([...toolStepsRef.current]);
        setIsStreaming(false);
        throw err;
      } finally {
        // Every settled turn consumed AI tokens/requests, so refresh the quota
        // bars. Runs on success, abort, and error alike; the backend event
        // (usage_changed) covers other tabs.
        queryClient.invalidateQueries({ queryKey: queryKeys.entitlements.me });
      }
    },
    [queryClient, queryKey, source],
  );

  const cancelStream = useCallback(() => {
    abortRef.current?.abort();
    setIsStreaming(false);
    setStreamingContent("");
  }, []);

  return {
    sessions,
    sessionsLoading,
    createSession,
    deleteSession,
    updateSession,
    streamChat,
    cancelStream,
    isStreaming,
    streamingContent,
    toolSteps,
    lastToolSteps,
    cooldownSeconds,
  };
}
