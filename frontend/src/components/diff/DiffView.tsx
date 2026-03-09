import { useState, useRef, useCallback } from "react";
import { useProjectDiff } from "../../hooks/useDiff";
import { DiffStats } from "./DiffStats";
import { FileList } from "./FileList";
import { FileDiff } from "./FileDiff";
import styles from "./DiffView.module.css";

interface Props {
  slug: string;
  branch: string;
  onDiscuss?: () => void;
}

export function DiffView({ slug, branch, onDiscuss }: Props) {
  const { data, isLoading, error } = useProjectDiff(slug, branch);
  const [activeFile, setActiveFile] = useState(0);
  const fileRefs = useRef<Map<number, HTMLDivElement>>(new Map());

  const handleFileSelect = useCallback((index: number) => {
    setActiveFile(index);
    const el = fileRefs.current.get(index);
    el?.scrollIntoView({ behavior: "smooth", block: "start" });
  }, []);

  const setFileRef = useCallback((index: number, el: HTMLDivElement | null) => {
    if (el) {
      fileRefs.current.set(index, el);
    } else {
      fileRefs.current.delete(index);
    }
  }, []);

  if (isLoading) {
    return <div className={styles.loading}>Loading diff...</div>;
  }

  if (error) {
    return <div className={styles.error}>Failed to load diff: {error.message}</div>;
  }

  if (!data || data.files.length === 0) {
    return <div className={styles.empty}>No changes on this branch</div>;
  }

  return (
    <div className={styles.container}>
      <div className={styles.topBar}>
        <DiffStats stats={data.stats} truncated={data.truncated} />
        {onDiscuss && (
          <button className={styles.discussBtn} onClick={onDiscuss}>
            Discuss
          </button>
        )}
      </div>
      <div className={styles.layout}>
        <div className={styles.sidebar}>
          <FileList files={data.files} activeIndex={activeFile} onSelect={handleFileSelect} />
        </div>
        <div className={styles.content}>
          {data.files.map((file, i) => (
            <div key={file.path} ref={(el) => setFileRef(i, el)}>
              <FileDiff file={file} />
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
