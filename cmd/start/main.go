package main

import (
	"amigo"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
	"github.com/kr/pretty"
)

func init() {
	http.HandleFunc("/bundle/bundle.js", func(rw http.ResponseWriter, r *http.Request) {
		http.ServeFile(rw, r, "web/bundle/bundle.js")
	})

	http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		http.ServeFile(rw, r, "web/index.html")
	})
}

func init() {

	// pretty.Println(amigo.Diff(amigo.StaticDynamic{Dynamic: []interface{}{"hello", "malte"}}, amigo.StaticDynamic{Dynamic: []interface{}{"hello", "donald"}}))
	// os.Exit(1)

	// tt, err := template.New("test").Parse(`
	// 	<p> {{.Assets.FromMap}} </p>

	// 	<p> {{.In.FromStruct}} </p>
	// `)
	// fmt.Println(err)

	// type in interface{}

	// type comp struct {
	// 	FromStruct string
	// }

	// data := struct {
	// 	Assets map[string]interface{}
	// 	In     in
	// }{
	// 	Assets: map[string]interface{}{
	// 		"FromMap": "hello from map",
	// 	},
	// 	In: comp{
	// 		FromStruct: "hello from struct",
	// 	},
	// }

	// err = tt.Execute(os.Stdout, data)
	// fmt.Println(err)

	// os.Exit(1)
}

func main() {

	errors := make(chan error, 2)

	http.HandleFunc("/ws", func(rw http.ResponseWriter, r *http.Request) {

		upgrader := websocket.Upgrader{}

		conn, err := upgrader.Upgrade(rw, r, nil)
		if err != nil {
			log.Println(err)
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		events := make(chan amigo.Event)

		ctx, canc := context.WithCancel(context.Background())

		onMount := make(chan amigo.StaticDynamic)

		patchesStream := amigo.New(ctx, &TestSite{}, events, errors, onMount)

		// send mount message

		{

			mountedWith := <-onMount
			fmt.Println("mounted: ", mountedWith.String())

			payload, err := json.Marshal(mountedWith)
			if err != nil {
				errors <- err
			}

			err = conn.WriteMessage(websocket.BinaryMessage, payload)
			if err != nil {
				errors <- err
			}
			close(onMount)
		}

		go func() {
			for patches := range patchesStream {
				pretty.Println(patches)

				payload, err := json.Marshal(patches)
				if err != nil {
					errors <- err
				}

				err = conn.WriteMessage(websocket.BinaryMessage, payload)
				if err != nil {
					errors <- err
				}

			}
			errors <- nil
		}()

		go func() {
			for {
				_, err := bufio.NewReader(os.Stdin).ReadByte()
				if err != nil {
					errors <- err
				}
				events <- amigo.Event{Name: "event1"}
			}
		}()

		// events <- amigo.Event{Name: "event1"}
		// events <- amigo.Event{Name: "event1"}
		// events <- amigo.Event{Name: "event1"}
		// events <- amigo.Event{Name: "event1"}

		go func() {
			for {
				var msg = map[string]string{}

				err := conn.ReadJSON(&msg)
				if err != nil {
					errors <- err
					continue
				}

				fmt.Println(msg)

				t := msg["type"]
				delete(msg, "type")

				events <- amigo.Event{
					Name: t,
					Data: msg,
				}
			}

		}()

		fmt.Printf("connection error: %v", <-errors)
		canc()
		conn.Close()
		close(events)

		close(errors)
	})

	errors <- http.ListenAndServe(":8080", nil)

}