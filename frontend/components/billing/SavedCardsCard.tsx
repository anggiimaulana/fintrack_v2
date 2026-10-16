"use client";

import * as React from "react";
import { CreditCard, Trash2, CheckCircle2, ShieldCheck, AlertCircle } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { ConfirmDialog } from "@/components/shared/ConfirmDialog";
import {
  useMyCards,
  useDeleteCard,
  useSetDefaultCard,
  type SavedCard,
} from "@/hooks/use-billing";

interface SavedCardsCardProps {
  className?: string;
}

export function SavedCardsCard({ className }: SavedCardsCardProps) {
  const { data: cards, isLoading, isError } = useMyCards();
  const deleteCard = useDeleteCard();
  const setDefaultCard = useSetDefaultCard();

  const [deleteTarget, setDeleteTarget] = React.useState<SavedCard | null>(null);

  const defaultCard = React.useMemo(() => {
    return cards?.find((c) => c.is_default) ?? cards?.[0];
  }, [cards]);

  const handleDelete = async () => {
    if (!deleteTarget) return;
    await deleteCard.mutateAsync(deleteTarget.id);
    setDeleteTarget(null);
  };

  const handleSetDefault = async (cardId: string) => {
    await setDefaultCard.mutateAsync(cardId);
  };

  return (
    <>
      <Card className={`border-2 rounded-md border-foreground shadow-[4px_4px_0px_0px] shadow-foreground ${className ?? ""}`}>
        <CardHeader>
          <div className="flex flex-wrap items-center justify-between gap-2">
            <CardTitle className="flex items-center gap-2 text-lg font-black">
              <CreditCard className="h-5 w-5" aria-hidden="true" />
              Kartu Tersimpan &amp; Perpanjangan Otomatis
            </CardTitle>
            {defaultCard ? (
              <Badge variant="default" className="gap-1 bg-emerald-600 hover:bg-emerald-600 text-white">
                <ShieldCheck className="h-3.5 w-3.5" aria-hidden="true" />
                Auto-Charge Aktif
              </Badge>
            ) : (
              <Badge variant="outline" className="text-muted-foreground">
                Perpanjangan Manual
              </Badge>
            )}
          </div>
          <CardDescription>
            Kartu tersimpan digunakan untuk perpanjangan langganan otomatis saat siklus berakhir tanpa perlu redirect.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {isLoading ? (
            <div className="space-y-2">
              <Skeleton className="h-14 w-full" />
              <Skeleton className="h-14 w-full" />
            </div>
          ) : isError ? (
            <div className="flex items-center gap-2 text-xs font-bold text-red-600">
              <AlertCircle className="h-4 w-4" />
              Gagal memuat kartu tersimpan.
            </div>
          ) : !cards || cards.length === 0 ? (
            <div className="rounded-md border-2 border-dashed border-muted-foreground/30 p-4 text-center">
              <CreditCard className="mx-auto h-8 w-8 text-muted-foreground/50 mb-2" />
              <p className="text-xs font-bold text-foreground">Belum ada kartu tersimpan</p>
              <p className="text-xs text-muted-foreground mt-1 max-w-md mx-auto">
                Saat Anda melakukan pembayaran dengan kartu kredit/debit via Midtrans, kartu akan otomatis tersimpan dengan aman untuk pembayaran 1-klik dan perpanjangan otomatis.
              </p>
            </div>
          ) : (
            <div className="space-y-2.5">
              {cards.map((card) => {
                const isDefault = card.is_default;
                return (
                  <div
                    key={card.id}
                    className={`flex flex-wrap items-center justify-between gap-3 p-3 rounded-md border-2 ${
                      isDefault
                        ? "border-foreground bg-primary/10 shadow-[2px_2px_0px_0px] shadow-foreground"
                        : "border-border bg-card"
                    }`}
                  >
                    <div className="flex items-center gap-3">
                      <div className="h-9 w-12 rounded border border-foreground/30 bg-muted flex items-center justify-center font-mono text-xs font-black uppercase text-foreground">
                        {card.bank || "CARD"}
                      </div>
                      <div>
                        <div className="flex items-center gap-2">
                          <span className="font-mono text-sm font-bold tracking-wider text-foreground">
                            •••• {card.masked_card?.slice(-4) || card.masked_card}
                          </span>
                          {card.card_type && (
                            <Badge variant="outline" className="text-[10px] font-mono uppercase py-0 px-1">
                              {card.card_type}
                            </Badge>
                          )}
                          {isDefault && (
                            <Badge variant="default" className="text-[10px] font-bold py-0 px-1.5 gap-1">
                              <CheckCircle2 className="h-2.5 w-2.5" /> Utama
                            </Badge>
                          )}
                        </div>
                        <p className="text-[11px] text-muted-foreground font-medium">
                          Gateway: {card.gateway}
                        </p>
                      </div>
                    </div>

                    <div className="flex items-center gap-2">
                      {!isDefault && (
                        <Button
                          variant="outline"
                          size="sm"
                          className="text-xs font-bold cursor-pointer"
                          disabled={setDefaultCard.isPending}
                          onClick={() => handleSetDefault(card.id)}
                        >
                          Jadikan Utama
                        </Button>
                      )}
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8 text-muted-foreground hover:text-red-600 cursor-pointer"
                        disabled={deleteCard.isPending}
                        onClick={() => setDeleteTarget(card)}
                        aria-label={`Hapus kartu ${card.masked_card}`}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </CardContent>
      </Card>

      <ConfirmDialog
        open={deleteTarget !== null}
        onOpenChange={(o) => !o && setDeleteTarget(null)}
        onConfirm={handleDelete}
        title="Hapus Kartu Tersimpan"
        message={`Apakah Anda yakin ingin menghapus kartu •••• ${deleteTarget?.masked_card?.slice(-4) || ""}? Jika dihapus, perpanjangan otomatis tidak dapat menggunakan kartu ini.`}
        confirmText="Hapus Kartu"
        variant="danger"
        loading={deleteCard.isPending}
      />
    </>
  );
}
