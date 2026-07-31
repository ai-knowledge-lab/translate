package translate

import "context"

type Translator interface {
	Translate(ctx context.Context, text, from, to, model string) (chan TranslateResponse, error)
}
