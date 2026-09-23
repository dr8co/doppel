package scanner

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestReadFilesFrom(t *testing.T) {
	longPath := strings.Repeat("a", 64*1024)
	tests := []struct {
		name          string
		input         string
		nullDelimited bool
		want          []string
	}{
		{name: "newline", input: "first\nsecond\n", want: []string{"first", "second"}},
		{name: "crlf", input: "first\r\nsecond\r\n", want: []string{"first", "second"}},
		{name: "trimmed spaces", input: "  first  \n\tpath with spaces\t\n", want: []string{"first", "path with spaces"}},
		{name: "unterminated", input: "first\nsecond", want: []string{"first", "second"}},
		{name: "nul", input: "  first  \x00\tsecond\t\x00", nullDelimited: true, want: []string{"first", "second"}},
		{name: "embedded newline", input: "first\npart\x00second\x00", nullDelimited: true, want: []string{"first\npart", "second"}},
		{name: "long record", input: longPath + "\n", want: []string{longPath}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadFilesFrom(strings.NewReader(tt.input), tt.nullDelimited, false)
			if err != nil {
				t.Fatalf("ReadFilesFrom() error = %v", err)
			}
			if strings.Join(got, "\x00") != strings.Join(tt.want, "\x00") {
				t.Fatalf("ReadFilesFrom() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReadFilesFromRejectsEmptyRecords(t *testing.T) {
	for _, tt := range []struct {
		name          string
		input         string
		nullDelimited bool
	}{
		{name: "empty line", input: "first\n\n"},
		{name: "empty nul record", input: "first\x00\x00", nullDelimited: true},
		{name: "empty final line", input: "\n"},
		{name: "whitespace-only line", input: " \t\n"},
		{name: "whitespace-only nul record", input: " \t\x00", nullDelimited: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ReadFilesFrom(strings.NewReader(tt.input), tt.nullDelimited, false)
			if err == nil || !strings.Contains(err.Error(), "empty path") {
				t.Fatalf("ReadFilesFrom() error = %v, want empty path error", err)
			}
		})
	}
}

func TestReadFilesFromIgnoresEmptyRecords(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		nullDelimited bool
		want          []string
	}{
		{name: "newline", input: "first\n\n \t\nsecond\n", want: []string{"first", "second"}},
		{name: "nul", input: "first\x00\x00 \t\x00second\x00", nullDelimited: true, want: []string{"first", "second"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadFilesFrom(strings.NewReader(tt.input), tt.nullDelimited, true)
			if err != nil {
				t.Fatalf("ReadFilesFrom() error = %v", err)
			}
			if strings.Join(got, "\x00") != strings.Join(tt.want, "\x00") {
				t.Fatalf("ReadFilesFrom() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReadFilesFromReturnsReadErrors(t *testing.T) {
	wantErr := errors.New("read failed")
	reader := errorReader{err: wantErr}

	_, err := ReadFilesFrom(reader, false, false)
	if err == nil || !errors.Is(err, wantErr) {
		t.Fatalf("ReadFilesFrom() error = %v, want wrapped read error", err)
	}
}

type errorReader struct {
	err error
}

func (r errorReader) Read([]byte) (int, error) {
	return 0, r.err
}

var _ io.Reader = errorReader{}
