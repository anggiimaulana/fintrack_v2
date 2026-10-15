import ChatWindow from "@/components/ai/ChatWindow";

export const metadata = {
  title: "AI Chat - FinTrack",
  description: "AI Financial Assistant",
};

export default function AiChatPage() {
  return (
    <div className="flex h-full w-full max-w-7xl mx-auto border-2 rounded-md border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground overflow-hidden">
      <ChatWindow />
    </div>
  );
}
