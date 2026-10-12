import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { ChatMessage } from "./ChatMessage";

describe("ChatMessage markdown link sanitization (TASK-FIX-01)", () => {
  it("allows safe http/https links", () => {
    render(<ChatMessage author="assistant" content="[Google](https://google.com)" />);
    const link = screen.getByRole("link", { name: "Google" });
    expect(link).toHaveAttribute("href", "https://google.com");
  });

  it("neutralizes javascript: URLs to #", () => {
    render(<ChatMessage author="assistant" content="[Evil](javascript:alert(1))" />);
    const link = screen.getByRole("link", { name: "Evil" });
    expect(link).toHaveAttribute("href", "#");
  });

  it("neutralizes data: URLs to #", () => {
    render(<ChatMessage author="assistant" content="[Data](data:text/html,evil)" />);
    const link = screen.getByRole("link", { name: "Data" });
    expect(link).toHaveAttribute("href", "#");
  });
});
