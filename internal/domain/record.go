package domain

import (
	"database/sql/driver"
	"encoding/json"
)

type Record struct {
	Entries map[string]any
}

func (rec *Record) normalize() {
	if len(rec.Entries) == 0 {
		rec.Entries = nil
	}
}

func (rec *Record) UnmarshalJSON(data []byte) error {
	err := json.Unmarshal(data, &rec.Entries)
	if err != nil {
		return err
	}
	rec.normalize()
	return nil
}

func (rec Record) MarshalJSON() ([]byte, error) {
	if rec.Entries == nil {
		return json.Marshal(make(map[string]any))
	}
	return json.Marshal(rec.Entries)
}

func (rec Record) Value() (driver.Value, error) {
	if rec.Entries == nil {
		return make(map[string]any), nil
	}
	return rec.Entries, nil
}
