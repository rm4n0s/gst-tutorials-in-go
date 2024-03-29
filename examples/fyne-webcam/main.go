package main

import (
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"github.com/go-gst/go-glib/glib"
	"github.com/go-gst/go-gst/gst"
	gstApp "github.com/go-gst/go-gst/gst/app"
)

var rgba = image.NewRGBA(image.Rect(0, 0, 640, 480))

func runPipeline(mainLoop *glib.MainLoop, screen *canvas.Image) error {
	gst.Init(nil)
	pipeline, err := gst.NewPipeline("")
	if err != nil {
		return err
	}

	camsrc, err := gst.NewElement("v4l2src")
	if err != nil {
		return err
	}
	camsrc.Set("caps", "video/x-raw,format=RGB,width=640,height=360,framerate=30/1")

	jpegenc, err := gst.NewElement("jpegenc")
	if err != nil {
		return err
	}

	queue, err := gst.NewElement("queue")
	if err != nil {
		return err
	}
	sink, err := gstApp.NewAppSink()
	if err != nil {
		return err
	}

	pipeline.AddMany(camsrc, jpegenc, queue, sink.Element)
	err = gst.ElementLinkMany(camsrc, jpegenc, queue, sink.Element)
	if err != nil {
		return err
	}

	sink.SetCallbacks(&gstApp.SinkCallbacks{
		// Add a "new-sample" callback
		NewSampleFunc: func(sink *gstApp.Sink) gst.FlowReturn {

			// Pull the sample that triggered this callback
			sample := sink.PullSample()
			if sample == nil {
				return gst.FlowEOS
			}
			//defer sample.Unref()

			// Retrieve the buffer from the sample
			buffer := sample.GetBuffer()
			if buffer == nil {
				return gst.FlowError
			}
			defer buffer.Unmap()

			img, err := jpeg.Decode(buffer.Reader())
			if err != nil {
				log.Fatal(err)
			}
			draw.Draw(rgba, rgba.Bounds(), img, image.Point{0, 0}, draw.Over)
			screen.Refresh()
			return gst.FlowOK
		},
	})
	pipeline.SetState(gst.StatePlaying)
	return mainLoop.RunError()
}

func webcamGst(screen *canvas.Image) {
	mainLoop := glib.NewMainLoop(glib.MainContextDefault(), false)

	if err := runPipeline(mainLoop, screen); err != nil {
		fmt.Println("ERROR!", err)
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	var width float32 = 640
	var height float32 = 480
	a := app.New()
	w := a.NewWindow("Video Player")
	w.Resize(fyne.NewSize(width, height))
	w.SetPadded(false)
	screen := canvas.NewImageFromImage(rgba)
	screen.ScaleMode = canvas.ImageScaleFastest
	go webcamGst(screen)

	w.SetContent(screen)
	w.ShowAndRun()

}
