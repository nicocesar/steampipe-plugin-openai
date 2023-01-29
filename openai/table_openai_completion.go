package openai

import (
	"context"
	"encoding/json"

	gogpt "github.com/sashabaranov/go-gpt3"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
)

func tableOpenAiCompletion(ctx context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "openai_completion",
		Description: "Completions available in OpenAI.",
		List: &plugin.ListConfig{
			Hydrate: listCompletion,
			KeyColumns: []*plugin.KeyColumn{
				{Name: "prompt", Require: plugin.Optional},
				{Name: "settings", Require: plugin.Optional},
			},
		},
		Columns: []*plugin.Column{
			// Top columns
			/*
				{Name: "id", Type: proto.ColumnType_STRING, Description: ""},
				{Name: "object", Type: proto.ColumnType_STRING, Description: ""},
				{Name: "created", Type: proto.ColumnType_TIMESTAMP, Transform: transform.FromField("CreatedAt").Transform(transform.UnixToTimestamp), Description: "Timestamp of when the model was created."},
				{Name: "model", Type: proto.ColumnType_STRING, Description: ""},
				{Name: "usage", Type: proto.ColumnType_STRING, Description: ""},
			*/
			{Name: "completion", Type: proto.ColumnType_STRING, Transform: transform.FromField("Text"), Description: ""},
			{Name: "index", Type: proto.ColumnType_INT, Transform: transform.FromField("Index"), Description: ""},
			{Name: "finish_reason", Type: proto.ColumnType_STRING, Description: ""},
			{Name: "log_probs", Type: proto.ColumnType_JSON, Description: ""},
			{Name: "prompt", Type: proto.ColumnType_STRING, Transform: transform.FromQual("prompt"), Description: ""},
			{Name: "settings", Type: proto.ColumnType_JSON, Transform: transform.FromQual("settings"), Description: ""},
		},
	}
}

type CompletionRequestQual struct {
	Model            *string        `json:"model"`
	Prompt           *string        `json:"prompt,omitempty"`
	Suffix           *string        `json:"suffix,omitempty"`
	MaxTokens        *int           `json:"max_tokens,omitempty"`
	Temperature      *float32       `json:"temperature,omitempty"`
	TopP             *float32       `json:"top_p,omitempty"`
	N                *int           `json:"n,omitempty"`
	Stream           *bool          `json:"stream,omitempty"`
	LogProbs         *int           `json:"logprobs,omitempty"`
	Echo             *bool          `json:"echo,omitempty"`
	Stop             []string       `json:"stop,omitempty"`
	PresencePenalty  *float32       `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float32       `json:"frequency_penalty,omitempty"`
	BestOf           *int           `json:"best_of,omitempty"`
	LogitBias        map[string]int `json:"logit_bias,omitempty"`
	User             *string        `json:"user,omitempty"`
}

type CompletionRow struct {
	gogpt.CompletionChoice
	Prompt string
}

func listCompletion(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	conn, err := connect(ctx, d)
	if err != nil {
		plugin.Logger(ctx).Error("openai_completion.listCompletion", "connection_error", err)
		return nil, err
	}

	// Default settings taken from the playground UI
	// https://beta.openai.com/playground
	cr := gogpt.CompletionRequest{
		Model:            "text-davinci-003",
		Temperature:      0.7,
		MaxTokens:        256,
		Stop:             []string{},
		TopP:             1,
		FrequencyPenalty: 0,
		PresencePenalty:  0,
		BestOf:           1,
	}

	settingsString := d.EqualsQuals["settings"].GetJsonbValue()
	if settingsString != "" {
		// Overwrite any settings provided in the settings qual. If a field
		// is not passed in the settings, then default to the settings above.
		var crQual CompletionRequestQual
		err := json.Unmarshal([]byte(settingsString), &crQual)
		if err != nil {
			plugin.Logger(ctx).Error("openai_completion.listCompletion", "unmarshal_error", err)
			return nil, err
		}
		if crQual.Model != nil {
			cr.Model = *crQual.Model
		}
		if crQual.Prompt != nil {
			cr.Prompt = *crQual.Prompt
		}
		if crQual.Suffix != nil {
			cr.Suffix = *crQual.Suffix
		}
		if crQual.MaxTokens != nil {
			cr.MaxTokens = *crQual.MaxTokens
		}
		if crQual.Temperature != nil {
			cr.Temperature = *crQual.Temperature
		}
		if crQual.TopP != nil {
			cr.TopP = *crQual.TopP
		}
		if crQual.N != nil {
			cr.N = *crQual.N
		}
		if crQual.Stream != nil {
			cr.Stream = *crQual.Stream
		}
		if crQual.LogProbs != nil {
			cr.LogProbs = *crQual.LogProbs
		}
		if crQual.Echo != nil {
			cr.Echo = *crQual.Echo
		}
		if crQual.Stop != nil {
			cr.Stop = crQual.Stop
		}
		if crQual.PresencePenalty != nil {
			cr.PresencePenalty = *crQual.PresencePenalty
		}
		if crQual.FrequencyPenalty != nil {
			cr.FrequencyPenalty = *crQual.FrequencyPenalty
		}
		if crQual.BestOf != nil {
			cr.BestOf = *crQual.BestOf
		}
		if crQual.LogitBias != nil {
			cr.LogitBias = crQual.LogitBias
		}
		if crQual.User != nil {
			cr.User = *crQual.User
		}
	}

	// Both are valid, but the order of precedence is:
	// 1. prompt = "my prompt"
	// 2. settings = '{"prompt": "my prompt"}'
	if d.EqualsQuals["prompt"] != nil {
		cr.Prompt = d.EqualsQualString("prompt")
	}

	if cr.Prompt == "" {
		// No prompt, so return zero rows
		return nil, nil
	}

	plugin.Logger(ctx).Debug("openai_completion.listCompletion", "prompt", cr)
	resp, err := conn.CreateCompletion(ctx, cr)
	if err != nil {
		plugin.Logger(ctx).Error("openai_completion.listCompletion", "prompt", cr, "completion_error", err)
		return nil, err
	}
	plugin.Logger(ctx).Debug("openai_completion.listCompletion", "completion_response", resp)

	for _, i := range resp.Choices {
		row := CompletionRow{i, cr.Prompt}
		d.StreamListItem(ctx, row)
	}

	return nil, nil
}
