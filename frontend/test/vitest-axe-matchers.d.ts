// Register vitest-axe's `toHaveNoViolations()` on Vitest's typed `expect`.
//
// vitest-axe augments the legacy global `Vi` namespace, which Vitest 3 no
// longer uses for matcher typing — the real interface now lives in the
// `vitest` module. Without this augmentation `tsc` fails with:
//   TS2339: Property 'toHaveNoViolations' does not exist on type 'Assertion<AxeResults>'
import type { AxeMatchers } from "vitest-axe/matchers";

declare module "vitest" {
  interface Assertion<T = any> extends AxeMatchers {}
}

export {};
