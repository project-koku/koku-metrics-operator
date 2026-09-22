//
// Copyright 2024 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package crhchttp

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	metricscfgv1beta1 "github.com/project-koku/koku-metrics-operator/api/v1beta1"
	"github.com/project-koku/koku-metrics-operator/internal/packaging"
)

func TestGetClient(t *testing.T) {
	getClientTests := []struct {
		name               string
		config             AuthConfig
		conErr             error
		insecureSkipVerify bool
	}{
		{
			name:               "no validate cert returns insecureSkipVerify false",
			config:             AuthConfig{ValidateCert: false},
			insecureSkipVerify: true,
		},
		{
			name:               "validate cert returns insecureSkipVerify true",
			config:             AuthConfig{ValidateCert: true},
			insecureSkipVerify: false,
		},
	}
	for _, tt := range getClientTests {
		t.Run(tt.name, func(t *testing.T) {

			result := GetClient(tt.config.ValidateCert)
			client, ok := result.(*http.Client)
			if !ok {
				t.Errorf("'%s' expected client to be http.Client type, got %T", tt.name, result)
			}
			transport, ok := client.Transport.(*http.Transport)
			if !ok {
				t.Errorf("'%s' expected transport to be http.Transport type, got %T", tt.name, client.Transport)
			}
			if tt.insecureSkipVerify != transport.TLSClientConfig.InsecureSkipVerify {
				t.Errorf("'%s' expected insecureSkipVerify to be %v, got %v", tt.name, tt.insecureSkipVerify, transport.TLSClientConfig.InsecureSkipVerify)
			}
		})
	}
}

func TestGetMultiPartBodyAndHeaders(t *testing.T) {
	getMultiPartBodyAndHeadersTests := []struct {
		name                string
		filename            string
		expectedNotNil      bool
		expectedErrNotNil   bool
		expectedContentType string
	}{
		{
			name:                "valid file returns correct things",
			filename:            "config.go",
			expectedNotNil:      true,
			expectedErrNotNil:   false,
			expectedContentType: "multipart/form-data",
		},
		{
			name:                "invalid file raises error",
			filename:            "file-does-not-exist.go",
			expectedNotNil:      false,
			expectedErrNotNil:   true,
			expectedContentType: "",
		},
	}
	for _, tt := range getMultiPartBodyAndHeadersTests {
		t.Run(tt.name, func(t *testing.T) {
			body, s, contentLength, err := GetMultiPartBodyAndHeaders(tt.filename)
			if tt.expectedNotNil != (body != nil) {
				t.Errorf("'%s' test expected not-nil body, got %v", tt.name, body)
			}
			if tt.expectedErrNotNil != (err != nil) {
				t.Errorf("'%s' test expected error, got %v", tt.name, err)
			}
			if tt.expectedContentType != "" && !strings.Contains(s, tt.expectedContentType) {
				t.Errorf("'%s' test expected content-type %s, got %v", tt.name, tt.expectedContentType, s)
			} else if tt.expectedContentType == "" && s != tt.expectedContentType {
				t.Errorf("'%s' test expected empty content-type, got %v", tt.name, s)
			}
			if body == nil {
				return
			}
			// Drain the streamed body: this releases the writer goroutine
			// and lets us verify the payload made it through the pipe.
			data, err := io.ReadAll(body)
			if err != nil {
				t.Errorf("'%s' test failed reading streamed body: %v", tt.name, err)
				return
			}
			original, err := os.ReadFile(tt.filename)
			if err != nil {
				t.Errorf("'%s' test failed reading fixture: %v", tt.name, err)
				return
			}
			if !strings.Contains(string(data), string(original)) {
				t.Errorf("'%s' test expected streamed body to contain the file payload", tt.name)
			}
			if int64(len(data)) != contentLength {
				t.Errorf("'%s' test expected framed length %d, got %d", tt.name, contentLength, len(data))
			}
		})
	}
}

func TestGetMultiPartBodyWriterErrors(t *testing.T) {
	t.Run("copy failure surfaces read error", func(t *testing.T) {
		// os.Open succeeds on directories but reads fail, which exercises
		// the writer goroutine copy-error branch.
		body, _, _, err := GetMultiPartBodyAndHeaders(t.TempDir())
		if err != nil {
			t.Fatalf("expected open to succeed on directory, got: %v", err)
		}
		_, err = io.Copy(io.Discard, body)
		if err == nil || !strings.Contains(err.Error(), "failed to copy file") {
			t.Errorf("expected failed-to-copy read error, got: %v", err)
		}
	})
	t.Run("abandoned reader surfaces writer error", func(t *testing.T) {
		body, _, _, err := GetMultiPartBodyAndHeaders("config.go")
		if err != nil {
			t.Fatalf("failed to build multipart body: %v", err)
		}
		// The pipe is unbuffered, so the writer is stuck in its first
		// write; closing the reader fails it deterministically.
		body.(io.Closer).Close()
		if _, err := io.Copy(io.Discard, body); err == nil {
			t.Errorf("expected error reading abandoned stream, got nil")
		}
	})
	t.Run("abandoning after payload surfaces close error", func(t *testing.T) {
		payload := filepath.Join(t.TempDir(), "payload.tar.gz")
		content := []byte("mid-stream close fixture payload")
		if err := os.WriteFile(payload, content, 0644); err != nil {
			t.Fatalf("failed to write fixture: %v", err)
		}
		// Drain once to learn the framed stream and its closing boundary.
		first, contentType, _, err := GetMultiPartBodyAndHeaders(payload)
		if err != nil {
			t.Fatalf("failed to build multipart body: %v", err)
		}
		full, err := io.ReadAll(first)
		if err != nil {
			t.Fatalf("failed to drain multipart body: %v", err)
		}
		boundary := strings.Split(contentType, "boundary=")[1]
		idx := bytes.LastIndex(full, []byte("--"+boundary+"--\r\n"))
		if idx < 0 {
			t.Fatalf("framed stream missing closing boundary")
		}
		// Consume everything the writer produces before its final write,
		// then abandon: the closing-boundary write must fail.
		// NOTE: each stream gets a random boundary, so the second stream
		// is validated against its own framing, not the drained one.
		second, secondContentType, _, err := GetMultiPartBodyAndHeaders(payload)
		if err != nil {
			t.Fatalf("failed to build multipart body: %v", err)
		}
		secondBoundary := strings.Split(secondContentType, "boundary=")[1]
		prefix := make([]byte, idx)
		if _, err := io.ReadFull(second, prefix); err != nil {
			t.Fatalf("failed to read stream prefix: %v", err)
		}
		if !bytes.HasPrefix(prefix, []byte("--"+secondBoundary)) {
			t.Errorf("stream prefix missing its boundary")
		}
		// NOTE: the prefix ends with the CRLF terminating the part body,
		// which belongs to the closing delimiter, not the payload.
		if !bytes.HasSuffix(prefix, append(content, '\r', '\n')) {
			t.Errorf("stream prefix missing the payload bytes")
		}
		second.(io.Closer).Close()
	})
}

func TestCloseBody(t *testing.T) {
	// Non-closer bodies are a no-op and must not panic.
	closeBody(strings.NewReader("not a closer"))
	// Closer bodies are released.
	pr, pw := io.Pipe()
	_ = pw.Close()
	closeBody(pr)
}

func TestUpload(t *testing.T) {
	payload := filepath.Join(t.TempDir(), "test-payload.tar.gz")
	content := []byte("fake tar.gz payload")
	if err := os.WriteFile(payload, content, 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}
	var gotMethod, gotContentType string
	var gotContentLength int64
	var gotTransferEncoding []string
	var gotBody []byte
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		gotContentLength = r.ContentLength
		gotTransferEncoding = r.TransferEncoding
		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("x-rh-insights-request-id", "test-request-id")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"account":"12345"}`))
	}))
	defer ts.Close()

	auth := &AuthConfig{
		Authentication:    metricscfgv1beta1.Token,
		BearerTokenString: "token",
		ValidateCert:      true,
		OperatorCommit:    "abc123",
		ClusterID:         "cluster-1",
	}
	body, contentType, contentLength, err := GetMultiPartBodyAndHeaders(payload)
	if err != nil {
		t.Fatalf("failed to build multipart body: %v", err)
	}
	info := packaging.FileInfoManifest{
		Files:     []string{"test-payload.tar.gz"},
		UUID:      "uuid-1",
		ClusterID: "cluster-1",
	}
	status, _, requestID, err := Upload(auth, contentType, "POST", ts.URL, body, contentLength, info, "test-payload.tar.gz")
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}
	if status != "202 Accepted" {
		t.Errorf("expected status 202 Accepted, got %q", status)
	}
	if requestID != "test-request-id" {
		t.Errorf("expected request ID test-request-id, got %q", requestID)
	}
	if gotMethod != "POST" {
		t.Errorf("expected POST, got %q", gotMethod)
	}
	if !strings.Contains(gotContentType, "multipart/form-data") {
		t.Errorf("expected multipart content type, got %q", gotContentType)
	}
	if !bytes.Contains(gotBody, content) {
		t.Errorf("server did not receive the payload bytes")
	}
	if len(gotTransferEncoding) != 0 {
		t.Errorf("expected Content-Length wire format, got chunked: %q", gotTransferEncoding)
	}
	if gotContentLength != contentLength {
		t.Errorf("expected Content-Length %d, got %d", contentLength, gotContentLength)
	}
	if int64(len(gotBody)) != contentLength {
		t.Errorf("expected framed body length %d, got %d", contentLength, len(gotBody))
	}
}

func TestUploadSetupFailure(t *testing.T) {
	payload := filepath.Join(t.TempDir(), "test-payload.tar.gz")
	if err := os.WriteFile(payload, []byte("fake tar.gz payload"), 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}
	auth := &AuthConfig{Authentication: metricscfgv1beta1.Token}
	body, contentType, contentLength, err := GetMultiPartBodyAndHeaders(payload)
	if err != nil {
		t.Fatalf("failed to build multipart body: %v", err)
	}
	// Unparsable URI fails request setup; the streamed body must still be
	// released instead of leaking the writer goroutine.
	_, _, _, err = Upload(auth, contentType, "POST", "://bad-uri", body, contentLength, packaging.FileInfoManifest{}, "test-payload.tar.gz")
	if err == nil || !strings.Contains(err.Error(), "could not setup the request") {
		t.Errorf("expected setup error, got: %v", err)
	}
}

// BenchmarkGetMultiPartBodyAndHeaders measures heap allocation while building
// the upload body. With io.Pipe streaming the allocation stays near-constant
// regardless of payload size; run with -benchmem to see it, e.g.:
//
//	go test ./internal/crhchttp/ -run XXX -bench BenchmarkGetMultiPartBodyAndHeaders -benchmem
func BenchmarkGetMultiPartBodyAndHeaders(b *testing.B) {
	sizes := []struct {
		name string
		size int64
	}{
		{"1MB", 1 << 20},
		{"10MB", 10 << 20},
		{"50MB", 50 << 20},
	}
	for _, sz := range sizes {
		b.Run(sz.name, func(b *testing.B) {
			payload := filepath.Join(b.TempDir(), "payload.tar.gz")
			f, err := os.Create(payload)
			if err != nil {
				b.Fatalf("failed to create payload fixture: %v", err)
			}
			if err := f.Truncate(sz.size); err != nil {
				b.Fatalf("failed to size payload fixture: %v", err)
			}
			if err := f.Close(); err != nil {
				b.Fatalf("failed to close payload fixture: %v", err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				body, _, contentLength, err := GetMultiPartBodyAndHeaders(payload)
				if err != nil {
					b.Fatalf("failed to build multipart body: %v", err)
				}
				n, err := io.Copy(io.Discard, body)
				if err != nil {
					b.Fatalf("failed to drain multipart body: %v", err)
				}
				if n != contentLength {
					b.Fatalf("expected framed length %d, got %d", contentLength, n)
				}
			}
		})
	}
}
