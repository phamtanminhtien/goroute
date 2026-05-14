import type { Transition, Variants } from "motion/react";

const fadeTransition: Transition = {
  duration: 0.42,
};

export const fadeInUp: Variants = {
  hidden: { opacity: 0, y: 18 },
  visible: {
    opacity: 1,
    y: 0,
    transition: fadeTransition,
  },
};

export const staggerContainer: Variants = {
  hidden: {},
  visible: {
    transition: {
      delayChildren: 0.06,
      staggerChildren: 0.08,
    },
  },
};

export const softSpring = {
  stiffness: 340,
  damping: 24,
  mass: 0.9,
};
