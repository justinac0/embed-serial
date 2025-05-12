package main

import (
	"fmt"
	"log"
	"net/http"
	"serialembed/web/templates"
	"time"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	"go.bug.st/serial"
)

type AppState struct {
	Rx          []byte
	CurrentPort serial.Port
}

func NewAppState() *AppState {
	return &AppState{
		Rx:          make([]byte, 0),
		CurrentPort: nil,
	}
}

func (app *AppState) ConnectPort(name string) error {
	// TODO: add info to set config
	mode := &serial.Mode{
		BaudRate: 115200,
		Parity:   serial.EvenParity,
		DataBits: 7,
		StopBits: serial.OneStopBit,
	}

	var err error
	app.CurrentPort, err = serial.Open(name, mode)
	if err != nil {
		log.Fatal(err)
	}

	// start reading? sse?

	return nil
}

func (app *AppState) DisconnectPort(name string) error {
	return nil
}

// util func to render templ components
func RenderTemplate(ctx echo.Context, component templ.Component) error {
	return component.Render(ctx.Request().Context(), ctx.Response().Writer)
}

func setupEcho() {
	e := echo.New()

	state := NewAppState()

	e.Static("static/css", "web/static/css")
	e.Static("static/js", "web/static/js")

	// handlers
	e.GET("/", func(ctx echo.Context) error {
		return RenderTemplate(ctx, templates.Index())
	})

	e.GET("/send", func(ctx echo.Context) error {
		return ctx.HTML(http.StatusOK, "<p>form posted</p>")
	})

	e.GET("/RxSSE", func(ctx echo.Context) error {
		ctx.Response().Header().Set("Content-Type", "text/event-stream")
		ctx.Response().Header().Set("Cache-Control", "no-cache")
		ctx.Response().Header().Set("Connection", "keep-alive")
		ctx.Response().WriteHeader(http.StatusOK)
		ctx.Response().Flush()

		// just read
		go func() {
			fmt.Println("thread entered...")
			for {
				// no port connected, dont read
				if state.CurrentPort == nil {
					continue
				}

				buffer := make([]byte, 1024)
				n, err := state.CurrentPort.Read(buffer)
				if err != nil {
					continue
				}

				if n > 0 {
					event := fmt.Sprintf("event: %s\ndata: %v\n\n", "message", string(buffer))
					ctx.Response().Write([]byte(event))
					ctx.Response().Flush()
					time.Sleep(time.Duration(n) * time.Millisecond)
				}
			}
		}()

		for {
		}

		return ctx.NoContent(http.StatusOK)
	})

	e.GET("/scan", func(ctx echo.Context) error {
		ports, err := serial.GetPortsList()
		if err != nil {
			log.Fatal(err)
		}
		if len(ports) == 0 {
			log.Fatal("No serial ports found!")
		}
		for _, port := range ports {
			fmt.Printf("Found port: %v\n", port)
		}

		// DEBUG: just connect to the first one please
		err = state.ConnectPort(ports[0])
		if err != nil {
			panic(err)
		}

		return ctx.HTML(http.StatusOK, fmt.Sprintf("<div id='ports'>%v</div>", ports))
	})

	e.Logger.Fatal(e.Start("127.0.0.1:8000"))
}

func main() {
	setupEcho()
}
