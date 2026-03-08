import type { ConfigResponse, UpdateConfigRequest } from "../types/api";
import styles from "./TracingToggle.module.css";

interface Props {
  config: ConfigResponse | null;
  loading: boolean;
  updating: boolean;
  onUpdate: (req: UpdateConfigRequest) => Promise<boolean>;
}

export function TracingToggle({ config, loading, updating, onUpdate }: Props) {
  const enabled = config?.tracing_enabled ?? true;
  const disabled = loading || updating;

  const handleClick = () => {
    if (disabled) return;
    onUpdate({ tracing_enabled: !enabled });
  };

  return (
    <div className={styles.container}>
      <span className={styles.label}>Tracing</span>
      <button
        className={styles.toggle}
        data-enabled={enabled}
        disabled={disabled}
        onClick={handleClick}
        aria-label={`Tracing ${enabled ? "enabled" : "disabled"}`}
      >
        <span className={styles.knob} />
      </button>
      <span className={styles.label}>{enabled ? "On" : "Off"}</span>
      <span className={styles.note}>Changes take effect on next restart</span>
    </div>
  );
}
