import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { ConnectionForm } from "./ConnectionForm";

describe("ConnectionForm", () => {
  it("renders new connection form", () => {
    render(
      <ConnectionForm
        connection={null}
        onSaved={vi.fn()}
        onCancel={vi.fn()}
      />
    );

    expect(screen.getByText("New Connection")).toBeInTheDocument();
    expect(
      screen.getByPlaceholderText("My Database")
    ).toBeInTheDocument();
    expect(screen.getByText("PostgreSQL")).toBeInTheDocument();
    expect(screen.getByText("MongoDB")).toBeInTheDocument();
    expect(screen.getByText("Cassandra")).toBeInTheDocument();
    expect(screen.getByText("ScyllaDB")).toBeInTheDocument();
  });

  it("renders edit connection form with data", () => {
    const conn = {
      id: "1",
      name: "Test PG",
      type: "postgres" as const,
      host: "db.example.com",
      port: 5432,
      username: "admin",
      password: "",
      database: "mydb",
      created: "",
      updated: "",
    };

    render(
      <ConnectionForm
        connection={conn}
        onSaved={vi.fn()}
        onCancel={vi.fn()}
      />
    );

    expect(screen.getByText("Edit Connection")).toBeInTheDocument();
    expect(screen.getByDisplayValue("Test PG")).toBeInTheDocument();
    expect(screen.getByDisplayValue("db.example.com")).toBeInTheDocument();
    expect(screen.getByDisplayValue("5432")).toBeInTheDocument();
  });

  it("calls onCancel when cancel button clicked", () => {
    const onCancel = vi.fn();
    render(
      <ConnectionForm
        connection={null}
        onSaved={vi.fn()}
        onCancel={onCancel}
      />
    );

    fireEvent.click(screen.getByText("Cancel"));
    expect(onCancel).toHaveBeenCalledTimes(1);
  });

  it("disables save when name or host is empty", () => {
    render(
      <ConnectionForm
        connection={null}
        onSaved={vi.fn()}
        onCancel={vi.fn()}
      />
    );

    const saveButton = screen.getByText("Save & Connect");
    expect(saveButton).toBeDisabled();
  });

  it("changes default port when database type changes", () => {
    render(
      <ConnectionForm
        connection={null}
        onSaved={vi.fn()}
        onCancel={vi.fn()}
      />
    );

    // Default is postgres = 5432
    expect(screen.getByDisplayValue("5432")).toBeInTheDocument();

    // Switch to MongoDB
    fireEvent.click(screen.getByText("MongoDB"));
    expect(screen.getByDisplayValue("27017")).toBeInTheDocument();

    // Switch to Cassandra
    fireEvent.click(screen.getByText("Cassandra"));
    expect(screen.getByDisplayValue("9042")).toBeInTheDocument();
  });
});
