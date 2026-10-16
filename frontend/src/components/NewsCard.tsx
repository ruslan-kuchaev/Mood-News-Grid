import type { NewsArticle, NewsFact } from "../type";

interface NewsCardProps {
  article: NewsArticle;
  facts?: NewsFact;
  hasFacts: boolean;
  onClick: () => void;
  hasError: boolean;
}

function NewsCard({
  article,
  facts,
  hasFacts,
  onClick,
  hasError,
}: NewsCardProps) {
  const formattedDate = formatDate(article.pubdate || "");

  return (
    <div
      onClick={onClick}
      className="relative flex flex-col gap-3 rounded-2xl bg-neutral-900 border border-neutral-800 p-5 hover:border-neutral-600 transition-colors cursor-pointer"
    >
      {hasError && (
        <div className="absolute top-3 right-3 flex items-center gap-1.5 px-2 py-1 rounded-full bg-red-500/10 text-red-500 text-xs">
          <span>⚠</span>
          <span>Ошибка</span>
        </div>
      )}

      {!hasError && !hasFacts && (
        <div className="absolute top-3 right-3 flex items-center gap-1.5 px-2 py-1 rounded-full bg-yellow-500/10 text-yellow-500 text-xs">
          <span>Анализируем...</span>
        </div>
      )}

      {!hasError && hasFacts && facts && (
        <div className="absolute top-3 right-3 flex items-center gap-1.5 px-2 py-1 rounded-full bg-green-500/10 text-green-500 text-xs">
          <span>✓</span>
          <span>Факты готовы</span>
        </div>
      )}

      <div className="flex items-center justify-between text-xs text-neutral-500 mt-6">
        <span>{article.author}</span>
        <span>{formattedDate}</span>
      </div>
      <h2 className="text-lg font-semibold text-neutral-100 leading-snug line-clamp-2">
        {article.title}
      </h2>
      <p className="text-sm text-neutral-400 line-clamp-4">{article.content}</p>
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

export default NewsCard;
