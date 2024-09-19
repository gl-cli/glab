package expirationdate

import (
	"time"
)

// ExpirationDate specifies the expiration date for an access token
// the String(), Set() and Type() functions are required for Cobra
// to parse the expiration date as command line argument.
type ExpirationDate time.Time

func (a *ExpirationDate) String() string {
	return time.Time(*a).Format(time.DateOnly)
}

func (a *ExpirationDate) Set(value string) error {
	v, err := time.Parse(time.DateOnly, value)
	if err != nil {
		return err
	}
	*a = ExpirationDate(v)
	return nil
}

func (a *ExpirationDate) Type() string {
	return "DATE"
}
