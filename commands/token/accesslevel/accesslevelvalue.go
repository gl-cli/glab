package accesslevel

import (
	"fmt"
	"strings"

	"github.com/xanzy/go-gitlab"
)

type AccessLevel struct {
	Value gitlab.AccessLevelValue
}

var accessLevels = map[string]AccessLevel{
	"no":         {gitlab.NoPermissions},
	"minimal":    {gitlab.MinimalAccessPermissions},
	"guest":      {gitlab.GuestPermissions},
	"reporter":   {gitlab.ReporterPermissions},
	"developer":  {gitlab.DeveloperPermissions},
	"maintainer": {gitlab.MaintainerPermissions},
	"owner":      {gitlab.OwnerPermissions},
	"admin":      {gitlab.AdminPermissions},
}

func CreateAccessLevelFromString(s string) (AccessLevel, error) {
	if level, ok := accessLevels[strings.ToLower(s)]; ok {
		return level, nil
	}
	return AccessLevel{gitlab.NoPermissions}, fmt.Errorf("invalid access level: %s", s)
}

func (a *AccessLevel) String() string {
	for name, level := range accessLevels {
		if level.Value == a.Value {
			return name
		}
	}
	return fmt.Sprintf("%d", a.Value)
}

func (a *AccessLevel) Set(value string) (err error) {
	*a, err = CreateAccessLevelFromString(value)
	return err
}

func (a *AccessLevel) Type() string {
	return "AccessLevel"
}
