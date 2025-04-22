package totp

import (
	"testing"
	"time"
)

func TestGenerateAuthCode(t *testing.T) {
	//goland:noinspection SpellCheckingInspection
	state, err := NewState("cnOgv/KdpLoP6Nbh0GMkXkPXALQ=", "")
	if err != nil {
		t.Error(err)
	}

	code, err := state.GenerateTotpCode("conf", time.Now())
	if err != nil {
		t.Error(err)
	}

	if len(code) != 5 {
		t.Errorf("len(code)=%d, expected 5 digit code", len(code))
	}
}
