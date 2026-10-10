"use client";

import { useState } from "react";
import { Plus, MessageSquare, Trash2, Edit2, Check, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import api from "@/lib/api";
import { Skeleton } from "@/components/ui/skeleton";

interface Session {
  id: string;
  title: string;
  created_at: string;
  updated_at: string;
}

export default function SessionsPage() {
  const queryClient = useQueryClient();
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editTitle, setEditTitle] = useState("");
  const [newTitle, setNewTitle] = useState("");

  const { data: sessions, isLoading } = useQuery({
    queryKey: ["chat-sessions"],
    queryFn: async () => {
      try { const res = await api.get("/ai/sessions"); return res.data?.data || []; }
      catch { return []; }
    },
    placeholderData: (prev: any) => prev,
  });

  const createMutation = useMutation({
    mutationFn: async (title: string) => { const res = await api.post("/ai/sessions", { title }); return res.data; },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ["chat-sessions"] }); setNewTitle(""); },
  });

  const updateMutation = useMutation({
    mutationFn: async ({ id, title }: { id: string; title: string }) => { await api.put(`/ai/sessions/${id}`, { title }); },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ["chat-sessions"] }); setEditingId(null); },
  });

  const deleteMutation = useMutation({
    mutationFn: async (id: string) => { await api.delete(`/ai/sessions/${id}`); },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ["chat-sessions"] }); },
  });

  const startEdit = (s: Session) => { setEditingId(s.id); setEditTitle(s.title); };
  const cancelEdit = () => { setEditingId(null); setEditTitle(""); };

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-black uppercase tracking-wide text-foreground">Sesi Chat</h2>
          <p className="text-xs font-bold text-muted-foreground mt-1">Kelola sesi percakapan AI Anda.</p>
        </div>
      </div>

      <div className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-4">
        <form
          onSubmit={(e) => { e.preventDefault(); if (newTitle.trim()) createMutation.mutate(newTitle.trim()); }}
          className="flex gap-3"
        >
          <input
            type="text"
            value={newTitle}
            onChange={(e) => setNewTitle(e.target.value)}
            placeholder="Nama sesi baru..."
            className="flex-1 border-2 border-foreground bg-background px-4 py-2.5 text-sm font-bold focus:shadow-[2px_2px_0px_0px] focus:shadow-foreground focus:outline-none placeholder:text-muted-foreground/70"
          />
          <Button type="submit" disabled={!newTitle.trim() || createMutation.isPending}>
            <Plus className="w-4 h-4 mr-2" /> Buat Sesi
          </Button>
        </form>
      </div>

      <div className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground divide-y-2 divide-foreground">
        {isLoading ? (
          Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="p-4 flex items-center gap-4">
              <Skeleton className="w-10 h-10 border-2 border-foreground" />
              <div className="flex-1 space-y-1">
                <Skeleton className="h-5 w-48 border-2 border-foreground" />
                <Skeleton className="h-3 w-32 border-2 border-foreground" />
              </div>
            </div>
          ))
        ) : sessions && sessions.length > 0 ? (
          sessions.map((s: Session) => (
            <div key={s.id} className="p-4 flex items-center gap-4 hover:bg-muted/50 transition-colors group">
              <div className="w-10 h-10 border-2 border-foreground bg-primary/10 flex items-center justify-center shrink-0">
                <MessageSquare className="w-5 h-5 text-primary" />
              </div>

              {editingId === s.id ? (
                <div className="flex-1 flex gap-2">
                  <input
                    type="text"
                    value={editTitle}
                    onChange={(e) => setEditTitle(e.target.value)}
                    className="flex-1 border-2 border-foreground bg-background px-3 py-1.5 text-sm font-bold focus:shadow-[2px_2px_0px_0px] focus:shadow-foreground focus:outline-none"
                    autoFocus
                  />
                  <Button size="sm" variant="ghost" onClick={() => updateMutation.mutate({ id: s.id, title: editTitle })} disabled={!editTitle.trim()}>
                    <Check className="w-4 h-4" />
                  </Button>
                  <Button size="sm" variant="ghost" onClick={cancelEdit}>
                    <X className="w-4 h-4" />
                  </Button>
                </div>
              ) : (
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-bold text-foreground truncate">{s.title}</p>
                  <p className="text-xs font-bold text-muted-foreground">
                    {new Date(s.updated_at || s.created_at).toLocaleDateString("id-ID", { day: "numeric", month: "short", year: "numeric", hour: "2-digit", minute: "2-digit" })}
                  </p>
                </div>
              )}

              {editingId !== s.id && (
                <div className="flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                  <Button size="sm" variant="ghost" onClick={() => startEdit(s)}>
                    <Edit2 className="w-4 h-4" />
                  </Button>
                  <Button size="sm" variant="ghost" className="text-red-600 hover:text-red-700" onClick={() => { if (confirm("Hapus sesi ini?")) deleteMutation.mutate(s.id); }}>
                    <Trash2 className="w-4 h-4" />
                  </Button>
                </div>
              )}
            </div>
          ))
        ) : (
          <div className="p-8 text-center text-sm font-bold text-muted-foreground">
            <MessageSquare className="w-8 h-8 text-muted-foreground mx-auto mb-3" />
            Belum ada sesi chat. Buat sesi baru untuk memulai.
          </div>
        )}
      </div>
    </div>
  );
}
