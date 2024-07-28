package main

//https://gist.github.com/theCalcaholic/bea95753cc90da8b7562046f9175fcc2
import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-gst/go-gst/gst"
	"github.com/godbus/dbus/v5"
)

const request_iface = "org.freedesktop.portal.Request"

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	var (
		mins int
		out  string
	)
	flag.IntVar(&mins, "mins", 0, "the minutes until closing recording")
	flag.StringVar(&out, "out", "", "the path for the output, for example -out=test.mp4")
	flag.Parse()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	if mins > 0 {
		time.AfterFunc(time.Duration(mins)*time.Minute, func() {
			log.Printf("%d minutes have passed", mins)
			stop()
		})
	}
	log.Println("save output to ", out)
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go mainRoutine(ctx, wg, out)
	wg.Wait()
	fmt.Println("Bye!")
}

func mainRoutine(ctx context.Context, wg *sync.WaitGroup, filePath string) {
	gst.Init(nil)
	defer wg.Done()
	bus, err := dbus.ConnectSessionBus()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to connect to session bus:", err)
		os.Exit(1)
	}
	defer bus.Close()
	portal := bus.Object("org.freedesktop.portal.Desktop",
		"/org/freedesktop/portal/desktop")

	createSessionCall := portal.Call("org.freedesktop.portal.ScreenCast.CreateSession", 0, map[string]dbus.Variant{
		"session_handle_token": dbus.MakeVariant("u2"),
		"handle_token":         dbus.MakeVariant("u1"),
	})

	if createSessionCall.Err != nil {
		log.Fatal(createSessionCall.Err)
	}
	reqCSC := createSessionCall.Body[0].(dbus.ObjectPath)
	respCSC := getResponse(bus, reqCSC)
	if respCSC.Body[0].(uint32) != 0 {
		log.Fatal("failed to create session")
	}
	resultCSC := respCSC.Body[1].(map[string]dbus.Variant)
	sessionHandle := resultCSC["session_handle"].Value().(string)
	selectSourceCall := portal.Call("org.freedesktop.portal.ScreenCast.SelectSources", 0, dbus.ObjectPath(sessionHandle), map[string]any{
		"multiple": false,
		"types":    uint32(1 | 2),
	})
	if selectSourceCall.Err != nil {
		log.Fatal(selectSourceCall.Err)
	}

	reqSCS := selectSourceCall.Body[0].(dbus.ObjectPath)
	respSCS := getResponse(bus, reqSCS)
	if respSCS.Body[0].(uint32) != 0 {
		log.Fatal("failed to select sources")
	}

	startCall := portal.Call("org.freedesktop.portal.ScreenCast.Start", 0, dbus.ObjectPath(sessionHandle), "", map[string]any{})
	if startCall.Err != nil {
		log.Fatal(startCall.Err)
	}
	reqStart := startCall.Body[0].(dbus.ObjectPath)
	respStart := getResponse(bus, reqStart)
	if respSCS.Body[0].(uint32) != 0 {
		log.Fatal("failed to select sources")
	}
	resultStart := respStart.Body[1].(map[string]dbus.Variant)
	arr := resultStart["streams"].Value().([][]interface{})[0]
	nodeId := arr[0].(uint32)

	openPipeWireRemoteCall := portal.Call("org.freedesktop.portal.ScreenCast.OpenPipeWireRemote", 0, dbus.ObjectPath(sessionHandle), map[string]any{})
	if openPipeWireRemoteCall.Err != nil {
		log.Fatal(openPipeWireRemoteCall.Err)
	}
	fd := openPipeWireRemoteCall.Body[0].(dbus.UnixFD)
	pipe := fmt.Sprintf("pipewiresrc fd=%d path=%d  do-timestamp=true ! videoconvert ! x264enc ! mp4mux ! filesink location=%s", fd, nodeId, filePath)
	log.Println(pipe)
	pipeline, err := gst.NewPipelineFromString(pipe)
	if err != nil {
		log.Fatal("Failed to create pipeline: ", err)
	}

	err = pipeline.SetState(gst.StatePlaying)
	if err != nil {
		log.Fatal("Failed to start pipeline: ", err)
	}
	log.Println("start playing")
	<-ctx.Done()
	log.Println("Send EOS")
	pipeline.SendEvent(gst.NewEOSEvent())
	time.Sleep(time.Second)
}

func getResponse(bus *dbus.Conn, req dbus.ObjectPath) *dbus.Signal {
	if err := bus.AddMatchSignal(
		dbus.WithMatchMember("Response"),
		dbus.WithMatchInterface(request_iface),
		dbus.WithMatchSender("org.freedesktop.portal.Desktop"),
		dbus.WithMatchObjectPath(req),
	); err != nil {
		log.Fatal(err)
	}

	c := make(chan *dbus.Signal, 10)
	bus.Signal(c)
	v := <-c
	return v
}
