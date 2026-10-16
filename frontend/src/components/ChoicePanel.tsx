import { useState, useRef, useEffect } from "react";
import EmojiButton from "./EmotjiButton";

type EmotionKey = "happy" | "sad" | "angry";

interface Emotion {
  key: EmotionKey;
  from: string;
  to: string;
  shadow: string;
}

const emotions: Emotion[] = [
  {
    key: "happy",
    from: "from-green-400",
    to: "to-emerald-600",
    shadow: "shadow-emerald-500/50",
  },
  {
    key: "sad",
    from: "from-blue-400",
    to: "to-indigo-600",
    shadow: "shadow-indigo-500/50",
  },
  {
    key: "angry",
    from: "from-red-400",
    to: "to-rose-600",
    shadow: "shadow-rose-500/50",
  },
];

interface FlyingState {
  emotion: Emotion;
  startTop: number;
  startLeft: number;
}

interface ChoicePanelProps {
  onEmotionSelect?: (emotion: EmotionKey) => void;
}

export const ChoicePanel = ({ onEmotionSelect }: ChoicePanelProps) => {
  const [selected, setSelected] = useState<EmotionKey | null>(null);
  const [flying, setFlying] = useState<FlyingState | null>(null);
  const slotRef = useRef<HTMLDivElement | null>(null);
  const buttonRefs = useRef<Record<string, HTMLButtonElement | null>>({});

  const handleSelect = (emotion: Emotion) => {
    if (emotion.key === selected) return;

    const btn = buttonRefs.current[emotion.key];
    const slot = slotRef.current;
    if (!btn || !slot) return;

    const startRect = btn.getBoundingClientRect();

    setFlying({
      emotion,
      startTop: startRect.top,
      startLeft: startRect.left,
    });
    setSelected(emotion.key);

    if (onEmotionSelect) {
      onEmotionSelect(emotion.key);
    }
  };

  const selectedEmotion = emotions.find((e) => e.key === selected);

  return (
    <div className="flex flex-col items-center gap-16 px-4 py-6">
      {/* Слот назначения */}
      <div
        ref={slotRef}
        className={`
          relative w-14 h-14 rounded-full flex items-center justify-center
          border-2 border-dashed border-gray-400/50 bg-gray-500/10
          transition-all duration-300
          ${selectedEmotion ? "border-none" : ""}
        `}
      >
        {selectedEmotion && (
          <div
            key={selectedEmotion.key}
            className={`
              absolute inset-0 rounded-full flex items-center justify-center
              bg-gradient-to-br ${selectedEmotion.from} ${selectedEmotion.to}
              ${selectedEmotion.shadow} shadow-lg
              animate-[popIn_0.3s_ease-out]
            `}
          >
            <div className="flex items-center justify-center text-2xl pointer-events-none">
              <EmojiButton
                onClick={() => {}}
                emoji={selectedEmotion.key}
                color="white"
                className="pointer-events-none"
              />
            </div>
          </div>
        )}
      </div>

      {/* Столбик выбора */}
      <div className="flex flex-col-reverse items-center gap-3">
        {emotions.map((e) => {
          const isHidden = selected === e.key;

          return (
            <button
              key={e.key}
              ref={(el) => {
                buttonRefs.current[e.key] = el;
              }}
              onClick={() => handleSelect(e)}
              disabled={isHidden}
              aria-label={`Выбрать эмоцию ${e.key}`}
              className={`
                w-14 h-14 rounded-full flex items-center justify-center
                bg-gradient-to-br ${e.from} ${e.to} ${e.shadow}
                transition-all duration-300 ease-out shadow-lg
                ${
                  isHidden
                    ? "opacity-0 scale-50 pointer-events-none"
                    : "opacity-100 scale-100 hover:scale-110"
                }
              `}
            >
              <div className="flex items-center justify-center text-2xl pointer-events-none">
                <EmojiButton
                  onClick={() => {}}
                  emoji={e.key}
                  color="white"
                  className="pointer-events-none"
                />
              </div>
            </button>
          );
        })}
      </div>

      {/* Летящая копия */}
      {flying && (
        <FlyingEmoji
          key={flying.emotion.key + flying.startTop}
          emotion={flying.emotion}
          startTop={flying.startTop}
          startLeft={flying.startLeft}
          targetRef={slotRef}
          onDone={() => setFlying(null)}
        />
      )}
    </div>
  );
};

interface FlyingEmojiProps {
  emotion: Emotion;
  startTop: number;
  startLeft: number;
  targetRef: React.RefObject<HTMLDivElement | null>;
  onDone: () => void;
}

const FlyingEmoji = ({ emotion, startTop, startLeft, targetRef, onDone }: FlyingEmojiProps) => {
  const [pos, setPos] = useState({
    top: startTop,
    left: startLeft,
    opacity: 1,
    scale: 1,
  });

  useEffect(() => {
    if (!targetRef.current) return;
    const rect = targetRef.current.getBoundingClientRect();
    requestAnimationFrame(() => {
      setPos({ top: rect.top, left: rect.left, opacity: 0, scale: 0.8 });
    });
  }, [targetRef]);

  return (
    <div
      onTransitionEnd={onDone}
      className={`
        fixed w-14 h-14 rounded-full flex items-center justify-center
        bg-gradient-to-br ${emotion.from} ${emotion.to} shadow-lg
        transition-all duration-500 ease-in-out pointer-events-none z-50
      `}
      style={{
        top: pos.top,
        left: pos.left,
        opacity: pos.opacity,
        transform: `scale(${pos.scale})`,
      }}
    >
      <div className="flex items-center justify-center text-2xl">
        <EmojiButton
          onClick={() => {}}
          emoji={emotion.key}
          color="white"
          className="pointer-events-none"
        />
      </div>
    </div>
  );
};
