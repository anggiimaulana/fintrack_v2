'use client'

import { format } from 'date-fns'
import { id } from 'date-fns/locale'
import { UserX, UserCheck } from 'lucide-react'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'

interface User {
  id: string
  full_name: string
  email: string
  created_at: string
  is_active: boolean
  transaction_count?: number
}

interface UserTableProps {
  users: User[]
  onToggleStatus: (user: User) => void
  isLoading?: boolean
}

export function UserTable({ users, onToggleStatus, isLoading }: UserTableProps) {
  if (isLoading) {
    return (
      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow className="hover:bg-transparent bg-neutral-50">
              <TableHead>Nama</TableHead>
              <TableHead>Email</TableHead>
              <TableHead>Tanggal Daftar</TableHead>
              <TableHead>Status</TableHead>
              <TableHead className="text-right">Transaksi</TableHead>
              <TableHead className="text-right">Aksi</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {[1, 2, 3, 4, 5].map((i) => (
              <TableRow key={i}>
                <TableCell><Skeleton className="h-5 w-32" /></TableCell>
                <TableCell><Skeleton className="h-5 w-48" /></TableCell>
                <TableCell><Skeleton className="h-5 w-24" /></TableCell>
                <TableCell><Skeleton className="h-5 w-16" /></TableCell>
                <TableCell className="text-right"><Skeleton className="h-5 w-12 ml-auto" /></TableCell>
                <TableCell className="text-right"><Skeleton className="h-5 w-20 ml-auto" /></TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    )
  }

  return (
    <div className="rounded-md border bg-white">
      <Table>
        <TableHeader>
          <TableRow className="hover:bg-transparent bg-neutral-50">
            <TableHead>Nama</TableHead>
            <TableHead>Email</TableHead>
            <TableHead>Tanggal Daftar</TableHead>
            <TableHead>Status</TableHead>
            <TableHead className="text-right">Transaksi</TableHead>
            <TableHead className="text-right">Aksi</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {users.map((user) => (
            <TableRow key={user.id}>
              <TableCell className="font-medium">{user.full_name}</TableCell>
              <TableCell>{user.email}</TableCell>
              <TableCell>
                {format(new Date(user.created_at), 'dd MMM yyyy', { locale: id })}
              </TableCell>
              <TableCell>
                <Badge
                  variant={user.is_active ? 'default' : 'destructive'}
                  className={user.is_active ? "bg-success-500" : ""}
                >
                  {user.is_active ? 'Aktif' : 'Suspended'}
                </Badge>
              </TableCell>
              <TableCell className="text-right font-mono">
                {user.transaction_count || 0}
              </TableCell>
              <TableCell className="text-right">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => onToggleStatus(user)}
                  className={user.is_active ? "text-danger-600 hover:text-danger-700 hover:bg-danger-50" : "text-success-600 hover:text-success-700 hover:bg-success-50"}
                >
                  {user.is_active ? (
                    <>
                      <UserX className="mr-2 h-4 w-4" /> Suspend
                    </>
                  ) : (
                    <>
                      <UserCheck className="mr-2 h-4 w-4" /> Aktifkan
                    </>
                  )}
                </Button>
              </TableCell>
            </TableRow>
          ))}
          {users.length === 0 && (
            <TableRow>
              <TableCell colSpan={6} className="h-24 text-center text-neutral-500">
                Tidak ada user ditemukan.
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </div>
  )
}
