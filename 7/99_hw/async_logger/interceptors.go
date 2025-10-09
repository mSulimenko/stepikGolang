package main

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"strings"
)

func makeStatUnaryInterceptor(st *StatsTracker, lt *LogTracker,
) func(context.Context, interface{}, *grpc.UnaryServerInfo, grpc.UnaryHandler) (interface{}, error) {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
	) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "metadata required")
		}

		consumers := md.Get("consumer")
		if len(consumers) == 0 {
			return nil, status.Error(codes.Unauthenticated, "consumer metadata required")
		}

		consumer := consumers[0]
		Method := info.FullMethod
		host := getClientAddr(ctx)
		if len(consumers) == 0 {
			return nil, status.Error(codes.Unauthenticated, ":authority metadata required")
		}

		st.SendStats(DataPoint{
			Consumer: consumer,
			Method:   Method,
		})

		lt.SendEvents(&Event{
			Consumer: consumer,
			Method:   Method,
			Host:     host,
		})

		return handler(ctx, req)
	}
}

func makeStatStreamInterceptor(st *StatsTracker, lt *LogTracker,
) func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler,
) error {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		md, ok := metadata.FromIncomingContext(ss.Context())
		if !ok {
			return status.Error(codes.Unauthenticated, "metadata required")
		}

		consumers := md.Get("consumer")
		if len(consumers) == 0 {
			return status.Error(codes.Unauthenticated, "consumer data required")
		}

		consumer := consumers[0]
		Method := info.FullMethod
		host := getClientAddr(ss.Context())
		if len(consumers) == 0 {
			return status.Error(codes.Unauthenticated, ":authority metadata required")
		}

		st.SendStats(DataPoint{
			Consumer: consumer,
			Method:   Method,
		})

		lt.SendEvents(&Event{
			Consumer: consumer,
			Method:   Method,
			Host:     host,
		})

		return handler(srv, ss)
	}
}

func makeAclUnaryInterceptor(acl map[string][]string,
) func(context.Context, interface{}, *grpc.UnaryServerInfo, grpc.UnaryHandler) (interface{}, error) {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
	) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "metadata required")
		}

		consumers := md.Get("consumer")
		if len(consumers) == 0 {
			return nil, status.Error(codes.Unauthenticated, "consumer metadata required")
		}

		consumer := consumers[0]

		curMethod := info.FullMethod
		consumerAllowance, ok := acl[consumer]
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "consumer not in acl")
		}

		if !hasAccess(consumerAllowance, curMethod) {
			return nil, status.Error(codes.Unauthenticated, "consumer has no access to method")
		}

		return handler(ctx, req)
	}
}

func makeAuthStreamInterceptor(acl map[string][]string,
) func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler,
) error {

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		md, ok := metadata.FromIncomingContext(ss.Context())
		if !ok {
			return status.Error(codes.Unauthenticated, "metadata required")
		}

		consumers := md.Get("consumer")
		if len(consumers) == 0 {
			return status.Error(codes.Unauthenticated, "consumer metadata required")
		}

		consumer := consumers[0]

		curMethod := info.FullMethod
		consumerAllowance, ok := acl[consumer]

		if !ok {
			return status.Error(codes.Unauthenticated, "has no such consumer")
		}

		if !hasAccess(consumerAllowance, curMethod) {
			return status.Error(codes.Unauthenticated, "consumer has no access to method")
		}

		return handler(srv, ss)
	}

}

func hasAccess(consumerAllowance []string, curMethod string) bool {
	for _, allow := range consumerAllowance {
		if curMethod == allow {
			return true
		}

		if allow == "/*" {
			return true
		}

		if strings.HasSuffix(allow, "/*") && strings.HasPrefix(curMethod, allow[:len(allow)-2]) {
			return true
		}

	}
	return false
}

func getClientAddr(ctx context.Context) string {
	if p, ok := peer.FromContext(ctx); ok {
		return p.Addr.String()
	}
	return "unknown"
}
