package logs

import "testing"

func TestScoreMessage(t *testing.T) {
	t.Parallel()

	got := ScoreMessage("Blue Team", 100, "ES", "Web")
	want := "Team Blue Team scored 100 points for challenge ES (Web)"

	if got != want {
		t.Fatalf("ScoreMessage() = %q, want %q", got, want)
	}
}

func TestEnableMessage(t *testing.T) {
	t.Parallel()

	got := EnableMessage("ES", "Web", 100)
	want := "Challenge ES (Web) was enabled: 100 points"

	if got != want {
		t.Fatalf("EnableMessage() = %q, want %q", got, want)
	}
}

func TestDisableMessage(t *testing.T) {
	t.Parallel()

	got := DisableMessage("ES", "Web", 100)
	want := "Challenge ES (Web) was disabled: 100 points"

	if got != want {
		t.Fatalf("DisableMessage() = %q, want %q", got, want)
	}
}
