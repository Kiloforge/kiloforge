import styles from "./SlashCommandAutocomplete.module.css";

export interface SlashCommand {
  slashCommand: string;
  label: string;
  description: string;
}

interface Props {
  input: string;
  commands: SlashCommand[];
  selectedIndex: number;
  onSelect: (command: string) => void;
}

export function SlashCommandAutocomplete({ input, commands, selectedIndex, onSelect }: Props) {
  if (!input.startsWith("/")) return null;

  const query = input.toLowerCase();
  const filtered = commands.filter((c) => c.slashCommand.toLowerCase().startsWith(query));
  if (filtered.length === 0) return null;

  const wrappedIndex = selectedIndex % filtered.length;

  return (
    <ul className={styles.dropdown} role="listbox">
      {filtered.map((cmd, i) => (
        <li
          key={cmd.slashCommand}
          role="option"
          aria-selected={i === wrappedIndex}
          className={`${styles.item} ${i === wrappedIndex ? styles.selected : ""}`}
          onPointerDown={(e) => {
            e.preventDefault();
            onSelect(cmd.slashCommand);
          }}
        >
          <span className={styles.command}>{cmd.slashCommand}</span>
          <span className={styles.description}>{cmd.description}</span>
        </li>
      ))}
    </ul>
  );
}
