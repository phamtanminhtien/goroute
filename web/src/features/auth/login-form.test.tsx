import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it } from "vitest";

import { useAuthStore } from "@/features/auth/auth-store";
import { LoginForm } from "@/features/auth/login-form";
import { renderWithQueryClient } from "@/test/test-utils";

describe("login form", () => {
  beforeEach(() => {
    localStorage.clear();
    useAuthStore.setState({
      hydrated: true,
      isAuthenticated: false,
      token: null,
    });
  });

  it("shows validation error for empty token submission", async () => {
    const user = userEvent.setup();

    renderWithQueryClient(
      <MemoryRouter>
        <LoginForm />
      </MemoryRouter>,
    );

    await user.click(screen.getByRole("button", { name: /sign in/i }));

    expect(
      screen.getAllByText(/enter an admin token to continue/i).length,
    ).toBeGreaterThan(0);
  });

  it("toggles token visibility", async () => {
    const user = userEvent.setup();

    renderWithQueryClient(
      <MemoryRouter>
        <LoginForm />
      </MemoryRouter>,
    );

    const input = screen.getByPlaceholderText(/enter your admin token/i);
    expect(input).toHaveAttribute("type", "password");

    await user.click(screen.getByRole("button", { name: /show token/i }));

    expect(input).toHaveAttribute("type", "text");
  });

  it("disables submit while submitting", async () => {
    const user = userEvent.setup();

    renderWithQueryClient(
      <MemoryRouter>
        <LoginForm />
      </MemoryRouter>,
    );

    await user.type(
      screen.getByPlaceholderText(/enter your admin token/i),
      "secret-token",
    );
    const submitButton = screen.getByRole("button", { name: /sign in/i });

    const submitPromise = user.click(submitButton);

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: /signing in/i }),
      ).toBeDisabled();
    });
    await submitPromise;
  });
});
