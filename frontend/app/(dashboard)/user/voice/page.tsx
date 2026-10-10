"use client";

import { useState, useRef, useCallback, useEffect } from "react";
import { Mic, Square, Loader2, Volume2, Send, History, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { useMutation } from "@tanstack/react-query";
import api from "@/lib/api";

interface VoiceEntry {
  id: string;
  transcript: string;
  timestamp: Date;
}

export default function VoicePage() {
  const [isListening, setIsListening] = useState(false);
  const [isSupported, setIsSupported] = useState(true);
  const [transcript, setTranscript] = useState("");
  const [interim, setInterim] = useState("");
  const [history, setHistory] = useState<VoiceEntry[]>([]);
  const recognitionRef = useRef<any>(null);

  useEffect(() => {
    const SpeechRecognition = (window as any).SpeechRecognition || (window as any).webkitSpeechRecognition;
    if (!SpeechRecognition) setIsSupported(false);
  }, []);

  const chatMutation = useMutation({
    mutationFn: async (text: string) => {
      const res = await api.post("/ai/chat", { message: text });
      return res.data;
    },
  });

  const handleToggle = useCallback(() => {
    if (isListening && recognitionRef.current) {
      recognitionRef.current.stop();
      setIsListening(false);
      return;
    }

    const SpeechRecognition = (window as any).SpeechRecognition || (window as any).webkitSpeechRecognition;
    if (!SpeechRecognition) {
      toast.error("Browser tidak mendukung input suara. Gunakan Chrome atau Edge.");
      return;
    }

    const recognition = new SpeechRecognition();
    recognition.lang = "id-ID";
    recognition.continuous = true;
    recognition.interimResults = true;
    recognition.maxAlternatives = 1;

    recognition.onstart = () => {
      setIsListening(true);
      setTranscript("");
      setInterim("");
    };

    recognition.onresult = (event: any) => {
      let finalText = "";
      let interimText = "";
      for (let i = event.resultIndex; i < event.results.length; i++) {
        if (event.results[i].isFinal) {
          finalText += event.results[i][0].transcript;
        } else {
          interimText += event.results[i][0].transcript;
        }
      }
      if (finalText) setTranscript((prev) => prev + finalText);
      setInterim(interimText);
    };

    recognition.onerror = (event: any) => {
      setIsListening(false);
      if (event.error === "not-allowed") {
        toast.error("Izin mikrofon ditolak. Aktifkan di pengaturan browser Anda.");
      }
    };

    recognition.onend = () => setIsListening(false);

    recognitionRef.current = recognition;
    recognition.start();
  }, [isListening]);

  const saveTranscript = () => {
    if (!transcript.trim()) return;
    const entry: VoiceEntry = {
      id: Date.now().toString(),
      transcript: transcript.trim(),
      timestamp: new Date(),
    };
    setHistory((prev) => [entry, ...prev]);
    setTranscript("");
    setInterim("");
  };

  const clearHistory = () => setHistory([]);

  const deleteEntry = (id: string) => {
    setHistory((prev) => prev.filter((e) => e.id !== id));
  };

  return (
    <div className="max-w-3xl mx-auto space-y-6">
      <div>
        <h2 className="text-xl font-black uppercase tracking-wide text-foreground">Input Suara</h2>
        <p className="text-xs font-bold text-muted-foreground mt-1">
          Konversi suara ke teks dan kirim ke AI Assistant.
        </p>
      </div>

      {!isSupported ? (
        <div className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-8 text-center">
          <Volume2 className="w-10 h-10 text-muted-foreground mx-auto mb-3" />
          <p className="text-sm font-bold text-muted-foreground">
            Browser Anda tidak mendukung input suara.
          </p>
          <p className="text-xs font-bold text-muted-foreground mt-1">
            Gunakan Chrome, Edge, atau Safari terbaru.
          </p>
        </div>
      ) : (
        <div className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-6 text-center">
          <button
            onClick={handleToggle}
            className={`w-24 h-24 mx-auto border-2 border-foreground flex items-center justify-center transition-all cursor-pointer ${
              isListening
                ? "bg-red-100 text-red-600 border-red-600 shadow-[4px_4px_0px_0px] shadow-red-600 animate-pulse scale-105"
                : "bg-background text-foreground shadow-[4px_4px_0px_0px] shadow-foreground hover:bg-muted"
            }`}
          >
            {isListening ? <Square className="w-8 h-8" /> : <Mic className="w-8 h-8" />}
          </button>
          <p className="text-xs font-bold text-muted-foreground mt-3">
            {isListening ? "Klik untuk berhenti merekam" : "Klik untuk mulai berbicara"}
          </p>

          {(transcript || interim) && (
            <div className="mt-6">
              <div className="border-2 border-foreground bg-background p-4 min-h-[80px] text-left">
                {transcript && (
                  <p className="text-sm font-bold text-foreground">{transcript}</p>
                )}
                {interim && (
                  <p className="text-sm font-bold text-muted-foreground/60">{interim}</p>
                )}
                {!transcript && !interim && (
                  <p className="text-sm font-bold text-muted-foreground/40 italic">
                    Bicaralah...
                  </p>
                )}
              </div>
              <div className="flex justify-end gap-3 mt-4">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => { setTranscript(""); setInterim(""); }}
                  disabled={!transcript}
                >
                  <Trash2 className="w-4 h-4 mr-1" /> Hapus
                </Button>
                <Button size="sm" onClick={saveTranscript} disabled={!transcript.trim()}>
                  <Send className="w-4 h-4 mr-2" /> Simpan Teks
                </Button>
              </div>
            </div>
          )}
        </div>
      )}

      {history.length > 0 && (
        <div className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground">
          <div className="flex items-center justify-between px-4 py-3 border-b-2 border-foreground bg-muted">
            <h3 className="text-xs font-black uppercase tracking-wide text-foreground">
              <History className="w-4 h-4 inline mr-2" />
              Riwayat ({history.length})
            </h3>
            <Button variant="ghost" size="sm" className="text-red-500 text-xs" onClick={clearHistory}>
              <Trash2 className="w-3.5 h-3.5 mr-1" /> Hapus Semua
            </Button>
          </div>
          <div className="divide-y-2 divide-foreground max-h-[400px] overflow-y-auto">
            {history.map((entry) => (
              <div key={entry.id} className="p-4 flex items-start gap-3 group hover:bg-muted/50 transition-colors">
                <div className="w-8 h-8 border-2 border-foreground bg-primary/10 flex items-center justify-center shrink-0 mt-0.5">
                  <Volume2 className="w-4 h-4 text-primary" />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-bold text-foreground">{entry.transcript}</p>
                  <p className="text-xs font-bold text-muted-foreground mt-1">
                    {entry.timestamp.toLocaleDateString("id-ID", {
                      day: "numeric", month: "short", year: "numeric",
                      hour: "2-digit", minute: "2-digit",
                    })}
                  </p>
                </div>
                <div className="flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity shrink-0">
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() => {
                      chatMutation.mutate(entry.transcript);
                    }}
                    disabled={chatMutation.isPending}
                    title="Kirim ke AI"
                  >
                    <Send className="w-4 h-4" />
                  </Button>
                  <Button
                    size="sm"
                    variant="ghost"
                    className="text-red-600"
                    onClick={() => deleteEntry(entry.id)}
                  >
                    <Trash2 className="w-4 h-4" />
                  </Button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {chatMutation.data && (
        <div className="border-2 border-green-600 bg-green-50 p-4">
          <p className="text-xs font-bold text-green-700">Terkirim ke AI Assistant!</p>
        </div>
      )}
    </div>
  );
}
