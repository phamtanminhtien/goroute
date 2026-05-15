import { describe, expect, it } from "vitest";

import {
  defaultProviderConnectionFormEntry,
  getProviderConnectionFormEntry,
} from "@/features/providers/connection-form-registry";

describe("provider connection form registry", () => {
  it("returns the provider-specific entry when provider id is registered", () => {
    const entry = getProviderConnectionFormEntry("cx");

    expect(entry).not.toBe(defaultProviderConnectionFormEntry);
  });

  it("falls back to the default entry for unknown providers", () => {
    expect(getProviderConnectionFormEntry("custom-provider")).toBe(
      defaultProviderConnectionFormEntry,
    );
  });

  it("keeps create and edit renderers distinct", () => {
    const entry = getProviderConnectionFormEntry("openai");

    expect(entry.renderCreate).not.toBe(entry.renderEdit);
  });
});
