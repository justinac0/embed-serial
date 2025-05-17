package main

import (
	"fmt"
	"log"
	"net/http"
	"serialembed/web/templates"
	"serialembed/web/templates/components"
	"sync"
	"time"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	"go.bug.st/serial"
)

type OutputMode int

const (
	HEX OutputMode = iota
	ASCII
)

type AppState struct {
	PortMutex   sync.Mutex
	Rx          []byte
	CurrentPort serial.Port
	RxThreads   int
	Resetting   bool
}

func NewAppState() *AppState {
	return &AppState{
		Rx:          make([]byte, 0),
		CurrentPort: nil,
		RxThreads:   0,
		Resetting:   false,
	}
}

func (app *AppState) ConnectPort(name string) error {
	// TODO: add info to set config
	mode := &serial.Mode{
		BaudRate: 115200,
		Parity:   serial.NoParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
	}

	if app.CurrentPort != nil {
		app.Resetting = true
		time.Sleep(2 * time.Second)
	}

	var err error
	app.CurrentPort, err = serial.Open(name, mode)
	if err != nil {
		log.Println(err)
	}

	return nil
}

var state *AppState

// util func to render templ components
func RenderTemplate(ctx echo.Context, component templ.Component) error {
	return component.Render(ctx.Request().Context(), ctx.Response().Writer)
}

func convertByteToEventMsg(b byte, mode OutputMode) string {
	switch mode {
	case HEX:
		return fmt.Sprintf("%02X ", b)
	case ASCII:
		// special characters:
		if b == '\r' {
			return "<span class='special'>CR</span>"
		}
		if b == '\n' {
			return "<span class='special'>LF</span></br>"
		}
		return string(b)
	default:
		return ""
	}
}

func handleIncomingBytes(bs []byte, mode OutputMode) string {
	var output string
	for _, b := range bs {
		output += convertByteToEventMsg(b, mode)
	}

	return output
}

func setupHandlers(e *echo.Echo) {

	// index
	e.GET("/", func(ctx echo.Context) error {
		if state.CurrentPort != nil {
			state.Resetting = true
		}

		return RenderTemplate(ctx, templates.Index())
	})

	// send: sends data to connected com port
	e.POST("/send", func(ctx echo.Context) error {
		message := ctx.FormValue("message")
		message += "\r\n"

		fmt.Printf("%v\n", []byte(message))

		state.PortMutex.Lock()
		fmt.Println("message: ", message)
		n, err := state.CurrentPort.Write([]byte(message))
		fmt.Println(n, err)
		state.PortMutex.Unlock()

		return ctx.NoContent(http.StatusOK)
	})

	// clear: clears contents from terminal
	e.GET("/clear", func(ctx echo.Context) error {
		return ctx.HTML(http.StatusOK, "")
	})

	// open: open a selected com port
	e.POST("/open", func(ctx echo.Context) error {
		portName := ctx.QueryParam("port_name")

		err := state.ConnectPort(portName)
		if err != nil {
			log.Println(err)
		}

		return ctx.NoContent(http.StatusOK)
	})

	// Rx SSE: recieves data from connected device
	e.GET("/RxSSE", func(ctx echo.Context) error {
		ctx.Response().Header().Set("Content-Type", "text/event-stream")
		ctx.Response().Header().Set("Cache-Control", "no-cache")
		ctx.Response().Header().Set("Connection", "keep-alive")
		ctx.Response().WriteHeader(http.StatusOK)
		ctx.Response().Flush()

		if state.CurrentPort == nil || state.RxThreads > 0 {
			return ctx.NoContent(http.StatusOK)
		}

		state.CurrentPort.SetReadTimeout(time.Millisecond)

		// NOTE: I want this thread to only be opened once
		go func() {
			state.RxThreads++
			fmt.Println("thread entered...")
			for {
				data := make([]byte, 0)
				// nice hack bro
				if state.Resetting {
					state.PortMutex.Lock()
					state.CurrentPort.ResetInputBuffer()
					state.CurrentPort.ResetOutputBuffer()
					state.CurrentPort.Close()
					time.Sleep(time.Second)
					state.CurrentPort = nil
					state.PortMutex.Unlock()
					state.Resetting = false
					fmt.Println("resetting port...")
					break
				}

				// TODO: do we have to reinstatiate this every read?
				//
				// NOTE: the baud rate is 115200, so max data coming through every
				// millisecond is (baud/10)
				buffer := make([]byte, 2048) // this is not enough memory for high baud
				state.PortMutex.Lock()
				n, err := state.CurrentPort.Read(buffer)
				state.PortMutex.Unlock()
				if err != nil {
					continue
				}

				data = append(data, buffer[:n]...)

				if n > 0 {
					msg := handleIncomingBytes(data, ASCII)
					event := fmt.Sprintf("event: %s\ndata: %v\n\n", "message", msg)
					ctx.Response().Write([]byte(event))
					ctx.Response().Flush()
				}

				time.Sleep(100 * time.Millisecond)
			}

			fmt.Println("thread exit...")
			state.RxThreads--
		}()

		for state.CurrentPort != nil {
			time.Sleep(100 * time.Millisecond)
		}

		return ctx.NoContent(http.StatusOK)
	})

	// scan: scans connected com ports
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

		return RenderTemplate(ctx, components.Ports(ports))
	})
}

func setupEcho() {
	e := echo.New()
	e.HideBanner = true

	state = NewAppState()

	e.Static("static/css", "web/static/css")
	e.Static("static/js", "web/static/js")

	setupHandlers(e)

	e.Logger.Fatal(e.Start("127.0.0.1:8000"))
}

func main() {
	setupEcho()
}
