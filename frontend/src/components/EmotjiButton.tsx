import { Frown, Smile, Angry, type LucideIcon } from "lucide-react";

export type EmojiType = "sad" | "happy" | "angry";

const EMOJI_MAP: Record<EmojiType, LucideIcon> = {
  sad: Frown,
  happy: Smile,
  angry: Angry,
};

interface Props {
  onClick: () => void;
  emoji: EmojiType | string;
  className?: string;
  color?: string;
}

const EmojiButton = ({ onClick, emoji, className = "", color }: Props) => {
  const IconComponent = EMOJI_MAP[emoji as EmojiType];

  if (!IconComponent) {
    return null;
  }

  return (
    <button
      type="button"
      onClick={onClick}
      className={className}
      aria-label={`Выбрать эмоцию ${emoji}`}
    >
      <IconComponent size={24} color={color} />
    </button>
  );
};

export default EmojiButton;
