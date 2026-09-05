package app

import (
	"fmt"
	"strings"
)

// AllowedAvatarSeeds is the curated DiceBear seed gallery (ADR-0017).
// Keep in sync with frontend/src/components/avatarSeeds.ts.
var AllowedAvatarSeeds = []string{
	"fox", "owl", "panda", "tiger", "wolf", "hawk",
	"seal", "koala", "otter", "raven", "lynx", "bison",
	"coral", "maple", "comet", "dune", "ember", "jade",
}

var allowedAvatarSeedSet = func() map[string]struct{} {
	m := make(map[string]struct{}, len(AllowedAvatarSeeds))
	for _, s := range AllowedAvatarSeeds {
		m[s] = struct{}{}
	}
	return m
}()

// NormalizeAvatarSeed trims and lowercases; empty means legacy fallback.
func NormalizeAvatarSeed(seed string) string {
	return strings.ToLower(strings.TrimSpace(seed))
}

// ValidateAvatarSeed accepts empty (legacy) or a whitelisted seed.
func ValidateAvatarSeed(seed string) (string, error) {
	s := NormalizeAvatarSeed(seed)
	if s == "" {
		return "", nil
	}
	if _, ok := allowedAvatarSeedSet[s]; !ok {
		return "", fmt.Errorf("%w: avatar_seed is not in the allowed gallery", ErrValidation)
	}
	return s, nil
}

// EffectiveAvatarSeed returns the stored seed, or userID, or username.
func EffectiveAvatarSeed(stored, userID, username string) string {
	if s := NormalizeAvatarSeed(stored); s != "" {
		return s
	}
	if id := strings.TrimSpace(userID); id != "" {
		return id
	}
	return strings.TrimSpace(username)
}
