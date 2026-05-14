import { Eye, EyeOff, KeyRound, LogIn } from "lucide-react";
import { AnimatePresence, motion } from "motion/react";
import { type FormEvent, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";

import { useAuthStore } from "@/features/auth/auth-store";
import { fadeInUp, staggerContainer } from "@/shared/lib/motion";
import { Button } from "@/shared/ui/button";
import { Field } from "@/shared/ui/field";
import { Input } from "@/shared/ui/input";
import { SurfaceCard } from "@/shared/ui/surface-card";

export function LoginForm() {
  const navigate = useNavigate();
  const location = useLocation();
  const signIn = useAuthStore((state) => state.signIn);

  const [token, setToken] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isTokenVisible, setIsTokenVisible] = useState(false);

  const nextPath =
    ((location.state as { from?: { pathname?: string } } | null)?.from
      ?.pathname as string | undefined) ?? "/providers";

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const normalizedToken = token.trim();

    if (!normalizedToken) {
      setError("Enter an admin token to continue.");
      return;
    }

    setIsSubmitting(true);
    setError(null);

    await new Promise((resolve) => {
      window.setTimeout(resolve, 120);
    });

    signIn(normalizedToken);
    navigate(nextPath, { replace: true });
  }

  return (
    <motion.div
      animate="visible"
      className="w-full max-w-md"
      initial="hidden"
      variants={fadeInUp}
    >
      <SurfaceCard asChild className="overflow-hidden p-6 sm:p-7">
        <motion.section
          animate="visible"
          initial="hidden"
          variants={staggerContainer}
        >
          <motion.div
            className="flex items-start justify-between gap-4"
            variants={fadeInUp}
          >
            <div className="space-y-2">
              <div className="border-border/70 bg-bg-tertiary text-fg-secondary inline-flex items-center gap-2 rounded-full border px-3 py-1 text-xs font-semibold tracking-[0.18em] uppercase">
                <KeyRound className="size-3.5" />
                Token sign-in
              </div>
              <div className="space-y-1">
                <h2 className="text-fg-primary text-2xl font-semibold tracking-tight">
                  Sign in with your admin token
                </h2>
                <p className="text-fg-secondary text-sm leading-6">
                  Your token unlocks the admin dashboard and is attached to
                  authenticated admin requests in this browser session.
                </p>
              </div>
            </div>
          </motion.div>

          <motion.form
            className="mt-6 space-y-5"
            onSubmit={(event) => {
              void handleSubmit(event);
            }}
            variants={fadeInUp}
          >
            <Field
              error={error}
              help="The token is stored locally in this browser and reused for admin requests."
              label="Admin token"
            >
              <div className="relative">
                <Input
                  autoComplete="off"
                  className="pr-[3.25rem]"
                  disabled={isSubmitting}
                  onChange={(event) => {
                    setToken(event.target.value);
                    if (error) {
                      setError(null);
                    }
                  }}
                  placeholder="Enter your admin token"
                  spellCheck={false}
                  type={isTokenVisible ? "text" : "password"}
                  value={token}
                />
                <Button
                  aria-label={isTokenVisible ? "Hide token" : "Show token"}
                  className="absolute top-1/2 right-1.5 -translate-y-1/2"
                  disabled={isSubmitting}
                  iconOnly
                  leadingIcon={
                    isTokenVisible ? (
                      <EyeOff className="size-4" />
                    ) : (
                      <Eye className="size-4" />
                    )
                  }
                  onClick={() => setIsTokenVisible((value) => !value)}
                  ripple={!isSubmitting}
                  tone="ghost"
                  type="button"
                />
              </div>
            </Field>

            <AnimatePresence initial={false}>
              {error ? (
                <motion.p
                  animate={{ opacity: 1, y: 0 }}
                  className="rounded-2xl border border-rose-200/80 bg-rose-50/90 px-4 py-3 text-sm text-rose-700"
                  exit={{ opacity: 0, y: -6 }}
                  initial={{ opacity: 0, y: -6 }}
                >
                  {error}
                </motion.p>
              ) : null}
            </AnimatePresence>

            <motion.div className="flex items-center gap-3" variants={fadeInUp}>
              <Button
                className="min-w-36"
                disabled={isSubmitting}
                leadingIcon={<LogIn className="size-4" />}
                type="submit"
              >
                {isSubmitting ? "Signing in..." : "Sign in"}
              </Button>
              <p className="text-fg-muted text-xs leading-5">
                Empty values are rejected locally before a session is created.
              </p>
            </motion.div>
          </motion.form>
        </motion.section>
      </SurfaceCard>
    </motion.div>
  );
}
