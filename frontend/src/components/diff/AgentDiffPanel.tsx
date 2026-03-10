import { useState, useCallback } from "react";
import { useProjectDiff } from "../../hooks/useDiff";
import { FileList } from "./FileList";
import { FileDiff } from "./FileDiff";
import styles from "./AgentDiffPanel.module.css";

interface Props {
  slug: string;
  branch: string;
}

export function AgentDiffPanel({ slug, branch }: Props) {
  const { data, isLoading, error } = useProjectDiff(slug, branch);
  const [activeFile, setActiveFile] = useState(0);

  const handleFileSelect = useCallback((index: number) => {
    setActiveFile(index);
  }, []);

  if (isLoading) {
    return <div className={styles.empty}>Loading diff...</div>;
  }

  if (error) {
    return <div className={styles.error}>Failed to load diff: {error.message}</div>;
  }

  if (!data || data.files.length === 0) {
    return <div className={styles.empty}>No changes on this branch</div>;
  }

  const selectedFile = data.files[activeFile];

  return (
    <div className={styles.container}>
      <div className={styles.sidebar}>
        <FileList files={data.files} activeIndex={activeFile} onSelect={handleFileSelect} />
      </div>
      <div className={styles.content}>
        {selectedFile && <FileDiff file={selectedFile} />}
      </div>
    </div>
  );
}
