// package main
package jsonrpcv2

import (
//	"fmt"
//	"log"
	"encoding/json"
)

// Types:

const jsonRPCParseErrorCode int = -32700
const jsonRPCInvalidRequestErrorCode int = -32600
const jsonRPCMethodNotFoundErrorCode int = -32601
const jsonRPCInternalErrorCode int = -32603
const JSONRPCInvalidParamsErrorCode int = -32602
const JSONRPCServerErrorCode int = -32000

type JSONRPCResponseHandler = func(interface{})
type JSONRPCBatchResponseHandler = func([]*JSONRPCResponse)
type JSONRPCRequestHandler = func(interface{}) (interface{}, *JSONRPCError)

type JSONRPCError struct {
	Code int              `json:"code"`
	Message string        `json:"message"`
	Data interface{}     `json:"data,omitempty"`
}
type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID interface{}        `json:"id"`
	Result interface{}   `json:"result,omitempty"`
	Error *JSONRPCError   `json:"error,omitempty"`
}
type JSONRPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method interface{}    `json:"method"`
	ID interface{}       `json:"id"`
	Params interface{}    `json:"params"`
}

type JSONRPCServer interface {
	handleRequest(*JSONRPCRequest) *JSONRPCResponse
	
	HasMethod(string) bool
	AddMethod(string, JSONRPCRequestHandler)
	RemoveMethod(string)
	
	ReceiveBytes(*[]byte)
	ReceiveString(string)
	ReceiveSingle(*JSONRPCRequest)
	ReceiveMultiple([]*JSONRPCRequest)
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
	s.ResponseHandler = fn
	s.Methods = make(map[string]JSONRPCRequestHandler)
	return s
}

// JSONRPCServer implementation:

type server struct {
	ResponseHandler JSONRPCResponseHandler
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
		s.ResponseHandler(&JSONRPCResponse{"2.0", nil, nil, NewJSONRPCError(jsonRPCParseErrorCode, "Parse error")})
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
		
		s.ResponseHandler(response)
		
	} // else: Notification request, no response should be given
}
func (s *server) ReceiveMultiple(requests []*JSONRPCRequest) {
	
	responses := make([]*JSONRPCResponse, len(requests))
	
	for i, request := range requests {
		
		responses[i] = s.handleRequest(request)
	}
	
	s.ResponseHandler(responses)
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
