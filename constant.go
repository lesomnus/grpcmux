package grpcmux

// https://github.com/grpc/grpc/blob/master/doc/PROTOCOL-WEB.md#protocol-differences-vs-grpc-over-http2
const (
	GrpcContentType        = "application/grpc"
	GrpcWebContentType     = "application/grpc-web"
	GrpcWebTextContentType = "application/grpc-web-text"
)
