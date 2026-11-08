package xui

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

const maxKeepAliveSeconds = 65535

// KeepAliveWire is a panel keepAlive value: a JSON number, or an AmneziaWG
// range string such as "15-25". Empty means unset / zero on the wire.
type KeepAliveWire string

func (v KeepAliveWire) String() string { return string(v) }

func (v KeepAliveWire) IsZero() bool {
	return v == "" || v == "0"
}

func ParseKeepAliveWire(s string) (KeepAliveWire, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "0", nil
	}
	if err := (KeepAliveWire(s)).Validate(true); err != nil {
		return "", err
	}
	return KeepAliveWire(s), nil
}

func (v KeepAliveWire) Validate(allowRange bool) error {
	text := string(v)
	parts := strings.Split(text, "-")
	if len(parts) == 1 {
		n, err := strconv.Atoi(text)
		if err != nil || strconv.Itoa(n) != text || n < 0 || n > maxKeepAliveSeconds {
			return fmt.Errorf("keepAlive must be an integer from 0 to %d", maxKeepAliveSeconds)
		}
		return nil
	}
	if len(parts) != 2 || !allowRange {
		return fmt.Errorf("keepAlive ranges are supported only for AmneziaWG clients")
	}
	min, errMin := strconv.Atoi(parts[0])
	max, errMax := strconv.Atoi(parts[1])
	if errMin != nil || errMax != nil || strconv.Itoa(min) != parts[0] || strconv.Itoa(max) != parts[1] || min < 0 || max > maxKeepAliveSeconds || min > max {
		return fmt.Errorf("keepAlive range must be ordered integers from 0 to %d", maxKeepAliveSeconds)
	}
	return nil
}

func (v KeepAliveWire) MarshalJSON() ([]byte, error) {
	if v == "" {
		return []byte("0"), nil
	}
	if _, err := strconv.ParseInt(string(v), 10, 64); err == nil {
		return []byte(string(v)), nil
	}
	if err := v.Validate(true); err != nil || !strings.Contains(string(v), "-") {
		return nil, fmt.Errorf("invalid keepAlive %q", v)
	}
	return json.Marshal(string(v))
}

func (v *KeepAliveWire) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*v = ""
		return nil
	}
	if len(data) > 0 && data[0] == '"' {
		var text string
		if err := json.Unmarshal(data, &text); err != nil {
			return err
		}
		candidate := KeepAliveWire(text)
		if err := candidate.Validate(true); err != nil {
			return err
		}
		*v = candidate
		return nil
	}
	var number json.Number
	if err := json.Unmarshal(data, &number); err != nil {
		return fmt.Errorf("keepAlive must be a number or an AmneziaWG range string")
	}
	n, err := strconv.ParseInt(string(number), 10, 64)
	if err != nil {
		return fmt.Errorf("keepAlive must be an integer")
	}
	*v = KeepAliveWire(strconv.FormatInt(n, 10))
	return nil
}
