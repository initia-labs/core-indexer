package db

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

// JSON custom type for JSON fields
type JSON json.RawMessage

// Scan scans value into JSON, implements sql.Scanner interface
func (j *JSON) Scan(value any) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSON value:", value))
	}

	result := json.RawMessage{}
	err := json.Unmarshal(bytes, &result)
	*j = JSON(result)
	return err
}

// Value return json value, implement driver.Valuer interface
func (j JSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return json.RawMessage(j).MarshalJSON()
}

// JSONB custom type for PostgreSQL JSONB fields
type JSONB json.RawMessage

// Scan scans value into JSONB, implements sql.Scanner interface
func (j *JSONB) Scan(value any) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", value))
	}

	result := json.RawMessage{}
	err := json.Unmarshal(bytes, &result)
	*j = JSONB(result)
	return err
}

// Value return jsonb value, implement driver.Valuer interface
func (j JSONB) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return json.RawMessage(j).MarshalJSON()
}
