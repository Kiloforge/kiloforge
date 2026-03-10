import { useEffect, useState, useRef } from "react";
import { useTourContext } from "./TourProvider";
import { TOUR_STEPS } from "./tourSteps";
import styles from "./TourOverlay.module.css";

export function TourOverlay() {
  const { isActive, isPending, currentStep, totalSteps, startTour, dismissTour, nextStep, completeTour, tourState } = useTourContext();
  const [targetRect, setTargetRect] = useState<DOMRect | null>(null);
  const tooltipRef = useRef<HTMLDivElement>(null);

  const step = TOUR_STEPS[currentStep];
  const isWelcome = step?.id === "welcome";
  const isLast = currentStep === totalSteps - 1;

  // Find and track the target element
  useEffect(() => {
    if (!isActive || !step || isWelcome) {
      setTargetRect(null);
      return;
    }

    const findTarget = () => {
      const el = document.querySelector(step.target);
      if (el) {
        setTargetRect(el.getBoundingClientRect());
      } else {
        setTargetRect(null);
      }
    };

    findTarget();
    const interval = setInterval(findTarget, 500);
    window.addEventListener("resize", findTarget);
    window.addEventListener("scroll", findTarget, true);

    return () => {
      clearInterval(interval);
      window.removeEventListener("resize", findTarget);
      window.removeEventListener("scroll", findTarget, true);
    };
  }, [isActive, step, isWelcome, currentStep]);

  // Show welcome dialog
  if (isPending) {
    return (
      <div className={styles.overlay}>
        <div className={styles.welcomeDialog}>
          <h2 className={styles.welcomeTitle}>Welcome to Kiloforge!</h2>
          <p className={styles.welcomeText}>
            Take a quick guided tour to learn how to set up projects, generate implementation tracks,
            and manage your AI development agents.
          </p>
          <div className={styles.welcomeActions}>
            <button className={styles.startBtn} onClick={startTour}>
              Start Tour
            </button>
            <button className={styles.skipBtn} onClick={dismissTour}>
              Skip
            </button>
          </div>
        </div>
      </div>
    );
  }

  // Tour completed celebration
  if (tourState.status === "completed" && !tourState.completed_at) {
    return null; // Will be handled by completion state
  }

  if (!isActive || !step) return null;

  // Welcome step (centered dialog, no spotlight)
  if (isWelcome) {
    return (
      <div className={styles.overlay}>
        <div className={styles.welcomeDialog}>
          <h2 className={styles.welcomeTitle}>{step.title}</h2>
          <p className={styles.welcomeText}>{step.content}</p>
          <div className={styles.welcomeActions}>
            <button className={styles.startBtn} onClick={nextStep}>
              Let's Go
            </button>
            <button className={styles.skipBtn} onClick={dismissTour}>
              Skip Tour
            </button>
          </div>
        </div>
      </div>
    );
  }

  // Spotlight step
  const pad = 8;
  const spotlightStyle = targetRect
    ? {
        top: targetRect.top - pad + window.scrollY,
        left: targetRect.left - pad,
        width: targetRect.width + pad * 2,
        height: targetRect.height + pad * 2,
      }
    : undefined;

  // Tooltip positioning
  const placement = step.placement ?? "bottom";
  const tooltipStyle = targetRect
    ? computeTooltipPosition(targetRect, placement, pad)
    : { top: "50%", left: "50%", transform: "translate(-50%, -50%)" };

  const isWaitStep = step.action === "wait-for-drag";

  return (
    <>
      {/* Backdrop with spotlight cutout */}
      <div className={`${styles.backdrop}${spotlightStyle ? "" : ` ${styles.backdropNoTarget}`}`}>
        {spotlightStyle && (
          <div className={styles.spotlight} style={spotlightStyle} />
        )}
      </div>

      {/* Tooltip */}
      <div
        ref={tooltipRef}
        className={`${styles.tooltip} ${styles[`tooltip_${placement}`]}`}
        style={tooltipStyle as React.CSSProperties}
      >
        <div className={styles.tooltipHeader}>
          <span className={styles.stepCount}>
            {currentStep + 1} / {totalSteps}
          </span>
          <button className={styles.closeBtn} onClick={dismissTour} title="Close tour">
            &times;
          </button>
        </div>
        <h3 className={styles.tooltipTitle}>{step.title}</h3>
        <p className={styles.tooltipContent}>{step.content}</p>
        {!isWaitStep ? (
          <div className={styles.tooltipActions}>
            <button className={styles.nextBtn} onClick={isLast ? completeTour : nextStep}>
              {isLast ? "Finish" : "Next"}
            </button>
          </div>
        ) : (
          <div className={styles.tooltipActions}>
            <button className={styles.skipLink} onClick={completeTour}>
              Skip and finish tour
            </button>
          </div>
        )}
      </div>
    </>
  );
}

function computeTooltipPosition(
  rect: DOMRect,
  placement: string,
  pad: number,
): Record<string, number | string> {
  const gap = 12;
  switch (placement) {
    case "top":
      return {
        top: rect.top - gap - pad + window.scrollY,
        left: rect.left + rect.width / 2,
        transform: "translate(-50%, -100%)",
      };
    case "left":
      return {
        top: rect.top + rect.height / 2 + window.scrollY,
        left: rect.left - gap - pad,
        transform: "translate(-100%, -50%)",
      };
    case "right":
      return {
        top: rect.top + rect.height / 2 + window.scrollY,
        left: rect.right + gap + pad,
        transform: "translateY(-50%)",
      };
    case "bottom":
    default:
      return {
        top: rect.bottom + gap + pad + window.scrollY,
        left: rect.left + rect.width / 2,
        transform: "translateX(-50%)",
      };
  }
}

