package validation

import "errors"

type greaterThanOrEqualRule struct {
	message   string
	lessValue interface{}
}

// Validate checks if the given value is valid or not.
func (r *greaterThanOrEqualRule) Validate(value interface{}) error {
	greaterThanValue, err := getFloatSwitchOnly(value)
	if err != nil {
		return err
	}

	lessValue, err := getFloatSwitchOnly(r.lessValue)
	if err != nil {
		return err
	}

	if greaterThanValue < lessValue {
		return errors.New(r.message)
	}

	return nil
}

// Error sets the error message for the rule.
func GreaterThanOrEqualCreate(message string, lessValue interface{}) *greaterThanOrEqualRule {
	return &greaterThanOrEqualRule{
		message:   message,
		lessValue: lessValue,
	}
}
