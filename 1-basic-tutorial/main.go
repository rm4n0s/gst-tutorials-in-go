package main

import (
	"fmt"
	"os"

	"github.com/go-gst/go-glib/glib"
	"github.com/go-gst/go-gst/gst"
)

func runPipeline(mainLoop *glib.MainLoop) error {
	//   /* Initialize GStreamer */
	gst.Init(&os.Args)

	//   /* Build the pipeline */
	pipeline, err := gst.NewPipelineFromString("v4l2src device=/dev/video0 ! jpegdec ! xvimagesink")
	if err != nil {
		return err
	}

	pipeline.GetPipelineBus().AddWatch(func(msg *gst.Message) bool {
		switch msg.Type() {
		case gst.MessageEOS:
			pipeline.BlockSetState(gst.StateNull)
			mainLoop.Quit()
		case gst.MessageError:
			err := msg.ParseError()
			fmt.Println("ERROR:", err.Error())
			if debug := err.DebugString(); debug != "" {
				fmt.Println("DEBUG:", debug)
			}
			mainLoop.Quit()
		default:

			fmt.Println(msg)
		}
		return true
	})

	//   /* Start playing */
	pipeline.SetState(gst.StatePlaying)

	return mainLoop.RunError()
}

func main() {
	mainLoop := glib.NewMainLoop(glib.MainContextDefault(), false)

	if err := runPipeline(mainLoop); err != nil {
		fmt.Println("ERROR!", err)
	}
}
