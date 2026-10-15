import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Send, Square } from "lucide-react";
import { VoiceInput } from "./VoiceInput";
import { ReceiptUpload, type ReceiptResult } from "./ReceiptUpload";

interface ChatInputProps {
  onSend: (message: string) => void;
  onStop?: () => void;
  isSending: boolean;
  isStreaming?: boolean;
  onReceiptResult?: (result: ReceiptResult) => void;
}

export function ChatInput({ onSend, onStop, isSending, isStreaming, onReceiptResult }: ChatInputProps) {
  const [text, setText] = useState("");

  const handleSend = () => {
    if (text.trim() && !isSending && !isStreaming) {
      onSend(text.trim());
      setText("");
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      if (!isStreaming && !isSending) {
        handleSend();
      }
    }
  };

  const isBusy = isSending || Boolean(isStreaming);

  return (
    <div className="p-4 bg-card border-t-2 border-foreground">
      <div className="flex items-end gap-2">
        <div className="flex gap-1 pb-1">
          <VoiceInput onTranscript={(t: string) => setText((prev) => prev ? prev + " " + t : t)} disabled={isBusy} />
          <ReceiptUpload onResult={(res) => {
            if (onReceiptResult) {
              onReceiptResult(res);
            } else {
              onSend(`Struk berhasil dicatat: Total Rp ${res?.total || 0}`);
            }
          }} disabled={isBusy} />
        </div>
        <div className="flex-1 bg-background border-2 rounded-md border-foreground overflow-hidden flex items-end relative min-h-[44px]">
          <textarea
            value={text}
            onChange={(e) => setText(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={isStreaming ? "Sedang memproses respons..." : "Ketik perintah atau pertanyaan..."}
            className="w-full bg-transparent border-none focus:ring-0 resize-none py-3 px-4 max-h-32 min-h-[44px] font-medium placeholder:text-muted-foreground/70 disabled:opacity-60"
            rows={1}
            disabled={isBusy}
          />
        </div>
        <div className="pb-1">
          {isStreaming ? (
            <Button
              type="button"
              onClick={onStop}
              size="icon"
              aria-label="Hentikan respons"
              title="Hentikan respons"
              className="bg-red-600 hover:bg-red-700 text-white border-2 border-foreground shadow-[2px_2px_0px_0px] shadow-foreground active:translate-x-[1px] active:translate-y-[1px] cursor-pointer"
            >
              <Square className="w-3.5 h-3.5 fill-current" />
            </Button>
          ) : (
            <Button
              type="button"
              onClick={handleSend}
              disabled={!text.trim() || isSending}
              size="icon"
              aria-label="Kirim pesan"
              title="Kirim pesan"
              className="cursor-pointer disabled:cursor-not-allowed"
            >
              <Send className="w-4 h-4" />
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}
