module github.com/helixdevelopment/notification-service

go 1.25.3

require (
	github.com/gin-gonic/gin v1.12.0
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/golang-migrate/migrate/v4 v4.19.1
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.10.0
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/bytedance/sonic v1.15.0 // indirect
	github.com/bytedance/sonic/loader v0.5.0 // indirect
	github.com/cloudwego/base64x v0.1.6 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/gabriel-vasile/mimetype v1.4.12 // indirect
	github.com/gin-contrib/sse v1.1.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.30.1 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/jackc/pgerrcode v0.0.0-20220416144525-469b46aa5efa // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/rogpeppe/go-internal v1.15.0 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.3.1 // indirect
	golang.org/x/arch v0.22.0 // indirect
	golang.org/x/crypto v0.53.0 // indirect
	golang.org/x/net v0.56.0 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/text v0.38.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
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
// TIDIED FOR MODULE-MODE CORRECTNESS (HEL-002, 2026-07-23): this module
// was previously left un-tidied on purpose (the workspace masked missing
// Herald transitive requirements, and tidying forces the gin v1.10.0 →
// v1.12.0 bump). That left the module WORKSPACE-ONLY: `GOWORK=off go
// build` exited 1 ("updates to go.mod needed") and the recorded gin pin
// was a fiction — the workspace was ALREADY building gin v1.12.0 while
// go.mod said v1.10.0 (captured: HEL-002 evidence,
// specs/001-hel-002-gowork-off-build/evidence/). Per the previous
// revision of this very comment, the bump has now been "reviewed and
// tested on its own merits": `GOWORK=off go mod tidy` applied, and the
// full matrix (build/vet/test + -tags integration vet, in BOTH workspace
// and GOWORK=off modes) is green with the recorded and built versions
// identical in both modes. The manifest is tidy-stable (`go mod tidy
// -diff` is empty), so this drift class cannot silently return.
require github.com/vasic-digital/herald/commons_messaging v0.0.0

require (
	github.com/bytedance/gopkg v0.1.3 // indirect
	github.com/goccy/go-yaml v1.19.2 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/quic-go/qpack v0.6.0 // indirect
	github.com/quic-go/quic-go v0.59.0 // indirect
	github.com/slack-go/slack v0.16.0 // indirect
	github.com/vasic-digital/herald/commons v0.0.0
	go.mongodb.org/mongo-driver/v2 v2.5.0 // indirect
)

replace (
	digital.vasic.background => ../../submodules/herald/submodules/background
	digital.vasic.cache => ../../submodules/herald/submodules/cache
	digital.vasic.database => ../../submodules/herald/submodules/database
	digital.vasic.models => ../../submodules/herald/submodules/Models
	github.com/vasic-digital/herald/commons => ../../submodules/herald/commons
	github.com/vasic-digital/herald/commons_infra => ../../submodules/herald/commons_infra
	github.com/vasic-digital/herald/commons_messaging => ../../submodules/herald/commons_messaging
	github.com/vasic-digital/herald/commons_storage => ../../submodules/herald/commons_storage
	gopkg.in/telebot.v3 => ../../submodules/herald/submodules/telebot
)
