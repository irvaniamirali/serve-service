package deployments

import "testing"

func TestCanTransition(t *testing.T) {
	tests := []struct {
		from string
		to   string
		want bool
	}{
		{from: StatusQueued, to: StatusBuilding, want: true},
		{from: StatusQueued, to: StatusCancelled, want: true},
		{from: StatusBuilding, to: StatusDeploying, want: true},
		{from: StatusBuilding, to: StatusFailed, want: true},
		{from: StatusDeploying, to: StatusHealthy, want: true},
		{from: StatusDeploying, to: StatusCancelled, want: true},
		{from: StatusQueued, to: StatusHealthy, want: false},
		{from: StatusHealthy, to: StatusFailed, want: false},
		{from: StatusFailed, to: StatusBuilding, want: false},
		{from: StatusCancelled, to: StatusQueued, want: false},
		{from: "unknown", to: StatusBuilding, want: false},
	}
	for _, test := range tests {
		t.Run(test.from+"_to_"+test.to, func(t *testing.T) {
			if got := CanTransition(test.from, test.to); got != test.want {
				t.Errorf("CanTransition(%q, %q) = %t, want %t", test.from, test.to, got, test.want)
			}
		})
	}
}
