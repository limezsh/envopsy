package builtin

import "testing"

func TestIgnored(t *testing.T) {
	if !Ignored("NO_COLOR") {
		t.Fatal("NO_COLOR should be ignored")
	}
	if !Ignored("NODE_ENV") {
		t.Fatal("NODE_ENV should be ignored")
	}
	if !Ignored("PATH") {
		t.Fatal("PATH should be ignored")
	}
	if Ignored("PORT") {
		t.Fatal("PORT must not be ignored")
	}
	if Ignored("DATABASE_URL") {
		t.Fatal("DATABASE_URL must not be ignored")
	}
	if Ignored("GITHUB_TOKEN") {
		t.Fatal("GITHUB_TOKEN must not be ignored")
	}
}
