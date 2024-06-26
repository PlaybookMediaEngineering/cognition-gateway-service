package proxy

import (
	"encoding/json"

	goopenai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func logListModelsResponse(log *zap.Logger, data []byte, prod bool) {
	models := &goopenai.ModelsList{}
	err := json.Unmarshal(data, models)
	if err != nil {
		logError(log, "error when unmarshalling list models response", prod, err)
		return
	}

	if prod {
		fields := []zapcore.Field{
			zap.Any("models", models.Models),
		}

		log.Info("openai list models response", fields...)
	}
}

func logRetrieveModelRequest(log *zap.Logger, prod bool, model string) {
	if prod {
		fields := []zapcore.Field{
			zap.String("model", model),
		}

		log.Info("openai retrieve model resquest", fields...)
	}
}

func logRetrieveModelResponse(log *zap.Logger, data []byte, prod bool) {
	model := &goopenai.Model{}
	err := json.Unmarshal(data, model)
	if err != nil {
		logError(log, "error when unmarshalling retrieve model response", prod, err)
		return
	}

	if prod {
		fields := []zapcore.Field{
			zap.String("id", model.ID),
			zap.Int64("created", model.CreatedAt),
			zap.String("object", model.Object),
			zap.String("owned_by", model.OwnedBy),
		}

		log.Info("openai retrieve model response", fields...)
	}
}

func logDeleteModelRequest(log *zap.Logger, prod bool, model string) {
	if prod {
		fields := []zapcore.Field{
			zap.String("model", model),
		}

		log.Info("openai delete model resquest", fields...)
	}
}

type DeletionResponse struct {
	Id      string `json:"id"`
	Object  string `json:"object"`
	Deleted bool   `json:"deleted"`
}

func logDeleteModelResponse(log *zap.Logger, data []byte, prod bool) {
	resp := &DeletionResponse{}
	err := json.Unmarshal(data, resp)
	if err != nil {
		logError(log, "error when unmarshalling model deletion response", prod, err)
		return
	}

	if prod {
		fields := []zapcore.Field{
			zap.String("id", resp.Id),
			zap.String("object", resp.Object),
			zap.Bool("deleted", resp.Deleted),
		}

		log.Info("openai delete model response", fields...)
	}
}
