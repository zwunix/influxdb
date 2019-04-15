package tsm1

import (
	"testing"

	"github.com/influxdata/influxql"
)

func TestValidateTagPredicate(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		wantErr bool
	}{
		{
			expr:    `"_m" = 'foo'`,
			wantErr: false,
		},
		{
			expr:    `_m = 'foo'`,
			wantErr: false,
		},
		{
			expr:    `_m = foo`,
			wantErr: true,
		},
		{
			expr:    `_m = 5`,
			wantErr: true,
		},
		{
			expr:    `_m =~ //`,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateTagPredicate(influxql.MustParseExpr(tt.expr)); (err != nil) != tt.wantErr {
				t.Errorf("ValidateTagPredicate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
