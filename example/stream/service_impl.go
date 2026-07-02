package main

import (
	"fmt"
	"io"
	"log/slog"
	"time"
)

// ---------------------------------------------------------------------------
// streamServiceImpl implements StreamServiceServer.
//
// This is what a user would write after protoc-gen-goose generates the
// interface and handler layer. Each method contains the business logic for
// the corresponding streaming RPC.
// ---------------------------------------------------------------------------

// streamServiceImpl is the user-defined service implementation.
type streamServiceImpl struct {
	logger *slog.Logger
}

// NewStreamServiceImpl creates a new service implementation.
func NewStreamServiceImpl(logger *slog.Logger) StreamServiceServer {
	return &streamServiceImpl{logger: logger}
}

// ---------------------------------------------------------------------------
// ClientStream: client sends many requests, server responds with one
// aggregated response (e.g., batch upload, log ingestion).
// ---------------------------------------------------------------------------

func (s *streamServiceImpl) ClientStream(stream ServerClientStream[ListExpiredCreditBucketsRequest, ListExpiredCreditBucketsResponse]) error {
	var total int64
	var buckets []*CreditBucket

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			// Client has finished sending. Send the aggregated response.
			s.logger.Info("client-stream complete", slog.Int64("total", total))
			return stream.SendAndClose(&ListExpiredCreditBucketsResponse{
				Buckets: buckets,
				Total:   total,
			})
		}
		if err != nil {
			return err
		}

		s.logger.Info("client-stream received",
			slog.String("filter", req.Filter),
		)

		// Simulate processing: each request produces a credit bucket entry.
		buckets = append(buckets, &CreditBucket{
			BucketID: fmt.Sprintf("bucket-%d", total+1),
			Amount:   100,
			Expired:  true,
		})
		total++
	}
}

// ---------------------------------------------------------------------------
// ServerStream: client sends one request, server streams back many responses
// (e.g., real-time feed, paginated list push).
// ---------------------------------------------------------------------------

func (s *streamServiceImpl) ServerStream(req *ListExpiredCreditBucketsRequest, stream ServerServerStream[ListExpiredCreditBucketsResponse]) error {
	s.logger.Info("server-stream started", slog.String("filter", req.Filter))

	// Simulate streaming 5 credit bucket entries back to the client.
	for i := int64(1); i <= 5; i++ {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		default:
		}

		resp := &ListExpiredCreditBucketsResponse{
			Buckets: []*CreditBucket{
				{
					BucketID: fmt.Sprintf("bucket-%d", i),
					Amount:   i * 50,
					Expired:  true,
				},
			},
			Total: i,
		}

		if err := stream.Send(resp); err != nil {
			return err
		}

		s.logger.Info("server-stream sent", slog.Int64("seq", i))

		// Simulate some processing delay between pushes.
		time.Sleep(500 * time.Millisecond)
	}

	s.logger.Info("server-stream complete")
	return nil
}

// ---------------------------------------------------------------------------
// BidStream: full-duplex bidirectional communication (e.g., chat, collab).
// ---------------------------------------------------------------------------

func (s *streamServiceImpl) BidStream(stream ServerBidiStream[ListExpiredCreditBucketsRequest, ListExpiredCreditBucketsResponse]) error {
	s.logger.Info("bidi-stream started")

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			s.logger.Info("bidi-stream client closed")
			return nil
		}
		if err != nil {
			return err
		}

		s.logger.Info("bidi-stream received",
			slog.String("filter", req.Filter),
		)

		// Echo back an acknowledgement with a generated bucket.
		resp := &ListExpiredCreditBucketsResponse{
			Buckets: []*CreditBucket{
				{
					BucketID: fmt.Sprintf("ack-%s", req.Filter),
					Amount:   1,
					Expired:  false,
				},
			},
			Total: 1,
		}

		if err := stream.Send(resp); err != nil {
			return err
		}
	}
}
