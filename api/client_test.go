package api

import (
	"crypto/tls"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_tlsConfig(t *testing.T) {
	type args struct {
		host string
	}
	tests := []struct {
		name string
		args args
		want []uint16
	}{
		{
			name: "GitLab.com uses secure ciphers",
			args: args{
				host: "gitlab.com",
			},
			want: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			},
		},
		{
			name: "Other hosts aren't limited to secure ciphers",
			args: args{
				host: "gitlab.selfhosted.com",
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tlsConfig(tt.args.host)

			assert.Equal(t, tt.want, client.CipherSuites)
		})
	}
}
