import { fireEvent, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const motionState = vi.hoisted(() => ({
  reducedMotion: false,
}));

vi.mock("motion/react", async () => {
  const actual =
    await vi.importActual<typeof import("motion/react")>("motion/react");

  return {
    ...actual,
    useReducedMotion: () => motionState.reducedMotion,
  };
});

import { Button } from "@/shared/ui/button";
import { Field } from "@/shared/ui/field";
import { Input } from "@/shared/ui/input";
import { PageHeader } from "@/shared/ui/page-header";
import { SectionCard } from "@/shared/ui/section-card";
import { renderWithQueryClient } from "@/test/test-utils";

describe("shared ui primitives", () => {
  beforeEach(() => {
    motionState.reducedMotion = false;
  });

  it("renders field, page header, and section card in different contexts", () => {
    renderWithQueryClient(
      <div className="grid gap-4 md:grid-cols-2">
        <PageHeader
          description="Shared page description"
          eyebrow="Preview"
          title="Reusable title"
        />
        <SectionCard
          description="Shared section description"
          title="Section title"
          tone="solid"
        >
          <Field help="Helpful copy" label="API token" spacing="sm">
            <Input inputSize="lg" value="value" readOnly />
          </Field>
          <Button tone="secondary" size="lg" type="button">
            Save
          </Button>
        </SectionCard>
      </div>,
    );

    expect(screen.getByText(/reusable title/i)).toBeInTheDocument();
    expect(screen.getByText(/section title/i)).toBeInTheDocument();
    expect(screen.getByText(/api token/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /save/i })).toBeInTheDocument();
  });

  it("supports icon-only buttons with accessible names", () => {
    renderWithQueryClient(
      <Button
        aria-label="Reveal token"
        iconOnly
        leadingIcon={<span>+</span>}
      />,
    );

    expect(
      screen.getByRole("button", { name: /reveal token/i }),
    ).toBeInTheDocument();
  });

  it("creates a ripple on pointer down when enabled", () => {
    const { container } = renderWithQueryClient(
      <Button type="button">Create ripple</Button>,
    );

    const button = screen.getByRole("button", { name: /create ripple/i });
    fireEvent.pointerDown(button, { clientX: 16, clientY: 16 });

    expect(container.querySelector('span[aria-hidden="true"]')).not.toBeNull();
  });

  it("does not create a ripple when disabled", () => {
    const { container } = renderWithQueryClient(
      <Button disabled type="button">
        Disabled ripple
      </Button>,
    );

    fireEvent.pointerDown(
      screen.getByRole("button", { name: /disabled ripple/i }),
      {
        clientX: 16,
        clientY: 16,
      },
    );

    expect(container.querySelector('span[aria-hidden="true"]')).toBeNull();
  });

  it("does not create a ripple when reduced motion is preferred", () => {
    motionState.reducedMotion = true;

    const { container } = renderWithQueryClient(
      <Button type="button">Reduced motion</Button>,
    );

    fireEvent.pointerDown(
      screen.getByRole("button", { name: /reduced motion/i }),
      {
        clientX: 16,
        clientY: 16,
      },
    );

    expect(container.querySelector('span[aria-hidden="true"]')).toBeNull();
  });
});
