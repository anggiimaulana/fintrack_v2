import { render, screen, fireEvent } from "@testing-library/react"
import { describe, it, expect, vi } from "vitest"
import { RupiahInput } from "./RupiahInput"

describe("RupiahInput", () => {
  it("formats digits with thousands dots e.g. 50000 -> 50.000", () => {
    const handleChange = vi.fn()
    render(<RupiahInput value={50000} onChange={handleChange} />)

    const input = screen.getByRole("textbox") as HTMLInputElement
    expect(input.value).toBe("50.000")
  })

  it("calls onChange with numeric integer value when typed", () => {
    const handleChange = vi.fn()
    render(<RupiahInput onChange={handleChange} />)

    const input = screen.getByRole("textbox")
    fireEvent.change(input, { target: { value: "75000" } })

    expect(handleChange).toHaveBeenCalledWith(75000)
  })

  it("strips non-digits and strips leading zeros", () => {
    const handleChange = vi.fn()
    render(<RupiahInput onChange={handleChange} />)

    const input = screen.getByRole("textbox")
    fireEvent.change(input, { target: { value: "abc0012500" } })

    expect(handleChange).toHaveBeenCalledWith(12500)
  })

  it("displays error when value is below min=1", () => {
    render(<RupiahInput value={0} min={1} />)

    const alert = screen.getByRole("alert")
    expect(alert).toHaveTextContent("Minimal input adalah 1")
  })
})
