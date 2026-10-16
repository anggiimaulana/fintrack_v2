"use client"

import * as React from "react"
import { Trash2, ShieldAlert, PowerOff, Loader2 } from "lucide-react"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

export interface ConfirmActionModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  actionType: "delete" | "suspend" | "deactivate"
  title: string
  description: string
  entityName?: string
  confirmLabel?: string
  cancelLabel?: string
  isPending?: boolean
  onConfirm: () => void | Promise<void>
}

export function ConfirmActionModal({
  open,
  onOpenChange,
  actionType,
  title,
  description,
  entityName,
  confirmLabel,
  cancelLabel = "Batal",
  isPending = false,
  onConfirm,
}: ConfirmActionModalProps) {
  const getActionConfig = () => {
    switch (actionType) {
      case "delete":
        return {
          icon: <Trash2 className="w-5 h-5 text-destructive shrink-0" />,
          defaultConfirmLabel: "Hapus Data",
          btnClass: "bg-destructive text-destructive-foreground hover:bg-destructive/90",
        }
      case "suspend":
        return {
          icon: <ShieldAlert className="w-5 h-5 text-amber-500 shrink-0" />,
          defaultConfirmLabel: "Suspend Akun",
          btnClass: "bg-amber-600 text-white hover:bg-amber-700 dark:bg-amber-600 dark:hover:bg-amber-700",
        }
      case "deactivate":
        return {
          icon: <PowerOff className="w-5 h-5 text-muted-foreground shrink-0" />,
          defaultConfirmLabel: "Nonaktifkan",
          btnClass: "bg-secondary text-secondary-foreground hover:bg-secondary/80",
        }
    }
  }

  const { icon, defaultConfirmLabel, btnClass } = getActionConfig()
  const effectiveConfirmLabel = confirmLabel || defaultConfirmLabel

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md border-2 border-foreground shadow-[4px_4px_0px_0px] shadow-foreground">
        <DialogHeader className="gap-2">
          <div className="flex items-center gap-3">
            <div className="p-2 rounded-lg border-2 border-foreground/20 bg-muted shrink-0">
              {icon}
            </div>
            <DialogTitle className="text-lg font-black">{title}</DialogTitle>
          </div>
          <DialogDescription className="text-sm font-medium text-foreground/80 leading-relaxed pt-1">
            {description}
            {entityName && (
              <span className="block font-bold text-foreground mt-2 bg-muted/60 p-2 rounded border border-border">
                {entityName}
              </span>
            )}
          </DialogDescription>
        </DialogHeader>

        <DialogFooter className="gap-2 sm:gap-0 mt-4">
          <Button
            type="button"
            variant="outline"
            disabled={isPending}
            onClick={() => onOpenChange(false)}
            className="font-bold border-2"
          >
            {cancelLabel}
          </Button>
          <Button
            type="button"
            disabled={isPending}
            onClick={onConfirm}
            className={cn("font-bold border-2 border-foreground shadow-[2px_2px_0px_0px] shadow-foreground transition-all", btnClass)}
          >
            {isPending ? (
              <>
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                Memproses...
              </>
            ) : (
              effectiveConfirmLabel
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
