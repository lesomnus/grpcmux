package grpcmux

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"io"
	"net/http"
	"strings"

	"golang.org/x/net/http2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
)

type grpcWebHandler struct {
	s *grpc.Server
}

func (m grpcWebHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ProtoMajor = 2
	content_type_origin := r.Header.Get("content-type")
	content_type_normal := GrpcWebContentType
	is_proto_text := strings.HasPrefix(content_type_origin, GrpcWebTextContentType)
	if is_proto_text {
		content_type_normal = GrpcWebTextContentType
	}

	r.Header.Set("content-type", strings.Replace(content_type_origin, content_type_normal, GrpcContentType, 1))

	// Remove content-length header since it represents http1.1 payload size, not the sum of the h2
	// DATA frame payload lengths. https://http2.github.io/http2-spec/#malformed This effectively
	// switches to chunked encoding which is the default for h2
	r.Header.Del("content-length")

	if r.Method == http.MethodOptions {
		return
	}

	if is_proto_text {
		w_ := &grpcWebTextResponseWriter{ResponseWriter: w}
		w_.reset()
		w = w_

		decoder := base64.NewDecoder(base64.StdEncoding, r.Body)
		r.Body = struct {
			io.Reader
			io.Closer
		}{
			decoder,
			r.Body,
		}
	}
	gw := &grpcWebResponseWriter{
		base:   w,
		header: make(http.Header),

		content_type: content_type_normal,
	}

	m.s.ServeHTTP(gw, r)

	if gw.wrote_h || gw.wrote_b {
		trailers := http.Header{}
		for k, vs := range gw.header {
			if _, ok := r.Header[k]; ok {
				continue
			}

			k = strings.Replace(k, http2.TrailerPrefix, "", 1)
			// gRPC-Web spec says that must use lower-case header/trailer names. See
			// "HTTP wire protocols" section in
			// https://github.com/grpc/grpc/blob/master/doc/PROTOCOL-WEB.md#protocol-differences-vs-grpc-over-http2
			k = strings.ToLower(k)
			trailers[k] = vs
		}

		var trailer_header = [...]byte{1 << 7, 0, 0, 0, 0}

		buf := bytes.Buffer{}
		trailers.Write(&buf)
		binary.BigEndian.PutUint32(trailer_header[1:5], uint32(buf.Len()))
		w.Write(trailer_header[:])
		w.Write(buf.Bytes())
	} else {
		gw.WriteHeader(http.StatusOK)
	}

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

type grpcWebResponseWriter struct {
	// Flush must be called on this writer before returning to ensure encoded buffer is flushed
	base   http.ResponseWriter
	header http.Header

	// The standard "application/grpc" content-type will be replaced with this.
	content_type string

	wrote_h bool
	wrote_b bool
}

func (w *grpcWebResponseWriter) Header() http.Header {
	return w.header
}

func (w *grpcWebResponseWriter) Write(b []byte) (int, error) {
	if !w.wrote_h {
		w.prepareHeader()
	}
	w.wrote_b, w.wrote_h = true, true

	return w.base.Write(b)
}

func (w *grpcWebResponseWriter) WriteHeader(code int) {
	w.prepareHeader()
	w.wrote_h = true

	w.base.WriteHeader(code)
}

func (w *grpcWebResponseWriter) Flush() {
	if !(w.wrote_h || w.wrote_b) {
		return
	}

	// Work around the fact that WriteHeader and a call to Flush would have caused a 200 response.
	// This is the case when there is no payload.
	if f, ok := w.base.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *grpcWebResponseWriter) prepareHeader() {
	h := w.base.Header()

	w.header.Del("trailer")
	for k, vs := range w.header {
		k = strings.Replace(k, http2.TrailerPrefix, "", 1)
		if strings.EqualFold(k, "content-type") {
			for i, v := range vs {
				vs[i] = strings.Replace(v, GrpcContentType, w.content_type, 1)
			}
		}

		k = http.CanonicalHeaderKey(k)
		h[k] = vs
	}
	h.Set("Access-Control-Expose-Headers", "grpc-status, grpc-message")
}

type grpcWebTextResponseWriter struct {
	http.ResponseWriter
	encoder io.WriteCloser
}

func (w *grpcWebTextResponseWriter) reset() {
	w.encoder = base64.NewEncoder(base64.StdEncoding, w.ResponseWriter)
}

func (w *grpcWebTextResponseWriter) Write(b []byte) (int, error) {
	return w.encoder.Write(b)
}

func (w *grpcWebTextResponseWriter) WriteHeader(code int) {
	w.ResponseWriter.WriteHeader(code)
}

func (w *grpcWebTextResponseWriter) Flush() {
	// Flush the base64 encoder by closing it. Grpc-web permits multiple padded base64 parts:
	// https://github.com/grpc/grpc/blob/master/doc/PROTOCOL-WEB.md
	err := w.encoder.Close()
	if err != nil {
		// Must ignore this error since Flush() is not defined as returning an error.
		// The error occurs only when writing to underlying writer is failed.
		grpclog.Errorf("ignoring error Flushing base64 encoder: %v", err)
	}
	w.reset()
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
