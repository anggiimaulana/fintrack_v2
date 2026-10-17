import { cn } from "@/lib/utils";

export interface SectionHeadingProps {
  /** Small uppercase label above the title (JetBrains Mono). */
  eyebrow?: string;
  title: string;
  description?: string;
  /** Render the title as an H1 (hero) instead of an H2 (section). */
  as?: "h1" | "h2";
  align?: "left" | "center";
  id?: string;
  className?: string;
}

/**
 * Consistent section heading: eyebrow + title + subtitle. Server component.
 * Titles use the display font; the title is capped at a readable width.
 */
export function SectionHeading({
  eyebrow,
  title,
  description,
  as = "h2",
  align = "left",
  id,
  className,
}: SectionHeadingProps) {
  const Tag = as;
  return (
    <div
      className={cn(
        "flex flex-col gap-3",
        align === "center" && "items-center text-center",
        className,
      )}
    >
      {eyebrow ? (
        <span className="font-mono text-xs font-bold uppercase tracking-[0.08em] text-muted-foreground">
          {eyebrow}
        </span>
      ) : null}
      <Tag
        id={id}
        className={cn(
          "font-heading font-extrabold tracking-[-0.02em] text-balance",
          as === "h1"
            ? "text-[clamp(2.5rem,6vw,4.5rem)] leading-[1.05]"
            : "text-[clamp(1.875rem,4vw,3rem)] leading-[1.1]",
        )}
      >
        {title}
      </Tag>
      {description ? (
        <p
          className={cn(
            "text-base leading-relaxed text-muted-foreground",
            align === "center" ? "max-w-2xl" : "max-w-2xl",
          )}
        >
          {description}
        </p>
      ) : null}
    </div>
  );
}
