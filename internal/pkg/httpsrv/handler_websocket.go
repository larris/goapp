package httpsrv

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/csrf"
	"log"
	"net/http"

	"goapp/internal/pkg/watcher"

	"github.com/gorilla/websocket"
)

func (s *Server) handlerWebSocket(w http.ResponseWriter, r *http.Request) {
	//Get the token from the request
	csrfToken := r.URL.Query().Get("csrf_token")
	log.Printf("csrfToken = %s , csrf.Token(r) = %s", csrfToken, csrf.Token(r))
	// check the token
	//if csrfToken != csrf.Token(r) {
	//	s.error(w, http.StatusForbidden, fmt.Errorf("invalid CSRF token"))
	//	return
	//}
	// Create and start a watcher.
	var watch = watcher.New()
	if err := watch.Start(); err != nil {
		s.error(w, http.StatusInternalServerError, fmt.Errorf("failed to start watcher: %w", err))
		return
	}
	defer watch.Stop()

	s.addWatcher(watch)
	defer s.removeWatcher(watch)

	// Start WS.
	var upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.error(w, http.StatusInternalServerError, fmt.Errorf("failed to upgrade connection: %w", err))
		return
	}
	defer func() { _ = c.Close() }()

	connID := len(s.conn)
	s.connLock.Lock()
	s.conn[c] = &sessionStats{} //add the WS connection
	s.connLock.Unlock()
	defer func() {
		s.connLock.Lock()
		delete(s.conn, c)
		s.connLock.Unlock()
	}()

	log.Printf("websocket started for watcher %s\n", watch.GetWatcherId())
	defer func() {
		log.Printf("websocket stopped for watcher %s\n", watch.GetWatcherId())
	}()

	// Read done.
	readDoneCh := make(chan struct{})

	// All done.
	doneCh := make(chan struct{})
	defer close(doneCh)

	go func() {
		defer close(readDoneCh)
		for {
			select {
			default:
				_, p, err := c.ReadMessage()
				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived) {
						log.Printf("failed to read message: %v\n", err)
					}
					return
				}
				var m watcher.CounterReset
				if err := json.Unmarshal(p, &m); err != nil {
					log.Printf("failed to unmarshal message: %v\n", err)
					continue
				}
				watch.ResetCounter()
			case <-doneCh:
				return
			case <-s.quitChannel:
				return
			}
		}
	}()

	for {
		select {
		case cv := <-watch.Recv():
			data, _ := json.Marshal(cv)
			msg := fmt.Sprintf("[conn #%d] value: %s", connID, string(data))
			err = c.WriteMessage(websocket.TextMessage, []byte(msg))
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("failed to write message: %v\n", err)
				}
				return
			}
		case <-readDoneCh:
			return
		case <-s.quitChannel:
			return
		}
	}
}
