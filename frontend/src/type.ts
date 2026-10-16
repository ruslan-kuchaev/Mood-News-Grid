export interface NewsArticle {
  title: string | null
  author: string | null
  pubdate: string | null
  language: string | null
  content: string | null
  url: string | null
  url_hash: string
}

export interface KeyValue {
  value: string
  context: string
}

export interface NewsFact {
  url: string
  title: string
  dates?: string[]
  people?: string[]
  organizations?: string[]
  locations?: string[]
  main_thesis: string
  key_numbers?: KeyValue[]
  keywords?: string[]
  categories?: string[]
  processed_at: string
}

export interface NewsResponse {
  status: number
  numResults: number
  articles: NewsArticle[]
}

export interface RewriteRequest {
  url_hash: string
  emotion: 'happy' | 'sad' | 'angry'
  title: string
  content: string
}

export interface RewriteResponse {
  original_content: string
  rewritten_content: string
  emotion: string
}
