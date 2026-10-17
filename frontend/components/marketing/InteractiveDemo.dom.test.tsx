import { describe, it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { InteractiveDemo } from "./InteractiveDemo";
import { MICROCOPY } from "@/lib/marketing/copy";

describe("InteractiveDemo", () => {
  it("renders three tabs with the first selected", () => {
    render(<InteractiveDemo />);
    const tabs = screen.getAllByRole("tab");
    expect(tabs.map((t) => t.textContent?.trim())).toEqual(["Chat", "Suara", "Struk"]);
    expect(screen.getByRole("tab", { name: "Chat" })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByRole("tab", { name: "Suara" })).toHaveAttribute("aria-selected", "false");
  });

  it("keeps all three panels in the DOM (readable without interaction)", () => {
    const { container } = render(<InteractiveDemo />);
    const panels = container.querySelectorAll('[role="tabpanel"]');
    expect(panels).toHaveLength(3);
    const visible = [...panels].filter((panel) => !panel.hasAttribute("hidden"));
    expect(visible).toHaveLength(1);
  });

  it("wires tabs to their panels via aria attributes", () => {
    render(<InteractiveDemo />);
    screen.getAllByRole("tab").forEach((tab) => {
      const id = tab.getAttribute("id");
      const panelId = tab.getAttribute("aria-controls");
      expect(panelId).toBeTruthy();
      const panel = document.getElementById(panelId!);
      expect(panel).not.toBeNull();
      expect(panel).toHaveAttribute("aria-labelledby", id);
    });
  });

  it("switches the panel on click", () => {
    render(<InteractiveDemo />);
    fireEvent.click(screen.getByRole("tab", { name: "Struk" }));
    expect(screen.getByRole("tab", { name: "Struk" })).toHaveAttribute("aria-selected", "true");
    expect(document.getElementById("demo-panel-receipt")).not.toHaveAttribute("hidden");
    expect(document.getElementById("demo-panel-chat")).toHaveAttribute("hidden");
    expect(screen.getByText(/Rp 312\.500 tercatat/)).toBeInTheDocument();
  });

  it("moves selection with arrow keys and Home/End (roving tabindex)", () => {
    render(<InteractiveDemo />);
    const chat = screen.getByRole("tab", { name: "Chat" });
    expect(chat).toHaveAttribute("tabindex", "0");

    fireEvent.keyDown(chat, { key: "ArrowRight" });
    const voice = screen.getByRole("tab", { name: "Suara" });
    expect(voice).toHaveAttribute("aria-selected", "true");
    expect(voice).toHaveFocus();
    expect(voice).toHaveAttribute("tabindex", "0");

    fireEvent.keyDown(voice, { key: "End" });
    expect(screen.getByRole("tab", { name: "Struk" })).toHaveAttribute("aria-selected", "true");

    fireEvent.keyDown(screen.getByRole("tab", { name: "Struk" }), { key: "Home" });
    expect(screen.getByRole("tab", { name: "Chat" })).toHaveAttribute("aria-selected", "true");

    fireEvent.keyDown(screen.getByRole("tab", { name: "Chat" }), { key: "ArrowLeft" });
    expect(screen.getByRole("tab", { name: "Struk" })).toHaveAttribute("aria-selected", "true");
  });

  it("labels the mock data as an example", () => {
    render(<InteractiveDemo />);
    expect(screen.getByText(MICROCOPY.demoLabel)).toBeInTheDocument();
  });
});
