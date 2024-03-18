package main

import (
	"fmt"
	"log"

	"github.com/go-gst/go-glib/glib"
	"github.com/go-gst/go-gst/gst"
)

func runPipeline(mainLoop *glib.MainLoop) error {
	gst.Init(nil)

	elements, err := gst.NewElementMany("videotestsrc", "vertigotv", "videoconvert", "autovideosink")
	if err != nil {
		log.Fatal(err)
	}
	source := elements[0]
	vertigotv := elements[1]
	converter := elements[2]
	sink := elements[3]

	pipeline, err := gst.NewPipeline("test_pipeline")
	if err != nil {
		log.Fatal(err)
	}

	err = pipeline.AddMany(source, vertigotv, converter, sink)
	if err != nil {
		log.Fatal(err)
	}

	err = gst.ElementLinkMany(source, vertigotv, converter, sink)
	if err != nil {
		log.Fatal(err)
	}

	t, err := source.GetPropertyType("pattern")
	if err != nil {
		log.Fatal(err)
	}
	val, err := glib.ValueInit(t)
	if err != nil {
		log.Fatal(err)
	}
	val.SetEnum(18)
	err = source.SetPropertyValue("pattern", val)
	if err != nil {
		log.Fatal(err)
	}

	pipeline.GetBus().AddWatch(func(msg *gst.Message) bool {
		switch msg.Type() {
		case gst.MessageEOS:
			pipeline.BlockSetState(gst.StateNull)
			mainLoop.Quit()
		case gst.MessageError:
			err := msg.ParseError()
			log.Println("Error:", err.Error())
			debug := err.DebugString()
			if len(debug) > 0 {
				log.Println("Debug: ", debug)
			}
		default:
			log.Println("Unexpected message received")
		}
		return true
	})

	err = pipeline.SetState(gst.StatePlaying)
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
