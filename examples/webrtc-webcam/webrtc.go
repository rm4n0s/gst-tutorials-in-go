package main

import (
	"fmt"
	"log"

	"github.com/go-gst/go-gst/gst"
	"github.com/go-gst/go-gst/gst/app"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/rm4n0s/gst-tutorials-in-go/examples/webrtc-webcam/signal"
)

type Webrtc struct {
	opusPipeline *gst.Pipeline
	h264Pipeline *gst.Pipeline
}

func NewWebrtc() *Webrtc {
	return &Webrtc{}
}

func (wr *Webrtc) start(b64Offer string) string {

	audioSrc := "alsasrc ! audioparse ! decodebin ! audioconvert ! audioresample"
	videoSrc := "v4l2src device=/dev/video0 ! video/x-raw,width=640,height=360,framerate=30/1 ! videoconvert"
	var err error
	// Prepare the configuration
	config := webrtc.Configuration{}
	// Create a new RTCPeerConnection
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}

	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("Connection State has changed %s \n", connectionState.String())
		if connectionState.String() == "disconnected" {
			peerConnection.Close()
		}
	})

	// Create a audio track
	audioTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: "audio/opus"}, "audio", "pion1")
	if err != nil {
		panic(err)
	}

	_, err = peerConnection.AddTrack(audioTrack)
	if err != nil {
		panic(err)
	}

	// Create a video track
	videoTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: "video/h264"}, "video", "pion2")
	if err != nil {
		panic(err)
	}

	_, err = peerConnection.AddTrack(videoTrack)
	if err != nil {
		panic(err)
	}

	// Wait for the offer to be pasted

	offer := webrtc.SessionDescription{}
	signal.Decode(b64Offer, &offer)

	// Set the remote SessionDescription
	err = peerConnection.SetRemoteDescription(offer)
	if err != nil {
		panic(err)
	}

	// Create an answer
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		panic(err)
	}

	// Create channel that is blocked until ICE Gathering is complete
	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)

	// Sets the LocalDescription, and starts our UDP listeners
	err = peerConnection.SetLocalDescription(answer)
	if err != nil {
		panic(err)
	}

	<-gatherComplete

	// Start pushing buffers on these tracks
	wr.pipelineForCodec("opus", []*webrtc.TrackLocalStaticSample{audioTrack}, audioSrc)
	wr.pipelineForCodec("h264", []*webrtc.TrackLocalStaticSample{videoTrack}, videoSrc)

	// Output the answer in base64 so we can paste it in browser
	return signal.Encode(*peerConnection.LocalDescription())
}

// Create the appropriate GStreamer pipeline depending on what codec we are working with
func (wr *Webrtc) pipelineForCodec(codecName string, tracks []*webrtc.TrackLocalStaticSample, pipelineSrc string) {
	var err error
	var pipeline *gst.Pipeline
	pipelineStr := "appsink name=appsink"
	switch codecName {
	case "h264":
		pipelineStr = pipelineSrc + " ! video/x-raw,format=I420 ! x264enc speed-preset=ultrafast tune=zerolatency key-int-max=20 ! video/x-h264,stream-format=byte-stream ! " + pipelineStr
		pipeline, err = gst.NewPipelineFromString(pipelineStr)
		if err != nil {
			panic(err)
		}
		wr.h264Pipeline = pipeline

	case "opus":
		pipelineStr = pipelineSrc + " ! opusenc ! " + pipelineStr
		pipeline, err = gst.NewPipelineFromString(pipelineStr)
		if err != nil {
			panic(err)
		}
		wr.opusPipeline = pipeline

	default:
		panic("Unhandled codec " + codecName) //nolint
	}

	if err = pipeline.SetState(gst.StatePlaying); err != nil {
		panic(err)
	}

	appSink, err := pipeline.GetElementByName("appsink")
	if err != nil {
		panic(err)
	}

	sink := app.SinkFromElement(appSink)
	sink.SetCallbacks(&app.SinkCallbacks{
		NewSampleFunc: func(sink *app.Sink) gst.FlowReturn {
			sample := sink.PullSample()
			if sample == nil {
				return gst.FlowEOS
			}

			buffer := sample.GetBuffer()
			if buffer == nil {
				return gst.FlowError
			}

			samples := buffer.Map(gst.MapRead).Bytes()
			defer buffer.Unmap()

			for _, t := range tracks {
				if err := t.WriteSample(media.Sample{Data: samples, Duration: *buffer.Duration().AsDuration()}); err != nil {
					panic(err) //nolint
				}
			}

			return gst.FlowOK
		},
	})
}

func (wr *Webrtc) stop() {
	log.Println("stop")
	if wr.h264Pipeline != nil {
		wr.h264Pipeline.SendEvent(gst.NewEOSEvent())
		wr.h264Pipeline.SetState(gst.StateNull)
		wr.h264Pipeline.Clear()
	}

	if wr.opusPipeline != nil {
		wr.opusPipeline.SendEvent(gst.NewEOSEvent())
		wr.opusPipeline.SetState(gst.StateNull)
		wr.opusPipeline.Clear()
	}

}
