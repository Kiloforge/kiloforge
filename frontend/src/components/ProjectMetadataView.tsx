import { useState, useMemo } from "react";
import type { ProjectMetadata } from "../types/api";
import { MarkdownContent } from "./terminal/MarkdownContent";
import styles from "./ProjectMetadataView.module.css";

interface Props {
  metadata: ProjectMetadata;
}

type TabId = "product" | "guidelines" | "tech_stack" | "workflow" | "style_guides";

interface TabDef {
  id: TabId;
  label: string;
  available: boolean;
}

export function ProjectMetadataView({ metadata }: Props) {
  const tabs = useMemo<TabDef[]>(() => {
    const t: TabDef[] = [
      { id: "product", label: "Product", available: true },
      { id: "guidelines", label: "Guidelines", available: !!metadata.product_guidelines },
      { id: "tech_stack", label: "Tech Stack", available: true },
      { id: "workflow", label: "Workflow", available: !!metadata.workflow },
      { id: "style_guides", label: "Style Guides", available: !!metadata.style_guides?.length },
    ];
    return t.filter((tab) => tab.available);
  }, [metadata]);

  const [activeTab, setActiveTab] = useState<TabId>("product");
  const [activeGuide, setActiveGuide] = useState(0);

  // If active tab is no longer available, reset to first
  const currentTab = tabs.find((t) => t.id === activeTab) ? activeTab : tabs[0]?.id ?? "product";

  const { track_summary: summary } = metadata;
  const done = summary.completed + summary.archived;

  return (
    <div>
      {/* Track summary stats */}
      <div className={styles.summaryRow}>
        <div className={styles.summaryItem}>
          <span>Done</span>
          <span className={`${styles.summaryValue} ${styles.summaryCompleted}`}>{done}/{summary.total}</span>
        </div>
        <div className={styles.summaryItem}>
          <span>In Progress</span>
          <span className={`${styles.summaryValue} ${styles.summaryProgress}`}>{summary.in_progress}</span>
        </div>
        <div className={styles.summaryItem}>
          <span>Pending</span>
          <span className={`${styles.summaryValue} ${styles.summaryPending}`}>{summary.pending}</span>
        </div>
      </div>

      {/* Quick links */}
      {metadata.quick_links.length > 0 && (
        <div className={styles.quickLinks}>
          {metadata.quick_links.map((link) => (
            <a
              key={link.path}
              href={link.path}
              className={styles.quickLink}
              target="_blank"
              rel="noopener noreferrer"
            >
              {link.label}
            </a>
          ))}
        </div>
      )}

      {/* Tabs */}
      <div className={styles.tabs}>
        {tabs.map((tab) => (
          <button
            key={tab.id}
            className={`${styles.tab} ${currentTab === tab.id ? styles.tabActive : ""}`}
            onClick={() => setActiveTab(tab.id)}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* Tab content */}
      <div className={styles.content}>
        {currentTab === "product" && (
          <MarkdownContent text={metadata.product} />
        )}
        {currentTab === "guidelines" && metadata.product_guidelines && (
          <MarkdownContent text={metadata.product_guidelines} />
        )}
        {currentTab === "tech_stack" && (
          <MarkdownContent text={metadata.tech_stack} />
        )}
        {currentTab === "workflow" && metadata.workflow && (
          <MarkdownContent text={metadata.workflow} />
        )}
        {currentTab === "style_guides" && metadata.style_guides && (
          <>
            {metadata.style_guides.length > 1 && (
              <div className={styles.subTabs}>
                {metadata.style_guides.map((guide, i) => (
                  <button
                    key={guide.name}
                    className={`${styles.subTab} ${activeGuide === i ? styles.subTabActive : ""}`}
                    onClick={() => setActiveGuide(i)}
                  >
                    {guide.name}
                  </button>
                ))}
              </div>
            )}
            {metadata.style_guides[activeGuide] && (
              <MarkdownContent text={metadata.style_guides[activeGuide].content} />
            )}
          </>
        )}
      </div>
    </div>
  );
}
