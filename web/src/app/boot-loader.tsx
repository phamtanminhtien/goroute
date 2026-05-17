import { AnimatePresence, motion, useReducedMotion } from "motion/react";
import { type ReactNode, useEffect, useRef, useState } from "react";

const BOOT_LOADER_MIN_DURATION_MS = 1700;

type AppBootLoaderProps = {
  children: ReactNode;
};

export function AppBootLoader({ children }: AppBootLoaderProps) {
  const prefersReducedMotion = useReducedMotion() ?? false;
  const finishTimeoutRef = useRef<number | null>(null);
  const isFinishingRef = useRef(false);
  const [isBooting, setIsBooting] = useState(
    () => typeof window !== "undefined",
  );

  useEffect(() => {
    if (!isBooting || typeof window === "undefined") {
      return;
    }

    const startTime = window.performance.now();
    const finishBoot = () => {
      if (isFinishingRef.current) {
        return;
      }

      isFinishingRef.current = true;
      const elapsed = window.performance.now() - startTime;
      const remaining = Math.max(
        0,
        (prefersReducedMotion ? 320 : BOOT_LOADER_MIN_DURATION_MS) - elapsed,
      );

      finishTimeoutRef.current = window.setTimeout(() => {
        finishTimeoutRef.current = null;
        setIsBooting(false);
      }, remaining);
    };

    if (document.readyState === "complete") {
      finishBoot();
      return;
    }

    window.addEventListener("load", finishBoot, { once: true });

    return () => {
      if (finishTimeoutRef.current !== null) {
        window.clearTimeout(finishTimeoutRef.current);
        finishTimeoutRef.current = null;
      }
      window.removeEventListener("load", finishBoot);
    };
  }, [isBooting, prefersReducedMotion]);

  return (
    <>
      {children}

      <AnimatePresence>
        {isBooting ? (
          <BootLoaderOverlay reducedMotion={prefersReducedMotion} />
        ) : null}
      </AnimatePresence>
    </>
  );
}

function BootLoaderOverlay({ reducedMotion }: { reducedMotion: boolean }) {
  return (
    <motion.div
      animate={{ opacity: 1 }}
      className="fixed inset-0 z-[120] overflow-hidden"
      exit={{
        opacity: 0,
        filter: reducedMotion ? "none" : "blur(10px)",
        transition: {
          duration: reducedMotion ? 0.16 : 0.48,
          ease: "easeInOut",
        },
      }}
      initial={{ opacity: 0 }}
    >
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_left,_rgba(255,149,94,0.22),_transparent_30%),radial-gradient(circle_at_82%_18%,_rgba(159,215,104,0.12),_transparent_20%),linear-gradient(135deg,_#110e0d_0%,_#171312_38%,_#1d1715_100%)]" />
      <div className="absolute inset-0 bg-[linear-gradient(rgba(255,255,255,0.04)_1px,transparent_1px),linear-gradient(90deg,rgba(255,255,255,0.04)_1px,transparent_1px)] bg-[size:48px_48px] opacity-20" />
      <div className="bg-primary/18 absolute inset-x-[-20%] top-[14%] h-56 rounded-full blur-3xl" />
      <div className="bg-accent/12 absolute right-[-8%] bottom-[-14%] h-72 w-72 rounded-full blur-3xl" />

      <div className="relative flex min-h-screen items-center justify-center px-6">
        <motion.div
          animate={reducedMotion ? undefined : { opacity: [0.88, 1, 0.9] }}
          className="flex w-full max-w-md flex-col items-center text-center"
          initial={{
            opacity: 0.92,
            scale: reducedMotion ? 1 : 0.97,
            y: reducedMotion ? 0 : 12,
          }}
          transition={{
            duration: reducedMotion ? 0 : 1.6,
            ease: "easeInOut",
            repeat: reducedMotion ? 0 : Number.POSITIVE_INFINITY,
            repeatType: "mirror",
          }}
        >
          <div className="relative flex h-52 w-full items-center justify-center sm:h-56">
            <div className="absolute inset-x-6 top-1/2 h-px -translate-y-1/2 bg-gradient-to-r from-transparent via-white/18 to-transparent" />
            <motion.div
              animate={
                reducedMotion
                  ? { scale: 1, rotate: 0 }
                  : { scale: [0.97, 1, 0.985], rotate: [0, -1, 0] }
              }
              className="relative z-10"
              transition={{
                duration: 2.4,
                ease: "easeInOut",
                repeat: reducedMotion ? 0 : Number.POSITIVE_INFINITY,
              }}
            >
              <AnimatedLogo reducedMotion={reducedMotion} />
            </motion.div>
          </div>

          <motion.h1
            animate={reducedMotion ? undefined : { opacity: [0.82, 1, 0.86] }}
            className="mt-2 text-3xl font-semibold tracking-[0.45em] text-white uppercase sm:text-4xl"
            transition={{
              duration: reducedMotion ? 0 : 1.8,
              ease: "easeInOut",
              repeat: reducedMotion ? 0 : Number.POSITIVE_INFINITY,
            }}
          >
            GOROUTE
          </motion.h1>
          <p className="mt-3 max-w-xs text-sm leading-6 text-white/58 sm:text-base">
            Smart routing is getting everything ready for you.
          </p>
        </motion.div>
      </div>
    </motion.div>
  );
}

function AnimatedLogo({ reducedMotion }: { reducedMotion: boolean }) {
  return (
    <svg
      aria-hidden="true"
      className="h-40 w-40 drop-shadow-[0_26px_60px_rgba(223,122,82,0.34)] sm:h-44 sm:w-44"
      fill="none"
      viewBox="0 0 288 288"
      xmlns="http://www.w3.org/2000/svg"
    >
      <g filter="url(#boot-loader-filter0)">
        <rect
          fill="url(#boot-loader-paint0)"
          height="240"
          rx="84"
          width="240"
          x="24"
          y="12"
        />
      </g>
      <g filter="url(#boot-loader-filter1)">
        <rect
          fill="white"
          fillOpacity="0.08"
          height="232.4"
          rx="80"
          width="232.4"
          x="28"
          y="16"
        />
      </g>
      <motion.g
        animate={
          reducedMotion
            ? { x: 0, opacity: 1 }
            : { x: [-8, 10], opacity: [0.78, 1] }
        }
        filter="url(#boot-loader-filter2)"
        transition={{
          duration: reducedMotion ? 0 : 1.9,
          ease: [0.2, 0.9, 0.2, 1],
          repeat: reducedMotion ? 0 : Number.POSITIVE_INFINITY,
          repeatType: "mirror",
        }}
      >
        <path
          d="M103.698 68C105.22 68 106.659 68.6931 107.608 69.883L153.551 127.508C154.988 129.311 155.006 131.863 153.594 133.686L107.607 193.062C106.661 194.284 105.201 195 103.654 195H81.2101C77.0517 195 74.7106 190.219 77.2603 186.934L118.586 133.691C120.002 131.866 119.984 129.308 118.542 127.504L77.4885 76.121C74.8724 72.8467 77.2037 68 81.3948 68H103.698Z"
          fill="#F5F5F5"
        />
      </motion.g>
      <motion.g
        animate={
          reducedMotion
            ? { x: 0, opacity: 0.5 }
            : { x: [-14, 8], opacity: [0.3, 0.72] }
        }
        filter="url(#boot-loader-filter3)"
        transition={{
          delay: reducedMotion ? 0 : 0.12,
          duration: reducedMotion ? 0 : 2.35,
          ease: [0.2, 0.9, 0.2, 1],
          repeat: reducedMotion ? 0 : Number.POSITIVE_INFINITY,
          repeatType: "mirror",
        }}
      >
        <path
          d="M168.698 68C170.22 68 171.659 68.6931 172.608 69.883L218.551 127.508C219.988 129.311 220.006 131.863 218.594 133.686L172.607 193.062C171.661 194.284 170.201 195 168.654 195H146.21C142.052 195 139.711 190.219 142.26 186.934L183.586 133.691C185.002 131.866 184.984 129.308 183.542 127.504L142.489 76.121C139.872 72.8467 142.204 68 146.395 68H168.698Z"
          fill="#F5F5F5"
          fillOpacity="0.5"
          shapeRendering="crispEdges"
        />
      </motion.g>
      <defs>
        <filter
          colorInterpolationFilters="sRGB"
          filterUnits="userSpaceOnUse"
          height="288"
          id="boot-loader-filter0"
          width="288"
          x="0"
          y="0"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
          <feColorMatrix
            in="SourceAlpha"
            result="hardAlpha"
            type="matrix"
            values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0"
          />
          <feOffset dy="3" />
          <feGaussianBlur stdDeviation="3" />
          <feComposite in2="hardAlpha" operator="out" />
          <feColorMatrix
            type="matrix"
            values="0 0 0 0 1 0 0 0 0 1 0 0 0 0 1 0 0 0 0.18 0"
          />
          <feBlend
            in2="BackgroundImageFix"
            mode="normal"
            result="effect1_dropShadow_2002_1438"
          />
          <feColorMatrix
            in="SourceAlpha"
            result="hardAlpha"
            type="matrix"
            values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0"
          />
          <feOffset dy="12" />
          <feGaussianBlur stdDeviation="12" />
          <feComposite in2="hardAlpha" operator="out" />
          <feColorMatrix
            type="matrix"
            values="0 0 0 0 0.658824 0 0 0 0 0.301961 0 0 0 0 0.164706 0 0 0 0.35 0"
          />
          <feBlend
            in2="effect1_dropShadow_2002_1438"
            mode="normal"
            result="effect2_dropShadow_2002_1438"
          />
          <feBlend
            in="SourceGraphic"
            in2="effect2_dropShadow_2002_1438"
            mode="normal"
            result="shape"
          />
        </filter>
        <filter
          colorInterpolationFilters="sRGB"
          filterUnits="userSpaceOnUse"
          height="240.4"
          id="boot-loader-filter1"
          width="232.4"
          x="28"
          y="8"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
          <feBlend in2="BackgroundImageFix" mode="normal" result="shape" />
          <feColorMatrix
            in="SourceAlpha"
            result="hardAlpha"
            type="matrix"
            values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0"
          />
          <feOffset dy="-8" />
          <feGaussianBlur stdDeviation="8" />
          <feComposite in2="hardAlpha" k2="-1" k3="1" operator="arithmetic" />
          <feColorMatrix
            type="matrix"
            values="0 0 0 0 0.603922 0 0 0 0 0.243137 0 0 0 0 0.141176 0 0 0 0.35 0"
          />
          <feBlend
            in2="shape"
            mode="normal"
            result="effect1_innerShadow_2002_1438"
          />
        </filter>
        <filter
          colorInterpolationFilters="sRGB"
          filterUnits="userSpaceOnUse"
          height="159"
          id="boot-loader-filter2"
          width="110.439"
          x="60.2012"
          y="60"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
          <feColorMatrix
            in="SourceAlpha"
            result="hardAlpha"
            type="matrix"
            values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0"
          />
          <feOffset dy="8" />
          <feGaussianBlur stdDeviation="8" />
          <feComposite in2="hardAlpha" operator="out" />
          <feColorMatrix
            type="matrix"
            values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0.25 0"
          />
          <feBlend
            in2="BackgroundImageFix"
            mode="normal"
            result="effect1_dropShadow_2002_1438"
          />
          <feBlend
            in="SourceGraphic"
            in2="effect1_dropShadow_2002_1438"
            mode="normal"
            result="shape"
          />
        </filter>
        <filter
          colorInterpolationFilters="sRGB"
          filterUnits="userSpaceOnUse"
          height="159"
          id="boot-loader-filter3"
          width="110.439"
          x="125.201"
          y="60"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
          <feColorMatrix
            in="SourceAlpha"
            result="hardAlpha"
            type="matrix"
            values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0"
          />
          <feOffset dy="8" />
          <feGaussianBlur stdDeviation="8" />
          <feComposite in2="hardAlpha" operator="out" />
          <feColorMatrix
            type="matrix"
            values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0.15 0"
          />
          <feBlend
            in2="BackgroundImageFix"
            mode="normal"
            result="effect1_dropShadow_2002_1438"
          />
          <feBlend
            in="SourceGraphic"
            in2="effect1_dropShadow_2002_1438"
            mode="normal"
            result="shape"
          />
        </filter>
        <linearGradient
          gradientUnits="userSpaceOnUse"
          id="boot-loader-paint0"
          x1="24"
          x2="264"
          y1="12"
          y2="252"
        >
          <stop stopColor="#FF955E" />
          <stop offset="1" stopColor="#DF7A52" />
        </linearGradient>
      </defs>
    </svg>
  );
}
