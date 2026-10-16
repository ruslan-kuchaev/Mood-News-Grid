import { useState, useEffect, useRef } from "react";
import SearchInput from "./components/SearchInput";
import NewsCard from "./components/NewsCard";
import ModalNews from "./components/ModalNews";
import type { NewsResponse, NewsArticle, NewsFact } from "./type";

const apiUrl = import.meta.env.VITE_API_URL;

if (!apiUrl) {
  console.error("VITE_API_URL не настроен. Проверьте файл .env");
}

function App() {
  const [query, setQuery] = useState("");
  const [articles, setArticles] = useState<NewsArticle[]>([]);
  const [facts, setFacts] = useState<Record<string, NewsFact>>({});
  const [processingErrors, setProcessingErrors] = useState<Set<string>>(new Set());
  const [loading, setLoading] = useState(false);
  const [searched, setSearched] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [selectedArticle, setSelectedArticle] = useState<NewsArticle | null>(
    null,
  );
  const eventSourceRef = useRef<EventSource | null>(null);
  const abortControllerRef = useRef<AbortController | null>(null);

  useEffect(() => {
    if (!apiUrl) {
      setError("API URL не настроен. Проверьте переменные окружения.");
      return;
    }

    let eventSource: EventSource | null = null;
    let reconnectTimer: number | null = null;
    let reconnectAttempts = 0;
    const maxReconnectAttempts = 10;

    const connect = () => {
      if (eventSource && eventSource.readyState !== EventSource.CLOSED) {
        return;
      }

      if (reconnectAttempts >= maxReconnectAttempts) {
        console.error("[SSE] Достигнут лимит попыток переподключения");
        return;
      }

      console.log("[SSE] Создание нового соединения...");
      eventSource = new EventSource(`${apiUrl}/events`);
      eventSourceRef.current = eventSource;

      eventSource.onopen = () => {
        console.log("[SSE] Соединение открыто");
        reconnectAttempts = 0;
      };

      eventSource.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          console.log("[SSE] Получено событие:", data);

          if (data.event === "connected") {
            console.log("[SSE] Подключено к серверу");
            return;
          }

          if (data.event === "fact_ready") {
            console.log(`[SSE] Факты готовы для: ${data.url_hash}`);
            loadFacts(data.url_hash);
          }

          if (data.event === "processing_error") {
            console.warn(
              `[SSE] Ошибка обработки: ${data.url_hash} - ${data.error}`,
            );
            setProcessingErrors((prev) => new Set(prev).add(data.url_hash));
          }
        } catch (err) {
          console.error("[SSE] Ошибка парсинга события:", err);
        }
      };

      eventSource.onerror = (err) => {
        console.error(
          "[SSE] Ошибка соединения, переподключение через 3 сек...",
          err,
        );
        eventSource?.close();
        reconnectAttempts++;
        
        const delay = Math.min(3000 * Math.pow(1.5, reconnectAttempts - 1), 30000);
        reconnectTimer = window.setTimeout(connect, delay);
      };
    };

    connect();

    return () => {
      console.log("[SSE] Компонент размонтируется, закрытие соединения");
      if (reconnectTimer !== null) {
        clearTimeout(reconnectTimer);
      }
      if (eventSource) {
        eventSource.close();
      }
    };
  }, []);

  async function loadFacts(urlHash: string) {
    if (!apiUrl) return;

    try {
      const res = await fetch(`${apiUrl}/facts/${urlHash}`);

      if (!res.ok) {
        console.warn(`[Facts] Факты для ${urlHash} еще не готовы`);
        return;
      }

      const factData: NewsFact = await res.json();
      setFacts((prev) => ({ ...prev, [urlHash]: factData }));
      console.log(`[Facts] Загружены факты для ${urlHash}`, factData);
    } catch (err) {
      console.error(`[Facts] Ошибка загрузки фактов для ${urlHash}:`, err);
    }
  }

  async function handleSearch() {
    if (!query.trim()) {
      setError("Введите запрос для поиска");
      return;
    }

    if (!apiUrl) {
      setError("API URL не настроен");
      return;
    }

    // Отменяем предыдущий запрос если есть
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
    }

    // Создаем новый контроллер для текущего запроса
    const abortController = new AbortController();
    abortControllerRef.current = abortController;

    setLoading(true);
    setError(null);
    setFacts({});
    setProcessingErrors(new Set());

    try {
      const res = await fetch(
        `${apiUrl}/?q=${encodeURIComponent(query)}`,
        { signal: abortController.signal }
      );
      
      if (!res.ok) throw new Error(`Request failed: ${res.status}`);

      const data: NewsResponse = await res.json();
      
      // Проверяем что запрос не был отменен
      if (!abortController.signal.aborted) {
        setArticles(data.articles);
        setSearched(true);
      }
    } catch (err) {
      if (err instanceof Error && err.name === 'AbortError') {
        console.log('[Search] Запрос был отменен');
        return;
      }
      setError(err instanceof Error ? err.message : "Что-то пошло не так");
    } finally {
      setLoading(false);
      abortControllerRef.current = null;
    }
  }

  return (
    <div className="min-h-screen bg-black">
      {!searched ? (
        <div className="min-h-screen flex items-center justify-center">
          <SearchInput
            value={query}
            onChange={setQuery}
            onSubmit={handleSearch}
            loading={loading}
            variant="center"
          />
        </div>
      ) : (
        <div className="flex flex-col items-center gap-10 py-10 px-4">
          <SearchInput
            value={query}
            onChange={setQuery}
            onSubmit={handleSearch}
            loading={loading}
            variant="top"
          />

          <div
            className={`w-full max-w-6xl grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4 transition-opacity duration-500 ${
              loading ? "opacity-40" : "opacity-100"
            }`}
          >
            {articles.map((article, index) => {
              const articleFacts = facts[article.url_hash];
              const hasFacts = !!articleFacts;
              const hasError = processingErrors.has(article.url_hash);

              return (
                <NewsCard
                  key={article.url_hash || `article-${index}`}
                  article={article}
                  facts={articleFacts}
                  hasFacts={hasFacts}
                  hasError={hasError}
                  onClick={() => setSelectedArticle(article)}
                />
              );
            })}
          </div>
        </div>
      )}

      <ModalNews
        open={selectedArticle !== null}
        content={selectedArticle}
        facts={selectedArticle ? facts[selectedArticle.url_hash] : undefined}
        onClose={() => setSelectedArticle(null)}
      />

      {error && (
        <p className="fixed bottom-6 left-1/2 -translate-x-1/2 text-red-400 text-sm px-4 py-2 bg-red-900/30 rounded-lg border border-red-500/50">
          {error}
        </p>
      )}
    </div>
  );
}

export default App;
