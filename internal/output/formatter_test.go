package output

import "testing"

// TestFormatBytes tests the FormatBytes function to ensure it correctly converts byte counts into human-readable formats.
func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{
			name:     "zero bytes",
			bytes:    0,
			expected: "0 B",
		},
		{
			name:     "bytes",
			bytes:    500,
			expected: "500 B",
		},
		{
			name:     "kilobytes",
			bytes:    1500,
			expected: "1.5 KB",
		},
		{
			name:     "megabytes",
			bytes:    1500000,
			expected: "1.4 MB",
		},
		{
			name:     "gigabytes",
			bytes:    1500000000,
			expected: "1.4 GB",
		},
		{
			name:     "terabytes",
			bytes:    1500000000000,
			expected: "1.4 TB",
		},
		{
			name:     "petabytes",
			bytes:    1500000000000000,
			expected: "1.3 PB",
		},
		{
			name:     "exabytes",
			bytes:    1500000000000000000,
			expected: "1.3 EB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("FormatBytes(%d) = %s, want %s", tt.bytes, result, tt.expected)
			}
		})
	}
}
