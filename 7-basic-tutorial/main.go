package main

import (
	"fmt"
	"log"

	"github.com/go-gst/go-glib/glib"
	"github.com/go-gst/go-gst/gst"
)

func runPipeline(mainLoop *glib.MainLoop) error {
	gst.Init(nil)

	// audio_source = gst_element_factory_make ("audiotestsrc", "audio_source");
	audioSource, err := gst.NewElementWithName("audiotestsrc", "audio_source")
	if err != nil {
		log.Fatal(err)
	}
	// tee = gst_element_factory_make ("tee", "tee");
	tee, err := gst.NewElementWithName("tee", "tee")
	if err != nil {
		log.Fatal(err)
	}
	// audio_queue = gst_element_factory_make ("queue", "audio_queue");
	audioQueue, err := gst.NewElementWithName("queue", "audio_queue")
	if err != nil {
		log.Fatal(err)
	}
	// audio_convert = gst_element_factory_make ("audioconvert", "audio_convert");
	audioConvert, err := gst.NewElementWithName("audioconvert", "audio_convert")
	if err != nil {
		log.Fatal(err)
	}
	// audio_resample = gst_element_factory_make ("audioresample", "audio_resample");
	audioResample, err := gst.NewElementWithName("audioresample", "audio_resample")
	if err != nil {
		log.Fatal(err)
	}
	// audio_sink = gst_element_factory_make ("autoaudiosink", "audio_sink");
	audioSink, err := gst.NewElementWithName("autoaudiosink", "audio_sink")
	if err != nil {
		log.Fatal(err)
	}
	// video_queue = gst_element_factory_make ("queue", "video_queue");
	videoQueue, err := gst.NewElementWithName("queue", "video_queue")
	if err != nil {
		log.Fatal(err)
	}
	// visual = gst_element_factory_make ("wavescope", "visual");
	visual, err := gst.NewElementWithName("wavescope", "visual")
	if err != nil {
		log.Fatal(err)
	}
	// video_convert = gst_element_factory_make ("videoconvert", "csp");
	videoConvert, err := gst.NewElementWithName("videoconvert", "csp")
	if err != nil {
		log.Fatal(err)
	}
	// video_sink = gst_element_factory_make ("autovideosink", "video_sink");
	videoSink, err := gst.NewElementWithName("autovideosink", "video_sink")
	if err != nil {
		log.Fatal(err)
	}

	pipeline, err := gst.NewPipeline("test-pipeline")
	if err != nil {
		log.Fatal(err)
	}

	audioSource.Set("freq", 215.0)
	visual.Set("shader", 0)
	visual.Set("style", 1)

	err = pipeline.AddMany(audioSource, tee, audioQueue, audioConvert, audioResample, audioSink,
		videoQueue, visual, videoConvert, videoSink)
	if err != nil {
		log.Fatal(err)
	}

	err = gst.ElementLinkMany(audioSource, tee)
	if err != nil {
		log.Fatal(err)
	}
	err = gst.ElementLinkMany(audioQueue, audioConvert, audioResample, audioSink)
	if err != nil {
		log.Fatal(err)
	}
	err = gst.ElementLinkMany(videoQueue, visual, videoConvert, videoSink)
	if err != nil {
		log.Fatal(err)
	}

	teeAudioPad := tee.GetRequestPad("src_%u")
	log.Printf("Obtained request pad %s for audio branch.\n", teeAudioPad.GetName())

	queueAudioPad := audioQueue.GetStaticPad("sink")
	teeVideoPad := tee.GetRequestPad("src_%u")
	log.Printf("Obtained request pad %s for video branch.\n", teeVideoPad.GetName())
	queueVideoPad := videoQueue.GetStaticPad("sink")

	if teeAudioPad.Link(queueAudioPad) != gst.PadLinkOK {
		log.Fatal("Failed to link audio pads")
	}

	if teeVideoPad.Link(queueVideoPad) != gst.PadLinkOK {
		log.Fatal("Failed to link video pads")
	}

	pipeline.SetState(gst.StatePlaying)

	msg := pipeline.GetBus().TimedPopFiltered(gst.ClockTimeNone, gst.MessageError|gst.MessageEOS)

	tee.ReleaseRequestPad(teeAudioPad)
	tee.ReleaseRequestPad(teeVideoPad)
	pipeline.SetState(gst.StateNull)
	if msg != nil {
		msg.Unref()
	}
	pipeline.Unref()

	return mainLoop.RunError()
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	mainLoop := glib.NewMainLoop(glib.MainContextDefault(), false)

	if err := runPipeline(mainLoop); err != nil {
		fmt.Println("ERROR!", err)
	}
}
