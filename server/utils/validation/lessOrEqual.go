package validation

import (
	"database/sql/driver"
	"errors"
	"math"
	"reflect"
)

var (
	valuerType        = reflect.TypeOf((*driver.Valuer)(nil)).Elem()
	errUnexpectedType = errors.New("Non-numeric type could not be converted to float")
)

type lessOrEqualRule struct {
	message          string
	greaterThanValue interface{}
}

// Validate checks if the given value is valid or not.
func (r *lessOrEqualRule) Validate(value interface{}) error {
	lessValue, err := getFloatSwitchOnly(value)
	if err != nil {
		return err
	}

	greaterThanValue, err := getFloatSwitchOnly(r.greaterThanValue)
	if err != nil {
		return err
	}

	if lessValue > greaterThanValue {
		return errors.New(r.message)
	}

	return nil
}

// Error sets the error message for the rule.
func LessOrEqualCreate(message string, greaterThanValue interface{}) *lessOrEqualRule {
	return &lessOrEqualRule{
		message:          message,
		greaterThanValue: greaterThanValue,
	}
}

func getFloatSwitchOnly(unk interface{}) (float64, error) {
	switch i := unk.(type) {
	case float64:
		return i, nil
	case float32:
		return float64(i), nil
	case int64:
		return float64(i), nil
	case int32:
		return float64(i), nil
	case int:
		return float64(i), nil
	case uint64:
		return float64(i), nil
	case uint32:
		return float64(i), nil
	case uint:
		return float64(i), nil
	default:
		return math.NaN(), errUnexpectedType
	}
}
