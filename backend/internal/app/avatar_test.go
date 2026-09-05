package app

import "testing"

func TestValidateAvatarSeed(t *testing.T) {
	ok, err := ValidateAvatarSeed("Fox")
	if err != nil || ok != "fox" {
		t.Fatalf("Fox => %q %v", ok, err)
	}
	empty, err := ValidateAvatarSeed("  ")
	if err != nil || empty != "" {
		t.Fatalf("blank => %q %v", empty, err)
	}
	if _, err := ValidateAvatarSeed("not-a-real-seed"); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestEffectiveAvatarSeed(t *testing.T) {
	if got := EffectiveAvatarSeed("panda", "uid", "name"); got != "panda" {
		t.Fatalf("got %q", got)
	}
	if got := EffectiveAvatarSeed("", "uid", "name"); got != "uid" {
		t.Fatalf("got %q", got)
	}
	if got := EffectiveAvatarSeed("", "", "name"); got != "name" {
		t.Fatalf("got %q", got)
	}
}
