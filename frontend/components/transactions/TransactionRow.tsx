"use client";

import { Pencil, Trash2, Mic, Camera, Sparkles, User } from "lucide-react";

export interface TransactionData {
  id: string;
  type: "income" | "expense" | "savings";
  amount: number;
  category?: string;
  category_name?: string;
  subcategory?: string;
  description?: string;
  date?: string;
  transaction_date?: string;
  source?: "manual" | "ai" | "voice" | "receipt";
  expense_category_id?: string | null;
  income_category_id?: string | null;
  savings_category_id?: string | null;
}

interface TransactionRowProps {
  transaction: TransactionData;
}

export function TransactionRow({ transaction }: TransactionRowProps) {
  const isIncome = transaction.type === "income";
  const amountColor = isIncome ? "text-green-700" : "text-foreground";
  const prefix = isIncome ? "+" : "-";
  
  const formattedAmount = new Intl.NumberFormat("id-ID", {
    style: "currency",
    currency: "IDR",
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(transaction.amount);

  const getSourceBadge = (source?: string) => {
    switch(source) {
      case "ai": return <div className="flex items-center gap-1 px-1.5 py-0.5 border rounded-md border-foreground text-[10px] font-bold bg-primary/10 text-primary-strong"><Sparkles className="w-3 h-3" /> AI</div>;
      case "voice": return <div className="flex items-center gap-1 px-1.5 py-0.5 border rounded-md border-foreground text-[10px] font-bold bg-secondary text-secondary-foreground"><Mic className="w-3 h-3" /> Voice</div>;
      case "receipt": return <div className="flex items-center gap-1 px-1.5 py-0.5 border rounded-md border-foreground text-[10px] font-bold bg-amber-100 text-amber-700"><Camera className="w-3 h-3" /> Scan</div>;
      default: return null;
    }
  };

  return (
    <div className="flex flex-col sm:flex-row sm:items-center gap-4 p-4 hover:bg-muted transition-colors group">
      <div className={`hidden sm:flex w-10 h-10 border-2 rounded-md border-foreground items-center justify-center shrink-0 ${isIncome ? 'bg-green-100 text-green-700' : 'bg-neutral-100 text-neutral-700'}`}>
        <User className="w-5 h-5" />
      </div>
      
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 mb-1">
          <span className="font-bold text-sm text-foreground truncate">
            {transaction.description || transaction.category}
          </span>
          {getSourceBadge(transaction.source)}
        </div>
        <div className="flex items-center gap-2 text-xs font-bold text-muted-foreground">
          <span className="truncate max-w-[120px]">{transaction.category} {transaction.subcategory ? `- ${transaction.subcategory}` : ''}</span>
          <span>/</span>
          <span>{transaction.date}</span>
        </div>
      </div>
      
      <div className="flex items-center justify-between sm:justify-end gap-6 shrink-0">
        <span className={`font-mono font-black text-sm ${amountColor}`}>
          {prefix} {formattedAmount}
        </span>
        
        <div className="flex items-center gap-1 opacity-100 sm:opacity-0 group-hover:opacity-100 transition-opacity">
          <button className="p-1.5 text-muted-foreground hover:text-primary hover:bg-primary/10 border-2 rounded-md border-transparent hover:border-foreground cursor-pointer transition-all" title="Edit">
            <Pencil className="w-4 h-4" />
          </button>
          <button className="p-1.5 text-muted-foreground hover:text-red-700 hover:bg-red-100 border-2 rounded-md border-transparent hover:border-foreground cursor-pointer transition-all" title="Hapus">
            <Trash2 className="w-4 h-4" />
          </button>
        </div>
      </div>
    </div>
  );
}
