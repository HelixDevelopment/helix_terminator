module github.com/helixdevelopment/notification-service

go 1.25.0

require (
	github.com/gin-gonic/gin v1.10.0
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/golang-migrate/migrate/v4 v4.19.1
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.10.0
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/bytedance/sonic v1.11.6 // indirect
	github.com/bytedance/sonic/loader v0.1.1 // indirect
	github.com/cloudwego/base64x v0.1.4 // indirect
	github.com/cloudwego/iasm v0.2.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/gabriel-vasile/mimetype v1.4.3 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.20.0 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/jackc/pgerrcode v0.0.0-20220416144525-469b46aa5efa // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/cpuid/v2 v2.2.7 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.2.2 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/rogpeppe/go-internal v1.15.0 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.2.12 // indirect
	golang.org/x/arch v0.8.0 // indirect
	golang.org/x/crypto v0.53.0 // indirect
	golang.org/x/net v0.56.0 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/text v0.38.0 // indirect
	google.golang.org/protobuf v1.36.7 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// Slack delivery (internal/delivery/slack_herald.go, NO build tag —
// compiled by DEFAULT, Constitution §11.4.197: a wired feature is active
// by default, never present-but-off) imports Herald's live Slack channel
// adapter. The require + replace pair below resolves the module graph
// this needs, rooted at submodules/herald (which this checkout has
// initialized recursively — `git -C submodules/herald submodule update
// --init --recursive` — the project-wide submodule-init mandate,
// Constitution §11.4.27/§11.4.36, that every checkout of this repo is
// already expected to satisfy).
//
// DELIBERATE NON-REPLACEMENT of github.com/slack-go/slack (verified,
// Constitution §11.4.6 — not guessed): unlike every other herald-internal
// dependency below, github.com/slack-go/slack is NOT replaced to its
// local submodule copy at submodules/herald/submodules/slack-go. That
// copy is checked out at tag v0.27.0, which has drifted ahead of the API
// Herald's own commons_messaging/channels/slack source tree here was
// actually written against — v0.16.0, per
// submodules/herald/commons_messaging/go.mod's own `require` — and does
// NOT compile against it (`slack.UploadFileV2Parameters`/
// `.UploadFileV2Context` and `slackevents.MessageEvent.Files` are
// undefined at v0.27.0; captured verbatim in the implementation report,
// this is a Herald-repo-internal submodule-pin drift, out of this
// change's scope to fix). Leaving no replace here lets Go's MVS resolve
// the real public v0.16.0 release from the module proxy instead, which
// DOES compile — verified: `go build ./...` (default, no tags) is clean.
//
// DELIBERATELY NOT `go mod tidy`'d (Constitution §11.4.92 Pass 2 —
// regression-blast-radius): a bare `go mod tidy` against this module DOES
// now succeed (unlike an earlier round of this change, before Herald's
// submodules were initialized), but it forces gin-gonic/gin v1.10.0 →
// v1.12.0 (and several further transitive bumps: bytedance/sonic,
// go-playground/validator, golang.org/x/arch, etc.) project-wide, purely
// because Herald's own `commons` module requires newer versions of
// several shared dependencies — an unrequested, out-of-scope version-bump
// blast radius for an add-a-notification-channel change, affecting the
// HTTP framework every endpoint in this service depends on. This
// require+replace block was hand-verified instead (`go build`/`go vet`/
// `go test`, default and `-tags integration`, all green — see the
// implementation report) without pulling that bump in. If a future change
// legitimately needs gin v1.12+ (or `go mod tidy` is run for an unrelated
// reason), the resulting gin bump should be reviewed and tested on its
// own merits, not silently absorbed as a side effect of Slack support.
require github.com/vasic-digital/herald/commons_messaging v0.0.0

require github.com/vasic-digital/herald/commons v0.0.0 // indirect

replace (
	github.com/vasic-digital/herald/commons => ../../submodules/herald/commons
	github.com/vasic-digital/herald/commons_infra => ../../submodules/herald/commons_infra
	github.com/vasic-digital/herald/commons_messaging => ../../submodules/herald/commons_messaging
	github.com/vasic-digital/herald/commons_storage => ../../submodules/herald/commons_storage
	digital.vasic.background => ../../submodules/herald/submodules/background
	digital.vasic.cache => ../../submodules/herald/submodules/cache
	digital.vasic.database => ../../submodules/herald/submodules/database
	digital.vasic.models => ../../submodules/herald/submodules/Models
	gopkg.in/telebot.v3 => ../../submodules/herald/submodules/telebot
)
