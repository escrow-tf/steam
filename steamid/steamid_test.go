package steamid

import "testing"

func TestEmptySteamID64(t *testing.T) {
	_, err := ParseSteamID64("")
	if err == nil {
		t.Error("expected error, got none")
	}
}

func TestNoneNumberSteamID64(t *testing.T) {
	_, err := ParseSteamID64("not a number")
	if err == nil {
		t.Error("expected error, got none")
	}
}

func TestValidSteamID64(t *testing.T) {
	steamID, err := ParseSteamID64("76561197960287930")
	if err != nil {
		t.Error(err)
	}

	if !steamID.IsValid() {
		t.Error("steamID is not valid")
	}

	if !steamID.IsValidIndividual() {
		t.Error("steamID is not valid individual")
	}
}
