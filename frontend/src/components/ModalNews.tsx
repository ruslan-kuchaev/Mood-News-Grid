import { useEffect, useState } from "react";
import * as React from "react";
import type { NewsArticle, NewsFact, RewriteResponse } from "../type";
import { ChoicePanel } from "./ChoicePanel";

interface ModalNewsProps {
  open: boolean;
  content: NewsArticle | null;
  facts?: NewsFact;
  onClose: () => void;
}

export default function ModalNews({
  open,
  content,
  facts,
  onClose,
}: ModalNewsProps) {
  const [rewrittenText, setRewrittenText] = useState<RewriteResponse | null>(
    null,
  );
  const [isRewriting, setIsRewriting] = useState(false);
  const [validation, setValidation] = useState<Record<string, boolean>>({});
  const abortControllerRef = React.useRef<AbortController | null>(null);

  useEffect(() => {
    if (!open) return;

    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape") onClose();
    }

    document.addEventListener("keydown", handleKeyDown);
    document.body.style.overflow = "hidden";

    return () => {
      document.removeEventListener("keydown", handleKeyDown);
      document.body.style.overflow = "";
    };
  }, [open, onClose]);

  useEffect(() => {
    if (!open) {
      // Отменяем запрос если модалка закрывается
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }
      setRewrittenText(null);
      setIsRewriting(false);
      setValidation({});
    }
  }, [open]);

  const validateFacts = (rewrittenContent: string, facts: NewsFact) => {
    const lowerContent = rewrittenContent.toLowerCase();
    const results: Record<string, boolean> = {};

    facts.dates?.forEach((date, idx) => {
      results[`date_${idx}`] = lowerContent.includes(date.toLowerCase());
    });

    facts.people?.forEach((person, idx) => {
      results[`person_${idx}`] = lowerContent.includes(person.toLowerCase());
    });

    facts.organizations?.forEach((org, idx) => {
      results[`org_${idx}`] = lowerContent.includes(org.toLowerCase());
    });

    facts.locations?.forEach((loc, idx) => {
      results[`location_${idx}`] = lowerContent.includes(loc.toLowerCase());
    });

    facts.keywords?.forEach((keyword, idx) => {
      results[`keyword_${idx}`] = lowerContent.includes(keyword.toLowerCase());
    });

    return results;
  };

  const handleEmotionSelect = async (emotion: "happy" | "sad" | "angry") => {
    if (!content || !content.url_hash) return;
    if (isRewriting) return;

    // Отменяем предыдущий запрос если есть
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
    }

    const abortController = new AbortController();
    abortControllerRef.current = abortController;

    setIsRewriting(true);
    setRewrittenText(null);

    try {
      const apiUrl = import.meta.env.VITE_API_URL;
      const response = await fetch(`${apiUrl}/rewrite`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          url_hash: content.url_hash,
          emotion: emotion,
          title: content.title,
          content: content.content,
        }),
        signal: abortController.signal,
      });

      if (!response.ok) {
        throw new Error(`Failed to rewrite: ${response.status}`);
      }

      const data: RewriteResponse = await response.json();
      
      // Проверяем что запрос не был отменен
      if (!abortController.signal.aborted) {
        setRewrittenText(data);

        if (facts) {
          const validationResults = validateFacts(data.rewritten_content, facts);
          setValidation(validationResults);
        }
      }
    } catch (error) {
      if (error instanceof Error && error.name === 'AbortError') {
        console.log('[Rewrite] Запрос был отменен');
        return;
      }
      console.error("[Rewrite] Error:", error);
    } finally {
      if (!abortController.signal.aborted) {
        setIsRewriting(false);
      }
      abortControllerRef.current = null;
    }
  };

  if (!open || !content) return null;

  const formattedDate = formatDate(content.pubdate || "");

  return (
    <div className="fixed inset-0 z-30 flex items-center justify-center p-4">
      <div
        className="absolute inset-0 bg-black/70 backdrop-blur-sm animate-in fade-in duration-200"
        onClick={onClose}
      />
      <div className="flex flex-col items-center justify-center py-8">
        <ChoicePanel onEmotionSelect={handleEmotionSelect} />
      </div>

      <div className="relative z-10 w-full max-w-7xl max-h-[90vh] overflow-hidden rounded-2xl bg-neutral-900 border border-neutral-800 shadow-2xl">
        <button
          type="button"
          onClick={onClose}
          className="absolute top-4 right-4 z-20 text-neutral-500 hover:text-white transition-colors text-xl leading-none"
          aria-label="Закрыть"
        >
          ✕
        </button>

        <div className="flex h-[90vh] overflow-hidden">
          <div className="flex-1 overflow-y-auto p-6 pr-4">
            <div className="flex items-center justify-between text-xs text-neutral-500 mb-3 pr-8">
              <span>{content.author}</span>
              <span>{formattedDate}</span>
            </div>

            <h2 className="text-2xl font-semibold text-neutral-100 leading-snug mb-4 pr-8">
              {content.title}
            </h2>

            <p className="text-sm text-neutral-300 leading-relaxed whitespace-pre-line mb-6">
              {content.content}
            </p>

            {isRewriting && (
              <div className="mt-8 pt-6 border-t border-neutral-700">
                <div className="flex items-center gap-2 mb-4">
                  <div className="w-2 h-2 bg-blue-500 rounded-full animate-pulse" />
                  <span className="text-xs text-neutral-400">
                    Переписываем текст...
                  </span>
                </div>
                <div className="space-y-3">
                  <div className="h-4 bg-neutral-800 rounded animate-pulse w-full" />
                  <div className="h-4 bg-neutral-800 rounded animate-pulse w-11/12" />
                  <div className="h-4 bg-neutral-800 rounded animate-pulse w-full" />
                  <div className="h-4 bg-neutral-800 rounded animate-pulse w-10/12" />
                  <div className="h-4 bg-neutral-800 rounded animate-pulse w-full" />
                  <div className="h-4 bg-neutral-800 rounded animate-pulse w-9/12" />
                </div>
              </div>
            )}

            {rewrittenText && !isRewriting && (
              <div className="mt-8 pt-6 border-t border-neutral-700">
                <div className="flex items-center gap-2 mb-4">
                  <span className="text-xs font-medium text-blue-400">
                    {rewrittenText.emotion === "happy" &&
                      "😊 Позитивная версия"}
                    {rewrittenText.emotion === "sad" && "😔 Грустная версия"}
                    {rewrittenText.emotion === "angry" && "😠 Критичная версия"}
                  </span>
                </div>
                <p className="text-sm text-neutral-300 leading-relaxed whitespace-pre-line">
                  {rewrittenText.rewritten_content}
                </p>
              </div>
            )}

            <a
              href={content.url || "#"}
              target="_blank"
              rel="noreferrer"
              className="inline-block mt-6 text-sm text-neutral-400 hover:text-white underline underline-offset-4 transition-colors"
            >
              Читать источник →
            </a>
          </div>

          <div className="w-96 h-auto border-l border-neutral-800 bg-neutral-950/50 overflow-y-auto p-6">
            <h3 className="text-lg font-semibold text-neutral-100 mb-4 flex items-center gap-2">
              Ключевые факты
            </h3>

            {facts ? (
              <div className="space-y-4">
                <FactSection
                  title="Даты"
                  items={facts.dates?.length ? facts.dates : []}
                  validation={validation}
                  prefix="date"
                />

                <FactSection
                  title="Люди"
                  items={facts.people?.length ? facts.people : []}
                  validation={validation}
                  prefix="person"
                />

                <FactSection
                  title="Организации"
                  items={facts.organizations?.length ? facts.organizations : []}
                  validation={validation}
                  prefix="org"
                />

                <FactSection
                  title="Места"
                  items={facts.locations?.length ? facts.locations : []}
                  validation={validation}
                  prefix="location"
                />

                <FactSection
                  title="Ключевые слова"
                  items={facts.keywords?.length ? facts.keywords : []}
                  validation={validation}
                  prefix="keyword"
                />

                <FactSection
                  title="Категории"
                  items={facts.categories?.length ? facts.categories : []}
                  validation={{}}
                  prefix=""
                />

                <div className="p-4 rounded-lg bg-neutral-800/50 border border-neutral-700">
                  <p className="text-xs font-medium text-neutral-400 mb-2">
                    Главная мысль:
                  </p>
                  <p className="text-sm text-neutral-200 leading-relaxed">
                    {facts.main_thesis}
                  </p>
                </div>

                <div>
                  <p className="text-xs font-medium text-neutral-400 mb-2">
                    Ключевые цифры:
                  </p>
                  <div className="space-y-2">
                    {(facts.key_numbers ?? []).map((num, idx) => (
                      <div
                        key={idx}
                        className="p-3 rounded-lg bg-blue-500/10 border border-blue-500/20"
                      >
                        <span className="text-blue-400 font-semibold block mb-1">
                          {num.value}
                        </span>
                        <span className="text-xs text-neutral-400">
                          {num.context}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>

                <div className="pt-4 border-t border-neutral-800">
                  <p className="text-xs text-neutral-500">
                    Обработано:{" "}
                    {new Date(facts.processed_at).toLocaleString("ru-RU")}
                  </p>
                </div>
              </div>
            ) : (
              <div className="flex flex-col items-center justify-center py-12 text-center">
                <span className="text-sm text-yellow-500">
                  Анализируем факты...
                </span>
                <p className="text-xs text-neutral-500 mt-2">
                  Это может занять несколько секунд
                </p>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function FactSection({
  title,
  items,
  validation,
  prefix,
}: {
  title: string;
  items: string[];
  validation: Record<string, boolean>;
  prefix: string;
}) {
  return (
    <div className="p-3 rounded-lg bg-neutral-800/30 border border-neutral-700/50">
      <p className="text-xs font-medium text-neutral-400 mb-2">{title}</p>
      <ul className="space-y-1">
        {items.map((item, idx) => {
          // 1. Формируем ключ: например "date_0" или "person_1"
          const key = `${prefix}_${idx}`;

          // 2. Получаем результат валидации (true, false или undefined)
          const isValid = validation[key];

          return (
            <li
              key={idx}
              className={`px-2 py-1.5 text-xs rounded transition-colors ${
                isValid === true
                  ? "bg-green-500/20 text-green-300 border border-green-500/30"
                  : isValid === false
                    ? "bg-red-500/20 text-red-300 border border-red-500/30"
                    : "bg-neutral-700/50 text-neutral-200"
              }`}
            >
              {item}
            </li>
          );
        })}
      </ul>
    </div>
  );
}

function formatDate(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleDateString("ru-RU", {
    day: "numeric",
    month: "short",
    year: "numeric",
  });
}
