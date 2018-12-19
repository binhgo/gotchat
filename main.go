package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	// Import this library.
	"github.com/centrifugal/centrifuge"
)

func handleLog(e centrifuge.LogEntry) {
	log.Printf("%s: %v", e.Message, e.Fields)
}

// Wait until program interrupted. When interrupted gracefully shutdown Node.
func waitExitSignal(n *centrifuge.Node) {
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		n.Shutdown(ctx)
		done <- true
	}()
	<-done
}

func main() {
	// We use default config here as starting point. Default config contains
	// reasonable values for available options.
	cfg := centrifuge.DefaultConfig
	// In this example we want client to do all possible actions with server
	// without any authentication and authorization. Insecure flag DISABLES
	// many security related checks in library. This is only to make example
	// short. In real app you most probably want authenticate and authorize
	// access to server. See godoc and examples in repo for more details.
	cfg.ClientInsecure = true
	// By default clients can not publish messages into channels. Setting this
	// option to true we allow them to publish.
	cfg.Publish = true

	// Node is the core object in Centrifuge library responsible for many useful
	// things. Here we initialize new Node instance and pass config to it.
	node, _ := centrifuge.New(cfg)

	// On().Connect() method is a point where you create a binding between
	// Centrifuge and your app business logic. Callback function you pass
	// to On().Connect will be called every time new connection established
	// with server. Inside this callback function you can set various event
	// handlers for incoming client connection.
	node.On().Connect(func(ctx context.Context, client *centrifuge.Client, e centrifuge.ConnectEvent) centrifuge.ConnectReply {
		// Set Subscribe Handler to react on every channel subscription attempt
		// initiated by client. Here you can theoretically return an error or
		// disconnect client from server if needed. But now we just accept
		// all subscriptions.
		client.On().Subscribe(func(e centrifuge.SubscribeEvent) centrifuge.SubscribeReply {
			log.Printf("client subscribes on channel %s", e.Channel)
			return centrifuge.SubscribeReply{}
		})

		// Set Publish Handler to react on every channel Publication sent by client.
		// Inside this method you can validate client permissions to publish into
		// channel. But in our simple chat app we allow everyone to publish into
		// any channel.
		client.On().Publish(func(e centrifuge.PublishEvent) centrifuge.PublishReply {
			log.Printf("client publishes into channel %s: %s", e.Channel, string(e.Data))

			//out, err1 := exec.Command("python3", "chatbot.py", "-q", string(e.Data)).Output()
			//if err1 != nil {
			//	log.Println("ERROR")
			//	log.Fatal(err1)
			//}
			//fmt.Printf("Output \n%s", out)


			return centrifuge.PublishReply{}
		})

		// Set Disconnect Handler to react on client disconnect events.
		client.On().Disconnect(func(e centrifuge.DisconnectEvent) centrifuge.DisconnectReply {
			log.Printf("client disconnected")
			return centrifuge.DisconnectReply{}
		})

		// In our example transport will always be Websocket but it can also be SockJS.
		transportName := client.Transport().Name()
		// In our example clients connect with JSON protocol but it can also be Protobuf.
		transportEncoding := client.Transport().Encoding()

		log.Printf("client connected via %s (%s)", transportName, transportEncoding)
		return centrifuge.ConnectReply{}
	})

	// Centrifuge library exposes logs with different log level. In your app
	// you can set special function to handle these log entries in a way you want.
	node.SetLogHandler(centrifuge.LogLevelDebug, handleLog)

	// Run node will start node's underlying Engine, launch several
	// internal goroutines.
	if err := node.Run(); err != nil {
		panic(err)
	}

	// Configure http routes.

	// The first route is for handling Websocket connections.
	http.Handle("/connection/websocket", centrifuge.NewWebsocketHandler(node, centrifuge.WebsocketConfig{}))

	// The second route is for serving index.html file.
	http.Handle("/", http.FileServer(http.Dir("./")))

	// Start HTTP server.
	go func() {
		if err := http.ListenAndServe(":8000", nil); err != nil {
			panic(err)
		}
	}()

	// Run program until interrupted.
	waitExitSignal(node)
}
