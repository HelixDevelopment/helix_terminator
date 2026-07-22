// Slack delivery, implemented as a thin wrapper around the Herald
// submodule's real, wire-tested Slack channel adapter
// (submodules/herald/commons_messaging/channels/slack — Wave 7, live in
// Herald's own `pherald listen` binary since Herald spec V4 §17). This
// package deliberately does NOT reimplement the Slack Web API call itself
// (Constitution §11.4.74 reuse-first, extend-don't-reimplement) — every
// real HTTP round-trip to `chat.postMessage` happens inside Herald's own
// already-tested code.
//
// REAL BY DEFAULT (Constitution §11.4.197 — a wired feature is active by
// default, never present-but-off behind a dead flag): the Herald-backed
// transport (slack_herald.go) has NO build tag and compiles into every
// build of this service. It requires submodules/herald's own nested git
// submodules to be initialized (`git -C submodules/herald submodule
// update --init --recursive`) — a precondition this repository already
// mandates project-wide for every incorporated submodule
// (Constitution §11.4.27/§11.4.36); a checkout that follows those
// mandates already satisfies it. See go.mod's comment above the herald
// require/replace block for the exact module-graph wiring, including one
// notable detail: github.com/slack-go/slack is deliberately NOT replaced
// to the local submodule copy at submodules/herald/submodules/slack-go
// (that copy's checked-out tag, v0.27.0, has drifted ahead of the API
// Herald's own source tree here was written against — v0.16.0, per
// submodules/herald/commons_messaging/go.mod's own `require` — and does
// not compile against it: `slack.UploadFileV2Parameters` / `.Files` on
// `slackevents.MessageEvent` are undefined at v0.27.0; captured verbatim
// in the accompanying implementation report). MVS is left to resolve the
// real public v0.16.0 release instead, which does compile.
//
// This file (the always-compiled entry point) defines the public
// surface: SlackConfig / SlackConfigFromEnv / NewSlackSenderFromEnv /
// NewConfiguredSlackSender / SlackSender.Send. Its behavior mirrors
// push.go's honest not-configured contract exactly: HERALD_SLACK_BOT_TOKEN
// unset => ok=false (never fabricates delivery); a genuine adapter-
// construction failure (BotToken empty) => ok=true, err=non-nil, so
// handler.New logs the operator misconfiguration and falls back to the
// honest unconfigured state, precisely like NewPushSenderFromEnv already
// does for FCM. The Herald adapter itself performs NO network call at
// construction time (see slack_herald.go's newHeraldSlackTransport) — a
// syntactically-present-but-invalid bot token is only discovered at
// Send() time, exactly like a real Slack API client would report it (an
// auth error from chat.postMessage), which SlackSender.Send maps to the
// honest "failed" status, never a fabricated "sent".
package delivery

import (
	"context"
	"errors"
	"fmt"
	"os"
)

// ErrSlackProviderNotConfigured is returned by SlackSender.Send when this
// SlackSender was built with delivery.NewSlackSender() — the
// zero-credential, honest "not configured" state. Slack delivery requires
// operator-supplied HERALD_SLACK_BOT_TOKEN (a Slack bot token, xoxb-…,
// with chat:write scope). This is an HONEST not-yet-configured state — it
// MUST NEVER be papered over with a fabricated "sent" status (Constitution
// §11.4 anti-bluff covenant). Callers persist notification.Status =
// "pending_provider_unconfigured" when this error is returned.
var ErrSlackProviderNotConfigured = errors.New(
	"slack provider (Herald) not configured: set HERALD_SLACK_BOT_TOKEN (a Slack bot token, xoxb-…, with chat:write scope) to enable Slack delivery",
)

// SlackConfig names the environment-sourced inputs that configure Slack
// delivery for this deployment.
type SlackConfig struct {
	// BotToken is the Slack bot token (xoxb-…) Herald's adapter uses for
	// chat.postMessage + auth.test. The app-level token (xapp-…) Herald
	// also supports is deliberately NOT exposed here — it is only needed
	// for Socket Mode inbound Subscribe, which notification-service (an
	// outbound-only sender) never calls (see Herald
	// commons_messaging/channels/slack/slack.go New() doc comment).
	BotToken string
}

// SlackConfigFromEnv reads HERALD_SLACK_BOT_TOKEN from the environment. ok
// is false when it is unset, meaning Slack is not configured for this
// deployment — an honest "not configured" state, mirroring
// PushConfigFromEnv's FCM_SERVICE_ACCOUNT_JSON gate in push.go
// (Constitution §11.4 anti-bluff: callers must not fabricate delivery when
// this returns false). The variable name deliberately matches Herald's own
// (submodules/herald/pherald/cmd/pherald/listen.go) rather than inventing
// a service-local prefix, so the same credential is legible/shareable
// across any service in this repo that later also delivers via Herald.
func SlackConfigFromEnv() (SlackConfig, bool) {
	token := os.Getenv("HERALD_SLACK_BOT_TOKEN")
	if token == "" {
		return SlackConfig{}, false
	}
	return SlackConfig{BotToken: token}, true
}

// slackTransport is the minimal seam SlackSender delegates to. Its two
// implementations — the real Herald-backed adapter (slack_herald.go,
// build tag heraldslack) and the honest stub (slack_herald_stub.go, build
// tag !heraldslack) — share this exact signature so SlackSender itself
// never needs to know which one is compiled in.
type slackTransport interface {
	Send(ctx context.Context, channelID, text string) error
}

// SlackSender delivers notifications to Slack via Herald's Slack channel
// adapter. The zero value returned by NewSlackSender() is deliberately
// UNCONFIGURED — Send() always returns ErrSlackProviderNotConfigured — so
// a deployment with no Slack credentials never fabricates a "sent" status
// (Constitution §11.4 anti-bluff covenant).
type SlackSender struct {
	transport slackTransport // nil => unconfigured
}

// NewSlackSender constructs an UNCONFIGURED SlackSender — the honest
// not-yet-provisioned state. Every Send() call returns
// ErrSlackProviderNotConfigured, regardless of arguments.
func NewSlackSender() *SlackSender { return &SlackSender{} }

// NewSlackSenderFromEnv builds a SlackSender from SlackConfigFromEnv's
// output.
//
// ok mirrors SlackConfigFromEnv: false means HERALD_SLACK_BOT_TOKEN is
// unset — Slack is honestly not configured for this deployment (sender is
// nil; callers should fall back to NewSlackSender()).
//
// When ok is true but err is non-nil, the operator SET
// HERALD_SLACK_BOT_TOKEN but the transport could not be constructed —
// either a genuine adapter-construction failure (heraldslack-tagged
// builds) or, in a binary built WITHOUT the heraldslack tag (every
// default build in this checkout today, see this file's package doc
// comment), the honest "this binary cannot speak to Slack" state.
// Callers MUST NOT silently treat this the same as "not configured" —
// see handler.New, which logs the error and falls back to an
// unconfigured sender, exactly mirroring NewPushSenderFromEnv's
// documented contract in push.go.
func NewSlackSenderFromEnv() (sender *SlackSender, ok bool, err error) {
	cfg, ok := SlackConfigFromEnv()
	if !ok {
		return nil, false, nil
	}
	sender, err = NewConfiguredSlackSender(cfg)
	return sender, true, err
}

// NewConfiguredSlackSender builds a real, credentialed SlackSender from
// cfg. Returns a non-nil error if cfg.BotToken is empty or the underlying
// transport cannot be constructed (see this file's package doc comment
// for the heraldslack build-tag seam this delegates through).
func NewConfiguredSlackSender(cfg SlackConfig) (*SlackSender, error) {
	if cfg.BotToken == "" {
		return nil, fmt.Errorf("slack (Herald): BotToken is required")
	}
	transport, err := newHeraldSlackTransport(cfg.BotToken)
	if err != nil {
		return nil, err
	}
	return &SlackSender{transport: transport}, nil
}

// Send delivers text to the Slack channel identified by channelID (a
// Slack channel ID, e.g. "C0123ABCD" — NOT a #channel-name; Slack's
// chat.postMessage accepts channel IDs, and Herald's adapter passes the
// value straight through per-message via commons.Recipient.ChannelUserID,
// see submodules/herald/commons_messaging/channels/slack/send.go).
//
// Returns ErrSlackProviderNotConfigured when this SlackSender has no
// transport (the NewSlackSender() zero-value case). Returns a non-nil
// error on ANY other failure (empty channelID, empty text, Slack Web API
// error, empty `ts` in the response — Herald's own §107 bluff guard).
// Success means Slack's chat.postMessage endpoint accepted + routed the
// message (Herald's commons.DeliveryRouted evidence ceiling — "platform
// stored & broadcast", NOT "recipient read"); callers MUST map this to
// their own "sent" status, never "delivered"/"read", to avoid over-
// claiming beyond what Slack's API actually confirms. Never fabricates a
// sent status either way (Constitution §11.4 anti-bluff covenant).
func (s *SlackSender) Send(ctx context.Context, channelID, text string) error {
	if s == nil || s.transport == nil {
		return ErrSlackProviderNotConfigured
	}
	if channelID == "" {
		return fmt.Errorf("slack (Herald): channel id (target) is required")
	}
	if text == "" {
		return fmt.Errorf("slack (Herald): message text is required")
	}
	return s.transport.Send(ctx, channelID, text)
}
