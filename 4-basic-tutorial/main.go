package main

import (
	"fmt"
	"log"
	"time"

	"github.com/go-gst/go-glib/glib"
	"github.com/go-gst/go-gst/gst"
)

type CustomData struct {
	playbin     *gst.Element
	playing     bool
	terminate   bool          /* Should we terminate execution? */
	seekEnabled bool          /* Is seeking enabled for this media? */
	seekDone    bool          /* Have we performed the seek already? */
	duration    gst.ClockTime /* How long does this media last, in nanoseconds */
}

func handleMessage(msg *gst.Message, data *CustomData) {
	switch msg.Type() {
	case gst.MessageError:
		err := msg.ParseError()
		log.Printf("Error received from element %s: %s\n", msg.Source(), err.Error())
		debug := err.DebugString()
		if len(debug) > 0 {
			log.Printf("Debugging information: %s\n", debug)
		}
		data.terminate = true
	case gst.MessageEOS:
		log.Println("End-Of-Stream reached.")
		data.terminate = true
	case gst.MessageDurationChanged:
		data.duration = gst.ClockTimeNone
	case gst.MessageStateChanged:
		if msg.Source() == data.playbin.GetName() {
			oldState, newState := msg.ParseStateChanged()
			log.Printf("Pipeline state changed from %s to %s:\n", oldState.String(), newState.String())
			data.playing = newState == gst.StatePlaying
			/* Remember whether we are in the PLAYING state or not */
			if data.playing {
				query := gst.NewSeekingQuery(gst.FormatTime)
				if data.playbin.Query(query) {
					/* We just moved to PLAYING. Check if seeking is possible */
					_, seekable, start, end := query.ParseSeeking()
					data.seekEnabled = seekable
					if data.seekEnabled {
						log.Printf("Seeking is ENABLED from %d to %d\n",
							start, end)
					} else {
						log.Println("Seeking is DISABLED for this stream.")
					}
				} else {
					log.Println("Seeking query failed.")
				}
			}

		}
	default:
		log.Println("Unexpected message received.")
	}
}

func runPipeline(mainLoop *glib.MainLoop) error {
	gst.Init(nil)
	var err error
	data := &CustomData{}
	data.duration = gst.ClockTimeNone
	data.playbin, err = gst.NewElementWithName("playbin", "playbin")
	if err != nil {
		log.Fatal(err)
	}
	data.playbin.Set("uri", "https://gstreamer.freedesktop.org/data/media/sintel_trailer-480p.webm")
	data.playbin.SetState(gst.StatePlaying)

	bus := data.playbin.GetBus()
	for !data.terminate {
		msg := bus.TimedPopFiltered(gst.ClockTime(100*time.Millisecond), gst.MessageStateChanged|gst.MessageError|gst.MessageEOS|gst.MessageDurationChanged)
		if msg != nil {
			handleMessage(msg, data)
		} else {
			/* We got no message, this means the timeout expired */
			if data.playing {
				/* Query the current position of the stream */
				ok, current := data.playbin.QueryPosition(gst.FormatTime)
				if !ok {
					log.Fatal("Error: Could not query current position.\n")
				}

				/* If we didn't know it yet, query the stream duration */
				if gst.ClockTime(current) != gst.ClockTimeNone {
					ok, duration := data.playbin.QueryDuration(gst.FormatTime)
					if !ok {
						log.Fatal("Could not query current duration.")
					}
					data.duration = gst.ClockTime(duration)
				}

				/* Print current position and total duration */
				//log.Printf("Position %d / %d \n", current, data.duration)
				curDur, err := time.ParseDuration(fmt.Sprintf("%dns", current))
				if err != nil {
					log.Fatal(err)
				}
				dur, err := time.ParseDuration(fmt.Sprintf("%dns", data.duration))
				if err != nil {
					log.Fatal(err)
				}
				log.Printf("%f:%f:%f %f:%f:%f\n", curDur.Hours(), curDur.Minutes(), curDur.Seconds(), dur.Hours(), dur.Minutes(), dur.Seconds())
				if data.seekEnabled && data.seekDone && time.Duration(current) > 10*time.Second {
					log.Println("Reached 10s, performing seek...")
					data.playbin.SeekSimple(int64(30*time.Second), gst.FormatTime, gst.SeekFlagFlush|gst.SeekFlagKeyUnit)
					data.seekDone = true
				}
			}
		}
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
