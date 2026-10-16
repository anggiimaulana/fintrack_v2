import { describe, it, expect, vi } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { NumberInput, formatNumberWithDots } from "./NumberInput"

describe("formatNumberWithDots", () => {
  it("formats integer numbers with dot thousand separators", () => {
    expect(formatNumberWithDots(50000)).toBe("50.000")
    expect(formatNumberWithDots(100000)).toBe("100.000")
    expect(formatNumberWithDots(250000)).toBe("250.000")
    expect(formatNumberWithDots(30)).toBe("30")
    expect(formatNumberWithDots(0)).toBe("0")
    expect(formatNumberWithDots(-1)).toBe("-1")
  })
})

describe("NumberInput component", () => {
  it("displays formatted thousands on initial render", () => {
    render(<NumberInput value={50000} />)
    const input = screen.getByRole("textbox")
    expect(input).toHaveValue("50.000")
  })

  it("updates and formats input as user types digits", () => {
    const onChange = vi.fn()
    render(<NumberInput value={undefined} onChange={onChange} />)
    const input = screen.getByRole("textbox")

    fireEvent.change(input, { target: { value: "100000" } })
    expect(input).toHaveValue("100.000")
    expect(onChange).toHaveBeenCalledWith(100000)
  })

  it("handles negative numbers if allowNegative is enabled", () => {
    const onChange = vi.fn()
    render(<NumberInput value={-1} allowNegative onChange={onChange} />)
    const input = screen.getByRole("textbox")
    expect(input).toHaveValue("-1")
  })
})
