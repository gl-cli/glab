package expirationdate

import (
	"testing"
	"time"
)

func TestExpirationDate_Set(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name    string
		a       ExpirationDate
		args    args
		want    ExpirationDate
		wantErr bool
	}{
		{
			"valid date",
			ExpirationDate(time.Date(2000, 11, 1, 0, 0, 0, 0, time.UTC)),
			args{value: "2023-10-01"},
			ExpirationDate(time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC)),
			false,
		},
		{
			"invalid date",
			ExpirationDate(time.Date(2000, 11, 1, 0, 0, 0, 0, time.UTC)),
			args{value: "2023-99-88"},
			ExpirationDate(time.Date(2000, 11, 1, 0, 0, 0, 0, time.UTC)),
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.a.Set(tt.args.value); (err != nil) != tt.wantErr {
				t.Errorf("Set(%s) error = %v, wantErr %v", tt.args.value, err, tt.wantErr)
			}

			if tt.a != tt.want {
				t.Errorf("Set(%s) expected = %v, got %v", tt.args.value, tt.a, tt.want)
			}

			if tt.a.Type() != "DATE" {
				t.Errorf("Type() expected = %v, got %v", "DATE", tt.a.Type())
			}
			if !tt.wantErr && tt.a.String() != tt.args.value {
				t.Errorf("String() expected = %v, got %v", tt.args.value, tt.a.String())
			}
		})
	}
}
