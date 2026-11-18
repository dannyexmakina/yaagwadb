package main

import "testing"

const (
	hranaTestWithinLimit   = 1 << 20
	hranaTestOverLimit     = grpcMaximumMessageSize + 1
	hranaTestNegativeLimit = -1
	hranaTestResultFormat  = "limit %d: %v"
)

func TestCheckHranaConfig(t *testing.T) {
	configuration := &fileConfig{}
	configuration.Hrana.FileDataSizeLimit = hranaTestWithinLimit
	if err := checkHranaConfig(configuration); err != nil {
		t.Fatalf(hranaTestResultFormat, hranaTestWithinLimit, err)
	}
	for _, refused := range []int64{hranaTestOverLimit, noFileDataSizeLimit, hranaTestNegativeLimit} {
		configuration.Hrana.FileDataSizeLimit = refused
		if err := checkHranaConfig(configuration); err == nil {
			t.Errorf(hranaTestResultFormat, refused, err)
		}
	}
}
