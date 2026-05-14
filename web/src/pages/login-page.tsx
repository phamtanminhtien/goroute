import { ShieldEllipsis, Sparkles } from "lucide-react";
import { motion } from "motion/react";

import { LoginForm } from "@/features/auth/login-form";
import { fadeInUp, staggerContainer } from "@/shared/lib/motion";

export function LoginPage() {
  return (
    <main className="relative min-h-screen overflow-hidden px-4 py-8 sm:px-6 sm:py-10">
      <div className="pointer-events-none absolute inset-0">
        <div className="bg-primary/10 absolute top-[-5%] left-[-8%] size-72 rounded-full blur-3xl" />
        <div className="bg-accent/10 absolute right-[-6%] bottom-0 size-80 rounded-full blur-3xl" />
      </div>

      <motion.div
        animate="visible"
        className="relative mx-auto grid min-h-[calc(100vh-4rem)] w-full max-w-6xl items-center gap-8 lg:grid-cols-[1.1fr_0.9fr]"
        initial="hidden"
        variants={staggerContainer}
      >
        <motion.section className="space-y-6" variants={fadeInUp}>
          <div className="border-border/70 bg-bg-secondary text-fg-secondary inline-flex items-center gap-2 rounded-full border px-4 py-1.5 text-xs font-semibold tracking-[0.22em] uppercase">
            <ShieldEllipsis className="size-3.5" />
            Admin access
          </div>
          <div className="max-w-2xl space-y-4">
            <h1 className="text-fg-primary text-4xl font-semibold tracking-tight sm:text-5xl">
              Secure the admin dashboard with your access token.
            </h1>
            <p className="text-fg-secondary text-base leading-8 sm:text-lg">
              Sign in with an admin token to access provider controls, runtime
              configuration, and operational status in one protected dashboard.
            </p>
          </div>
          <div className="grid gap-3 sm:grid-cols-2">
            {[
              "Protected routes gate access to providers, runtime, and system controls.",
              "Your active token is stored locally and reused for authenticated admin requests.",
            ].map((item) => (
              <div
                className="border-border/70 bg-bg-secondary text-fg-secondary rounded-[24px] border px-4 py-4 text-sm leading-6 shadow-[var(--shadow-sm)]"
                key={item}
              >
                <Sparkles className="text-primary mb-3 size-4" />
                {item}
              </div>
            ))}
          </div>
        </motion.section>

        <div className="flex justify-center lg:justify-end">
          <LoginForm />
        </div>
      </motion.div>
    </main>
  );
}
