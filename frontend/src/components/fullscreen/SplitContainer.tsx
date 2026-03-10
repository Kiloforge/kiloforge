import type { PaneNode } from "../../hooks/useCommandPanes";
import type { Agent } from "../../types/api";
import { CommandPane } from "./CommandPane";
import styles from "./FullScreenCommand.module.css";

interface Props {
  node: PaneNode;
  agents: Agent[];
  activePaneId: string;
  onFocusPane: (id: string) => void;
  onAgentChange: (paneId: string, agentId: string | null) => void;
  onClosePane: (id: string) => void;
  leafCount: number;
}

export function SplitContainer({
  node,
  agents,
  activePaneId,
  onFocusPane,
  onAgentChange,
  onClosePane,
  leafCount,
}: Props) {
  if (node.kind === "leaf") {
    return (
      <CommandPane
        paneId={node.id}
        agentId={node.agentId}
        agents={agents}
        isFocused={activePaneId === node.id}
        onFocus={() => onFocusPane(node.id)}
        onAgentChange={(agentId) => onAgentChange(node.id, agentId)}
        onClose={() => onClosePane(node.id)}
        showCloseBtn={leafCount > 1}
      />
    );
  }

  const dirClass = node.direction === "horizontal" ? styles.splitHorizontal : styles.splitVertical;
  const dividerClass = node.direction === "horizontal" ? styles.dividerH : styles.dividerV;

  return (
    <div className={dirClass}>
      <div style={{ flex: node.ratio }}>
        <SplitContainer
          node={node.children[0]}
          agents={agents}
          activePaneId={activePaneId}
          onFocusPane={onFocusPane}
          onAgentChange={onAgentChange}
          onClosePane={onClosePane}
          leafCount={leafCount}
        />
      </div>
      <div className={dividerClass} />
      <div style={{ flex: 1 - node.ratio }}>
        <SplitContainer
          node={node.children[1]}
          agents={agents}
          activePaneId={activePaneId}
          onFocusPane={onFocusPane}
          onAgentChange={onAgentChange}
          onClosePane={onClosePane}
          leafCount={leafCount}
        />
      </div>
    </div>
  );
}
