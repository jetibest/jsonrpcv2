// package main
package jsonrpcv2

import (
	"encoding/json"
	"time"
	"sync"
	"math"
//	"log"
//	"fmt"
)

// Types:

const ParseErrorCode int = -32700
const InvalidRequestErrorCode int = -32600
const MethodNotFoundErrorCode int = -32601
const InternalErrorCode int = -32603
const InvalidParamsErrorCode int = -32602
const ServerErrorCode int = -32000
const RequestTimeoutErrorCode int = -32001


type IDGenerator = func() any
type NotificationHandler = func(*Notification)
type RequestHandler = func(any) (any, *Error)

type MessageSender = func(*Message) *Error
// type BatchMessageSender = func([]*Message) *Error


// Object structs

type Error struct {
	
	Code int              `json:"code"`
	Message string        `json:"message"`
	Data any              `json:"data,omitempty"`
}
type Response struct {
	
	JSONRPC string        `json:"jsonrpc"`
	ID any                `json:"id"`
	Result any            `json:"result,omitempty"`
	Error *Error          `json:"error,omitempty"`
}
type Notification struct {
	
	JSONRPC string        `json:"jsonrpc"`
	Method any            `json:"method"`
	Params any            `json:"params,omitempty"`
}
type Request struct {
	
	JSONRPC string        `json:"jsonrpc"`
	ID any                `json:"id"`
	Method any            `json:"method"`
	Params any            `json:"params,omitempty"`
}
type Message struct {
	
	JSONRPC string        `json:"jsonrpc"`
	ID any                `json:"id,omitempty"`
	Result any            `json:"result,omitempty"`
	Method any            `json:"method,omitempty"`
	Params any            `json:"params,omitempty"`
	Error *Error          `json:"error,omitempty"`
}


// Constructors:

func NewInvalidParamsError() *Error {
	
	return &Error{Code: InvalidParamsErrorCode, Message: "Invalid params"}
}
func NewServerError(message string) *Error {
	
	return &Error{Code: ServerErrorCode, Message: message}
}

func NewNotificationFromMessage(message *Message) *Notification {
	
	if message == nil || (message.Method == nil) {
		
		// Message is not a valid Notification
		return nil
	} else {
		
		switch message.Method.(type) {
			
			case string:
			default:
				// Method must be string type, or else Message is not a valid Notification
				return nil
		}
	}
	
	if message.ID != nil {
		
		// a Notification cannot have an ID
		return nil
	}
	
	return &Notification{
		JSONRPC: message.JSONRPC,
		Method: message.Method,
		Params: message.Params,
	}
}

func NewMessageFromRequest(request *Request) *Message {
	
	if request == nil {
		
		return nil
	}
	
	return &Message{
		JSONRPC: request.JSONRPC,
		ID: request.ID,
		Method: request.Method,
		Params: request.Params,
	}
}

func NewMessageFromResponse(response *Response) *Message {
	
	if response == nil {
		
		return nil
	}
	
	return &Message{
		JSONRPC: response.JSONRPC,
		ID: response.ID,
		Result: response.Result,
		Error: response.Error,
	}
}

func NewRequestFromMessage(message *Message) *Request {
	
	if message == nil || (message.Method == nil) {
		
		// Message is not a valid Request
		return nil
	} else {
		
		switch message.Method.(type) {
			
			case string:
			default:
				// Method must be string type, or else Message is not a valid Request
				return nil
		}
	}
	
	if message.ID == nil {
		
		// a Request must have an ID
		return nil
	}
	
	return &Request{
		JSONRPC: message.JSONRPC,
		ID: message.ID,
		Method: message.Method,
		Params: message.Params,
	}
}

func NewResponseFromMessage(message *Message) *Response {
	
	if message == nil || (message.Result == nil && message.Error == nil) {
		
		// Message is not a Response
		return nil
	}
	
	return &Response{
		JSONRPC: message.JSONRPC,
		ID: message.ID,
		Result: message.Result,
		Error: message.Error,
	}
}

func NewPeer() Peer {
	
	p := &peer{}
	
	// implement default ID generator:
	var requestIndex float64 = 0
	p.GenerateID = func() any {
		requestIndex++
		return requestIndex
	}
	
	p.ChannelsByNumber = make(map[float64]chan *Response)
	p.ChannelsByString = make(map[string]chan *Response)
	
	p.Methods = make(map[string]RequestHandler)
	
	return p
}

func NewClient() Client {
	
	c := &peer{}
	
	// implement default ID generator:
	var requestIndex float64 = 0
	c.GenerateID = func() any {
		requestIndex++
		return requestIndex
	}
	
	c.ChannelsByNumber = make(map[float64]chan *Response)
	c.ChannelsByString = make(map[string]chan *Response)
	
	return c
}

func NewServer() Server {
	
	s := &peer{}
	
	s.Methods = make(map[string]RequestHandler)
	
	return s
}


// Class implementations:

type Server interface {
	
	processRequest(*Request) *Response
	
	WithNotificationHandler(NotificationHandler) Peer
	WithMessageSender(MessageSender) Peer
//	WithBatchMessageSender(BatchMessageSender) Peer
	
	HasMethod(string) bool
	AddMethod(string, RequestHandler)
	RemoveMethod(string)
	
	ReceiveString(string)
	ReceiveBytes(*[]byte)
	ReceiveMessage(*Message)
//	ReceiveBatchMessage([]*Message)
	ReceiveRequest(*Request)
	ReceiveNotification(*Notification)
}
type Client interface {
	
	emitResponse(*Response)
	
	WithNotificationHandler(NotificationHandler) Peer
	WithIDGenerator(IDGenerator) Peer
	WithMessageSender(MessageSender) Peer
//	WithBatchMessageSender(BatchMessageSender) Peer
	
	Request(string, any, time.Duration) (*Response, *Error) // with timeout
	Notify(string, any) *Error // won't expect any response
	
	ReceiveString(string)
	ReceiveBytes(*[]byte)
	ReceiveMessage(*Message)
//	ReceiveBatchMessage([]*Message)
	ReceiveRequest(*Request)
	ReceiveNotification(*Notification)
}
// Peer implements both Server and Client interfaces
type Peer interface {
	
	processRequest(*Request) *Response // Server-specific
	emitResponse(*Response) // Client-specific
	
	WithNotificationHandler(NotificationHandler) Peer
	WithIDGenerator(IDGenerator) Peer // Client-specific
	WithMessageSender(MessageSender) Peer
//	WithBatchMessageSender(BatchMessageSender) Peer
	
	// Server:
	HasMethod(string) bool
	AddMethod(string, RequestHandler)
	RemoveMethod(string)
	
	// Client:
	Request(string, any, time.Duration) (*Response, *Error) // with timeout
	Notify(string, any) *Error // won't expect any response
	
	ReceiveString(string)
	ReceiveBytes(*[]byte)
	ReceiveMessage(*Message)
//	ReceiveBatchMessage([]*Message)
	ReceiveRequest(*Request)
	ReceiveNotification(*Notification)
	ReceiveResponse(*Response)
}
type peer struct {
	
	mu sync.Mutex
	
	HandleNotification NotificationHandler
	
	SendMessage MessageSender
//	SendBatchMessage BatchMessageSender
	
	GenerateID IDGenerator
	ChannelsByNumber map[float64]chan *Response
	ChannelsByString map[string]chan *Response
	
	Methods map[string]RequestHandler
}
func (p *peer) processRequest(request *Request) *Response {
	
	fn, ok := p.Methods[request.Method.(string)]
	
	if ok {
		result, err := fn(request.Params)
		
		if err != nil {
			
			return &Response{JSONRPC: "2.0", ID: request.ID, Error: err}
			
		} else {
			
			return &Response{JSONRPC: "2.0", ID: request.ID, Result: result}
		}
	} else {
		
		return &Response{JSONRPC: "2.0", ID: request.ID, Error: &Error{Code: MethodNotFoundErrorCode, Message: "Method not found"}}
	}
}
/*
func (p *peer) emitBatchResponse(responses []*Response) {
	
	if responses == nil || len(responses) == 0 {
		return // ignore invalid or empty responses
	}
	
	// we don't return []*Response for RequestMultiple([]*Request) []*Response
	// but instead we use:
	
	ch := make(chan *Response)
	b := peer.BatchRequestBuilder()
	b.Append("method1", {params}, func (res *Response) {
		// handle response here
	})
	b.Append("method2", {params}, ch)
	b.Append("method2", {params}, ch)
	b.Send(time.Duration(10) * time.Second) // send adds responses to channel, until close(ch) at the end
	
	for res := range ch {
		
	}
}
*/
func (p *peer) emitResponse(response *Response) {
	
	if response == nil || response.ID == nil {
		return // ignore this invalid response
	}
	
	var responseNumberID float64
	var responseStringID string
	
	isNumericID := false
	
	switch v := response.ID.(type) {
		case float64:
			
			responseNumberID = v
			isNumericID = true
			
			if math.IsNaN(responseNumberID) {
				
				// NaN is not allowed
				return
			}
			
		case string:
			
			responseStringID = v
			
			if responseStringID == "" {
				// empty string ID is not allowed
				return
			}
		
		default:
			
			// invalid ID type (must be string or float64)
			// response ID is maybe absent or null
			return
	}
	
	if isNumericID {
		
		ch, ok := p.ChannelsByNumber[responseNumberID]
		
		if ok {
			ch <- response
			
			close(ch)
		
		} // else: ignore unrecognized response ID, does not match a request ID
	} else {
		
		ch, ok := p.ChannelsByString[responseStringID]
		
		if ok {
			ch <- response
			
			close(ch)
			
		} // else: ignore unrecognized response ID, does not match a request ID
	}
}
func (p *peer) WithIDGenerator(fn IDGenerator) Peer {
	
	p.GenerateID = fn
	return p
}
func (p *peer) WithNotificationHandler(fn NotificationHandler) Peer {
	
	p.HandleNotification = fn
	return p
}
func (p *peer) WithMessageSender(fn MessageSender) Peer {
	
	p.SendMessage = fn
	return p
}
/*
func (p *peer) WithBatchMessageSender(fn BatchMessageSender) Peer {
	
	p.SendBatchMessage = fn
	return p
}
*/

func (p *peer) ReceiveString(data string) {
	
	arr := []byte(data)
	p.ReceiveBytes(&arr)
}
func (p *peer) ReceiveBytes(data *[]byte) {
	
	var message Message
	err := json.Unmarshal(*data, &message)
	
	if err == nil {
		
		p.ReceiveMessage(&message)
		return
	}
	
	messageArray := make([]*Message, 0)
	err = json.Unmarshal(*data, &messageArray)
	
	// possibly this is a batch of responses, currently this is not supported, since the client can also not send an array of requests as of now
	// we must first implement client.RequestBatchBuilder().Append(method string, params any).Append(method string, params any).Commit()
	// which is a convenient enclosure for client.RequestMultiple([]*Request) []*Response
	// so we can do for _, response := range client.RequestBatchBuilder()...Commit() {
	/*
	if err == nil {
		
		p.ReceiveBatchMessage(messageArray)
		return
	}
	*/
}
func (p *peer) ReceiveMessage(message *Message) {
		
	notification := NewNotificationFromMessage(message)
	
	if notification != nil {
		
		p.ReceiveNotification(notification)
		return
	}
	
	request := NewRequestFromMessage(message)
	
	if request != nil {
		
		p.ReceiveRequest(request)
		return
	}
	
	response := NewResponseFromMessage(message)
	
	if response != nil {
		
		p.ReceiveResponse(response)
		return
	}
}
/*
func (p *peer) ReceiveBatchMessage(messages []*Message) {
	
	// the messages could be responses for a batch request array
	// or the messages could be requests that we must process
	
	server_responses := make([]*Response, 0)
	client_responses := make([]*Response, 0)
	
	for _, message := range messages {
		
		req := NewRequestFromMessage(message)
		
		if req != nil {
			
			res := p.processRequest(req)
			
			server_responses = append(server_responses, res)
			
			continue
		}
		
		not := NewNotificationFromMessage(message)
		
		if not != nil {
			
			if p.HandleNotification != nil {
				
				p.HandleNotification(not)
			}
			
			server_responses = append(server_responses, nil)
			
			continue
		}
		
		res := NewResponseFromMessage(message)
		
		if res != nil {
			
			client_responses = append(client_responses, res)
		}
	}
	
	if len(server_responses) > 0 && p.SendBatchResponse != nil {
		
		p.SendBatchResponse(server_responses)
	}
	
	if len(client_responses) > 0 {
		
		p.emitBatchResponse(client_responses)
	}
}
*/
func (p *peer) ReceiveRequest(request *Request) {
	
	if request != nil {
		
		response := p.processRequest(request)
		
		if p.SendMessage != nil {
			
			message := NewMessageFromResponse(response)
			
			p.SendMessage(message)
		}
	}
}
func (p *peer) ReceiveResponse(response *Response) {
	
	if response != nil {
		
		p.emitResponse(response)
	}
}
func (p *peer) ReceiveNotification(notification *Notification) {
	
	if p.HandleNotification != nil {
		
		p.HandleNotification(notification)
	}
}

func (p *peer) HasMethod(name string) bool {
	
	_, ok := p.Methods[name]
	return ok
}
func (p *peer) AddMethod(name string, fn RequestHandler) {
	
	p.Methods[name] = fn
}
func (p *peer) RemoveMethod(name string) {
	
	delete(p.Methods, name)
}
func (p *peer) Notify(method string, params any) *Error {
	
	if p.SendMessage == nil {
		
		return &Error{Code: InternalErrorCode, Message: "Internal error: No MessageSender attached."}
	}
	
	request := &Request{JSONRPC: "2.0", Method: method, Params: params}
	
	message := NewMessageFromRequest(request)
	
	return p.SendMessage(message)
}
func (p *peer) Request(method string, params any, timeout time.Duration) (*Response, *Error) {
	
	if p.SendMessage == nil {
		
		return nil, &Error{Code: InternalErrorCode, Message: "Internal error: No MessageSender attached."}
	}
	
	ch := make(chan *Response, 1) // buffered
	
	requestID := p.GenerateID()
	
	if requestID == nil {
		return nil, &Error{Code: InternalErrorCode, Message: "Internal error"}
	}
	
	isNumericID := false
	var requestNumberID float64
	var requestStringID string
	
	switch v := requestID.(type) {
		case float64:
			
			requestNumberID = v
			isNumericID = true
			
			if math.IsNaN(requestNumberID) {
				// NaN is not allowed
				return nil, &Error{Code: InternalErrorCode, Message: "Internal error"}
			}
			
		case string:
			
			requestStringID = v
			
			if requestStringID == "" {
				// empty string ID is not allowed
				return nil, &Error{Code: InternalErrorCode, Message: "Internal error"}
			}
		
		default:
			
			// invalid ID type (must be string or float64)
			return nil, &Error{Code: InternalErrorCode, Message: "Internal error"}
	}
	
	p.mu.Lock()
	if isNumericID {
		
		_, ok := p.ChannelsByNumber[requestNumberID]
		
		if ok {
			// id already exists, that's very bad
			return nil, &Error{Code: InternalErrorCode, Message: "Internal error"}
		} else {
			
			p.ChannelsByNumber[requestNumberID] = ch
		}
		
	} else {
		
		_, ok := p.ChannelsByString[requestStringID]
		
		if ok {
			// id already exists, that's very bad
			return nil, &Error{Code: InternalErrorCode, Message: "Internal error"}
		} else {
			
			p.ChannelsByString[requestStringID] = ch
		}
	}
	p.mu.Unlock()
	
	// build request, and pass request to the request sender
	request := &Request{JSONRPC: "2.0", ID: requestID, Method: method, Params: params}
	message := NewMessageFromRequest(request)
	err := p.SendMessage(message)
	
	// if immediate error sending the request, passthrough the error
	if err != nil {
		
		p.mu.Lock()
		if isNumericID {
			delete(p.ChannelsByNumber, requestNumberID)
		} else {
			delete(p.ChannelsByString, requestStringID)
		}
		p.mu.Unlock()
		
		return nil, err
	}
	
	// return the result, or timeout err if timeout
	select {
		case res := <- ch:
			
			p.mu.Lock()
			if isNumericID {
				delete(p.ChannelsByNumber, requestNumberID)
			} else {
				delete(p.ChannelsByString, requestStringID)
			}
			p.mu.Unlock()
			
			if res.Error != nil {
				return nil, res.Error
			} else {
				return res, nil
			}
			
		case <- time.After(timeout):
			
			p.mu.Lock()
			if isNumericID {
				delete(p.ChannelsByNumber, requestNumberID)
			} else {
				delete(p.ChannelsByString, requestStringID)
			}
			p.mu.Unlock()
			
			// if a request still comes in with a certain id, then it will simply be ignored, memory is already cleaned up here anyway
			
			return nil, &Error{Code: RequestTimeoutErrorCode, Message: "Request timeout"}
	}
}



// Main test example:
/*
func main() {
	
	log.Println("Main(): begin")
	
	s := NewServer()
	c := NewClient()
	
	s.WithMessageSender(func (message *Message) *Error {
		
		encodedMessage, _ := json.Marshal(*message)
		fmt.Printf("[s] Message being sent: %+v\n", string(encodedMessage))
		
		c.ReceiveBytes(&encodedMessage)
		
		return nil
	})
	
	c.WithMessageSender(func (message *Message) *Error {
		
		encodedMessage, _ := json.Marshal(*message)
		fmt.Printf("[c] Message being sent: %+v\n", string(encodedMessage))
		
		s.ReceiveBytes(&encodedMessage)
		
		return nil
	})
	
	s.AddMethod("echo", func (requestParams any) (any, *Error) {
		
		params := requestParams.(map[string]any)
		
		return params["text"], nil
	})
	
	res, err := c.Request("echo", map[string]any{"text": "hello there"}, time.Duration(10) * time.Second)
	
	if err != nil {
		
		fmt.Printf("Client request error: %+v\n", err)
	} else {
		
		fmt.Printf("Client request result: %+v\n", res)
	}
	
	log.Println("Main(): end")
}
*/
