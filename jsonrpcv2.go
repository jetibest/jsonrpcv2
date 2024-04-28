// package main
package jsonrpcv2

import (
	"encoding/json"
	"time"
	"sync"
	"math"
)

// Types:

const jsonRPCParseErrorCode int = -32700
const jsonRPCInvalidRequestErrorCode int = -32600
const jsonRPCMethodNotFoundErrorCode int = -32601
const jsonRPCInternalErrorCode int = -32603
const JSONRPCInvalidParamsErrorCode int = -32602
const JSONRPCServerErrorCode int = -32000
const jsonRPCRequestTimeoutErrorCode int = -32001

type JSONRPCResponseHandler = func(interface{})
type JSONRPCBatchResponseHandler = func([]*JSONRPCResponse)
type JSONRPCRequestHandler = func(interface{}) (interface{}, *JSONRPCError)
type JSONRPCIDGenerator = func() interface{}
type JSONRPCRequestSender = func(*JSONRPCRequest) *JSONRPCError

type JSONRPCError struct {
	
	Code int              `json:"code"`
	Message string        `json:"message"`
	Data interface{}      `json:"data,omitempty"`
}
type JSONRPCResponse struct {
	
	JSONRPC string        `json:"jsonrpc"`
	ID interface{}        `json:"id"`
	Result interface{}    `json:"result,omitempty"`
	Error *JSONRPCError   `json:"error,omitempty"`
}
type JSONRPCRequest struct {
	
	JSONRPC string        `json:"jsonrpc"`
	ID interface{}        `json:"id"`
	Method interface{}    `json:"method"`
	Params interface{}    `json:"params,omitempty"`
}

type JSONRPCServer interface {
	
	handleRequest(*JSONRPCRequest) *JSONRPCResponse
	
	HasMethod(string) bool
	AddMethod(string, JSONRPCRequestHandler)
	RemoveMethod(string)
	
	ReceiveBytes(*[]byte) // only supports a single JSONRPCRequest
	ReceiveString(string) // only supports a single JSONRPCRequest
	ReceiveSingle(*JSONRPCRequest)
	ReceiveMultiple([]*JSONRPCRequest)
}

type JSONRPCClient interface {
	
	emitResponse(*JSONRPCResponse)
	
	WithIDGenerator(JSONRPCIDGenerator) JSONRPCClient
	
	Request(string, interface{}, time.Duration) (*JSONRPCResponse, *JSONRPCError) // with timeout
	Notify(string, interface{}) *JSONRPCError // won't expect any response
	
	ReceiveBytes(*[]byte) // only supports a single JSONRPCResponse
	ReceiveString(string) // only supports a single JSONRPCResponse
	ReceiveSingle(*JSONRPCResponse)
	ReceiveMultiple([]*JSONRPCResponse)
}


// Functions:

func NewJSONRPCInvalidParamsError() *JSONRPCError {
	
	err := JSONRPCError{JSONRPCInvalidParamsErrorCode, "Invalid params", nil}
	
	return &err
}
func NewJSONRPCServerError(message string) *JSONRPCError {
	
	err := JSONRPCError{JSONRPCServerErrorCode, message, nil}
	
	return &err
}

func NewJSONRPCError(code int, message string) *JSONRPCError {
	
	err := JSONRPCError{code, message, nil}
	
	return &err
}

func NewJSONRPCServer(fn JSONRPCResponseHandler) JSONRPCServer {
	
	s := &server{}
	
	s.HandleResponse = fn
	s.Methods = make(map[string]JSONRPCRequestHandler)
	
	return s
}

func NewJSONRPCClient(fn JSONRPCRequestSender) JSONRPCClient {
	
	c := &client{}
	
	// implement default ID generator:
	var requestIndex float64 = 1
	c.GenerateID = func() interface{} {
		requestIndex++
		return requestIndex
	}
	
	c.SendRequest = fn
	c.ChannelsByNumber = make(map[float64]chan *JSONRPCResponse)
	c.ChannelsByString = make(map[string]chan *JSONRPCResponse)
	
	return c
}


// JSONRPCServer implementation:

type server struct {
	
	HandleResponse JSONRPCResponseHandler
	Methods map[string]JSONRPCRequestHandler
}
func (s *server) HasMethod(name string) bool {
	
	_, ok := s.Methods[name]
	return ok
}
func (s *server) AddMethod(name string, fn JSONRPCRequestHandler) {
	
	s.Methods[name] = fn
}
func (s *server) RemoveMethod(name string) {
	
	delete(s.Methods, name)
}
func (s *server) ReceiveBytes(data *[]byte) {
	
	var request JSONRPCRequest
	err := json.Unmarshal(*data, &request)
	
	if err != nil {
		// strictly speaking, no response should be returned, as a response requires a valid ID
		// s.HandleResponse(&JSONRPCResponse{"2.0", nil, nil, NewJSONRPCError(jsonRPCParseErrorCode, "Parse error")})
	} else {
		s.ReceiveSingle(&request)
	}
}
func (s *server) ReceiveString(data string) {
	
	arr := []byte(data)
	s.ReceiveBytes(&arr)
}
func (s *server) ReceiveSingle(request *JSONRPCRequest) {
	
//	encodedRequest, _ := json.Marshal(*request)
//	fmt.Printf("Incoming JSON-RPC-2.0 request: %+v\n", string(encodedRequest))
	
	response := s.handleRequest(request)
	
	if response.ID != nil {
		
		s.HandleResponse(response)
		
	} // else: Notification request, no response should be given
}
func (s *server) ReceiveMultiple(requests []*JSONRPCRequest) {
	
	responses := make([]*JSONRPCResponse, len(requests))
	
	for i, request := range requests {
		
		responses[i] = s.handleRequest(request)
		
		if responses[i].ID != nil {
			
			s.HandleResponse(responses[i])
		} // else: Notification request, no response should be given
	}
}

func (s *server) handleRequest(request *JSONRPCRequest) *JSONRPCResponse {
	
	fn, ok := s.Methods[request.Method.(string)]
	
	if ok {
		result, err := fn(request.Params)
		
		if err != nil {
			
			return &JSONRPCResponse{"2.0", request.ID, nil, err}
			
		} else {
			
			return &JSONRPCResponse{"2.0", request.ID, result, nil}
		}
	} else {
		
		return &JSONRPCResponse{"2.0", request.ID, nil, NewJSONRPCError(jsonRPCMethodNotFoundErrorCode, "Method not found")}
	}
}

// JSONRPCClient implementation:
type client struct {
	
	mu sync.Mutex
	GenerateID JSONRPCIDGenerator
	SendRequest JSONRPCRequestSender
	ChannelsByNumber map[float64]chan *JSONRPCResponse
	ChannelsByString map[string]chan *JSONRPCResponse
}
func (c *client) WithIDGenerator(fn JSONRPCIDGenerator) JSONRPCClient {
	
	c.GenerateID = fn
	return c
}
func (c *client) Notify(method string, params interface{}) *JSONRPCError {
	
	return c.SendRequest(&JSONRPCRequest{"2.0", nil, method, params})
}
func (c *client) Request(method string, params interface{}, timeout time.Duration) (*JSONRPCResponse, *JSONRPCError) {
	
	ch := make(chan *JSONRPCResponse, 1) // buffered
	
	requestID := c.GenerateID()
	
	if requestID == nil {
		return nil, NewJSONRPCError(jsonRPCInternalErrorCode, "Internal error")
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
				return nil, NewJSONRPCError(jsonRPCInternalErrorCode, "Internal error")
			}
			
		case string:
			
			requestStringID = v
			
			if requestStringID == "" {
				// empty string ID is not allowed
				return nil, NewJSONRPCError(jsonRPCInternalErrorCode, "Internal error")
			}
		
		default:
			
			// invalid ID type (must be string or float64)
			return nil, NewJSONRPCError(jsonRPCInternalErrorCode, "Internal error")
	}
	
	c.mu.Lock()
	if isNumericID {
		
		_, ok := c.ChannelsByNumber[requestNumberID]
		
		if ok {
			// id already exists, that's very bad
			return nil, NewJSONRPCError(jsonRPCInternalErrorCode, "Internal error")
		} else {
			
			c.ChannelsByNumber[requestNumberID] = ch
		}
		
	} else {
		
		_, ok := c.ChannelsByString[requestStringID]
		
		if ok {
			// id already exists, that's very bad
			return nil, NewJSONRPCError(jsonRPCInternalErrorCode, "Internal error")
		} else {
			
			c.ChannelsByString[requestStringID] = ch
		}
	}
	c.mu.Unlock()
	
	// build request, and pass request to the request sender
	err := c.SendRequest(&JSONRPCRequest{"2.0", requestID, method, params})
	
	// if immediate error sending the request, passthrough the error
	if err != nil {
		
		c.mu.Lock()
		if isNumericID {
			delete(c.ChannelsByNumber, requestNumberID)
		} else {
			delete(c.ChannelsByString, requestStringID)
		}
		c.mu.Unlock()
		
		return nil, err
	}
	
	// return the result, or timeout err if timeout
	select {
		case res := <- ch:
			
			c.mu.Lock()
			if isNumericID {
				delete(c.ChannelsByNumber, requestNumberID)
			} else {
				delete(c.ChannelsByString, requestStringID)
			}
			c.mu.Unlock()
			
			if res.Error != nil {
				return nil, res.Error
			} else {
				return res, nil
			}
			
		case <- time.After(timeout):
			
			c.mu.Lock()
			if isNumericID {
				delete(c.ChannelsByNumber, requestNumberID)
			} else {
				delete(c.ChannelsByString, requestStringID)
			}
			c.mu.Unlock()
			
			// if a request still comes in with a certain id, then it will simply be ignored, memory is already cleaned up here anyway
			
			return nil, NewJSONRPCError(jsonRPCRequestTimeoutErrorCode, "Request timeout")
	}
}
func (c *client) ReceiveBytes(data *[]byte) {
	
	var response JSONRPCResponse
	err := json.Unmarshal(*data, &response)
	
	if err != nil {
		c.emitResponse(&JSONRPCResponse{"2.0", nil, nil, NewJSONRPCError(jsonRPCParseErrorCode, "Parse error")})
	} else {
		c.ReceiveSingle(&response)
	}
}
func (c *client) ReceiveString(data string) {
	
	arr := []byte(data)
	c.ReceiveBytes(&arr)
}
func (c *client) ReceiveSingle(response *JSONRPCResponse) {
	
	c.emitResponse(response)
}
func (c *client) ReceiveMultiple(responses []*JSONRPCResponse) {
	
	for _, response := range responses {
		
		c.emitResponse(response)
	}
}
func (c *client) emitResponse(response *JSONRPCResponse) {
	
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
				return // ignore this invalid response
			}
			
		case string:
			
			responseStringID = v
			
			if responseStringID == "" {
				// empty string ID is not allowed
				return // ignore this invalid response
			}
		
		default:
			
			// invalid ID type (must be string or float64)
			return // ignore this invalid response
	}
	
	if isNumericID {
		
		ch, ok := c.ChannelsByNumber[responseNumberID]
		
		if ok {
			ch <- response
		
		} // else: ignore unrecognized response ID, does not match a request ID
	} else {
		
		ch, ok := c.ChannelsByString[responseStringID]
		
		if ok {
			ch <- response
		
		} // else: ignore unrecognized response ID, does not match a request ID
	}
}

// Main test example:
/*func main() {
	
	log.Println("Main(): begin")
	
	server := NewJSONRPCServer(func (response *JSONRPCResponse) {
		if response.Error != nil {
			encodedResponse, _ := json.Marshal(*response)
			fmt.Printf("Error Response: %+v\n", string(encodedResponse))
		} else {
			encodedResponse, _ := json.Marshal(*response)
			fmt.Printf("Success Response: %+v\n", string(encodedResponse))
		}
	})
	server.AddMethod("echo", func (requestParams interface{}) (*interface{}, *JSONRPCError) {
		
		params := requestParams.(map[string]interface{})
		
		return params["text"], nil
	})
	
	incoming_message := `{"jsonrpc":"2.0","id":123,"method":"echo","params":{"text":"hello there"}}`
	server.ReceiveString(incoming_message)
	
	log.Println("Main(): end")
}*/
