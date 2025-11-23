package api

import (
	http_transport "mrs/internal/transport/http"
	ws_transport "mrs/internal/transport/ws"
	"net/http"
)

type API struct {
	mux *http.ServeMux
}

type Deps struct {
	HttpHandler *http_transport.Handler
	WsHandler   *ws_transport.WSHandler
}

func NewAPI(deps Deps) *API {
	apiMux := http.NewServeMux()

	apiMux.HandleFunc("/videos", Method(http.MethodGet, deps.HttpHandler.GetListVideo))
	apiMux.HandleFunc("/rooms", Method(http.MethodPost, deps.HttpHandler.CreateRoom))
	apiMux.HandleFunc("/rooms/queue", Method(http.MethodPost, deps.HttpHandler.AddVideoInQueue))
	apiMux.HandleFunc("/rooms/seek", Method(http.MethodPost, deps.HttpHandler.Seek))
	apiMux.HandleFunc("/rooms/info", Method(http.MethodGet, deps.HttpHandler.GetAllRoomsInfo))
	apiMux.HandleFunc("/rooms/delete", Method(http.MethodDelete, deps.HttpHandler.DeleteVideoInQueue))

	rootMux := http.NewServeMux()

	rootMux.Handle("/api/v1/", http.StripPrefix("/api/v1", apiMux))

	rootMux.HandleFunc("/ws/room", deps.WsHandler.RoomWS)

	return &API{mux: rootMux}
}

func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mux.ServeHTTP(w, r)
}

func Method(method string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		handler(w, r)
	}
}
