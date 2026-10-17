/**
 * Sanitizes assistant chat content before it reaches the user.
 *
 * The model is told to wrap tool invocations in <TOOL_CALL>/<TOOL_RESULT> tags,
 * and the backend strips those while streaming. But the model sometimes ignores
 * the contract and echoes internal payloads as plain text — a bare
 * `{"chart": {...}}` object, a ```json fenced block, or a data payload copied
 * out of the tool-result prompt. Those must never be shown as raw JSON: the
 * user asked for a chart, not a data dump.
 *
 * This is the last line of defence. It is intentionally conservative: it only
 * removes blocks that are unmistakably internal payloads, and leaves ordinary
 * prose (including numbers and bullet lists) untouched.
 */

const TOOL_TAG_PATTERNS: RegExp[] = [
  /<TOOL_CALL>[\s\S]*?<\/TOOL_CALL>/g,
  /<TOOL_RESULT>[\s\S]*?<\/TOOL_RESULT>/g,
  /<TOOL_CALL>[\s\S]*$/g, // partial: opening tag seen, closing tag not yet
  /<TOOL_RESULT>[\s\S]*$/g,
];

/** Fenced code block whose language tag is json/JSON. */
const JSON_FENCE = /```json\s*[\s\S]*?```/gi;

/**
 * A fenced block with no language tag that contains only a JSON object/array.
 * Requiring the fence body to start with `{`/`[` and end with `}`/`]` keeps
 * genuine code samples (which are not JSON) intact.
 */
const BARE_JSON_FENCE = /```\s*([{[][\s\S]*?[}\]])\s*```/g;

/**
 * A standalone JSON object on its own line that looks like an internal payload:
 * it must start with a known internal key so we never swallow a user's own
 * pasted JSON that happens to be valid.
 */
const INTERNAL_JSON_LINE = /^\s*\{\s*"(chart|chart_type|datasets|tool|data|success|labels|income|expense|savings)"[\s\S]*?\}\s*$/gm;

/** Internal commentary the model sometimes emits after a tool run. */
const INTERNAL_COMMENT = /^\s*(?:→|->)\s*(?:skipped|note|internal|todo)\b.*$/gim;

/**
 * sanitizeAssistantContent removes internal tool/JSON payloads from assistant
 * text. User-authored messages should NOT be passed through this.
 */
export function sanitizeAssistantContent(content: string): string {
  let out = content;
  for (const re of TOOL_TAG_PATTERNS) out = out.replace(re, "");
  out = out.replace(JSON_FENCE, "");
  out = out.replace(BARE_JSON_FENCE, "");
  out = out.replace(INTERNAL_JSON_LINE, "");
  out = out.replace(INTERNAL_COMMENT, "");

  // Collapse the blank lines left behind so the message does not open with a
  // large empty gap.
  out = out.replace(/\n{3,}/g, "\n\n").trim();

  // If the model answered with nothing but a payload, say something useful
  // rather than rendering an empty bubble.
  if (out === "") {
    return "Hasil sudah saya proses.";
  }
  return out;
}
