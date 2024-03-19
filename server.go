package main

import (
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"log"
	"os"
	"runtime/pprof"
	"time"
)

func echo(c *websocket.Conn) {

	defer c.Close()

	startTime := time.Now()

	ch := make(chan int)

	go func(ch chan int, c *websocket.Conn, startTime time.Time) {
		message := make([]byte, 1024)
		totalBytes := 0

		duration := 10 * time.Second
		timer := time.NewTimer(duration)
	loop:
		for {
			select {
			case <-timer.C:
				break loop
			default:
				err := c.WriteMessage(websocket.TextMessage, message)
				totalBytes += len(message)
				if err != nil {
					log.Println("write:", err)
					break
				}
			}
		}

		ch <- totalBytes
	}(ch, c, startTime)

	totalByteSent := <-ch

	throughput := float64(totalByteSent) / float64(10)

	log.Printf("throughput is %f kb/s", throughput/1024)
}

func main() {

	f, err := os.Create("cpu.prof")
	if err != nil {
		log.Fatal("couldnt create profile file: ", err)
	}
	defer f.Close()

	if err := pprof.StartCPUProfile(f); err != nil {
		log.Fatal("cpu profiling error: ", err)
	}
	go func() {
		time.Sleep(10 * time.Second)
		pprof.StopCPUProfile()
	}()

	app := fiber.New()

	app.Use("/", func(c *fiber.Ctx) error {

		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	app.Get("/", websocket.New(echo))

	log.Fatal(app.Listen(":3000"))
	// server at ws://localhost:3000/

}
