"use client";

import { useState } from "react";
import { Plus, GripVertical, Play, Trash2, Shield, Key, AlertTriangle, CheckCircle, XCircle, RefreshCw, X, PenLine } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/shared/ConfirmDialog";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import api from "@/lib/api";
import { Skeleton } from "@/components/ui/skeleton";

const PROVIDERS = [
  { name: "gemini", display_name: "Gemini" },
  { name: "openai", display_name: "OpenAI" },
  { name: "groq", display_name: "Groq" },
  { name: "openrouter", display_name: "OpenRouter" },
  { name: "claude", display_name: "Claude" },
  { name: "qwen", display_name: "Qwen" },
  { name: "deepseek", display_name: "DeepSeek" },
  { name: "mistral", display_name: "Mistral" },
  { name: "cohere", display_name: "Cohere" },
  { name: "perplexity", display_name: "Perplexity" },
  { name: "other", display_name: "Lainnya" },
];

export default function ApiKeysPage() {
  const queryClient = useQueryClient();
  const [alias, setAlias] = useState("");
  const [keyValue, setKeyValue] = useState("");
  const [provider, setProvider] = useState("gemini");
  const [model, setModel] = useState("");
  const [supportsText, setSupportsText] = useState(true);
  const [supportsAudio, setSupportsAudio] = useState(false);
  const [supportsVision, setSupportsVision] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [editingKey, setEditingKey] = useState<any | null>(null);
  const [editAlias, setEditAlias] = useState("");
  const [editKeyValue, setEditKeyValue] = useState("");
  const [editProvider, setEditProvider] = useState("gemini");
  const [editModel, setEditModel] = useState("");
  const [editSupportsText, setEditSupportsText] = useState(true);
  const [editSupportsAudio, setEditSupportsAudio] = useState(false);
  const [editSupportsVision, setEditSupportsVision] = useState(false);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null);

  const { data: keys, isLoading } = useQuery({
    queryKey: ["api-keys"],
    queryFn: async () => {
      try { const res = await api.get("/gemini-keys"); return res.data?.data || []; }
      catch { return []; }
    },
    placeholderData: (prev: any) => prev,
  });

  const { data: providers } = useQuery({
    queryKey: ["providers"],
    queryFn: async () => {
      try { const res = await api.get("/providers"); return res.data?.data || []; }
      catch { return []; }
    },
    staleTime: 5 * 60 * 1000,
  });

  const providerOptions = providers?.length > 0 ? providers : PROVIDERS;

  const createMutation = useMutation({
    mutationFn: async (data: any) => { const res = await api.post("/gemini-keys", data); return res.data; },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ["api-keys"] }); },
  });

  const deleteMutation = useMutation({
    mutationFn: async (id: string) => { await api.delete(`/gemini-keys/${id}`); },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ["api-keys"] }); },
  });

  const updateMutation = useMutation({
    mutationFn: async ({ id, data }: { id: string; data: any }) => { const res = await api.patch(`/gemini-keys/${id}`, data); return res.data; },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ["api-keys"] }); },
  });

  const testMutation = useMutation({
    mutationFn: async (id: string) => { const res = await api.post(`/gemini-keys/${id}/test`); return res.data; },
  });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!alias || !keyValue || !model) return;
    if (!supportsText && !supportsAudio && !supportsVision) {
      toast.error("Pilih minimal 1 kemampuan (Text, Audio, atau Image).");
      return;
    }
    try {
      await createMutation.mutateAsync({ alias, api_key: keyValue, provider, model, supports_text: supportsText, supports_audio: supportsAudio, supports_vision: supportsVision });
      setAlias(""); setKeyValue(""); setProvider("gemini"); setModel("");
      setSupportsText(true); setSupportsAudio(false); setSupportsVision(false);
      toast.success("Key berhasil ditambahkan!");
    } catch { toast.error("Gagal menambahkan key."); }
  };

  const handleDialogSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!alias || !keyValue || !model) return;
    if (!supportsText && !supportsAudio && !supportsVision) {
      toast.error("Pilih minimal 1 kemampuan (Text, Audio, atau Image).");
      return;
    }
    try {
      await createMutation.mutateAsync({ alias, api_key: keyValue, provider, model, supports_text: supportsText, supports_audio: supportsAudio, supports_vision: supportsVision });
      setAlias(""); setKeyValue(""); setProvider("gemini"); setModel("");
      setSupportsText(true); setSupportsAudio(false); setSupportsVision(false);
      toast.success("Key berhasil ditambahkan!");
      setShowForm(false);
    } catch { toast.error("Gagal menambahkan key."); }
  };

  const openEdit = (key: any) => {
    setEditingKey(key);
    setEditAlias(key.alias);
    setEditKeyValue("");
    setEditProvider(key.provider || "gemini");
    setEditModel(key.model || "");
    setEditSupportsText(key.supports_text ?? true);
    setEditSupportsAudio(key.supports_audio ?? false);
    setEditSupportsVision(key.supports_vision ?? false);
  };

  const handleEditSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!editingKey) return;
    if (!editAlias || !editModel) { toast.error("Alias dan Model harus diisi."); return; }
    if (!editSupportsText && !editSupportsAudio && !editSupportsVision) {
      toast.error("Pilih minimal 1 kemampuan (Text, Audio, atau Image).");
      return;
    }
    try {
      const payload: any = { alias: editAlias, provider: editProvider, model: editModel, supports_text: editSupportsText, supports_audio: editSupportsAudio, supports_vision: editSupportsVision };
      if (editKeyValue.trim()) payload.api_key = editKeyValue.trim();
      await updateMutation.mutateAsync({
        id: editingKey.id,
        data: payload,
      });
      toast.success("Key berhasil diperbarui!");
      setEditingKey(null);
    } catch { toast.error("Gagal memperbarui key."); }
  };

  const handleTest = async (id: string) => {
    try {
      const result = await testMutation.mutateAsync(id);
      toast.success(result?.message || "Key berhasil terverifikasi!");
    } catch { toast.error("Key tidak valid atau gagal terhubung."); }
  };

  const isAllExhausted = keys?.every((k: any) => k.status !== "active");

  const getStatusBadge = (status: string) => {
    switch (status) {
      case "active": return <div className="flex items-center gap-1.5 px-2.5 py-1 border-2 border-foreground text-xs font-bold bg-green-100 text-green-700"><CheckCircle className="w-3.5 h-3.5" /> Aktif</div>;
      case "quota_exceeded": return <div className="flex items-center gap-1.5 px-2.5 py-1 border-2 border-foreground text-xs font-bold bg-amber-100 text-amber-700"><AlertTriangle className="w-3.5 h-3.5" /> Limit</div>;
      case "error": return <div className="flex items-center gap-1.5 px-2.5 py-1 border-2 border-foreground text-xs font-bold bg-red-100 text-red-700"><XCircle className="w-3.5 h-3.5" /> Error</div>;
      default: return <div className="flex items-center gap-1.5 px-2.5 py-1 border-2 border-foreground text-xs font-bold bg-gray-100 text-gray-700"><RefreshCw className="w-3.5 h-3.5" /> {status}</div>;
    }
  };

  return (
    <div className="flex flex-col gap-6 w-full max-w-4xl mx-auto h-full">
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h2 className="text-xl font-black uppercase tracking-wide text-foreground">API Keys</h2>
          <p className="text-xs font-bold text-muted-foreground mt-1">Kelola API key untuk berbagai provider AI. Round-robin otomatis jika salah satu limit.</p>
        </div>
      </div>

      {isAllExhausted && (
        <div className="border-2 border-red-600 bg-red-100 p-4 flex gap-3">
          <AlertTriangle className="w-5 h-5 text-red-600 shrink-0 mt-0.5" />
          <div>
            <h4 className="text-sm font-black uppercase tracking-wide text-red-700">Semua key mencapai limit atau error.</h4>
            <p className="text-xs font-bold text-red-600 mt-1">Tambahkan API key baru agar AI Asisten dapat kembali bekerja.</p>
          </div>
        </div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="md:col-span-1 space-y-6">
          <div className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-5 flex flex-col gap-4">
            <div className="w-10 h-10 border-2 border-foreground bg-card flex items-center justify-center text-primary shrink-0">
              <Shield className="w-5 h-5" />
            </div>
            <h4 className="text-xs font-black uppercase tracking-wide text-foreground">Modality System</h4>
            <p className="text-xs font-medium text-muted-foreground leading-relaxed">
              Setiap key punya kemampuan (Text, Audio, Image). Sistem otomatis milih key yang tepat: Text buat chat, Audio buat voice, Image buat scan struk. Jika key pertama limit, otomatis beralih ke key berikutnya secara round-robin.
            </p>
            <p className="text-xs font-medium text-muted-foreground leading-relaxed">
              API Key dienkripsi dengan AES-256-GCM. Setiap user hanya bisa mengakses key miliknya sendiri.
            </p>
          </div>
        </div>

        <div className="md:col-span-2 space-y-4">
          <div className="flex items-center justify-between mb-2">
            <h3 className="text-xs font-black uppercase tracking-wide text-foreground flex items-center gap-2">
              <Key className="w-5 h-5 text-muted-foreground" />
              Daftar Key Tersimpan
            </h3>
            <Button onClick={() => setShowForm(true)} size="sm" className="text-xs cursor-pointer">
              <Plus className="w-3.5 h-3.5 mr-1" />
              Tambah Key Baru
            </Button>
          </div>

          {isLoading ? (
            <div className="space-y-3">
              {[1,2,3].map(i => (
                <div key={i} className="border-2 border-foreground bg-card shadow-[3px_3px_0px_0px] shadow-foreground p-4">
                  <Skeleton className="h-5 w-48 mb-2 border-2 border-foreground" />
                  <Skeleton className="h-4 w-64 border-2 border-foreground" />
                </div>
              ))}
            </div>
          ) : (
            <div className="space-y-3">
              {(keys || []).map((key: any) => (
                <div key={key.id} className="border-2 border-foreground bg-card shadow-[3px_3px_0px_0px] shadow-foreground p-4 flex flex-col sm:flex-row sm:items-center gap-4 hover:shadow-[4px_4px_0px_0px] hover:shadow-foreground transition-all">
                  <div className="hidden sm:flex text-muted-foreground">
                    <GripVertical className="w-5 h-5" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-3 mb-1 flex-wrap">
                      <span className="font-black text-sm text-foreground">{key.alias}</span>
                      <span className="text-xs font-bold bg-primary/10 text-primary px-2 py-0.5 border border-foreground uppercase">
                        {key.provider || "gemini"}
                      </span>
                      {key.model && (
                        <span className="text-[10px] font-mono font-bold bg-muted text-muted-foreground px-2 py-0.5 border border-foreground">
                          {key.model}
                        </span>
                      )}
                      <div className="flex gap-1">
                        {key.supports_text && <span className="text-[9px] font-bold px-1.5 py-0.5 border border-foreground bg-blue-50 text-blue-700">Text</span>}
                        {key.supports_audio && <span className="text-[9px] font-bold px-1.5 py-0.5 border border-foreground bg-purple-50 text-purple-700">Audio</span>}
                        {key.supports_vision && <span className="text-[9px] font-bold px-1.5 py-0.5 border border-foreground bg-green-50 text-green-700">Image</span>}
                      </div>
                      {getStatusBadge(key.status)}
                    </div>
                    <div className="font-mono text-xs font-bold text-muted-foreground bg-muted px-2 py-1 border border-foreground inline-block">
                      {key.key_preview || key.preview || key.key?.substring(0, 20) + "..."}
                    </div>
                  </div>
                  <div className="flex items-center gap-2 mt-3 sm:mt-0 pt-3 sm:pt-0 border-t-2 sm:border-0 border-foreground">
                    <Button variant="outline" size="sm" onClick={() => openEdit(key)} className="text-muted-foreground cursor-pointer">
                      <PenLine className="w-3.5 h-3.5" />
                    </Button>
                    <Button variant="outline" size="sm" onClick={() => handleTest(key.id)} disabled={testMutation.isPending} className="text-muted-foreground cursor-pointer">
                      <Play className="w-3.5 h-3.5 mr-1.5" />
                      Test
                    </Button>
                    <Button variant="outline" size="sm" onClick={() => { setConfirmDeleteId(key.id); setConfirmOpen(true); }} className="text-red-600 hover:text-red-700 hover:bg-red-100 cursor-pointer">
                      <Trash2 className="w-3.5 h-3.5" />
                    </Button>
                  </div>
                </div>
              ))}
              {(!keys || keys.length === 0) && (
                <div className="text-center p-8 border-2 border-dashed border-muted-foreground">
                  <Key className="w-8 h-8 text-muted-foreground mx-auto mb-3" />
                  <p className="text-sm font-bold text-muted-foreground">Belum ada API key.</p>
                </div>
              )}
            </div>
          )}

          <p className="text-xs font-bold text-muted-foreground mt-4">
            * Prioritas berdasarkan urutan. Key atas digunakan pertama. Jika limit, otomatis beralih ke key berikutnya secara round-robin.
          </p>
        </div>
      </div>

      <Dialog open={showForm} onOpenChange={setShowForm}>
        <DialogContent className="sm:max-w-md border-2 border-foreground shadow-[8px_8px_0px_0px] shadow-foreground p-0 gap-0">
          <DialogHeader className="border-b-2 border-foreground px-5 py-4">
            <div className="flex items-center justify-between">
              <DialogTitle className="text-xs font-black uppercase tracking-wide">Tambah Key Baru</DialogTitle>
              <button onClick={() => setShowForm(false)} className="p-1 hover:bg-muted border-2 border-foreground cursor-pointer">
                <X className="w-4 h-4" />
              </button>
            </div>
          </DialogHeader>
          <form onSubmit={handleDialogSubmit} className="p-5 space-y-4">
            <div className="space-y-2">
              <label className="text-xs font-bold uppercase tracking-wide text-foreground">Provider</label>
              <select
                value={provider}
                onChange={(e) => setProvider(e.target.value)}
                className="w-full border-2 border-foreground bg-background px-3 py-2.5 text-sm font-bold focus:shadow-[2px_2px_0px_0px] focus:shadow-foreground focus:outline-none"
              >
                {providerOptions.map((p: any) => (
                  <option key={p.name || p} value={p.name || p}>
                    {p.display_name || (p.charAt(0).toUpperCase() + p.slice(1))}
                  </option>
                ))}
              </select>
            </div>
            <div className="space-y-2">
              <label className="text-xs font-bold uppercase tracking-wide text-foreground">
                Model <span className="text-red-500">*</span>
              </label>
              <input
                type="text"
                value={model}
                onChange={(e) => setModel(e.target.value)}
                placeholder="Misal: gpt-4o-mini, qwen/qwen3.6-27b, gemini-2.0-flash"
                className="w-full border-2 border-foreground bg-background px-3 py-2.5 text-sm font-bold focus:shadow-[2px_2px_0px_0px] focus:shadow-foreground focus:outline-none placeholder:text-muted-foreground/70"
                required
              />
              <p className="text-[10px] font-bold text-muted-foreground">
                Tentukan model spesifik yang akan digunakan untuk key ini.
              </p>
            </div>
            <div className="space-y-2">
              <label className="text-xs font-bold uppercase tracking-wide text-foreground">Kemampuan <span className="text-red-500">*</span></label>
              <div className="flex flex-wrap gap-3 border-2 border-foreground bg-muted p-3">
                <label className="flex items-center gap-2 text-xs font-bold cursor-pointer">
                  <input type="checkbox" checked={supportsText} onChange={(e) => setSupportsText(e.target.checked)} className="w-4 h-4 accent-primary cursor-pointer" />
                  <span className={supportsText ? "text-foreground" : "text-muted-foreground"}>Text</span>
                </label>
                <label className="flex items-center gap-2 text-xs font-bold cursor-pointer">
                  <input type="checkbox" checked={supportsAudio} onChange={(e) => setSupportsAudio(e.target.checked)} className="w-4 h-4 accent-primary cursor-pointer" />
                  <span className={supportsAudio ? "text-foreground" : "text-muted-foreground"}>Audio</span>
                </label>
                <label className="flex items-center gap-2 text-xs font-bold cursor-pointer">
                  <input type="checkbox" checked={supportsVision} onChange={(e) => setSupportsVision(e.target.checked)} className="w-4 h-4 accent-primary cursor-pointer" />
                  <span className={supportsVision ? "text-foreground" : "text-muted-foreground"}>Image</span>
                </label>
              </div>
              <p className="text-[10px] font-bold text-muted-foreground">
                Centang sesuai kemampuan model: Text (chat), Audio (voice input), Image (scan struk).
              </p>
            </div>
            <div className="space-y-2">
              <label className="text-xs font-bold uppercase tracking-wide text-foreground">Nama Alias</label>
              <input
                type="text"
                value={alias}
                onChange={(e) => setAlias(e.target.value)}
                placeholder="Misal: Key Utama"
                className="w-full border-2 border-foreground bg-background px-3 py-2.5 text-sm font-bold focus:shadow-[2px_2px_0px_0px] focus:shadow-foreground focus:outline-none placeholder:text-muted-foreground/70"
                required
              />
            </div>
            <div className="space-y-2">
              <label className="text-xs font-bold uppercase tracking-wide text-foreground">API Key</label>
              <input
                type="password"
                value={keyValue}
                onChange={(e) => setKeyValue(e.target.value)}
                placeholder="Masukkan API key"
                className="w-full font-mono border-2 border-foreground bg-background px-3 py-2.5 text-sm font-bold focus:shadow-[2px_2px_0px_0px] focus:shadow-foreground focus:outline-none placeholder:text-muted-foreground/70"
                required
              />
            </div>
            <Button type="submit" disabled={createMutation.isPending} className="w-full cursor-pointer">
              {createMutation.isPending ? "Menyimpan..." : "Simpan"}
            </Button>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog open={!!editingKey} onOpenChange={(o) => { if (!o) setEditingKey(null); }}>
        <DialogContent className="sm:max-w-md border-2 border-foreground shadow-[8px_8px_0px_0px] shadow-foreground p-0 gap-0">
          <DialogHeader className="border-b-2 border-foreground px-5 py-4">
            <div className="flex items-center justify-between">
              <DialogTitle className="text-xs font-black uppercase tracking-wide">Edit Key</DialogTitle>
              <button onClick={() => setEditingKey(null)} className="p-1 hover:bg-muted border-2 border-foreground cursor-pointer">
                <X className="w-4 h-4" />
              </button>
            </div>
          </DialogHeader>
          <form onSubmit={handleEditSubmit} className="p-5 space-y-4">
            <div className="space-y-2">
              <label className="text-xs font-bold uppercase tracking-wide text-foreground">Provider</label>
              <select
                value={editProvider}
                onChange={(e) => setEditProvider(e.target.value)}
                className="w-full border-2 border-foreground bg-background px-3 py-2.5 text-sm font-bold focus:shadow-[2px_2px_0px_0px] focus:shadow-foreground focus:outline-none"
              >
                {providerOptions.map((p: any) => (
                  <option key={p.name || p} value={p.name || p}>
                    {p.display_name || (p.charAt(0).toUpperCase() + p.slice(1))}
                  </option>
                ))}
              </select>
            </div>
            <div className="space-y-2">
              <label className="text-xs font-bold uppercase tracking-wide text-foreground">
                Model <span className="text-red-500">*</span>
              </label>
              <input
                type="text"
                value={editModel}
                onChange={(e) => setEditModel(e.target.value)}
                placeholder="Misal: gpt-4o-mini, qwen/qwen3.6-27b, gemini-2.0-flash"
                className="w-full border-2 border-foreground bg-background px-3 py-2.5 text-sm font-bold focus:shadow-[2px_2px_0px_0px] focus:shadow-foreground focus:outline-none placeholder:text-muted-foreground/70"
                required
              />
            </div>
            <div className="space-y-2">
              <label className="text-xs font-bold uppercase tracking-wide text-foreground">Kemampuan <span className="text-red-500">*</span></label>
              <div className="flex flex-wrap gap-3 border-2 border-foreground bg-muted p-3">
                <label className="flex items-center gap-2 text-xs font-bold cursor-pointer">
                  <input type="checkbox" checked={editSupportsText} onChange={(e) => setEditSupportsText(e.target.checked)} className="w-4 h-4 accent-primary cursor-pointer" />
                  <span className={editSupportsText ? "text-foreground" : "text-muted-foreground"}>Text</span>
                </label>
                <label className="flex items-center gap-2 text-xs font-bold cursor-pointer">
                  <input type="checkbox" checked={editSupportsAudio} onChange={(e) => setEditSupportsAudio(e.target.checked)} className="w-4 h-4 accent-primary cursor-pointer" />
                  <span className={editSupportsAudio ? "text-foreground" : "text-muted-foreground"}>Audio</span>
                </label>
                <label className="flex items-center gap-2 text-xs font-bold cursor-pointer">
                  <input type="checkbox" checked={editSupportsVision} onChange={(e) => setEditSupportsVision(e.target.checked)} className="w-4 h-4 accent-primary cursor-pointer" />
                  <span className={editSupportsVision ? "text-foreground" : "text-muted-foreground"}>Image</span>
                </label>
              </div>
            </div>
            <div className="space-y-2">
              <label className="text-xs font-bold uppercase tracking-wide text-foreground">Nama Alias</label>
              <input
                type="text"
                value={editAlias}
                onChange={(e) => setEditAlias(e.target.value)}
                className="w-full border-2 border-foreground bg-background px-3 py-2.5 text-sm font-bold focus:shadow-[2px_2px_0px_0px] focus:shadow-foreground focus:outline-none"
                required
              />
            </div>
            <div className="space-y-2">
              <label className="text-xs font-bold uppercase tracking-wide text-foreground">API Key</label>
              <input
                type="password"
                value={editKeyValue}
                onChange={(e) => setEditKeyValue(e.target.value)}
                placeholder="Kosongkan jika tidak ingin mengubah key"
                className="w-full font-mono border-2 border-foreground bg-background px-3 py-2.5 text-sm font-bold focus:shadow-[2px_2px_0px_0px] focus:shadow-foreground focus:outline-none placeholder:text-muted-foreground/70"
              />
              <p className="text-[10px] font-bold text-muted-foreground">
                Isi hanya jika ingin mengganti API key. Kosongkan untuk tetap menggunakan key yang sama.
              </p>
            </div>
            <Button type="submit" disabled={updateMutation.isPending} className="w-full cursor-pointer">
              {updateMutation.isPending ? "Menyimpan..." : "Simpan Perubahan"}
            </Button>
          </form>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={confirmOpen}
        onOpenChange={setConfirmOpen}
        onConfirm={async () => {
          if (!confirmDeleteId) return;
          await deleteMutation.mutateAsync(confirmDeleteId);
          toast.success("Key berhasil dihapus");
          setConfirmOpen(false);
          setConfirmDeleteId(null);
        }}
        title="Hapus API Key"
        message="Yakin ingin menghapus key ini?"
        confirmText="Ya, Hapus"
        loading={deleteMutation.isPending}
      />
    </div>
  );
}
