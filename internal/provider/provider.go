package provider

import (
	"context"
	"io"

	"noveltts/internal/model"
)

type Provider interface {
	Name() string
	Type() string
	Synthesize(ctx context.Context, req *model.TTSRequest) (io.ReadCloser, string, error)
	ListModels(ctx context.Context) ([]model.ModelInfo, error)
	ListVoices(ctx context.Context, modelID string) ([]model.VoiceInfo, error)
}
