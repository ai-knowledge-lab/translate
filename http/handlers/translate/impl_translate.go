package translate

import (
	"io"
	"net/http"

	"github.com/ai-knowledge-lab/translate/internal/service/translate"
	"github.com/gin-gonic/gin"
)

func (h *Handler) Translate(ctx *gin.Context) {
	var body translate.TranslateRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": err.Error(),
			"data":    "",
		})
		return
	}

	ch, err := h.Service.Translate(ctx, body.Text, body.From, body.To, body.Model)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": err.Error(),
			"data":    "",
		})
		return
	}

	ctx.Stream(func(w io.Writer) bool {
		if msg, ok := <-ch; ok {
			ctx.SSEvent("message", msg)
			return true
		}
		return false
	})

	return
}
