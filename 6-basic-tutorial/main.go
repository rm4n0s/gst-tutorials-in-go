package main

// The tutorial is incomplete because of missing function that exist in gstreamer but do not exist in go-gst like:
//  - gst_structure_foreach
//	- gst_element_factory_get_num_pad_templates
//  - gst_element_factory_get_static_pad_templates
//  -
import (
	"fmt"
	"log"

	"github.com/go-gst/go-glib/glib"
	"github.com/go-gst/go-gst/gst"
)

func printPadTemplatesInformation(element *gst.Element) {
	fmt.Printf("Pad Templates for %s:\n", element.GetFactory().GetName())
	for _, v := range element.GetPadTemplates() {
		if v.Direction() == gst.PadDirectionSource {
			fmt.Printf("  SRC template: '%s'\n", v.Name())
		} else if v.Direction() == gst.PadDirectionSink {
			fmt.Printf("  SINK template: '%s'\n", v.Name())
		} else {
			fmt.Printf("  UNKNOWN!!! template: '%s'\n", v.Name())
		}

		if v.Presence() == gst.PadPresenceAlways {
			fmt.Printf("    Availability: Always\n")
		} else if v.Presence() == gst.PadPresenceSometimes {
			fmt.Printf("    Availability: Sometimes\n")
		} else if v.Presence() == gst.PadPresenceRequest {
			fmt.Printf("    Availability: On request\n")
		} else {
			fmt.Printf("    Availability: UNKNOWN!!!\n")
		}

		if v.Caps() != nil {
			fmt.Printf("    Capabilities:\n")
			printCaps(v.Caps(), "      ")
		}
	}
}

func printField(value *glib.Value) {
	str := gst.ValueSerialize(value)
	fmt.Println(str)
}

func printCaps(caps *gst.Caps, pfx string) {
	if caps == nil {
		return
	}

	if caps.IsAny() {
		fmt.Printf("%sANY\n", pfx)
		return
	}

	if caps.IsEmpty() {
		fmt.Printf("%sEMPTY\n", pfx)
		return
	}

	for i := 0; i < caps.GetSize(); i++ {
		gstruct := caps.GetStructureAt(i)
		fmt.Printf("%s%s\n", pfx, gstruct.Name())
		gval, _ := gstruct.ToGValue()
		printField(gval)
	}
}

func printPadCapabilities(element *gst.Element, padName string) {
	pad := element.GetStaticPad(padName)
	caps := pad.GetCurrentCaps()
	fmt.Printf("Caps for the %s pad:\n", padName)
	printCaps(caps, "      ")
	// caps.Unref()
	// pad.Unref()
}

func runPipeline(mainLoop *glib.MainLoop) error {
	gst.Init(nil)
	source, err := gst.NewElement("audiotestsrc")
	if err != nil {
		log.Fatal(err)
	}
	sink, err := gst.NewElement("autoaudiosink")
	if err != nil {
		log.Fatal(err)
	}

	printPadTemplatesInformation(source)
	printPadTemplatesInformation(sink)

	pipeline, err := gst.NewPipeline("test-pipeline")
	if err != nil {
		log.Fatal(err)
	}
	pipeline.AddMany(source, sink)
	gst.ElementLinkMany(source, sink)

	fmt.Print("In NULL state:\n")
	printPadCapabilities(sink, "sink")
	pipeline.GetBus().AddWatch(func(msg *gst.Message) bool {

		switch msg.Type() {
		case gst.MessageEOS:
			log.Println(msg.String())
			mainLoop.Quit()
		case gst.MessageError:
			err := msg.ParseError()
			log.Println("Error:", err.Error())
			debug := err.DebugString()
			if len(debug) > 0 {
				log.Println("Debug: ", debug)
			}

		case gst.MessageStateChanged:
			if msg.Source() == pipeline.GetName() {
				oldState, newState := msg.ParseStateChanged()
				log.Printf("Pipeline state changed from %s to %s:\n", oldState.String(), newState.String())
			}
		default:
			//log.Println("Unexpected message received.", msg.String())
		}
		return true
	})
	pipeline.SetState(gst.StatePlaying)

	return mainLoop.RunError()
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	mainLoop := glib.NewMainLoop(glib.MainContextDefault(), false)

	if err := runPipeline(mainLoop); err != nil {
		fmt.Println("ERROR!", err)
	}
}
