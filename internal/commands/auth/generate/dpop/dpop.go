package dpop

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"math/big"

	"github.com/golang-jwt/jwt/v5"
)

/*
MIT License

Copyright (c) 2023 Software go-dpop
Portions Copyright 2024 Gitlab B.V
Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

type ecdsaJWK struct {
	X   string `json:"x"`
	Y   string `json:"y"`
	Crv string `json:"crv"`
	Kty string `json:"kty"`
}

type rsaJWK struct {
	Exponent string `json:"e"`
	Modulus  string `json:"n"`
	Kty      string `json:"kty"`
}

type ed25519JWK struct {
	PublicKey string `json:"x"`
	Kty       string `json:"kty"`
}

type HTTPVerb string

// HTTP method supported by the package.
const (
	GET     HTTPVerb = "GET"
	POST    HTTPVerb = "POST"
	PUT     HTTPVerb = "PUT"
	DELETE  HTTPVerb = "DELETE"
	PATCH   HTTPVerb = "PATCH"
	HEAD    HTTPVerb = "HEAD"
	OPTIONS HTTPVerb = "OPTIONS"
	TRACE   HTTPVerb = "TRACE"
	CONNECT HTTPVerb = "CONNECT"
)

type ProofTokenClaims struct {
	*jwt.RegisteredClaims

	// the `htm` (HTTP Method) claim. See https://datatracker.ietf.org/doc/html/draft-ietf-oauth-dpop#section-4.2
	Method HTTPVerb `json:"htm"`

	// the `htu` (HTTP URL) claim. See https://datatracker.ietf.org/doc/html/draft-ietf-oauth-dpop#section-4.2
	URL string `json:"htu"`

	// the `ath` (Authorization Token Hash) claim. See https://datatracker.ietf.org/doc/html/draft-ietf-oauth-dpop#section-4.2
	AccessTokenHash string `json:"ath,omitempty"`

	// the `nonce` claim. See https://datatracker.ietf.org/doc/html/draft-ietf-oauth-dpop#section-4.2
	Nonce string `json:"nonce,omitempty"`
}

func Reflect(v any) (any, error) {
	switch v := v.(type) {
	case *ecdsa.PublicKey:
		return &ecdsaJWK{
			X:   base64.RawURLEncoding.EncodeToString(v.X.Bytes()),
			Y:   base64.RawURLEncoding.EncodeToString(v.Y.Bytes()),
			Crv: v.Curve.Params().Name,
			Kty: "EC",
		}, nil
	case *rsa.PublicKey:
		return &rsaJWK{
			Exponent: base64.RawURLEncoding.EncodeToString(big.NewInt(int64(v.E)).Bytes()),
			Modulus:  base64.RawURLEncoding.EncodeToString(v.N.Bytes()),
			Kty:      "RSA",
		}, nil
	case ed25519.PublicKey:
		return &ed25519JWK{
			PublicKey: base64.RawURLEncoding.EncodeToString(v),
			Kty:       "OKP",
		}, nil
	}
	return nil, fmt.Errorf("unsupported key type")
}
