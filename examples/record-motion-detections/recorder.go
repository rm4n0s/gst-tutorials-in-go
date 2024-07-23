package main

import (
	"context"
	"fmt"
	"log"
	"path"
	"time"

	"github.com/go-gst/go-gst/gst"
	"github.com/go-gst/go-gst/gst/app"
	"github.com/go-gst/go-gst/gst/video"
)

const srcToVideo = "appsrc emit-signals=True is-live=True name=appsrc1 ! queue max-size-buffers=4 ! videoconvert ! x264enc ! h264parse ! qtmux !  filesink location=%s.mp4"

type Img struct {
	Data      []byte
	HasMotion bool
	PTS       gst.ClockTime
}

type Recorder struct {
	inputQueueImgs chan Img
	srcImgs        chan Img
	startedTime    time.Time
	durFromLast    time.Duration
	running        bool
	ctxGst         context.Context
	cancelGst      context.CancelFunc
	videoFolder    string
}

func NewRecorder(durFromLast time.Duration, videoFolder string) *Recorder {
	return &Recorder{
		durFromLast:    durFromLast,
		inputQueueImgs: make(chan Img),
		srcImgs:        make(chan Img),
		videoFolder:    videoFolder,
	}
}

func (r *Recorder) Start(ctx context.Context) {
	go r.scheduler(ctx)
}

func (r *Recorder) QueueImg(img []byte, hasMotion bool, pts gst.ClockTime) {
	r.inputQueueImgs <- Img{
		Data:      img,
		HasMotion: hasMotion,
		PTS:       pts,
	}
}

func (r *Recorder) scheduler(ctx context.Context) {
	var pipeline *gst.Pipeline
	for img := range r.inputQueueImgs {
		log.Println("received input img", r.running, img.HasMotion, len(img.Data))
		if !r.running && img.HasMotion {
			r.running = true
			r.startedTime = time.Now()
			ctx, cancel := context.WithCancel(ctx)
			r.ctxGst = ctx
			r.cancelGst = cancel
			if pipeline != nil {
				pipeline.SetState(gst.StateNull)
				pipeline = nil
			}
			pipeline = r.pipeSrcToVideo(path.Join(r.videoFolder, time.Now().Format("2006_01_02_15_04_05")))
		}
		if r.running {
			canSend := true
			if img.HasMotion {
				r.startedTime = time.Now()
			} else {
				since := time.Since(r.startedTime)
				log.Println("passed time", since, r.durFromLast)
				if since > r.durFromLast {
					log.Println("end stream")
					r.cancelGst()
					r.running = false
					canSend = false
				}
			}

			if canSend {
				r.srcImgs <- img
			}

		}
	}
}

func (r *Recorder) pipeSrcToVideo(name string) *gst.Pipeline {
	p := fmt.Sprintf(srcToVideo, name)
	log.Println("creating video ", p)

	pipeline, err := gst.NewPipelineFromString(p)
	if err != nil {
		panic(err)
	}

	if err = pipeline.SetState(gst.StatePlaying); err != nil {
		panic(err)
	}

	appsrc, err := pipeline.GetElementByName("appsrc1")
	if err != nil {
		panic(err)
	}
	videoInfo := video.NewInfo().
		WithFormat(video.FormatRGBA, 640, 360).
		WithFPS(gst.Fraction(30, 1))

	src := app.SrcFromElement(appsrc)
	src.SetCaps(videoInfo.ToCaps())
	src.SetCallbacks(&app.SourceCallbacks{
		NeedDataFunc: func(self *app.Source, _ uint) {
			select {
			case <-r.ctxGst.Done():
				log.Println("end of pipeSrcToVideo")
				src.EndStream()
				return
			case img := <-r.srcImgs:
				log.Println("Producing video frame:", len(img.Data))

				buffer := gst.NewBufferWithSize(videoInfo.Size())
				buffer.SetPresentationTimestamp(img.PTS)
				buffer.Map(gst.MapWrite).WriteData(img.Data[:])

				buffer.Unmap()
				flow := src.PushBuffer(buffer)
				if flow == gst.FlowError {
					panic(flow)
				}
			}
		},
	})
	return pipeline
}
