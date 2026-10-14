import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { MultiSeriesChart } from "./MultiSeriesChart";

// Regression: the Go backend marshals nil slices to JSON `null`, so the chart
// used to receive `series={null}` and throw
// "Cannot read properties of null (reading 'forEach')" from `series.forEach`.
// These cases must degrade to the empty state instead of crashing the page.
describe("MultiSeriesChart null-safety", () => {
  it("renders the empty state when series is null", () => {
    render(
      <MultiSeriesChart
        labels={["2026-01"]}
        series={null as unknown as { name: string; data: number[]; color: string }[]}
        type="line"
      />
    );
    expect(screen.getByText("Tidak ada data")).toBeInTheDocument();
  });

  it("renders the empty state when labels is null", () => {
    render(
      <MultiSeriesChart
        labels={null as unknown as string[]}
        series={[{ name: "Makanan", data: [1000], color: "#3b5bdb" }]}
        type="line"
      />
    );
    expect(screen.getByText("Tidak ada data")).toBeInTheDocument();
  });

  it("renders the empty state for an empty series array", () => {
    render(<MultiSeriesChart labels={["2026-01"]} series={[]} type="area" />);
    expect(screen.getByText("Tidak ada data")).toBeInTheDocument();
  });

  it("drops malformed series entries without throwing", () => {
    render(
      <MultiSeriesChart
        labels={["2026-01"]}
        series={[{ name: "Broken" } as unknown as { name: string; data: number[]; color: string }]}
        type="bar"
      />
    );
    expect(screen.getByText("Tidak ada data")).toBeInTheDocument();
  });
});
