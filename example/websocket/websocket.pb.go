package websocket

type Request struct {
	Name string `json:"name,omitempty"`
}

type Response struct {
	Message string `json:"message,omitempty"`
}
