"use client";

import { Suspense, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { Bell, CheckCheck, Eye, MailOpen, Trash2, Inbox, MailQuestion, Mail } from "lucide-react";
import { toast } from "sonner";
import type { RowSelectionState, ColumnDef } from "@tanstack/react-table";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { ConfirmDialog } from "@/components/shared/ConfirmDialog";
import { DataTable } from "@/components/data-table/data-table";
import { DataTablePagination } from "@/components/data-table/data-table-pagination";
import { DataTableToolbar } from "@/components/data-table/data-table-toolbar";
import { notificationColumns, eventMeta, type NotificationColumnsOptions } from "@/components/notifications/NotificationColumns";
import { useDataTableUrlState, type UseDataTableUrlState } from "@/hooks/use-data-table-url-state";
import {
  useNotifications,
  useNotificationStats,
  useNotificationsRealtime,
  useMarkNotificationRead,
  useMarkNotificationsRead,
  useMarkAllNotificationsRead,
  useDeleteNotification,
  useDeleteNotifications,
  type NotificationItem,
} from "@/hooks/useNotifications";

function NotificationsContent() {
  const router = useRouter();
  // List state lives in the URL (survives refresh, shareable) — same contract
  // as the transactions table.
  const table = useDataTableUrlState({ perPage: 20, sort: "created_at", order: "desc" });
  const status = table.filters.status || "all";

  // Live cache patching: created/read/deleted events patch only the affected
  // rows instead of refetching the table.
  useNotificationsRealtime();

  const { data, isLoading } = useNotifications({
    page: table.page,
    perPage: table.perPage,
    sort: table.sort || "created_at",
    order: table.order,
    search: table.q,
    status: status as "all" | "read" | "unread",
  });
  const { data: stats } = useNotificationStats();

  const markRead = useMarkNotificationRead();
  const markManyRead = useMarkNotificationsRead();
  const markAllRead = useMarkAllNotificationsRead();
  const deleteOne = useDeleteNotification();
  const deleteMany = useDeleteNotifications();

  const [detail, setDetail] = useState<NotificationItem | null>(null);
  const [rowSelection, setRowSelection] = useState<RowSelectionState>({});
  const [confirmDelete, setConfirmDelete] = useState<{ ids: string[] } | null>(null);

  const items = data?.data ?? [];
  const selectedIds = useMemo(() => Object.keys(rowSelection).filter((k) => rowSelection[k]), [rowSelection]);
  const allOnPageSelected = items.length > 0 && selectedIds.length === items.length;

  const openDetail = (n: NotificationItem) => {
    setDetail(n);
    // Opening counts as reading: mark read (optimistically patched).
    if (!n.read_at) markRead.mutate(n.id);
  };

  const handleMarkSelected = async () => {
    if (selectedIds.length === 0) return;
    await markManyRead.mutateAsync(selectedIds);
    setRowSelection({});
    toast.success(`${selectedIds.length} notifikasi ditandai dibaca.`);
  };

  const handleMarkAll = async () => {
    await markAllRead.mutateAsync();
    setRowSelection({});
    toast.success("Semua notifikasi ditandai dibaca.");
  };

  const handleDeleteSelected = () => {
    if (selectedIds.length === 0) return;
    setConfirmDelete({ ids: selectedIds });
  };

  const confirmDeleteAction = async () => {
    if (!confirmDelete) return;
    const ids = confirmDelete.ids;
    if (ids.length === 1) await deleteOne.mutateAsync(ids[0]);
    else await deleteMany.mutateAsync(ids);
    setConfirmDelete(null);
    setRowSelection({});
    toast.success(ids.length === 1 ? "Notifikasi dihapus." : `${ids.length} notifikasi dihapus.`);
  };

  const toggleSelectAllOnPage = () => {
    if (allOnPageSelected) {
      setRowSelection({});
    } else {
      const next: RowSelectionState = {};
      for (const n of items) next[n.id] = true;
      setRowSelection(next);
    }
  };

  const columnOptions: NotificationColumnsOptions = {
    page: table.page,
    perPage: table.perPage,
    onDetail: openDetail,
    onMarkRead: (n) => markRead.mutate(n.id),
    onDelete: (id) => setConfirmDelete({ ids: [id] }),
  };

  const isDeleting = deleteOne.isPending || deleteMany.isPending;

  return (
    <div className="flex flex-col gap-6 w-full max-w-7xl mx-auto">
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h2 className="text-xl font-black uppercase tracking-wide text-foreground flex items-center gap-2">
            <Bell className="h-5 w-5 text-primary" /> Notifikasi
          </h2>
          <p className="text-xs font-bold text-muted-foreground mt-1">
            Semua pemberitahuan akun Anda — pembayaran, paket, kuota, dan lainnya.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            className="cursor-pointer disabled:cursor-not-allowed"
            onClick={handleMarkAll}
            disabled={markAllRead.isPending || (stats?.unread ?? 0) === 0}
          >
            <CheckCheck className="w-4 h-4 mr-2" />
            Tandai Semua Dibaca
          </Button>
        </div>
      </div>

      {/* Statistics */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <div className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-4 flex flex-col justify-between gap-2">
          <div className="flex items-center justify-between">
            <span className="text-xs font-black uppercase tracking-wide text-muted-foreground">Total</span>
            <div className="w-8 h-8 border-2 border-foreground flex items-center justify-center bg-muted text-foreground">
              <Inbox className="w-4 h-4" />
            </div>
          </div>
          <div className="text-lg sm:text-xl font-black font-mono text-foreground">{stats?.total ?? 0}</div>
        </div>

        <div className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-4 flex flex-col justify-between gap-2">
          <div className="flex items-center justify-between">
            <span className="text-xs font-black uppercase tracking-wide text-muted-foreground">Belum Dibaca</span>
            <div className="w-8 h-8 border-2 border-foreground flex items-center justify-center bg-amber-100 text-amber-800">
              <MailQuestion className="w-4 h-4" />
            </div>
          </div>
          <div className="text-lg sm:text-xl font-black font-mono text-amber-700">{stats?.unread ?? 0}</div>
        </div>

        <div className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-4 flex flex-col justify-between gap-2">
          <div className="flex items-center justify-between">
            <span className="text-xs font-black uppercase tracking-wide text-muted-foreground">Sudah Dibaca</span>
            <div className="w-8 h-8 border-2 border-foreground flex items-center justify-center bg-emerald-100 text-emerald-800">
              <Mail className="w-4 h-4" />
            </div>
          </div>
          <div className="text-lg sm:text-xl font-black font-mono text-foreground">{stats?.read ?? 0}</div>
        </div>
      </div>

      <div className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground flex flex-col overflow-hidden">
        {/* Filter tabs + toolbar */}
        <div className="p-4 border-b-2 border-foreground flex flex-col md:flex-row gap-4 justify-between items-center bg-muted">
          <div className="flex border-2 border-foreground p-1 bg-background w-full md:w-auto">
            {[
              { key: "all", label: "Semua" },
              { key: "unread", label: "Belum Dibaca" },
              { key: "read", label: "Sudah Dibaca" },
            ].map((f) => (
              <button
                key={f.key}
                onClick={() => table.setFilter("status", f.key === "all" ? "" : f.key)}
                className={`flex-1 md:flex-none px-4 py-1.5 text-xs font-bold cursor-pointer transition-colors ${
                  status === f.key
                    ? "bg-card text-foreground shadow-[2px_2px_0px_0px] shadow-foreground border-x-2 border-foreground"
                    : "text-muted-foreground hover:text-foreground"
                }`}
              >
                {f.label}
              </button>
            ))}
          </div>
          <DataTableToolbar
            search={table.q}
            onSearch={(q) => table.setSearch(q)}
            placeholder="Cari notifikasi..."
          />
        </div>

        {/* Bulk action bar (only when rows are selected) */}
        {selectedIds.length > 0 && (
          <div className="px-4 py-2.5 border-b-2 border-foreground bg-primary/5 flex flex-wrap items-center gap-3">
            <span className="text-xs font-black uppercase tracking-wide text-foreground">
              {selectedIds.length} dipilih
            </span>
            <Button size="sm" variant="outline" className="cursor-pointer" onClick={handleMarkSelected} disabled={markManyRead.isPending}>
              <MailOpen className="w-3.5 h-3.5 mr-1.5" /> Tandai Dibaca
            </Button>
            <Button
              size="sm"
              className="bg-red-600 hover:bg-red-700 text-white cursor-pointer"
              onClick={handleDeleteSelected}
              disabled={isDeleting}
            >
              <Trash2 className="w-3.5 h-3.5 mr-1.5" /> Hapus
            </Button>
            <button
              type="button"
              onClick={() => setRowSelection({})}
              className="ml-auto text-xs font-bold text-muted-foreground hover:text-foreground cursor-pointer"
            >
              Bersihkan pilihan
            </button>
          </div>
        )}

        {/* Select-all-on-page toggle */}
        {items.length > 0 && (
          <div className="px-4 py-2 border-b-2 border-foreground/20 flex items-center gap-2">
            <input
              type="checkbox"
              id="select-all-page"
              checked={allOnPageSelected}
              onChange={toggleSelectAllOnPage}
              className="h-4 w-4 cursor-pointer accent-primary"
              aria-label="Pilih semua di halaman ini"
            />
            <label htmlFor="select-all-page" className="text-xs font-bold text-muted-foreground cursor-pointer">
              Pilih semua di halaman ini
            </label>
          </div>
        )}

        <NotificationTable
          table={table}
          items={items}
          isLoading={isLoading}
          rowSelection={rowSelection}
          onRowSelectionChange={setRowSelection}
          columnOptions={columnOptions}
        />

        <DataTablePagination
          page={table.page}
          perPage={table.perPage}
          total={data?.total ?? 0}
          totalPages={Math.max(1, Math.ceil((data?.total ?? 0) / table.perPage))}
          onPageChange={table.setPage}
          onPerPageChange={table.setPerPage}
        />
      </div>

      {/* Detail dialog */}
      <Dialog open={!!detail} onOpenChange={(open) => { if (!open) setDetail(null); }}>
        <DialogContent className="sm:max-w-lg border-2 border-foreground shadow-[6px_6px_0px_0px] shadow-foreground">
          <DialogHeader className="pr-8">
            <DialogTitle className="font-black uppercase tracking-wide flex items-center gap-2.5">
              <span>Detail Notifikasi</span>
              {detail && (() => {
                const meta = eventMeta(detail.event);
                const Icon = meta.icon;
                return (
                  <span className={`inline-flex items-center gap-1 border-2 px-2 py-0.5 text-[10px] font-black uppercase ${meta.className}`}>
                    <Icon className="h-3 w-3" /> {meta.label}
                  </span>
                );
              })()}
            </DialogTitle>
          </DialogHeader>

          {detail && (
            <div className="py-2 space-y-4">
              <div className="border-2 border-foreground bg-muted p-4 space-y-1">
                <p className="font-black text-base text-foreground">{detail.title}</p>
                <p className="text-sm font-medium text-muted-foreground whitespace-pre-wrap">{detail.body}</p>
              </div>

              <div className="grid grid-cols-2 gap-3 text-xs">
                <div className="border-2 border-foreground p-3 bg-card space-y-1">
                  <p className="font-bold uppercase tracking-wider text-muted-foreground text-[10px]">Waktu</p>
                  <p className="font-black text-sm text-foreground">
                    {new Date(detail.created_at).toLocaleString("id-ID", { dateStyle: "medium", timeStyle: "short" })}
                  </p>
                </div>
                <div className="border-2 border-foreground p-3 bg-card space-y-1">
                  <p className="font-bold uppercase tracking-wider text-muted-foreground text-[10px]">Status</p>
                  <p className="font-black text-sm text-foreground">{detail.read_at ? "Sudah dibaca" : "Belum dibaca"}</p>
                </div>
              </div>

              <div className="flex gap-2 pt-2">
                <Button variant="outline" className="flex-1 cursor-pointer" onClick={() => setDetail(null)}>
                  Tutup
                </Button>
                {detail.href && (
                  <Button
                    className="flex-1 cursor-pointer"
                    onClick={() => {
                      const href = detail.href;
                      setDetail(null);
                      router.push(href);
                    }}
                  >
                    <Eye className="w-3.5 h-3.5 mr-1.5" /> Buka Tautan
                  </Button>
                )}
                <Button
                  className="border-2 border-red-600 bg-red-600 hover:bg-red-700 text-white cursor-pointer"
                  onClick={() => {
                    const id = detail.id;
                    setDetail(null);
                    setConfirmDelete({ ids: [id] });
                  }}
                >
                  <Trash2 className="w-3.5 h-3.5 mr-1.5" /> Hapus
                </Button>
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={!!confirmDelete}
        onOpenChange={(o) => { if (!o) setConfirmDelete(null); }}
        onConfirm={confirmDeleteAction}
        title="Hapus Notifikasi"
        message={
          confirmDelete && confirmDelete.ids.length > 1
            ? `Yakin ingin menghapus ${confirmDelete.ids.length} notifikasi ini? Tindakan ini tidak dapat dibatalkan.`
            : "Yakin ingin menghapus notifikasi ini? Tindakan ini tidak dapat dibatalkan."
        }
        confirmText="Ya, Hapus"
        loading={isDeleting}
      />
    </div>
  );
}

/**
 * The table + its per-row checkbox column. Split out so the checkbox column can
 * be built with the row-selection handlers in scope (columns are memoised).
 */
function NotificationTable({
  table,
  items,
  isLoading,
  rowSelection,
  onRowSelectionChange,
  columnOptions,
}: {
  table: UseDataTableUrlState;
  items: NotificationItem[];
  isLoading: boolean;
  rowSelection: RowSelectionState;
  onRowSelectionChange: (s: RowSelectionState) => void;
  columnOptions: NotificationColumnsOptions;
}) {
  const columns = useMemo<ColumnDef<NotificationItem, unknown>[]>(() => {
    const selectColumn: ColumnDef<NotificationItem, unknown> = {
      id: "select",
      header: () => <span className="sr-only">Pilih</span>,
      enableSorting: false,
      cell: ({ row }) => (
        <input
          type="checkbox"
          checked={row.getIsSelected()}
          onChange={row.getToggleSelectedHandler()}
          className="h-4 w-4 cursor-pointer accent-primary"
          aria-label="Pilih notifikasi"
        />
      ),
    };
    return [selectColumn, ...notificationColumns(columnOptions)];
  }, [columnOptions]);

  return (
    <DataTable
      columns={columns}
      data={items}
      isLoading={isLoading}
      emptyMessage="Tidak ada notifikasi ditemukan."
      sort={table.sort}
      order={table.order}
      onSort={table.setSort}
      enableRowSelection
      rowSelection={rowSelection}
      onRowSelectionChange={(updater) => {
        const next = typeof updater === "function" ? updater(rowSelection) : updater;
        onRowSelectionChange(next);
      }}
      getRowId={(n) => n.id}
    />
  );
}

export default function NotificationsPage() {
  return (
    <Suspense fallback={<div className="h-40 animate-pulse bg-muted border-2 border-foreground" />}>
      <NotificationsContent />
    </Suspense>
  );
}
