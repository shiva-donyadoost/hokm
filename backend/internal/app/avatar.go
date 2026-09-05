package app

import (
	"fmt"
	"strings"
)

// AllowedAvatarStyles is the DiceBear style gallery (ADR-0018).
// Keep in sync with frontend/src/components/avatarSeeds.ts.
var AllowedAvatarStyles = []string{"lorelei", "avataaars"}

// AllowedAvatarSeeds is the curated DiceBear seed gallery (ADR-0017/0018).
// Same seeds apply to every allowed style (URL path differs by style).
// Keep in sync with frontend/src/components/avatarSeeds.ts.
var AllowedAvatarSeeds = []string{
	"fox", "owl", "panda", "tiger", "wolf", "hawk",
	"seal", "koala", "otter", "raven", "lynx", "bison",
	"coral", "maple", "comet", "dune", "ember", "jade",
}

const DefaultAvatarStyle = "lorelei"

var allowedAvatarSeedSet = func() map[string]struct{} {
	m := make(map[string]struct{}, len(AllowedAvatarSeeds))
	for _, s := range AllowedAvatarSeeds {
		m[s] = struct{}{}
	}
	return m
}()

var allowedAvatarStyleSet = func() map[string]struct{} {
	m := make(map[string]struct{}, len(AllowedAvatarStyles))
	for _, s := range AllowedAvatarStyles {
		m[s] = struct{}{}
	}
	return m
}()

// NormalizeAvatarSeed trims and lowercases; empty means legacy fallback.
func NormalizeAvatarSeed(seed string) string {
	return strings.ToLower(strings.TrimSpace(seed))
}

// NormalizeAvatarStyle trims and lowercases.
func NormalizeAvatarStyle(style string) string {
	return strings.ToLower(strings.TrimSpace(style))
}

// EffectiveAvatarStyle returns stored style or lorelei for legacy/empty.
func EffectiveAvatarStyle(stored string) string {
	s := NormalizeAvatarStyle(stored)
	if s == "" {
		return DefaultAvatarStyle
	}
	return s
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

// ValidateAvatarStyle accepts empty (legacy => lorelei) or a whitelisted style.
func ValidateAvatarStyle(style string) (string, error) {
	s := NormalizeAvatarStyle(style)
	if s == "" {
		return "", nil
	}
	if _, ok := allowedAvatarStyleSet[s]; !ok {
		return "", fmt.Errorf("%w: avatar_style is not in the allowed gallery", ErrValidation)
	}
	return s, nil
}

// ValidateAvatarChoice validates style+seed together.
// Both empty => legacy. Seed set with empty style => lorelei (ADR-0017 rows).
// Style set with empty seed is rejected.
func ValidateAvatarChoice(style, seed string) (normStyle, normSeed string, err error) {
	normSeed, err = ValidateAvatarSeed(seed)
	if err != nil {
		return "", "", err
	}
	normStyle, err = ValidateAvatarStyle(style)
	if err != nil {
		return "", "", err
	}
	if normSeed == "" {
		if normStyle != "" {
			return "", "", fmt.Errorf("%w: avatar_style requires avatar_seed", ErrValidation)
		}
		return "", "", nil
	}
	if normStyle == "" {
		normStyle = DefaultAvatarStyle
	}
	return normStyle, normSeed, nil
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
