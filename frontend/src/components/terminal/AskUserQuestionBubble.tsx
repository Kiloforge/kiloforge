import { useState } from "react";
import type { WSMessage } from "../../hooks/useAgentWebSocket";
import styles from "./AskUserQuestionBubble.module.css";

interface AskOption {
  label: string;
  description?: string;
}

interface Props {
  msg: WSMessage;
  onSend: (text: string) => void;
}

export function AskUserQuestionBubble({ msg, onSend }: Props) {
  const [selectedLabel, setSelectedLabel] = useState<string | null>(null);

  const question = typeof msg.toolInput?.question === "string"
    ? msg.toolInput.question
    : "";

  const options: AskOption[] = Array.isArray(msg.toolInput?.options)
    ? (msg.toolInput.options as AskOption[])
    : [];

  const answered = selectedLabel !== null;

  const handleClick = (label: string) => {
    if (answered) return;
    setSelectedLabel(label);
    onSend(label);
  };

  return (
    <div className={styles.container}>
      <span className={styles.icon}>ask</span>
      <div className={styles.body}>
        <p className={styles.question}>{question}</p>
        {options.length > 0 && (
          <div className={styles.options}>
            {options.map((opt) => (
              <button
                key={opt.label}
                className={`${styles.optionBtn} ${answered ? styles.answered : ""} ${selectedLabel === opt.label ? styles.selected : ""}`}
                onClick={() => handleClick(opt.label)}
                disabled={answered}
              >
                <span className={styles.optionLabel}>{opt.label}</span>
                {opt.description && (
                  <span className={styles.optionDesc}>{opt.description}</span>
                )}
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
