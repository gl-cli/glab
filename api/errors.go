package api

import "errors"

// ErrIssuableUserNotSubscribed received when trying to unsubscribe from an issue the user is not subscribed to
var ErrIssuableUserNotSubscribed = errors.New("you are not subscribed to this issue")

// ErrIssuableUserAlreadySubscribed received when trying to subscribe to an issue the user is already subscribed to
var ErrIssuableUserAlreadySubscribed = errors.New("you are already subscribed to this issue")
