package docker

import (
	"testing"
)

func TestParseStatsLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		wantName string
		wantCPU  string
		wantMem  string
		wantErr  bool
	}{
		{
			name:     "basic",
			line:     "/my-container\t0.50%\t50MiB / 1GiB\t5.00%\t1.2kB / 680B\t5.5MB / 0B",
			wantName: "my-container",
			wantCPU:  "0.50%",
			wantMem:  "50MiB / 1GiB",
			wantErr:  false,
		},
		{
			name:     "no_slash",
			line:     "web\t0.00%\t10MiB / 500MiB\t2.00%\t100B / 200B\t0B / 0B",
			wantName: "web",
			wantCPU:  "0.00%",
			wantMem:  "10MiB / 500MiB",
			wantErr:  false,
		},
		{
			name:    "too_few_fields",
			line:    "/c1\t0.50%\t50MiB",
			wantErr: true,
		},
		{
			name:    "empty",
			line:    "",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stats, err := parseStatsLine(tc.name, tc.line)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if stats.Name != tc.wantName {
				t.Errorf("Name = %q, want %q", stats.Name, tc.wantName)
			}
			if stats.CPUPerc != tc.wantCPU {
				t.Errorf("CPUPerc = %q, want %q", stats.CPUPerc, tc.wantCPU)
			}
			if stats.MemUsage != tc.wantMem {
				t.Errorf("MemUsage = %q, want %q", stats.MemUsage, tc.wantMem)
			}
		})
	}
}

func TestContainerStatsStruct(t *testing.T) {
	s := &ContainerStats{
		Name:     "test",
		CPUPerc:  "1.00%",
		MemUsage: "100MiB / 1GiB",
		MemPerc:  "10.00%",
		NetIO:    "1kB / 2kB",
		BlockIO:  "0B / 0B",
	}
	if s.Name != "test" || s.CPUPerc != "1.00%" {
		t.Error("ContainerStats fields not set correctly")
	}
}