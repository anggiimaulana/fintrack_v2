import { defineConfig, globalIgnores } from "eslint/config";
import nextVitals from "eslint-config-next/core-web-vitals";
import nextTs from "eslint-config-next/typescript";
import jsxA11y from "eslint-plugin-jsx-a11y";

const eslintConfig = defineConfig([
  ...nextVitals,
  ...nextTs,
  // Accessibility rules (P07 slice C). eslint-config-next already registers the
  // jsx-a11y plugin, so only the rules are added here (re-registering the plugin
  // is a config error). `recommended` covers the WCAG-relevant static checks:
  // alt text, label association, ARIA validity, keyboard handlers on
  // interactive elements. Contrast is NOT covered by static lint — it is
  // computed manually from the design tokens (see the slice C spec §1.1).
  {
    files: ["**/*.tsx", "**/*.jsx"],
    rules: {
      ...jsxA11y.flatConfigs.recommended.rules,
    },
  },
  // Override default ignores of eslint-config-next.
  globalIgnores([
    // Default ignores of eslint-config-next:
    ".next/**",
    "out/**",
    "build/**",
    "next-env.d.ts",
  ]),
]);

export default eslintConfig;
