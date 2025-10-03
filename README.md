# grpcmux

```
go get github.com/lesomnus/grpcmux
```

It helps to serve both gRPC server and HTTP server on the same port with gRPC-web support.
It includes code adapted from [improbable-eng/grpc-web/go/grpcweb](https://github.com/improbable-eng/grpc-web/tree/1d9bbb09a0990bdaff0e37499570dbc7d6e58ce8/go/grpcweb), licensed under the Apache License 2.0, with WebSocket functionality has been removed.

## Usage

```go
import (
	...
	"github.com/lesomnus/grpcmux"
)

func main() {
	grpc_server := grpc.NewServer()

	// Register your server.
	pb.RegisterRouteGuideServer(grpcServer, newServer())

	mux := grpcmux.New(grpc_server,
		// Given function is invoked for gRPC-web or normal HTTP request.
		grpcmux.WithWebMiddleware(func(w http.ResponseWriter, r *http.Request, next http.Handler) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-User-Agent, X-Grpc-Web")

			next.ServeHTTP(w, r)
		}
		grpcmux.WithHttpHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Your HTTP handler here.	
		})),
	))

	listener, err := net.Listen("tcp", ":15300")
	if err != nil {
		panic(err)
	}

	http_server := http.Server{
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}
	http_server.Serve(listener)
}
```
