import { useState, useRef, useCallback } from "react";
import { Mic, Square } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";

interface SpeechRecognitionEvent {
  resultIndex: number;
  results: {
    length: number;
    [index: number]: {
      isFinal: boolean;
      [index: number]: { transcript: string };
    };
  };
}

interface SpeechRecognitionErrorEvent {
  error: string;
}

interface SpeechRecognition {
  lang: string;
  continuous: boolean;
  interimResults: boolean;
  maxAlternatives: number;
  onstart: () => void;
  onresult: (event: SpeechRecognitionEvent) => void;
  onerror: (event: SpeechRecognitionErrorEvent) => void;
  onend: () => void;
  start: () => void;
  stop: () => void;
  abort: () => void;
}

type WindowWithSpeech = Window & typeof globalThis & {
  SpeechRecognition?: new () => SpeechRecognition;
  webkitSpeechRecognition?: new () => SpeechRecognition;
};
interface VoiceInputProps {
  onTranscript: (text: string) => void;
  disabled?: boolean;
}

export function VoiceInput({ onTranscript, disabled }: VoiceInputProps) {
  const [isListening, setIsListening] = useState(false);
  const [isSupported] = useState(() => {
    if (typeof window === "undefined") return true;
    return !!((window as WindowWithSpeech).SpeechRecognition || (window as WindowWithSpeech).webkitSpeechRecognition);
  });
  const recognitionRef = useRef<SpeechRecognition | null>(null);
  const handleToggle = useCallback(() => {
    if (isListening && recognitionRef.current) {
      recognitionRef.current.stop();
      setIsListening(false);
      return;
    }

    const SpeechRecognition = (window as WindowWithSpeech).SpeechRecognition || (window as WindowWithSpeech).webkitSpeechRecognition;
    if (!SpeechRecognition) {
      toast.error("Browser tidak mendukung input suara. Gunakan Chrome atau Edge.");
      return;
    }

    const recognition = new SpeechRecognition();
    recognition.lang = "id-ID";
    recognition.continuous = false;
    recognition.interimResults = true;
    recognition.maxAlternatives = 1;

    recognition.onstart = () => setIsListening(true);

    recognition.onresult = (event: SpeechRecognitionEvent) => {
      let finalTranscript = "";
      for (let i = event.resultIndex; i < event.results.length; i++) {
        if (event.results[i].isFinal) {
          finalTranscript += event.results[i][0].transcript;
        }
      }
      if (finalTranscript) {
        onTranscript(finalTranscript);
      }
    };

    recognition.onerror = (event: SpeechRecognitionErrorEvent) => {
      setIsListening(false);
      if (event.error === "not-allowed") {
        toast.error("Izin mikrofon ditolak. Aktifkan di pengaturan browser Anda.");
      } else if (event.error === "no-speech") {
        // silent fail - user can try again
      } else if (event.error === "network") {
        toast.error("Gagal terhubung ke layanan suara. Periksa koneksi internet Anda dan coba lagi.", { duration: 3000 });
      } else if (event.error === "aborted") {
        // user or code aborted, no error needed
      } else {
        toast.error(`Error pengenalan suara: ${event.error}. Silakan coba lagi.`);
      }
    };

    recognition.onend = () => setIsListening(false);

    recognitionRef.current = recognition;
    recognition.start();
  }, [isListening, onTranscript]);

  if (!isSupported) return null;

  return (
    <Button
      variant="ghost"
      size="icon"
      className={`border-2 border-foreground shadow-[2px_2px_0px_0px] shadow-foreground active:shadow-none active:translate-x-[2px] active:translate-y-[2px] ${
        isListening ? "bg-red-100 text-red-600 border-red-600" : "text-muted-foreground"
      }`}
      title={isListening ? "Stop recording" : "Voice Input"}
      onClick={handleToggle}
      disabled={disabled && !isListening}
    >
      {isListening ? <Square className="w-4 h-4" /> : <Mic className="w-4 h-4" />}
    </Button>
  );
}
