'use client'

import * as Icons from 'lucide-react'
import { LucideIcon, Pencil, Trash2 } from 'lucide-react'
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
import { cn } from '@/lib/utils'
import Link from 'next/link'

interface Category {
  id: string
  name: string
  icon: string
  color: string
  is_active: boolean
}

interface CategoryTableProps {
  categories: Category[]
  onEdit: (category: Category) => void
  onDelete: (id: string) => void
  isLoading?: boolean
  type?: 'income' | 'expense'
}

export function CategoryTable({ categories, onEdit, onDelete, isLoading, type = 'income' }: CategoryTableProps) {
  return (
    <div className="rounded-md border">
      <Table>
        <TableHeader>
          <TableRow className="hover:bg-transparent bg-neutral-50">
            <TableHead>Nama</TableHead>
            <TableHead>Ikon</TableHead>
            <TableHead>Warna</TableHead>
            <TableHead>Status</TableHead>
            {type === 'expense' && <TableHead>Sub-kategori</TableHead>}
            <TableHead className="text-right">Aksi</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {categories.map((category) => {
            const Icon = (Icons as any)[category.icon] as LucideIcon
            return (
              <TableRow key={category.id}>
                <TableCell className="font-medium">{category.name}</TableCell>
                <TableCell>
                  <div className="flex items-center justify-center h-8 w-8 rounded-md bg-neutral-100 text-neutral-600">
                    {Icon && <Icon size={18} />}
                  </div>
                </TableCell>
                <TableCell>
                  <div className="flex items-center gap-2">
                    <div
                      className="h-4 w-4 rounded-full border border-neutral-200"
                      style={{ backgroundColor: category.color }}
                    />
                    <span className="text-xs font-mono text-neutral-500">{category.color}</span>
                  </div>
                </TableCell>
                <TableCell>
                  <Badge
                    variant={category.is_active ? 'default' : 'secondary'}
                    className={cn(category.is_active ? "bg-success-500" : "bg-neutral-200 text-neutral-600")}
                  >
                    {category.is_active ? 'Aktif' : 'Nonaktif'}
                  </Badge>
                </TableCell>
                {type === 'expense' && (
                  <TableCell>
                    <Button variant="outline" size="sm" asChild className="h-8 text-xs">
                      <Link href={`/admin/categories/expense/${category.id}/subcategories`}>
                        Kelola
                      </Link>
                    </Button>
                  </TableCell>
                )}
                <TableCell className="text-right">
                  <div className="flex justify-end gap-2">
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => onEdit(category)}
                      className="h-8 w-8 text-neutral-500 hover:text-primary-600"
                    >
                      <Pencil size={16} />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => onDelete(category.id)}
                      className="h-8 w-8 text-neutral-500 hover:text-danger-600"
                    >
                      <Trash2 size={16} />
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            )
          })}
          {categories.length === 0 && !isLoading && (
            <TableRow>
              <TableCell colSpan={5} className="h-24 text-center text-neutral-500">
                Belum ada kategori.
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </div>
  )
}
