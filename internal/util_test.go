package internal

import (
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestGetContentType(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		// 字体：之前会以 application/octet-stream 返回
		{"ajax/libs/font-awesome/6.4.0/webfonts/fa-solid-900.woff2", "font/woff2"},
		{"font.woff", "font/woff"},
		{"font.ttf", "font/ttf"},
		// selfhst 的图标就是 webp
		{"icons-local/selfhst-icons/webp/openobserve.webp", "image/webp"},
		{"bundle.js", "text/javascript; charset=utf-8"},
		{"bundle.css", "text/css; charset=utf-8"},
		{"index.html", "text/html; charset=utf-8"},
		{"bundle.js.map", "application/json; charset=utf-8"},
		{"logo.svg", "image/svg+xml"},
		{"logo.png", "image/png"},
		{"favicon.ico", "image/x-icon"},
		{"unknown.bin", "application/octet-stream"},
		{"noext", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.Eq(t, tt.want, getContentType(tt.path))
		})
	}
}
