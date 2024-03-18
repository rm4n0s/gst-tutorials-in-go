package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/go-gst/go-glib/glib"
	"github.com/go-gst/go-gst/gst"
)

type CustomData struct {
	pipeline *gst.Pipeline
	source   *gst.Element
	convert  *gst.Element
	sink     *gst.Element
}

func runPipeline(mainLoop *glib.MainLoop) error {
	gst.Init(nil)
	data := CustomData{}
	var err error
	data.source, err = gst.NewElementWithName("uridecodebin", "source")
	if err != nil {
		log.Fatal(err)
	}

	data.convert, err = gst.NewElementWithName("videoconvert", "convert")
	if err != nil {
		log.Fatal(err)
	}

	data.sink, err = gst.NewElementWithName("autovideosink", "sink")
	if err != nil {
		log.Fatal(err)
	}

	data.pipeline, err = gst.NewPipeline("test_pipeline")
	if err != nil {
		log.Fatal(err)
	}

	err = data.pipeline.AddMany(data.source, data.convert, data.sink)
	if err != nil {
		log.Fatal(err)
	}

	err = gst.ElementLinkMany(data.convert, data.sink)
	if err != nil {
		log.Fatal(err)
	}
	err = data.source.Set("uri", "https://gstreamer.freedesktop.org/data/media/sintel_trailer-480p.webm")
	if err != nil {
		log.Fatal(err)
	}

	data.source.Connect("pad-added", func(src *gst.Element, newPad *gst.Pad) {
		sinkPad := data.convert.GetStaticPad("sink")
		defer sinkPad.Unref()
		if sinkPad.IsLinked() {
			log.Println("sink is already linked")
			return
		}

		log.Printf("Received new pad '%s' from '%s':\n", newPad.GetName(), src.GetName())
		newPadCaps := newPad.GetCurrentCaps()
		defer newPadCaps.Unref()
		newPadStruct := newPadCaps.GetStructureAt(0)
		newPadType := newPadStruct.Name()
		if !strings.HasPrefix(newPadType, "video/x-raw") {
			log.Printf("It has type '%s' which is not raw audio. Ignoring.\n", newPadType)
			return
		}

		ret := newPad.Link(sinkPad)
		if ret == gst.PadLinkOK {
			log.Printf("Link succeeded (type '%s').\n", newPadType)

		} else {
			log.Printf("Type is '%s' but link failed.\n", newPadType)
		}
	})

	data.pipeline.GetBus().AddWatch(func(msg *gst.Message) bool {

		switch msg.Type() {
		case gst.MessageEOS:
			log.Println(msg.String())
			data.pipeline.BlockSetState(gst.StateNull)
			mainLoop.Quit()
		case gst.MessageError:
			err := msg.ParseError()
			log.Println("Error:", err.Error())
			debug := err.DebugString()
			if len(debug) > 0 {
				log.Println("Debug: ", debug)
			}

		case gst.MessageStateChanged:
			if msg.Source() == data.pipeline.GetName() {
				oldState, newState := msg.ParseStateChanged()
				log.Printf("Pipeline state changed from %s to %s:\n", oldState.String(), newState.String())
			}
		default:
			log.Println("Unexpected message received.", msg.String())
		}
		return true
	})

	err = data.pipeline.SetState(gst.StatePlaying)
	if err != nil {
		log.Fatal(err)
	}

	return mainLoop.RunError()
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	mainLoop := glib.NewMainLoop(glib.MainContextDefault(), false)

	if err := runPipeline(mainLoop); err != nil {
		fmt.Println("ERROR!", err)
	}
}
