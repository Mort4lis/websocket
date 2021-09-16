// Package websocket implements the WebSocket protocol defined in RFC 6455.
//
// How to use
//
// The Conn type represents the WebSocket connection. if you are developing a server
// application you should use Upgrade function in your http handler to switching protocol
// to WebSocket.
//
//  func handler(w http.ResponseWriter, req *http.Request) {
//      conn, err := websocket.Upgrade(w, req)
//      if err != nil {
//          log.Println(err)
//          return
//      }
//      ...
//  }
//
// Otherwise, if you are interesting to use websocket package as a client you should
// invoke Dialer.Dial at first (or Dialer.DialContext). For example:
//
//  func main() {
//     dialer := &websocket.Dialer{
//         HandshakeTimeout: 10 * time.Second,
//     }
//     conn, err := dialer.Dial("ws://localhost:8080")
//     if err != nil {
//         log.Fatal(err)
//     }
//     ...
//  }
//
// Having Conn instance you can send and receive message due WebSocket protocol, calling
// Conn.WriteMessage and Conn.ReadMessage.
//
//  typ, payload, err := conn.ReadMessage()
//  if err != nil {
//      log.Println(err)
//      return
//  }
//  if err = conn.WriteMessage(typ, payload); err != nil {
//      log.Println(err)
//      return
//  }
//
// Also you can use Conn.NextWriter and Conn.NextReader for fragmented sending and receiving.
package websocket
