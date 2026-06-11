import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { Sidebar } from "./Sidebar";

// Mock the API
vi.mock("@/api/client", () => ({
  api: {
    connections: {
      list: vi.fn().mockResolvedValue([]),
      connect: vi.fn().mockResolvedValue({ status: "connected" }),
      disconnect: vi.fn().mockResolvedValue({ status: "disconnected" }),
      delete: vi.fn().mockResolvedValue(undefined),
    },
    db: {
      databases: vi.fn().mockResolvedValue([]),
      tables: vi.fn().mockResolvedValue([]),
      switchDatabase: vi.fn().mockResolvedValue({ status: "switched", database: "" }),
      createTable: vi.fn().mockResolvedValue({ status: "created", name: "" }),
      dropTable: vi.fn().mockResolvedValue({ status: "dropped", name: "" }),
    },
  },
}));

describe("Sidebar", () => {
  it("renders title and new connection button", () => {
    render(
      <Sidebar
        activeSelection={null}
        onSelectTable={vi.fn()}
        onOpenQuery={vi.fn()}
        onNewConnection={vi.fn()}
        onEditConnection={vi.fn()}
      />
    );

    expect(screen.getByText("dbUI")).toBeInTheDocument();
    expect(screen.getByText("+ New Connection")).toBeInTheDocument();
  });

  it("calls onNewConnection when button clicked", () => {
    const onNew = vi.fn();
    render(
      <Sidebar
        activeSelection={null}
        onSelectTable={vi.fn()}
        onOpenQuery={vi.fn()}
        onNewConnection={onNew}
        onEditConnection={vi.fn()}
      />
    );

    screen.getByText("+ New Connection").click();
    expect(onNew).toHaveBeenCalledTimes(1);
  });
});
