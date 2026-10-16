package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type NewsArticle struct {
	Title    string
	Author   string
	PubDate  time.Time
	Language string
	Content  string
	URL      string
}

func SaveArticles(ctx context.Context, articles []NewsArticle) error {
	if len(articles) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for _, a := range articles {
		batch.Queue(`
			INSERT INTO news (title, author, pubdate, language, content, url)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (url) DO NOTHING
		`, a.Title, a.Author, a.PubDate, a.Language, a.Content, a.URL)
	}

	br := Pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < len(articles); i++ {
		_, err := br.Exec()
		if err != nil {
			return fmt.Errorf("batch exec at index %d: %w", i, err)
		}
	}

	return nil
}
