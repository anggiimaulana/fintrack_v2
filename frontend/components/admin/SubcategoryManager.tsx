'use client'

import { Pencil, Trash2 } from 'lucide-react'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'

interface Subcategory {
  id: string
  name: string
  is_active: boolean
}

interface SubcategoryManagerProps {
  subcategories: Subcategory[]
  onEdit: (sub: Subcategory) => void
  onDelete: (id: string) => void
  isLoading?: boolean
}

export function SubcategoryManager({ subcategories, onEdit, onDelete, isLoading }: SubcategoryManagerProps) {
  return (
    <div className="rounded-md border bg-white">
      <Table>
        <TableHeader>
          <TableRow className="hover:bg-transparent bg-neutral-50">
            <TableHead>Nama Sub-kategori</TableHead>
            <TableHead>Status</TableHead>
            <TableHead className="text-right">Aksi</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading ? (
            <>
              {[1, 2, 3].map((i) => (
                <TableRow key={i}>
                  <TableCell><Skeleton className="h-5 w-48" /></TableCell>
                  <TableCell><Skeleton className="h-5 w-16" /></TableCell>
                  <TableCell className="text-right"><Skeleton className="h-5 w-20 ml-auto" /></TableCell>
                </TableRow>
              ))}
            </>
          ) : (
            <>
              {subcategories?.map((sub) => (
                <TableRow key={sub.id}>
                  <TableCell className="font-medium">{sub.name}</TableCell>
                  <TableCell>
                    <Badge
                      variant={sub.is_active ? 'default' : 'secondary'}
                      className={sub.is_active ? "bg-success-500" : "bg-neutral-200 text-neutral-600"}
                    >
                      {sub.is_active ? 'Aktif' : 'Nonaktif'}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex justify-end gap-2">
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => onEdit(sub)}
                        className="h-8 w-8 text-neutral-500 hover:text-primary-600"
                      >
                        <Pencil size={16} />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => onDelete(sub.id)}
                        className="h-8 w-8 text-neutral-500 hover:text-danger-600"
                      >
                        <Trash2 size={16} />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
              {subcategories?.length === 0 && (
                <TableRow>
                  <TableCell colSpan={3} className="h-24 text-center text-neutral-500">
                    Belum ada sub-kategori.
                  </TableCell>
                </TableRow>
              )}
            </>
          )}
        </TableBody>
      </Table>
    </div>
  )
}
