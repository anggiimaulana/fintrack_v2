'use client'

import { useState } from 'react'
import { Search, UserX, UserCheck, ShieldAlert } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
interface UserStats {
  id: string;
  full_name: string;
  email: string;
  role?: string;
  is_active: boolean;
  created_at: string;
  total_transactions?: number;
  last_transaction_date?: string | null;
}

import {
  Tabs,

  TabsList,
  TabsTrigger,
} from '@/components/ui/tabs'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { UserTable } from '@/components/admin/UserTable'
import { useAdminUsers } from '@/hooks/useAdminUsers'

export default function UserManagementPage() {
  const { users, isLoading, toggleUserStatus, isMutating } = useAdminUsers()
  const [searchQuery, setSearchQuery] = useState('')
  const [filter, setFilter] = useState('all')
  const [confirmDialog, setConfirmDialog] = useState<{ open: boolean, user: UserStats | null }>({
    open: false,
    user: null,
  })

  const filteredUsers = users?.filter((user: UserStats) => {
    const matchesSearch = 
      user.full_name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      user.email.toLowerCase().includes(searchQuery.toLowerCase())
    
    if (filter === 'all') return matchesSearch
    if (filter === 'active') return matchesSearch && user.is_active
    if (filter === 'suspended') return matchesSearch && !user.is_active
    
    return matchesSearch
  }) || []

  const handleToggleStatus = async () => {
    if (!confirmDialog.user) return

    try {
      await toggleUserStatus({
        id: confirmDialog.user.id,
        is_active: !confirmDialog.user.is_active
      })
      setConfirmDialog({ open: false, user: null })
    } catch (error) {
      console.error('Failed to toggle user status:', error)
    }
  }

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-xl font-black uppercase tracking-wide text-foreground">Manajemen User</h1>
        <p className="text-xs font-bold text-muted-foreground">Pantau dan kelola akses pengguna FinTrack</p>
      </div>
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div className="relative w-full max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={18} />
          <Input
            placeholder="Cari nama atau email..."
            className="pl-10"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
          />
        </div>
        <Tabs defaultValue="all" className="w-fit" onValueChange={setFilter}>
          <TabsList>
            <TabsTrigger value="all">Semua</TabsTrigger>
            <TabsTrigger value="active">Aktif</TabsTrigger>
            <TabsTrigger value="suspended">Suspended</TabsTrigger>
          </TabsList>
        </Tabs>
      </div>

      <UserTable 
        users={filteredUsers} 
        onToggleStatus={(user) => setConfirmDialog({ open: true, user })}
        isLoading={isLoading}
      />

      <Dialog open={confirmDialog.open} onOpenChange={(open) => !open && setConfirmDialog({ open: false, user: null })}>
        <DialogContent className="sm:max-w-[425px] border-2 border-foreground">
          <DialogHeader>
            <div className="flex justify-center mb-4">
              <div className={confirmDialog.user?.is_active ? "p-3 border-2 border-foreground bg-red-100 text-red-700" : "p-3 border-2 border-foreground bg-green-100 text-green-700"}>
                <ShieldAlert size={32} />
              </div>
            </div>
            <DialogTitle className="text-center font-black uppercase tracking-wide">
              {confirmDialog.user?.is_active ? 'Suspend User?' : 'Aktifkan User?'}
            </DialogTitle>
            <DialogDescription className="text-center font-medium">
              {confirmDialog.user?.is_active 
                ? `Apakah Anda yakin ingin menonaktifkan akun ${confirmDialog.user?.full_name}? User tidak akan bisa masuk ke sistem.`
                : `Apakah Anda yakin ingin mengaktifkan kembali akun ${confirmDialog.user?.full_name}?`}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="sm:justify-center gap-2">
            <Button variant="ghost" onClick={() => setConfirmDialog({ open: false, user: null })} disabled={isMutating}>
              Batal
            </Button>
            <Button
              variant={confirmDialog.user?.is_active ? "destructive" : "default"}
              className={!confirmDialog.user?.is_active ? "bg-green-600 hover:bg-green-700" : ""}
              onClick={handleToggleStatus}
              disabled={isMutating}
            >
              {isMutating ? 'Memproses...' : (confirmDialog.user?.is_active ? 'Ya, Suspend' : 'Ya, Aktifkan')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
