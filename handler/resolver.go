package handler

import (
	"GPTBot/api/telegram"
	"GPTBot/storage"
)

// IntentResolver analyses a normalized Request and returns the list of
// intents the pipeline should execute. The resolver handles structural
// routing (private vs group, voice echo); content-level intent detection
// (generate image, voice, etc.) is delegated to GPT function tools.
type IntentResolver struct{}

// Resolve maps a normalized Request to a list of Intent values.
func (r *IntentResolver) Resolve(ctx *telegram.UpdateContext, chat *storage.Chat, req *Request) []Intent {
	var intents []Intent

	// ── Transport: voice echo ──
	if req.OriginalMedia == MediaVoice {
		intents = append(intents, Intent{Type: IntentEchoTranscription})
		if req.IsForwarded {
			return intents // forwarded voice → transcription only
		}
	}

	// ── Uploaded image → analyze (exclusive path) ──
	if req.ImageURL != "" {
		if !ctx.IsGroup || req.BotAddressed {
			intents = append(intents, Intent{Type: IntentAnalyzeImage})
		}
		return intents
	}

	// ── Group gate ──
	if ctx.IsGroup && !req.BotAddressed {
		if chat.Settings.GroupAutoReply {
			intents = append(intents, Intent{Type: IntentGroupAutoReply})
		}
		return intents
	}

	// ── Main intent: GPT call with function tools ──
	if ctx.IsGroup {
		intents = append(intents, Intent{Type: IntentGroupReply})
	} else {
		intents = append(intents, Intent{Type: IntentChat})
	}

	return intents
}
