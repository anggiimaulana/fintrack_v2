'use client'

import { Suspense, useCallback } from 'react'
import { usePathname, useRouter, useSearchParams } from 'next/navigation'
import { Coins, BookOpenText } from 'lucide-react'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { TokenBalancesTable } from '@/components/super-admin/tokens/TokenBalancesTable'
import { TokenLedgerTable } from '@/components/super-admin/tokens/TokenLedgerTable'

function SuperAdminTokensContent() {
  const router = useRouter()
  const pathname = usePathname()
  const searchParams = useSearchParams()

  const currentTab = searchParams.get('tab') || 'balances'

  // Tab 1 (Balances) params: page, per_page, q, sort, order
  const balancePage = parseInt(searchParams.get('page') || '1', 10) || 1
  const balancePerPage = parseInt(searchParams.get('per_page') || '20', 10) || 20
  const balanceSearch = searchParams.get('q') || ''
  const balanceSort = searchParams.get('sort') || ''
  const balanceOrder = (searchParams.get('order') === 'asc' ? 'asc' : 'desc') as 'asc' | 'desc'

  // Tab 2 (Ledger) params: ledger_page, ledger_per_page, ledger_entry_type, ledger_modality, ledger_user_id, ledger_start_date, ledger_end_date, ledger_sort, ledger_order
  const ledgerPage = parseInt(searchParams.get('ledger_page') || '1', 10) || 1
  const ledgerPerPage = parseInt(searchParams.get('ledger_per_page') || '20', 10) || 20
  const ledgerEntryType = searchParams.get('ledger_entry_type') || ''
  const ledgerModality = searchParams.get('ledger_modality') || ''
  const ledgerUserId = searchParams.get('ledger_user_id') || ''
  const ledgerStartDate = searchParams.get('ledger_start_date') || ''
  const ledgerEndDate = searchParams.get('ledger_end_date') || ''
  const ledgerSort = searchParams.get('ledger_sort') || 'created_at'
  const ledgerOrder = (searchParams.get('ledger_order') === 'asc' ? 'asc' : 'desc') as 'asc' | 'desc'

  // Helper to commit param updates to URL without clobbering other tab's params
  const updateQuery = useCallback(
    (updates: Record<string, string | number | null | undefined>) => {
      const next = new URLSearchParams(searchParams.toString())
      for (const [k, v] of Object.entries(updates)) {
        if (v === null || v === undefined || v === '') {
          next.delete(k)
        } else {
          next.set(k, String(v))
        }
      }
      const qs = next.toString()
      router.replace(qs ? `${pathname}?${qs}` : pathname, { scroll: false })
    },
    [router, pathname, searchParams]
  )

  const handleTabChange = (val: string) => {
    updateQuery({ tab: val === 'balances' ? null : val })
  }

  // Balances handlers
  const handleBalancePageChange = (p: number) => {
    updateQuery({ page: p === 1 ? null : p })
  }

  const handleBalancePerPageChange = (pp: number) => {
    updateQuery({ per_page: pp === 20 ? null : pp, page: null })
  }

  const handleBalanceSearchChange = (q: string) => {
    updateQuery({ q: q || null, page: null })
  }

  const handleBalanceSortChange = (sort: string, order: 'asc' | 'desc') => {
    updateQuery({
      sort: sort || null,
      order: order === 'desc' ? null : order,
      page: null,
    })
  }

  // Ledger handlers
  const handleLedgerPageChange = (p: number) => {
    updateQuery({ ledger_page: p === 1 ? null : p })
  }

  const handleLedgerPerPageChange = (pp: number) => {
    updateQuery({ ledger_per_page: pp === 20 ? null : pp, ledger_page: null })
  }

  const handleLedgerFilterChange = (key: string, val: string) => {
    updateQuery({
      [`ledger_${key}`]: val || null,
      ledger_page: null,
    })
  }

  const handleLedgerSortChange = (sort: string, order: 'asc' | 'desc') => {
    updateQuery({
      ledger_sort: sort === 'created_at' ? null : sort,
      ledger_order: order === 'desc' ? null : order,
      ledger_page: null,
    })
  }

  const handleLedgerResetFilters = () => {
    updateQuery({
      ledger_entry_type: null,
      ledger_modality: null,
      ledger_user_id: null,
      ledger_start_date: null,
      ledger_end_date: null,
      ledger_page: null,
    })
  }

  return (
    <div className="space-y-6 p-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Manajemen Token AI</h1>
        <p className="text-sm text-muted-foreground">
          Kelola kuota, saldo per periode aktif, dan riwayat audit ledger transaksi token AI pengguna.
        </p>
      </div>

      <Tabs value={currentTab} onValueChange={handleTabChange} className="space-y-4">
        <TabsList className="grid w-full max-w-md grid-cols-2">
          <TabsTrigger value="balances" className="gap-2">
            <Coins className="h-4 w-4" />
            Saldo per Pengguna
          </TabsTrigger>
          <TabsTrigger value="ledger" className="gap-2">
            <BookOpenText className="h-4 w-4" />
            Ledger Transaksi
          </TabsTrigger>
        </TabsList>

        <TabsContent value="balances" className="space-y-4">
          <TokenBalancesTable
            page={balancePage}
            perPage={balancePerPage}
            search={balanceSearch}
            sort={balanceSort}
            order={balanceOrder}
            onPageChange={handleBalancePageChange}
            onPerPageChange={handleBalancePerPageChange}
            onSearchChange={handleBalanceSearchChange}
            onSortChange={handleBalanceSortChange}
          />
        </TabsContent>

        <TabsContent value="ledger" className="space-y-4">
          <TokenLedgerTable
            page={ledgerPage}
            perPage={ledgerPerPage}
            entryType={ledgerEntryType}
            modality={ledgerModality}
            userId={ledgerUserId}
            startDate={ledgerStartDate}
            endDate={ledgerEndDate}
            sort={ledgerSort}
            order={ledgerOrder}
            onPageChange={handleLedgerPageChange}
            onPerPageChange={handleLedgerPerPageChange}
            onFilterChange={handleLedgerFilterChange}
            onSortChange={handleLedgerSortChange}
            onResetFilters={handleLedgerResetFilters}
          />
        </TabsContent>
      </Tabs>
    </div>
  )
}

// The page reads useSearchParams, so it must sit under a Suspense boundary for
// the route to stay statically prerenderable.
export default function SuperAdminTokensPage() {
  return (
    <Suspense fallback={<div className="h-40 animate-pulse bg-muted border-2 border-foreground" />}>
      <SuperAdminTokensContent />
    </Suspense>
  )
}
