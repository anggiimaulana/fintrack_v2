import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Send, Mic, Camera } from "lucide-react";
import { VoiceInput } from "./VoiceInput";
import { ReceiptUpload } from "./ReceiptUpload";

interface ChatInputProps {
  onSend: (message: string) => void;
  isSending: boolean;
  onReceiptResult?: (result: any) => void;
}

export function ChatInput({ onSend, isSending, onReceiptResult }: ChatInputProps) {
  const [text, setText] = useState("");

  const handleSend = () => {
    if (text.trim() && !isSending) {
      onSend(text.trim());
      setText("");
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  return (
    <div className="p-4 bg-card border-t-2 border-foreground">
      <div className="flex items-end gap-2">
        <div className="flex gap-1 pb-1">
          <VoiceInput onTranscript={(t) => setText((prev) => prev ? prev + " " + t : t)} disabled={isSending} />
          <ReceiptUpload onResult={(res) => {
            if (onReceiptResult) {
              onReceiptResult(res);
            } else {
              onSend(`Struk berhasil dicatat: Total Rp ${res?.total || 0}`);
            }
          }} disabled={isSending} />
        </div>
        <div className="flex-1 bg-background border-2 border-foreground overflow-hidden flex items-end relative min-h-[44px]">
          <textarea
            value={text}
            onChange={(e) => setText(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Ketik perintah atau pertanyaan..."
            className="w-full bg-transparent border-none focus:ring-0 resize-none py-3 px-4 max-h-32 min-h-[44px] font-medium placeholder:text-muted-foreground/70"
            rows={1}
            disabled={isSending}
          />
        </div>
        <div className="pb-1">
          <Button
            onClick={handleSend}
            disabled={!text.trim() || isSending}
            size="icon"
            className="rounded-none"
          >
            <Send className="w-4 h-4" />
          </Button>
        </div>
      </div>
    </div>
  );
}
