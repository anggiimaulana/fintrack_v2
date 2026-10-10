import { useState, useRef } from "react";
import { Camera, X, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { toast } from "sonner";
import api from "@/lib/api";

async function compressImage(file: File): Promise<File> {
  return new Promise((resolve) => {
    const img = new Image();
    img.onload = () => {
      let w = img.width, h = img.height;
      const maxDim = 1920;
      if (w > maxDim || h > maxDim) {
        if (w > h) { h = Math.round(h * maxDim / w); w = maxDim; }
        else { w = Math.round(w * maxDim / h); h = maxDim; }
      }
      const canvas = document.createElement("canvas");
      canvas.width = w; canvas.height = h;
      const ctx = canvas.getContext("2d");
      if (!ctx) { resolve(file); return; }
      ctx.drawImage(img, 0, 0, w, h);
      canvas.toBlob((blob) => {
        if (!blob) { resolve(file); return; }
        resolve(new File([blob], file.name.replace(/\.[^.]+$/, ".webp"), { type: "image/webp" }));
      }, "image/webp", 0.8);
    };
    img.onerror = () => resolve(file);
    const reader = new FileReader();
    reader.onload = (e) => { img.src = e.target?.result as string; };
    reader.onerror = () => resolve(file);
    reader.readAsDataURL(file);
  });
}

export interface ReceiptResult {
  total?: number;
  type?: string;
  merchant?: string;
  category?: string;
  date?: string;
  [key: string]: unknown;
}

interface ReceiptUploadProps {
  onResult: (result: ReceiptResult) => void;
  disabled?: boolean;
}

export function ReceiptUpload({ onResult, disabled }: ReceiptUploadProps) {
  const [file, setFile] = useState<File | null>(null);
  const [preview, setPreview] = useState<string | null>(null);
  const [isAnalyzing, setIsAnalyzing] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const validTypes = ["image/jpeg", "image/png", "image/webp", "image/heic"];
  const maxSize = 2 * 1024 * 1024;

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0];
    if (!f) return;

    if (!validTypes.includes(f.type)) {
      toast.error("Format file tidak didukung. Gunakan JPG, PNG, WEBP, atau HEIC.");
      return;
    }
    if (f.size > maxSize) {
      toast.error("Ukuran file maksimal 2MB. Kompres dulu gambar Anda.");
      return;
    }

    const compressed = await compressImage(f);
    setFile(compressed);
    setPreview(URL.createObjectURL(compressed));
  };

  const handleCancel = () => {
    setFile(null);
    setPreview(null);
    if (fileInputRef.current) fileInputRef.current.value = "";
  };

  const handleAnalyze = async () => {
    if (!file) return;
    setIsAnalyzing(true);
    try {
      const formData = new FormData();
      formData.append("image", file);
      const res = await api.post("/ai/receipt", formData, {
        timeout: 30000,
      });
      onResult(res.data?.data || res.data);
      handleCancel();
    } catch {
      toast.error("Gagal menganalisis struk. Pastikan file terbaca dengan baik.");
    } finally {
      setIsAnalyzing(false);
    }
  };

  return (
    <>
      <input
        type="file"
        accept="image/jpeg,image/png,image/webp,image/heic"
        className="hidden"
        ref={fileInputRef}
        onChange={handleFileChange}
      />
      <Button
        variant="ghost"
        size="icon"
        className="text-muted-foreground hover:text-foreground"
        title="Upload Struk"
        onClick={() => fileInputRef.current?.click()}
        disabled={disabled}
      >
        <Camera className="w-5 h-5" />
      </Button>

      {preview && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
          <div className="border-2 border-foreground bg-card shadow-[6px_6px_0px_0px] shadow-foreground w-full max-w-md mx-4">
            <div className="flex items-center justify-between px-5 py-4 border-b-2 border-foreground">
              <h3 className="text-sm font-black uppercase tracking-wide text-foreground">Preview Foto Struk</h3>
              <button onClick={handleCancel} className="text-muted-foreground hover:text-foreground cursor-pointer" disabled={isAnalyzing}>
                <X className="w-5 h-5" />
              </button>
            </div>
            <div className="p-5">
              <img src={preview} alt="Preview struk" className="w-full max-h-[300px] object-contain border-2 border-foreground bg-muted" />
              <p className="text-xs font-bold text-muted-foreground mt-3">{file?.name} ({(file!.size / 1024 / 1024).toFixed(1)} MB)</p>
            </div>
            <div className="flex justify-end gap-3 px-5 py-4 border-t-2 border-foreground bg-muted">
              <Button variant="outline" onClick={handleCancel} disabled={isAnalyzing}>
                Batal
              </Button>
              <Button onClick={handleAnalyze} disabled={isAnalyzing}>
                {isAnalyzing ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Menganalisis...
                  </>
                ) : (
                  "Konfirmasi dan Kirim"
                )}
              </Button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
