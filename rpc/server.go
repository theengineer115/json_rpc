//rpc/server.go
package rpc

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"

	"rpcproxy/validation"
    "rpcproxy/kafka"
    "rpcproxy/model"
    "rpcproxy/uuid"
)

type Server struct {
    httpServer    *http.Server
    kafkaProducer *kafka.KafkaProducer
}

func NewServer(kafkaProducer *kafka.KafkaProducer) *Server {
    return &Server{
        httpServer:    &http.Server{},
        kafkaProducer: kafkaProducer,
    }
}

func (s *Server) Start(port string) error {
    mux := http.NewServeMux()
    mux.HandleFunc("/", s.handleRequest)
    mux.HandleFunc("/health", s.handleHealthCheck)

    s.httpServer = &http.Server{
        Addr:    port,
        Handler: mux,
    }

    err := s.httpServer.ListenAndServe()
    if err != nil && err != http.ErrServerClosed {
        return err
    }
    return nil
}

func (s *Server) Shutdown() error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    return s.httpServer.Shutdown(ctx)
}

func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
        return
    }

    var request struct {
        Method string        `json:"method"`
        Params []interface{} `json:"params"`
    }

    err := json.NewDecoder(r.Body).Decode(&request)
    if err != nil {
        http.Error(w, "Invalid request payload", http.StatusBadRequest)
        return
    }
    defer r.Body.Close()

    fmt.Printf("Received request: %+v\n", request)

    switch request.Method {
    case "eth_sendRawTransaction":
        if len(request.Params) != 1 {
            http.Error(w, "Invalid transaction parameters", http.StatusBadRequest)
            return
        }
        rawTx, ok := request.Params[0].(string)
        if !ok || rawTx == "" {
            http.Error(w, "Invalid transaction parameters", http.StatusBadRequest)
            return
        }
        fmt.Printf("Processing raw transaction: %s\n", rawTx)
        s.handleSubmitTransaction(w, r, rawTx)
    case "eth_cancelTransaction":
        if len(request.Params) != 1 {
            http.Error(w, "Invalid cancellation parameters", http.StatusBadRequest)
            return
        }
        uuid, ok := request.Params[0].(string)
        if !ok || uuid == "" {
            http.Error(w, "Invalid cancellation parameters", http.StatusBadRequest)
            return
        }
        s.handleCancelTransaction(w, r, uuid)
    default:
        http.Error(w, "Invalid method", http.StatusBadRequest)
    }
}

func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
        return
    }

    // Ping Pong
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("pong"))
}

func (s *Server) handleSubmitTransaction(w http.ResponseWriter, r *http.Request, rawTx string) {
    // Logging
    fmt.Println("Received raw transaction:", rawTx)

    // Validate the raw transaction
    validationResult, err := validation.ValidateTransaction(rawTx)
    if err != nil {
        http.Error(w, fmt.Sprintf("Transaction validation failed: %v", err), http.StatusBadRequest)
        return
    }

    validationJSON, err := json.Marshal(validationResult)
    if err != nil {
        http.Error(w, fmt.Sprintf("Failed to marshal validation result: %v", err), http.StatusInternalServerError)
        return
    }

    // Check if all fields are valid
    if !validationResult.Nonce || !validationResult.GasPrice || !validationResult.GasLimit || !validationResult.To || !validationResult.Value {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusBadRequest)
        w.Write(validationJSON)
        return
    }

    fmt.Println("Transaction validation passed")

    // Generate a new UUID for the transaction
    uuid := uuid.GenerateUUID()


    tx := &model.Transaction{
        UUID:  uuid,
        RawTx: rawTx,
    }

    message := map[string]interface{}{
        "type": "transaction",
        "data": tx,
    }

    err = s.kafkaProducer.SendMessage(r.Context(), message)
    if err != nil {
        http.Error(w, fmt.Sprintf("Failed to submit transaction: %v", err), http.StatusInternalServerError)
        return
    }

    // Return the UUID
    response := map[string]string{
        "uuid": uuid,
    }
    jsonResponse, err := json.Marshal(response)
    if err != nil {
        http.Error(w, fmt.Sprintf("Failed to marshal response: %v", err), http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    w.Write(jsonResponse)
}

func isValidUUID(u string) bool {
    return uuid.IsValidUUID(u)
}

func (s *Server) handleCancelTransaction(w http.ResponseWriter, r *http.Request, uuid string) {
    if !isValidUUID(uuid) {
        http.Error(w, "Invalid UUID", http.StatusBadRequest)
        return
    }


    cancellationMsg := &model.CancellationMessage{
        UUID: uuid,
    }


    message := map[string]interface{}{
        "type": "cancellation",
        "data": cancellationMsg,
    }

    err := s.kafkaProducer.SendMessage(r.Context(), message)
    if err != nil {
        http.Error(w, fmt.Sprintf("Failed to cancel transaction: %v", err), http.StatusInternalServerError)
        return
    }

    // Send success response
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    response := map[string]string{
        "status": "cancelled",
        "uuid":   uuid,
    }
    jsonResponse, err := json.Marshal(response)
    if err != nil {
        http.Error(w, fmt.Sprintf("Failed to marshal response: %v", err), http.StatusInternalServerError)
        return
    }
    w.Write(jsonResponse)
}
