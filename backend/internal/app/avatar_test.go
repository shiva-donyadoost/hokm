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

func TestValidateAvatarChoice(t *testing.T) {
	st, sd, err := ValidateAvatarChoice("Avataaars", "Ember")
	if err != nil || st != "avataaars" || sd != "ember" {
		t.Fatalf("got %q %q %v", st, sd, err)
	}
	st, sd, err = ValidateAvatarChoice("", "panda")
	if err != nil || st != "lorelei" || sd != "panda" {
		t.Fatalf("legacy style default => %q %q %v", st, sd, err)
	}
	st, sd, err = ValidateAvatarChoice("", "")
	if err != nil || st != "" || sd != "" {
		t.Fatalf("empty => %q %q %v", st, sd, err)
	}
	if _, _, err := ValidateAvatarChoice("nope", "fox"); err == nil {
		t.Fatal("expected bad style error")
	}
	if _, _, err := ValidateAvatarChoice("lorelei", "nope"); err == nil {
		t.Fatal("expected bad seed error")
	}
	if _, _, err := ValidateAvatarChoice("avataaars", ""); err == nil {
		t.Fatal("expected style-without-seed error")
	}
}

func TestEffectiveAvatarStyle(t *testing.T) {
	if got := EffectiveAvatarStyle(""); got != "lorelei" {
		t.Fatalf("got %q", got)
	}
	if got := EffectiveAvatarStyle("avataaars"); got != "avataaars" {
		t.Fatalf("got %q", got)
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
