package translate

import "github.com/ai-knowledge-lab/translate/internal/service/translate"

type Handler struct {
	Service translate.Translator
}

func NewHandler() *Handler {
	return &Handler{
		Service: translate.NewService(),
	}
}
