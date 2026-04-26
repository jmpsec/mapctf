package logs

import "testing"

func TestGenerateMessage(t *testing.T) {
	got := GenerateMessage(ActivityTeamScore, "Blue Team", 100, "Spain Challenge", "ES")
	want := "Team Blue Team scored 100 points for challenge Spain Challenge (ES)"

	if got != want {
		t.Fatalf("GenerateMessage() = %q, want %q", got, want)
	}
}
