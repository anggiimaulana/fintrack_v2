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

describe("ChatMessage assistant mode badge", () => {
  it("shows a Penalaran badge for reasoning replies", () => {
    render(<ChatMessage author="assistant" content="jawaban" mode="reasoning" modelUsed="deepseek-r1" />);
    expect(screen.getByText("Penalaran")).toBeInTheDocument();
  });

  it("shows a Cepat badge for fast replies", () => {
    render(<ChatMessage author="assistant" content="jawaban" mode="fast" />);
    expect(screen.getByText("Cepat")).toBeInTheDocument();
  });

  it("renders no badge when the mode is unknown (legacy messages)", () => {
    render(<ChatMessage author="assistant" content="jawaban" />);
    expect(screen.queryByText("Cepat")).not.toBeInTheDocument();
    expect(screen.queryByText("Penalaran")).not.toBeInTheDocument();
  });

  it("never badges user messages", () => {
    render(<ChatMessage author="user" content="halo" mode="reasoning" />);
    expect(screen.queryByText("Penalaran")).not.toBeInTheDocument();
  });
});
