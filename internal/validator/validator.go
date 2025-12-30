package validator

import "github.com/garrettladley/thoop/internal/xerrors"

type Validator interface {
	// Validate validates the fields of the struct and returns a map of errors.
	// returns nil if no errors are found
	Validate() map[string]string
}

func Validate(v Validator) *xerrors.Error {
	if err := v.Validate(); err != nil {
		return xerrors.Validation(err)
	}
	return nil
}
