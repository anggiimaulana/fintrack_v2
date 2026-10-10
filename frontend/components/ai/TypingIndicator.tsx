export function TypingIndicator() {
  return (
    <div className="flex justify-start">
      <div className="bg-neutral-100 rounded-2xl rounded-bl-sm px-4 py-3 flex gap-1 items-center">
        <div className="w-2 h-2 bg-neutral-400 rounded-full animate-bounce" style={{ animationDelay: "0ms" }} />
        <div className="w-2 h-2 bg-neutral-400 rounded-full animate-bounce" style={{ animationDelay: "150ms" }} />
        <div className="w-2 h-2 bg-neutral-400 rounded-full animate-bounce" style={{ animationDelay: "300ms" }} />
      </div>
    </div>
  );
}
