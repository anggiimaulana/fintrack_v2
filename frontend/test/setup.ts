import "@testing-library/jest-dom/vitest";
import { afterEach, expect } from "vitest";
import { cleanup } from "@testing-library/react";
import { axe } from "vitest-axe";
import * as matchers from "vitest-axe/matchers";

// `toHaveNoViolations()` for axe results.
expect.extend(matchers);

// Vitest runs without globals here, so Testing Library's automatic cleanup is
// not registered — do it explicitly, otherwise each test sees the previous
// test's DOM (which makes queries match more than one element).
afterEach(cleanup);
if (typeof globalThis.ResizeObserver === "undefined") {
  globalThis.ResizeObserver = class ResizeObserver {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
}


export { axe };
