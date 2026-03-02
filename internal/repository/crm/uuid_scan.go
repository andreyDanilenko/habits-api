package crm

import (
	"database/sql/driver"
	"fmt"

	"github.com/google/uuid"
)

// uuidString scans PostgreSQL UUID into string (lib/pq returns UUID as []byte or string).
type uuidString string

func (u *uuidString) Scan(value interface{}) error {
	if value == nil {
		*u = ""
		return nil
	}
	switch v := value.(type) {
	case []byte:
		if len(v) == 16 {
			uid, err := uuid.FromBytes(v)
			if err != nil {
				return err
			}
			*u = uuidString(uid.String())
			return nil
		}
		*u = uuidString(v)
		return nil
	case string:
		*u = uuidString(v)
		return nil
	default:
		return fmt.Errorf("uuidString: cannot scan type %T", value)
	}
}

func (u uuidString) Value() (driver.Value, error) {
	if u == "" {
		return nil, nil
	}
	return string(u), nil
}

// uuidBytesToString converts driver UUID (16 bytes or 36-char string) to string.
func uuidBytesToString(b []byte) string {
	if b == nil || len(b) == 0 {
		return ""
	}
	if len(b) == 16 {
		uid, err := uuid.FromBytes(b)
		if err != nil {
			return ""
		}
		return uid.String()
	}
	return string(b)
}
