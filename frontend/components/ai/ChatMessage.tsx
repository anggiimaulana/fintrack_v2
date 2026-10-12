import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";

interface ChatMessageProps {
  /** Message author. Named `author` (not `role`) to avoid being read as the ARIA role attribute. */
  author: string;
  content: string;
}

type CodeProps = {
  node?: unknown;
  inline?: boolean;
  className?: string;
  children?: React.ReactNode;
} & React.HTMLAttributes<HTMLElement>;

function isSafeUrl(url?: string): boolean {
  if (!url) return false;
  const trimmed = url.trim().toLowerCase();
  if (trimmed.startsWith("javascript:") || trimmed.startsWith("data:") || trimmed.startsWith("vbscript:")) {
    return false;
  }
  try {
    const parsed = new URL(url, "https://fintrack.internal");
    return parsed.protocol === "http:" || parsed.protocol === "https:" || parsed.protocol === "mailto:";
  } catch {
    return false;
  }
}

export function ChatMessage({ author, content }: ChatMessageProps) {
  const isUser = author === "user";

  // Strip complete and partial tool call blocks (security: never expose internal tool JSON to user)
  const cleanContent = content
    .replace(/<TOOL_CALL>[\s\S]*?<\/TOOL_CALL>/g, "")
    .replace(/<TOOL_RESULT>[\s\S]*?<\/TOOL_RESULT>/g, "")
    .replace(/<TOOL_CALL>[\s\S]*$/g, "")          // partial: no closing tag yet
    .replace(/<TOOL_RESULT>[\s\S]*$/g, "")         // partial: no closing tag yet
    .trim();

  return (
    <div className={`flex w-full ${isUser ? "justify-end" : "justify-start"}`}>
      <div
        className={`w-full sm:max-w-[80%] px-4 py-2.5 border-2 border-foreground break-words ${
          isUser
            ? "bg-primary text-primary-foreground shadow-[3px_3px_0px_0px] shadow-foreground"
            : "bg-card text-foreground shadow-[3px_3px_0px_0px] shadow-foreground"
        }`}
      >
        <div className="text-sm font-medium markdown-content">
          <ReactMarkdown
            remarkPlugins={[remarkGfm]}
            components={{
              strong: ({ children, node, ...props }: React.ComponentPropsWithoutRef<"strong"> & { node?: unknown }) => (
                <strong className="font-black" {...props}>{children}</strong>
              ),
              em: ({ children, node, ...props }: React.ComponentPropsWithoutRef<"em"> & { node?: unknown }) => (
                <em className="italic" {...props}>{children}</em>
              ),
              code: ({ inline, className, children, ...props }: CodeProps) => {
                if (inline) {
                  return (
                    <code className="bg-muted px-1.5 py-0.5 border border-foreground text-xs font-mono font-bold break-all" {...props}>
                      {children}
                    </code>
                  );
                }
                return (
                  <pre className="bg-muted p-3 border-2 border-foreground overflow-x-auto my-2 text-xs font-mono">
                    <code className={className} {...props}>{children}</code>
                  </pre>
                );
              },
              ul: ({ children, node, ...props }: React.ComponentPropsWithoutRef<"ul"> & { node?: unknown }) => (
                <ul className="list-disc list-inside space-y-1 my-1" {...props}>{children}</ul>
              ),
              ol: ({ children, node, ...props }: React.ComponentPropsWithoutRef<"ol"> & { node?: unknown }) => (
                <ol className="list-decimal list-inside space-y-1 my-1" {...props}>{children}</ol>
              ),
              li: ({ children, node, ...props }: React.ComponentPropsWithoutRef<"li"> & { node?: unknown }) => (
                <li className="text-sm font-medium" {...props}>{children}</li>
              ),
              h1: ({ children, node, ...props }: React.ComponentPropsWithoutRef<"h1"> & { node?: unknown }) => (
                <h1 className="text-lg font-black mt-3 mb-1" {...props}>{children}</h1>
              ),
              h2: ({ children, node, ...props }: React.ComponentPropsWithoutRef<"h2"> & { node?: unknown }) => (
                <h2 className="text-base font-black mt-2 mb-1" {...props}>{children}</h2>
              ),
              h3: ({ children, node, ...props }: React.ComponentPropsWithoutRef<"h3"> & { node?: unknown }) => (
                <h3 className="text-sm font-black mt-2 mb-1" {...props}>{children}</h3>
              ),
              p: ({ children, node, ...props }: React.ComponentPropsWithoutRef<"p"> & { node?: unknown }) => {
                const childrenArr = Array.isArray(children) ? children : [children];
                const hasBlock = childrenArr.some((c: unknown) =>
                  typeof c === "object" && c !== null && "type" in c &&
                  ((c as Record<string, unknown>).type === 'pre' || (c as Record<string, unknown>).type === 'table' || (c as Record<string, unknown>).type === 'ul' || (c as Record<string, unknown>).type === 'ol' || (c as Record<string, unknown>).type === 'div')
                );
                if (hasBlock) return <>{children}</>;
                return <div className="text-sm font-medium mb-1 last:mb-0" {...props}>{children}</div>;
              },
              blockquote: ({ children, node, ...props }: React.ComponentPropsWithoutRef<"blockquote"> & { node?: unknown }) => (
                <blockquote className="border-l-4 border-foreground pl-3 my-2 italic text-muted-foreground" {...props}>{children}</blockquote>
              ),
              table: ({ children, node, ...props }: React.ComponentPropsWithoutRef<"table"> & { node?: unknown }) => (
                <div className="overflow-x-auto my-2">
                  <table className="min-w-full border-2 border-foreground text-xs" {...props}>{children}</table>
                </div>
              ),
              thead: ({ children, node, ...props }: React.ComponentPropsWithoutRef<"thead"> & { node?: unknown }) => (
                <thead className="bg-muted font-bold" {...props}>{children}</thead>
              ),
              th: ({ children, node, ...props }: React.ComponentPropsWithoutRef<"th"> & { node?: unknown }) => (
                <th className="border border-foreground px-2 py-1 text-left" {...props}>{children}</th>
              ),
              td: ({ children, node, ...props }: React.ComponentPropsWithoutRef<"td"> & { node?: unknown }) => (
                <td className="border border-foreground px-2 py-1 break-words" {...props}>{children}</td>
              ),
              hr: ({ node, ...props }: React.ComponentPropsWithoutRef<"hr"> & { node?: unknown }) => (
                <hr className="my-3 border-t-2 border-foreground/20" {...props} />
              ),
              a: ({ children, href, node, ...props }: React.ComponentPropsWithoutRef<"a"> & { node?: unknown }) => {
                const safeHref = isSafeUrl(href) ? href : "#";
                return (
                  <a
                    href={safeHref}
                    className="cursor-pointer underline text-primary hover:text-primary/80 font-bold"
                    target="_blank"
                    rel="noopener noreferrer"
                    {...props}
                  >
                    {children}
                  </a>
                );
              },
            }}
          >
            {cleanContent}
          </ReactMarkdown>
        </div>
      </div>
    </div>
  );
}