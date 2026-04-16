package usage

type ParsedUsage struct {
	PromptTokens     int64
	CompletionTokens int64
	TotalTokens      int64
}

func Normalize(in ParsedUsage) ParsedUsage {
	out := ParsedUsage{
		PromptTokens:     clampNonNegative(in.PromptTokens),
		CompletionTokens: clampNonNegative(in.CompletionTokens),
		TotalTokens:      clampNonNegative(in.TotalTokens),
	}

	sum := out.PromptTokens + out.CompletionTokens
	if out.TotalTokens == 0 {
		out.TotalTokens = sum
	}

	if sum == 0 && out.TotalTokens > 0 {
		out.CompletionTokens = out.TotalTokens
		sum = out.TotalTokens
	}

	if out.TotalTokens > 0 {
		if out.PromptTokens == 0 && out.CompletionTokens > 0 && out.CompletionTokens < out.TotalTokens {
			out.PromptTokens = out.TotalTokens - out.CompletionTokens
		}
		if out.CompletionTokens == 0 && out.PromptTokens > 0 && out.PromptTokens < out.TotalTokens {
			out.CompletionTokens = out.TotalTokens - out.PromptTokens
		}
	}

	sum = out.PromptTokens + out.CompletionTokens
	if sum > 0 {
		out.TotalTokens = sum
	}
	return out
}

func Aggregate(usages ...ParsedUsage) ParsedUsage {
	total := ParsedUsage{}
	for _, usage := range usages {
		normalized := Normalize(usage)
		total.PromptTokens += normalized.PromptTokens
		total.CompletionTokens += normalized.CompletionTokens
		total.TotalTokens += normalized.TotalTokens
	}
	return total
}

func clampNonNegative(value int64) int64 {
	if value < 0 {
		return 0
	}
	return value
}
