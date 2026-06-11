import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { QueryEditor } from "./QueryEditor";

describe("QueryEditor", () => {
  it("renders query input and execute button", () => {
    render(<QueryEditor connectionId="test-conn" />);

    expect(
      screen.getByPlaceholderText(/Enter your query/i)
    ).toBeInTheDocument();
    expect(screen.getByText("Execute")).toBeInTheDocument();
  });

  it("execute button is disabled when query is empty", () => {
    render(<QueryEditor connectionId="test-conn" />);

    const executeButton = screen.getByText("Execute");
    expect(executeButton).toBeDisabled();
  });
});
