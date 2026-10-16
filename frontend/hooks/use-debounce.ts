"use client";

import { useEffect, useMemo, useState } from "react";

/**
 * Debounces a value by delay ms (default 300).
 * Handles primitives and serialized objects.
 */
export function useDebounce<T>(value: T, delay = 300): T {
  const [debounced, setDebounced] = useState<T>(value);
  const serialized = useMemo(() => {
    try {
      return JSON.stringify(value);
    } catch {
      return value;
    }
  }, [value]);

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebounced(value);
    }, delay);
    return () => clearTimeout(timer);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [serialized, delay]);

  return debounced;
}
