package handler

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// mfaChallengeTTL bounds how long a login-issued MFA challenge (see
// Login's MFARequired response) remains redeemable via VerifyMFA before
// the caller must restart the login flow.
const mfaChallengeTTL = 5 * time.Minute

// mfaChallenge binds a login-time MFA challengeId to the user that
// authenticated with a correct password and is now proving a second
// factor before receiving real tokens.
type mfaChallenge struct {
	userID    uuid.UUID
	expiresAt time.Time
}

// mfaChallengeStore is a thread-safe, single-process, short-lived
// challenge->user binding for the unauthenticated login-completion step
// (POST /mfa/verify). It exists because that endpoint is deliberately
// reachable with NO bearer token: Login() withholds real tokens for any
// MFA-enabled user until MFA verification succeeds, so a caller at that
// point in the flow structurally cannot present a JWT the
// jwtValidationMiddleware could resolve a userID from. Gating
// /mfa/verify behind that middleware (the fix applied to /logout, whose
// caller IS already authenticated) would make MFA-enabled login
// permanently impossible - a strictly worse regression than the bug
// being fixed. Consistent with this service's existing in-memory
// degrade-gracefully mode (see server.New), an in-process store is a
// real, working implementation for a single-instance deployment; a
// multi-instance deployment would back this with a shared store
// (Postgres table or Redis) using the same create/lookup/consume
// contract - tracked as a follow-up, not a blocker for this fix.
type mfaChallengeStore struct {
	mu         sync.Mutex
	challenges map[string]mfaChallenge
}

func newMFAChallengeStore() *mfaChallengeStore {
	return &mfaChallengeStore{challenges: make(map[string]mfaChallenge)}
}

// create mints a fresh, single-use challenge for userID and returns its
// ID. Called by Login() when it withholds tokens pending MFA.
func (s *mfaChallengeStore) create(userID uuid.UUID) string {
	id := uuid.New().String()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.challenges[id] = mfaChallenge{
		userID:    userID,
		expiresAt: time.Now().UTC().Add(mfaChallengeTTL),
	}
	return id
}

// lookup resolves challengeID to its bound userID without consuming it,
// so an incorrect MFA code can be retried until the challenge expires.
// An expired entry is reaped and reported as not-found.
func (s *mfaChallengeStore) lookup(challengeID string) (uuid.UUID, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	c, ok := s.challenges[challengeID]
	if !ok {
		return uuid.Nil, false
	}
	if time.Now().UTC().After(c.expiresAt) {
		delete(s.challenges, challengeID)
		return uuid.Nil, false
	}
	return c.userID, true
}

// consume permanently invalidates challengeID. Called only after a
// successful MFA verification, so a single login challenge yields
// exactly one token pair and cannot be replayed.
func (s *mfaChallengeStore) consume(challengeID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.challenges, challengeID)
}
