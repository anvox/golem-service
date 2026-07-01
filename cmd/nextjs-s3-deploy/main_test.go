package main

import (
	"reflect"
	"testing"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    *Options
		wantErr bool
	}{
		{
			name: "valid arguments with space",
			args: []string{"dist/apps/kds", "--s3-bucket", "my-bucket", "--log", "upload.log"},
			want: &Options{
				WorkingDir: "dist/apps/kds",
				S3Bucket:   "my-bucket",
				LogFile:    "upload.log",
			},
			wantErr: false,
		},
		{
			name: "valid arguments with equal sign",
			args: []string{"dist/apps/kds", "--s3-bucket=my-bucket", "--log=upload.log"},
			want: &Options{
				WorkingDir: "dist/apps/kds",
				S3Bucket:   "my-bucket",
				LogFile:    "upload.log",
			},
			wantErr: false,
		},
		{
			name: "valid arguments with verbose",
			args: []string{"dist/apps/kds", "--s3-bucket=my-bucket", "--verbose"},
			want: &Options{
				WorkingDir: "dist/apps/kds",
				S3Bucket:   "my-bucket",
				Verbose:    true,
			},
			wantErr: false,
		},
		{
			name:    "missing working dir",
			args:    []string{"--s3-bucket=my-bucket"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "missing bucket",
			args:    []string{"dist/apps/kds"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "too many positionals",
			args:    []string{"dist/apps/kds", "extra-arg", "--s3-bucket=my-bucket"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "unknown flag",
			args:    []string{"dist/apps/kds", "--s3-bucket=my-bucket", "--unknown-flag"},
			want:    nil,
			wantErr: true,
		},
		{
			name: "help flag",
			args: []string{"--help"},
			want: &Options{
				Help: true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetContentType(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"index.html", "text/html"},
		{"style.css", "text/css"},
		{"app.js", "application/javascript"},
		{"data.json", "application/json"},
		{"image.png", "image/png"},
		{"photo.jpg", "image/jpeg"},
		{"photo.jpeg", "image/jpeg"},
		{"vector.svg", "image/svg+xml"},
		{"favicon.ico", "image/x-icon"},
		{"note.txt", "text/plain"},
		{"feed.xml", "text/xml"},
		{"font.woff", "font/woff"},
		{"font.woff2", "font/woff2"},
		{"font.ttf", "font/ttf"},
		{"font.otf", "font/otf"},
		{"app.js.map", "application/json"},
		{"unknown.xyz", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := getContentType(tt.path)
			if tt.path == "unknown.xyz" {
				if got == "" {
					t.Errorf("getContentType(%q) returned empty string", tt.path)
				}
			} else if got != tt.want {
				t.Errorf("getContentType(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
