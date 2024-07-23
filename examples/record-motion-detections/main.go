package main

import (
	"context"
	"flag"
	"fmt"
	"image"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-gst/go-glib/glib"
	"github.com/go-gst/go-gst/gst"
	"github.com/go-gst/go-gst/gst/app"
	"github.com/go-gst/go-gst/gst/video"
	"gocv.io/x/gocv"
)

const MinimumArea = 3000

const (
	camToSink   = "v4l2src device=/dev/video0 ! clockoverlay time-format='%D %H:%M:%S' color=0xFF00FF00 halignment=right valignment=bottom ! videoconvert ! video/x-raw,format=RGBA,width=640,height=360,framerate=30/1 ! appsink name=appsink"
	srcToWindow = "appsrc name=appsrc ! videoconvert ! xvimagesink"
)

func pipeSrcToWindow(ctx context.Context, queueWindow chan []byte) {
	pipeline, err := gst.NewPipelineFromString(srcToWindow)
	if err != nil {
		panic(err)
	}

	if err = pipeline.SetState(gst.StatePlaying); err != nil {
		panic(err)
	}

	appsrc, err := pipeline.GetElementByName("appsrc")
	if err != nil {
		panic(err)
	}
	videoInfo := video.NewInfo().
		WithFormat(video.FormatRGBA, 640, 360).
		WithFPS(gst.Fraction(30, 1))

	src := app.SrcFromElement(appsrc)
	src.SetCaps(videoInfo.ToCaps())
	i := 0
	src.SetCallbacks(&app.SourceCallbacks{
		NeedDataFunc: func(self *app.Source, _ uint) {
			select {
			case <-ctx.Done():
				log.Println("end of pipeSrcToWindow")
				src.EndStream()
				return
			case img := <-queueWindow:
				log.Println("Producing frame:", i)

				// Create a buffer that can hold exactly one video RGBA frame.
				buffer := gst.NewBufferWithSize(videoInfo.Size())
				buffer.Map(gst.MapWrite).WriteData(img)
				buffer.Unmap()

				// Push the buffer onto the pipeline.
				self.PushBuffer(buffer)
				i++
			}
		},
	})
}
func pipeCamToSink(ctx context.Context, mainLoop *glib.MainLoop, rec *Recorder, queueWindow chan []byte) error {
	pipeline, err := gst.NewPipelineFromString(camToSink)
	if err != nil {
		panic(err)
	}

	if err = pipeline.SetState(gst.StatePlaying); err != nil {
		panic(err)
	}

	appSink, err := pipeline.GetElementByName("appsink")
	if err != nil {
		panic(err)
	}

	imgDelta := gocv.NewMat()
	defer imgDelta.Close()

	imgThresh := gocv.NewMat()
	defer imgThresh.Close()
	mog2 := gocv.NewBackgroundSubtractorMOG2()
	defer mog2.Close()

	sink := app.SinkFromElement(appSink)
	sink.SetCallbacks(&app.SinkCallbacks{
		NewSampleFunc: func(sink *app.Sink) gst.FlowReturn {
			select {
			case <-ctx.Done():
				log.Println("closed sink")
				return gst.FlowEOS
			default:

				sample := sink.PullSample()
				if sample == nil {
					return gst.FlowEOS
				}
				caps := sample.GetCaps()
				s := caps.GetStructureAt(0)
				width, err := s.GetValue("width")
				if err != nil {
					log.Fatal(err)
				}
				height, err := s.GetValue("height")
				if err != nil {
					log.Fatal(err)
				}
				buffer := sample.GetBuffer()
				if buffer == nil {
					return gst.FlowError
				}

				samples := buffer.Map(gst.MapRead).Bytes()
				defer buffer.Unmap()
				mat, err := gocv.NewMatFromBytes(height.(int), width.(int), gocv.MatTypeCV8UC4, samples)
				if err != nil {
					panic(err)
				}
				defer mat.Close()
				if mat.Empty() {
					panic("empty")
				}
				mog2.Apply(mat, &imgDelta)
				gocv.Threshold(imgDelta, &imgThresh, 25, 255, gocv.ThresholdBinary)

				kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(3, 3))
				gocv.Dilate(imgThresh, &imgThresh, kernel)
				kernel.Close()

				// now find contours
				contours := gocv.FindContours(imgThresh, gocv.RetrievalExternal, gocv.ChainApproxSimple)
				hasMotion := false
				for i := 0; i < contours.Size(); i++ {
					area := gocv.ContourArea(contours.At(i))
					if area < MinimumArea {
						continue
					}

					log.Println("Motion detected")
					hasMotion = true
				}

				contours.Close()
				pts := buffer.PresentationTimestamp()
				log.Println(width.(int), height.(int), len(samples), hasMotion, pts)
				rec.QueueImg(samples, hasMotion, pts)
				queueWindow <- samples
				return gst.FlowOK
			}
		},
	})

	return mainLoop.RunError()

}

func main() {
	videoFolder := flag.String("path", "", "path to save the videos")
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	gst.Init(nil)
	rec := NewRecorder(10*time.Second, *videoFolder)
	rec.Start(ctx)

	queueWindow := make(chan []byte)
	go pipeSrcToWindow(ctx, queueWindow)

	mainLoop := glib.NewMainLoop(glib.MainContextDefault(), false)
	go func(ctx context.Context, mainLoop *glib.MainLoop) {
		<-ctx.Done()
		time.Sleep(time.Second)
		mainLoop.Quit()
		fmt.Println("Bye!")
	}(ctx, mainLoop)
	if err := pipeCamToSink(ctx, mainLoop, rec, queueWindow); err != nil {
		fmt.Println("ERROR:", err)
	}
}
