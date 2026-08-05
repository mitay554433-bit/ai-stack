package cosl

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"emergion-sovereign-runtime/internal/core"
)

const Prefix = "@COSL1|"

func canonical(e core.Event) ([]byte, error) {
	e.SelfHash = ""
	return json.Marshal(e)
}

func Seal(e core.Event) (core.Event, error) {
	b, err := canonical(e)
	if err != nil {
		return core.Event{}, err
	}
	s := sha256.Sum256(b)
	e.SelfHash = hex.EncodeToString(s[:])
	return e, nil
}

func Encode(e core.Event) (string, error) {
	sealed, err := Seal(e)
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(sealed)
	if err != nil {
		return "", err
	}
	return Prefix + string(b), nil
}

func Decode(line string) (core.Event, error) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, Prefix) {
		return core.Event{}, fmt.Errorf("invalid COSL prefix")
	}
	var e core.Event
	if err := json.Unmarshal([]byte(strings.TrimPrefix(line, Prefix)), &e); err != nil {
		return core.Event{}, err
	}
	claimed := e.SelfHash
	sealed, err := Seal(e)
	if err != nil {
		return core.Event{}, err
	}
	if claimed == "" || claimed != sealed.SelfHash {
		return core.Event{}, fmt.Errorf("event hash mismatch")
	}
	return e, nil
}
