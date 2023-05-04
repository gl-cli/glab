package api

import "errors"

// ErrIssuableUserNotSubscribed received when trying to unsubscribe from issue the user not subscribed to
var ErrIssuableUserNotSubscribed = errors.New("you are not subscribed to this issue")
