'use client'

import { useState } from 'react'

/**
 * Reset dialog-local form state when `open` flips to true.
 *
 * Uses React's "adjust state during render" pattern instead of a
 * setState-in-effect, so no cascading render pass is scheduled. Pass
 * `signature` (e.g. the edited row id) when opening a different record
 * while the dialog stays mounted must also reset the form.
 */
export function useResetOnOpen(
  open: boolean,
  reset: () => void,
  signature?: string | number,
): void {
  const key = open ? String(signature ?? '') : null
  const [seenKey, setSeenKey] = useState<string | null>(key)

  if (key !== seenKey) {
    setSeenKey(key)
    if (key !== null) reset()
  }
}
