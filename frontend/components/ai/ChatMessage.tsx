import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";

interface ChatMessageProps {
  role: string;
  content: string;
}

type CodeProps = {
  node?: any;
  inline?: boolean;
  className?: string;
  children?: React.ReactNode;
} & React.HTMLAttributes<HTMLElement>;

type ListProps = {
  node?: any;
  ordered?: boolean;
  children?: React.ReactNode;
} & React.HTMLAttributes<HTMLElement>;

export function ChatMessage({ role, content }: ChatMessageProps) {
  const isUser = role === "user";

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
              strong: ({ children, ...props }: any) => (
                <strong className="font-black" {...props}>{children}</strong>
              ),
              em: ({ children, ...props }: any) => (
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
              ul: ({ children, ...props }: any) => (
                <ul className="list-disc list-inside space-y-1 my-1" {...props}>{children}</ul>
              ),
              ol: ({ children, ...props }: any) => (
                <ol className="list-decimal list-inside space-y-1 my-1" {...props}>{children}</ol>
              ),
              li: ({ children, ...props }: any) => (
                <li className="text-sm font-medium" {...props}>{children}</li>
              ),
              h1: ({ children, ...props }: any) => (
                <h1 className="text-lg font-black mt-3 mb-1" {...props}>{children}</h1>
              ),
              h2: ({ children, ...props }: any) => (
                <h2 className="text-base font-black mt-2 mb-1" {...props}>{children}</h2>
              ),
              h3: ({ children, ...props }: any) => (
                <h3 className="text-sm font-black mt-2 mb-1" {...props}>{children}</h3>
              ),
              p: ({ children, ...props }: any) => {
                const childrenArr = Array.isArray(children) ? children : [children];
                const hasBlock = childrenArr.some((c: any) =>
                  c?.type === 'pre' || c?.type === 'table' || c?.type === 'ul' || c?.type === 'ol' || c?.type === 'div'
                );
                if (hasBlock) return <>{children}</>;
                return <div className="text-sm font-medium mb-1 last:mb-0" {...props}>{children}</div>;
              },
              blockquote: ({ children, ...props }: any) => (
                <blockquote className="border-l-4 border-foreground pl-3 my-2 italic text-muted-foreground" {...props}>{children}</blockquote>
              ),
              table: ({ children, ...props }: any) => (
                <div className="overflow-x-auto my-2">
                  <table className="min-w-full border-2 border-foreground text-xs" {...props}>{children}</table>
                </div>
              ),
              thead: ({ children, ...props }: any) => (
                <thead className="bg-muted font-bold" {...props}>{children}</thead>
              ),
              th: ({ children, ...props }: any) => (
                <th className="border border-foreground px-2 py-1 text-left" {...props}>{children}</th>
              ),
              td: ({ children, ...props }: any) => (
                <td className="border border-foreground px-2 py-1 break-words" {...props}>{children}</td>
              ),
              hr: (props: any) => (
                <hr className="my-3 border-t-2 border-foreground/20" {...props} />
              ),
              a: ({ children, href, ...props }: any) => (
                <a href={href} className="underline text-primary hover:text-primary/80 font-bold" target="_blank" rel="noopener noreferrer" {...props}>{children}</a>
              ),
            }}
          >
            {cleanContent}
          </ReactMarkdown>
        </div>
      </div>
    </div>
  );
}