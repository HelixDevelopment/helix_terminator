// This file provides the REAL slackTransport implementation, wrapping
// Herald's live, wire-tested Slack channel adapter directly — Constitution
// §11.4.74 reuse-first: no Slack Web API call is reimplemented here, every
// byte on the wire is Herald's own already-tested
// commons_messaging/channels/slack.Adapter.Send.
//
// This file compiles by DEFAULT — no build tag — per Constitution
// §11.4.197 (a wired feature must be active by default, never present-but-
// off behind a dead flag). It requires submodules/herald's own nested git
// submodules to be initialized (`git -C submodules/herald submodule
// update --init --recursive`), which this repository already mandates
// project-wide (§11.4.27/§11.4.36) — a project checked out per those
// mandates already satisfies this precondition. See go.mod's comment
// immediately above the herald require/replace block for the exact
// resolution details (notably: github.com/slack-go/slack itself is NOT
// replaced to the local submodule copy — that copy's checked-out tag
// (v0.27.0) has drifted ahead of the API this Herald source tree was
// written against (v0.16.0, per submodules/herald/commons_messaging/
// go.mod's own `require`); MVS is left to resolve the real public
// v0.16.0 release instead, which is what actually compiles).
package delivery

import (
	"context"
	"fmt"

	heraldcommons "github.com/vasic-digital/herald/commons"
	heraldslack "github.com/vasic-digital/herald/commons_messaging/channels/slack"
)

// heraldSlackAdapter adapts Herald's *heraldslack.Adapter to this
// package's slackTransport seam.
type heraldSlackAdapter struct {
	adapter *heraldslack.Adapter
}

// newHeraldSlackTransport constructs a REAL, credentialed transport
// backed by Herald's Slack channel adapter. appToken/channelID are
// deliberately left empty — Herald's New() only requires appToken for
// Socket Mode Subscribe (inbound), which this outbound-only sender never
// calls; the destination channel is supplied per-call by
// SlackSender.Send's channelID parameter (Herald honors a per-message
// override via commons.Recipient.ChannelUserID — see
// submodules/herald/commons_messaging/channels/slack/send.go), so no
// adapter-level default channel is needed.
func newHeraldSlackTransport(botToken string) (slackTransport, error) {
	if botToken == "" {
		return nil, fmt.Errorf("slack (Herald): bot token is required")
	}
	return &heraldSlackAdapter{adapter: heraldslack.New(botToken, "", "")}, nil
}

// NewSlackSenderForTesting builds a fully real (non-mocked) SlackSender
// whose Herald adapter is pointed at a caller-supplied httptest.Server
// URL (via Herald's own NewWithBaseURL seam — see
// submodules/herald/commons_messaging/channels/slack/slack.go), so tests
// exercise the REAL Herald Send() code path (Constitution §11.4.27
// no-fakes-beyond-unit-tests — the only test double is the remote HTTP
// server itself, not this package's or Herald's logic) without depending
// on network access to Slack. Mirrors push.go's NewPushSenderForTesting.
//
// This constructor MUST NEVER be used in production wiring — handler.New
// always uses NewSlackSenderFromEnv / NewConfiguredSlackSender, both of
// which resolve to the live Slack Web API (empty baseURL).
func NewSlackSenderForTesting(botToken, baseURL string) (*SlackSender, error) {
	if botToken == "" {
		return nil, fmt.Errorf("slack (Herald): bot token is required")
	}
	return &SlackSender{transport: &heraldSlackAdapter{
		adapter: heraldslack.NewWithBaseURL(botToken, "", "", baseURL),
	}}, nil
}

// Send delegates to Herald's Adapter.Send, translating this package's
// (channelID, text) shape into Herald's commons.OutboundMessage / Recipient
// / Body value types (submodules/herald/commons/types.go). text is sent as
// Slack mrkdwn (Body.Markdown) — Herald's adapter prefers Markdown over
// Plain when both are supplied (see send.go: "Markdown first (Slack
// renders mrkdwn)").
func (h *heraldSlackAdapter) Send(ctx context.Context, channelID, text string) error {
	_, err := h.adapter.Send(ctx, heraldcommons.OutboundMessage{
		To: []heraldcommons.Recipient{
			{Channel: string(heraldcommons.ChannelSlack), ChannelUserID: channelID},
		},
		Body: heraldcommons.Body{Markdown: text},
	})
	if err != nil {
		return fmt.Errorf("slack (Herald): %w", err)
	}
	return nil
}
