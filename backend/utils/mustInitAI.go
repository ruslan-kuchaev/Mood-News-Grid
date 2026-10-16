package utils

import (
	"backend/openrouter"
	"context"
	"log"
)

func MustInitAI(ctx context.Context, Model string) *openrouter.RouterAi {
	ai := openrouter.NewRouterAi(Model)
	if ai == nil {
		log.Printf("Ошибка создания клиента OpenRouter для модели %s", Model)
		return nil
	}

	if err := ai.Check(ctx); err != nil {
		log.Printf("Ошибка проверки доступности модели %s: %v", Model, err)
		return nil
	}

	log.Printf("Модель %s успешно инициализирована", Model)
	return ai
}
