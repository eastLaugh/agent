package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/eastlaugh/agent/pkg/agents"
	"github.com/eastlaugh/agent/pkg/openai"
	"github.com/eastlaugh/agent/pkg/tools"
	"github.com/google/uuid"
)

var (
	convMu        sync.Mutex
	conversations = map[string][]openai.Message{}
)

type ChatRequest struct {
	ConversationId string `json:"conversationId"`
	Question       string `json:"question"`
}

type SSEData struct {
	State   string `json:"state"`
	Content string `json:"content"`
}

func corsOpts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.WriteHeader(http.StatusOK)
}

func main() {
	client := openai.NewClient(os.Getenv("OPENAI_BASE_URL"), os.Getenv("OPENAI_API_KEY"), os.Getenv("OPENAI_MODEL"))

	agt := agents.New(client, nil,
		rand.IntN, "",
		time.Now().Format, "",
		tools.SearchInternet, "在互联网上搜索信息",
		tools.HttpGet, "发送 HTTP GET 请求",
	)

	http.HandleFunc("OPTIONS /api/conversations", corsOpts)
	http.HandleFunc("OPTIONS /api/chat", corsOpts)
	http.HandleFunc("GET /api/conversations/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		id := r.PathValue("id")
		convMu.Lock()
		msgs := conversations[id]
		convMu.Unlock()
		if msgs == nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"messages": msgs})
	})
	http.HandleFunc("POST /api/conversations", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		id := uuid.New().String()
		convMu.Lock()
		conversations[id] = nil
		convMu.Unlock()
		json.NewEncoder(w).Encode(map[string]string{"id": id})
	})
	http.HandleFunc("POST /api/chat", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		var req ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.ConversationId == "" {
			http.Error(w, "conversationId required", http.StatusBadRequest)
			return
		}

		convMu.Lock()
		history := conversations[req.ConversationId]
		convMu.Unlock()
		// 只传历史，不传当前 user；Iter 内部会追加 user 并在 len(messages)==0 时注入 system prompt
		iter, ch := agt.Iter(history, req.Question)

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		iter2 := agents.ReactIter(iter)
		for state, chunk := range iter2 {
			jsonData, _ := json.Marshal(SSEData{State: state.String(), Content: chunk})
			fmt.Fprintf(w, "data: %s\n\n", jsonData)
			flusher.Flush()
		}
		msgs := <-ch
		convMu.Lock()
		conversations[req.ConversationId] = msgs
		convMu.Unlock()

		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
	})

	addr := ":8080"

	log.Printf("Server running on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
