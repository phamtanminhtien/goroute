import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it } from "vitest";

import { useSyncTheme, useUIStore } from "@/shared/store/ui-store";
import { renderWithQueryClient } from "@/test/test-utils";

function ThemeHarness() {
  useSyncTheme();
  const theme = useUIStore((state) => state.theme);
  const toggleTheme = useUIStore((state) => state.toggleTheme);

  return (
    <button onClick={toggleTheme} type="button">
      Current theme: {theme}
    </button>
  );
}

describe("ui theme store", () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.dataset.theme = "light";
    useUIStore.setState({ theme: "light" });
  });

  it("syncs the root data-theme attribute and local storage", async () => {
    const user = userEvent.setup();

    renderWithQueryClient(<ThemeHarness />);

    expect(document.documentElement.dataset.theme).toBe("light");
    expect(localStorage.getItem("goroute.theme")).toBe("light");

    await user.click(
      screen.getByRole("button", { name: /current theme: light/i }),
    );

    expect(document.documentElement.dataset.theme).toBe("dark");
    expect(localStorage.getItem("goroute.theme")).toBe("dark");
  });
});
