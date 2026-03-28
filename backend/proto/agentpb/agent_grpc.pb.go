// Code generated to match agent.proto service definition. DO NOT EDIT.
// source: agent.proto
//
// This file provides server and client interfaces that mirror the gRPC service
// defined in agent.proto, implemented over a lightweight JSON-over-TCP
// framing protocol so we avoid a hard dependency on google.golang.org/grpc.
//
// Wire format per frame:
//   [4-byte big-endian length] [JSON payload]
//
// The protocol uses a single persistent TCP connection with multiplexed RPCs
// identified by method name in the envelope.

package agentpb

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// ---------------------------------------------------------------------------
// Frame codec — length-prefixed JSON over TCP
// ---------------------------------------------------------------------------

const maxFrameSize = 4 * 1024 * 1024 // 4 MB

// Envelope wraps every message on the wire.
type Envelope struct {
	Method string          `json:"method"`           // RPC method name
	ID     uint64          `json:"id,omitempty"`     // request/response correlation ID
	Error  string          `json:"error,omitempty"`  // non-empty on error response
	Body   json.RawMessage `json:"body,omitempty"`   // payload
	Stream bool            `json:"stream,omitempty"` // true when more messages follow
}

// writeFrame writes a length-prefixed JSON frame to the writer.
func writeFrame(w io.Writer, env *Envelope) error {
	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}
	if len(data) > maxFrameSize {
		return fmt.Errorf("frame size %d exceeds maximum %d", len(data), maxFrameSize)
	}
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, uint32(len(data)))
	if _, err := w.Write(header); err != nil {
		return fmt.Errorf("write header: %w", err)
	}
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("write body: %w", err)
	}
	return nil
}

// readFrame reads a length-prefixed JSON frame from the reader.
func readFrame(r io.Reader) (*Envelope, error) {
	header := make([]byte, 4)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, err
	}
	size := binary.BigEndian.Uint32(header)
	if size > uint32(maxFrameSize) {
		return nil, fmt.Errorf("frame size %d exceeds maximum %d", size, maxFrameSize)
	}
	data := make([]byte, size)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	env := &Envelope{}
	if err := json.Unmarshal(data, env); err != nil {
		return nil, fmt.Errorf("unmarshal envelope: %w", err)
	}
	return env, nil
}

// ---------------------------------------------------------------------------
// Server interfaces and implementation
// ---------------------------------------------------------------------------

// AgentServiceServer is the server-side interface matching the proto service.
type AgentServiceServer interface {
	// Register handles agent registration.
	Register(ctx context.Context, req *RegisterRequest) (*RegisterResponse, error)

	// Heartbeat handles the bidirectional heartbeat stream.
	// The implementation receives heartbeats via the stream and sends commands back.
	Heartbeat(stream HeartbeatStream) error

	// ExecuteJob sends a job to execute and streams back events.
	ExecuteJob(ctx context.Context, req *ExecuteJobRequest, stream JobEventSender) error

	// ReportStatus handles the final status report for a job.
	ReportStatus(ctx context.Context, req *ReportStatusRequest) (*ReportStatusResponse, error)
}

// HeartbeatStream provides bidirectional communication for heartbeat.
type HeartbeatStream interface {
	// Recv receives the next heartbeat from the agent. Blocks until available.
	Recv() (*HeartbeatRequest, error)

	// Send sends a command/response back to the agent.
	Send(resp *HeartbeatResponse) error

	// Context returns the stream context (cancelled when connection drops).
	Context() context.Context
}

// JobEventSender sends job events back to the server from the agent.
type JobEventSender interface {
	// Send sends a job event.
	Send(event *JobEvent) error
}

// GRPCServer listens for agent connections over TCP and dispatches RPCs
// to the registered AgentServiceServer implementation.
type GRPCServer struct {
	listener net.Listener
	handler  AgentServiceServer
	wg       sync.WaitGroup
	quit     chan struct{}
}

// NewGRPCServer creates a new server that will listen on the given address.
func NewGRPCServer(handler AgentServiceServer) *GRPCServer {
	return &GRPCServer{
		handler: handler,
		quit:    make(chan struct{}),
	}
}

// Serve starts accepting connections on the given listener. Blocks until
// the listener is closed or Stop() is called.
func (s *GRPCServer) Serve(lis net.Listener) error {
	s.listener = lis
	for {
		conn, err := lis.Accept()
		if err != nil {
			select {
			case <-s.quit:
				return nil // graceful shutdown
			default:
				return fmt.Errorf("accept: %w", err)
			}
		}
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.handleConnection(conn)
		}()
	}
}

// Stop gracefully shuts down the server.
func (s *GRPCServer) Stop() {
	close(s.quit)
	if s.listener != nil {
		s.listener.Close()
	}
	s.wg.Wait()
}

// handleConnection processes RPCs on a single connection.
func (s *GRPCServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// writeMu serialises writes to the connection.
	var writeMu sync.Mutex

	sendFrame := func(env *Envelope) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return writeFrame(conn, env)
	}

	// Track active heartbeat streams by request ID so the main loop can keep
	// routing heartbeat frames without surrendering connection ownership.
	heartbeatMu := &sync.RWMutex{}
	heartbeatStreams := make(map[uint64]chan *HeartbeatRequest)

	closeHeartbeatStreams := func() {
		heartbeatMu.Lock()
		defer heartbeatMu.Unlock()
		for id, ch := range heartbeatStreams {
			close(ch)
			delete(heartbeatStreams, id)
		}
	}

	for {
		select {
		case <-s.quit:
			closeHeartbeatStreams()
			return
		default:
		}

		env, err := readFrame(conn)
		if err != nil {
			closeHeartbeatStreams()
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				return
			}
			return
		}

		switch env.Method {
		case "Register":
			s.wg.Add(1)
			go func(env *Envelope) {
				defer s.wg.Done()
				s.handleRegister(ctx, env, sendFrame)
			}(env)

		case "Heartbeat":
			heartbeatMu.RLock()
			ch, exists := heartbeatStreams[env.ID]
			heartbeatMu.RUnlock()
			if exists {
				var req HeartbeatRequest
				if err := json.Unmarshal(env.Body, &req); err != nil {
					continue
				}
				select {
				case ch <- &req:
				case <-ctx.Done():
				}
				continue
			}

			incoming := make(chan *HeartbeatRequest, 16)
			var firstReq HeartbeatRequest
			if err := json.Unmarshal(env.Body, &firstReq); err != nil {
				sendFrame(&Envelope{Method: "Heartbeat", ID: env.ID, Error: err.Error()})
				continue
			}

			heartbeatMu.Lock()
			heartbeatStreams[env.ID] = incoming
			heartbeatMu.Unlock()

			incoming <- &firstReq

			s.wg.Add(1)
			go func(streamID uint64, in chan *HeartbeatRequest) {
				defer s.wg.Done()
				defer func() {
					heartbeatMu.Lock()
					if stream, ok := heartbeatStreams[streamID]; ok {
						close(stream)
						delete(heartbeatStreams, streamID)
					}
					heartbeatMu.Unlock()
				}()
				s.handleHeartbeat(ctx, streamID, in, sendFrame)
			}(env.ID, incoming)

		case "ExecuteJob":
			s.wg.Add(1)
			go func(env *Envelope) {
				defer s.wg.Done()
				s.handleExecuteJob(ctx, env, sendFrame)
			}(env)

		case "ReportStatus":
			s.wg.Add(1)
			go func(env *Envelope) {
				defer s.wg.Done()
				s.handleReportStatus(ctx, env, sendFrame)
			}(env)

		default:
			sendFrame(&Envelope{
				Method: env.Method,
				ID:     env.ID,
				Error:  fmt.Sprintf("unknown method: %s", env.Method),
			})
		}
	}
}

func (s *GRPCServer) handleRegister(ctx context.Context, env *Envelope, send func(*Envelope) error) {
	var req RegisterRequest
	if err := json.Unmarshal(env.Body, &req); err != nil {
		send(&Envelope{Method: "Register", ID: env.ID, Error: err.Error()})
		return
	}
	resp, err := s.handler.Register(ctx, &req)
	if err != nil {
		send(&Envelope{Method: "Register", ID: env.ID, Error: err.Error()})
		return
	}
	body, _ := json.Marshal(resp)
	send(&Envelope{Method: "Register", ID: env.ID, Body: body})
}

func (s *GRPCServer) handleHeartbeat(ctx context.Context, streamID uint64, incoming chan *HeartbeatRequest, send func(*Envelope) error) {
	streamCtx, streamCancel := context.WithCancel(ctx)
	defer streamCancel()

	stream := &heartbeatStreamImpl{
		ctx:      streamCtx,
		incoming: incoming,
		send:     send,
		id:       streamID,
	}

	err := s.handler.Heartbeat(stream)
	if err != nil {
		send(&Envelope{Method: "Heartbeat", ID: streamID, Error: err.Error()})
	}
}

func (s *GRPCServer) handleExecuteJob(ctx context.Context, env *Envelope, send func(*Envelope) error) {
	var req ExecuteJobRequest
	if err := json.Unmarshal(env.Body, &req); err != nil {
		send(&Envelope{Method: "ExecuteJob", ID: env.ID, Error: err.Error()})
		return
	}

	sender := &jobEventSenderImpl{
		method: "ExecuteJob",
		id:     env.ID,
		send:   send,
	}

	err := s.handler.ExecuteJob(ctx, &req, sender)
	if err != nil {
		send(&Envelope{Method: "ExecuteJob", ID: env.ID, Error: err.Error()})
		return
	}
	// Send final "stream done" marker.
	send(&Envelope{Method: "ExecuteJob", ID: env.ID, Stream: false})
}

func (s *GRPCServer) handleReportStatus(ctx context.Context, env *Envelope, send func(*Envelope) error) {
	var req ReportStatusRequest
	if err := json.Unmarshal(env.Body, &req); err != nil {
		send(&Envelope{Method: "ReportStatus", ID: env.ID, Error: err.Error()})
		return
	}
	resp, err := s.handler.ReportStatus(ctx, &req)
	if err != nil {
		send(&Envelope{Method: "ReportStatus", ID: env.ID, Error: err.Error()})
		return
	}
	body, _ := json.Marshal(resp)
	send(&Envelope{Method: "ReportStatus", ID: env.ID, Body: body})
}

// heartbeatStreamImpl implements HeartbeatStream.
type heartbeatStreamImpl struct {
	ctx      context.Context
	incoming chan *HeartbeatRequest
	send     func(*Envelope) error
	id       uint64
}

func (h *heartbeatStreamImpl) Recv() (*HeartbeatRequest, error) {
	select {
	case req, ok := <-h.incoming:
		if !ok {
			return nil, io.EOF
		}
		return req, nil
	case <-h.ctx.Done():
		return nil, h.ctx.Err()
	}
}

func (h *heartbeatStreamImpl) Send(resp *HeartbeatResponse) error {
	body, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	return h.send(&Envelope{Method: "Heartbeat", ID: h.id, Body: body, Stream: true})
}

func (h *heartbeatStreamImpl) Context() context.Context {
	return h.ctx
}

// jobEventSenderImpl implements JobEventSender.
type jobEventSenderImpl struct {
	method string
	id     uint64
	send   func(*Envelope) error
}

func (j *jobEventSenderImpl) Send(event *JobEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return j.send(&Envelope{Method: j.method, ID: j.id, Body: body, Stream: true})
}

// ---------------------------------------------------------------------------
// Client implementation
// ---------------------------------------------------------------------------

// ClientConn manages the connection from agent to server.
type ClientConn struct {
	addr    string
	conn    net.Conn
	mu      sync.Mutex
	nextID  atomic.Uint64
	pending sync.Map // id -> chan *Envelope

	// reconnect settings
	maxRetries     int
	baseDelay      time.Duration
	maxDelay       time.Duration
	onReconnect    func() // optional callback after successful reconnect
	connectTimeout time.Duration
}

// DialOption configures a ClientConn.
type DialOption func(*ClientConn)

// WithMaxRetries sets the maximum reconnection attempts (0 = unlimited).
func WithMaxRetries(n int) DialOption {
	return func(c *ClientConn) { c.maxRetries = n }
}

// WithBaseDelay sets the initial backoff delay for reconnection.
func WithBaseDelay(d time.Duration) DialOption {
	return func(c *ClientConn) { c.baseDelay = d }
}

// WithMaxDelay sets the maximum backoff delay for reconnection.
func WithMaxDelay(d time.Duration) DialOption {
	return func(c *ClientConn) { c.maxDelay = d }
}

// WithOnReconnect sets a callback invoked after a successful reconnection.
func WithOnReconnect(fn func()) DialOption {
	return func(c *ClientConn) { c.onReconnect = fn }
}

// WithConnectTimeout sets the timeout for each connection attempt.
func WithConnectTimeout(d time.Duration) DialOption {
	return func(c *ClientConn) { c.connectTimeout = d }
}

// Dial creates a new client connection to the gRPC server.
func Dial(addr string, opts ...DialOption) (*ClientConn, error) {
	cc := &ClientConn{
		addr:           addr,
		maxRetries:     0, // unlimited by default
		baseDelay:      1 * time.Second,
		maxDelay:       30 * time.Second,
		connectTimeout: 10 * time.Second,
	}
	for _, opt := range opts {
		opt(cc)
	}
	conn, err := net.DialTimeout("tcp", addr, cc.connectTimeout)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}
	cc.conn = conn
	go cc.readLoop()
	return cc, nil
}

// Close closes the underlying connection.
func (cc *ClientConn) Close() error {
	if cc.conn != nil {
		return cc.conn.Close()
	}
	return nil
}

// reconnect attempts to re-establish the connection with exponential backoff.
func (cc *ClientConn) reconnect() error {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if cc.conn != nil {
		cc.conn.Close()
		cc.conn = nil
	}

	delay := cc.baseDelay
	attempts := 0

	for {
		attempts++
		if cc.maxRetries > 0 && attempts > cc.maxRetries {
			return fmt.Errorf("reconnection failed after %d attempts", attempts-1)
		}

		conn, err := net.DialTimeout("tcp", cc.addr, cc.connectTimeout)
		if err == nil {
			cc.conn = conn
			go cc.readLoop()
			if cc.onReconnect != nil {
				cc.onReconnect()
			}
			return nil
		}

		time.Sleep(delay)
		delay *= 2
		if delay > cc.maxDelay {
			delay = cc.maxDelay
		}
	}
}

// readLoop reads response frames and dispatches them to pending callers.
func (cc *ClientConn) readLoop() {
	conn := cc.conn
	for {
		env, err := readFrame(conn)
		if err != nil {
			// Connection lost — notify all pending callers.
			cc.pending.Range(func(key, value any) bool {
				ch := value.(chan *Envelope)
				select {
				case ch <- &Envelope{Error: "connection lost"}:
				default:
				}
				return true
			})
			return
		}

		if val, ok := cc.pending.Load(env.ID); ok {
			ch := val.(chan *Envelope)
			ch <- env
			// If the stream is done, remove from pending.
			if !env.Stream && env.Error == "" {
				// Keep the channel open for the caller to read the final frame.
			}
		}
	}
}

// call sends a unary RPC and waits for the response.
func (cc *ClientConn) call(method string, req any) (*Envelope, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	id := cc.nextID.Add(1)
	respCh := make(chan *Envelope, 1)
	cc.pending.Store(id, respCh)
	defer cc.pending.Delete(id)

	env := &Envelope{
		Method: method,
		ID:     id,
		Body:   body,
	}

	cc.mu.Lock()
	err = writeFrame(cc.conn, env)
	cc.mu.Unlock()
	if err != nil {
		return nil, err
	}

	select {
	case resp := <-respCh:
		if resp.Error != "" {
			return nil, errors.New(resp.Error)
		}
		return resp, nil
	case <-time.After(30 * time.Second):
		return nil, errors.New("rpc timeout")
	}
}

// AgentServiceClient provides the client-side API matching the proto service.
type AgentServiceClient struct {
	cc *ClientConn
}

// NewAgentServiceClient creates a new client from a connection.
func NewAgentServiceClient(cc *ClientConn) *AgentServiceClient {
	return &AgentServiceClient{cc: cc}
}

// Register sends a registration request to the server.
func (c *AgentServiceClient) Register(ctx context.Context, req *RegisterRequest) (*RegisterResponse, error) {
	env, err := c.cc.call("Register", req)
	if err != nil {
		return nil, err
	}
	var resp RegisterResponse
	if err := json.Unmarshal(env.Body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// HeartbeatClientStream handles the client side of the bidirectional heartbeat.
type HeartbeatClientStream struct {
	cc     *ClientConn
	respCh chan *Envelope
	id     uint64
}

// Send sends a heartbeat to the server.
func (s *HeartbeatClientStream) Send(req *HeartbeatRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	env := &Envelope{
		Method: "Heartbeat",
		ID:     s.id,
		Body:   body,
		Stream: true,
	}
	s.cc.mu.Lock()
	defer s.cc.mu.Unlock()
	return writeFrame(s.cc.conn, env)
}

// Recv receives a heartbeat response from the server.
func (s *HeartbeatClientStream) Recv() (*HeartbeatResponse, error) {
	select {
	case env, ok := <-s.respCh:
		if !ok {
			return nil, io.EOF
		}
		if env.Error != "" {
			return nil, errors.New(env.Error)
		}
		var resp HeartbeatResponse
		if err := json.Unmarshal(env.Body, &resp); err != nil {
			return nil, err
		}
		return &resp, nil
	case <-time.After(60 * time.Second):
		return nil, errors.New("heartbeat recv timeout")
	}
}

// Close ends the heartbeat stream.
func (s *HeartbeatClientStream) Close() {
	s.cc.pending.Delete(s.id)
}

// Heartbeat opens a bidirectional heartbeat stream.
func (c *AgentServiceClient) Heartbeat(ctx context.Context) (*HeartbeatClientStream, error) {
	id := c.cc.nextID.Add(1)
	respCh := make(chan *Envelope, 16)
	c.cc.pending.Store(id, respCh)

	return &HeartbeatClientStream{
		cc:     c.cc,
		respCh: respCh,
		id:     id,
	}, nil
}

// ExecuteJobStream receives streamed job events from the server.
type ExecuteJobStream struct {
	respCh chan *Envelope
	cc     *ClientConn
	id     uint64
}

// Recv receives the next job event from the stream.
func (s *ExecuteJobStream) Recv() (*JobEvent, error) {
	env, ok := <-s.respCh
	if !ok {
		return nil, io.EOF
	}
	if env.Error != "" {
		return nil, errors.New(env.Error)
	}
	if !env.Stream && len(env.Body) == 0 {
		return nil, io.EOF // stream complete
	}
	var event JobEvent
	if err := json.Unmarshal(env.Body, &event); err != nil {
		return nil, err
	}
	return &event, nil
}

// Close ends the job event stream.
func (s *ExecuteJobStream) Close() {
	s.cc.pending.Delete(s.id)
}

// ExecuteJob sends a job execution request and returns a stream of events.
// NOTE: On the server side this is used to push jobs to agents. On the client
// (agent) side, the agent calls this after receiving a job push.
func (c *AgentServiceClient) ExecuteJob(ctx context.Context, req *ExecuteJobRequest) (*ExecuteJobStream, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	id := c.cc.nextID.Add(1)
	respCh := make(chan *Envelope, 256)
	c.cc.pending.Store(id, respCh)

	env := &Envelope{
		Method: "ExecuteJob",
		ID:     id,
		Body:   body,
	}
	c.cc.mu.Lock()
	err = writeFrame(c.cc.conn, env)
	c.cc.mu.Unlock()
	if err != nil {
		c.cc.pending.Delete(id)
		return nil, err
	}

	return &ExecuteJobStream{
		respCh: respCh,
		cc:     c.cc,
		id:     id,
	}, nil
}

// ReportStatus sends a final status report for a completed job.
func (c *AgentServiceClient) ReportStatus(ctx context.Context, req *ReportStatusRequest) (*ReportStatusResponse, error) {
	env, err := c.cc.call("ReportStatus", req)
	if err != nil {
		return nil, err
	}
	var resp ReportStatusResponse
	if err := json.Unmarshal(env.Body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Conn returns the underlying connection for direct access if needed.
func (cc *ClientConn) Conn() net.Conn {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	return cc.conn
}

// Reconnect exposes the reconnection method for the agent to call explicitly.
func (cc *ClientConn) Reconnect() error {
	return cc.reconnect()
}
