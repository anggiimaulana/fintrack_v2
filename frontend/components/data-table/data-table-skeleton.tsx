"use client";

import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHeader, TableRow, TableHead } from "@/components/ui/table";

interface DataTableSkeletonProps {
  columns: number;
  rows?: number;
  /** Optional column headers to render above the skeleton rows. */
  headers?: string[];
}

/** Loading placeholder shown while table data is being fetched. */
export function DataTableSkeleton({ columns, rows = 5, headers }: DataTableSkeletonProps) {
  return (
    <div className="border-2 border-foreground bg-background">
      <Table>
        {headers && (
          <TableHeader>
            <TableRow className="hover:bg-transparent bg-neutral-50">
              {headers.map((h, i) => (
                <TableHead key={i}>{h}</TableHead>
              ))}
            </TableRow>
          </TableHeader>
        )}
        <TableBody>
          {Array.from({ length: rows }).map((_, r) => (
            <TableRow key={r}>
              {Array.from({ length: columns }).map((_, c) => (
                <TableCell key={c}>
                  <Skeleton className="h-5 w-full" />
                </TableCell>
              ))}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
