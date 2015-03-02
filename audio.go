package main

import (
	"sync"

	sp "github.com/op/go-libspotify/spotify"
	"github.com/wlcx/ao"
)

var (
	inputBufferSize = 8
)

type audio struct {
	format sp.AudioFormat
	frames []byte
}

type audioWriter struct {
	input  chan audio
	quit   chan bool
	wg     sync.WaitGroup
	device *audioDevice
}

func AudioInit() {
	ao.Init()
}

func AudioDeinit() {
	ao.Shutdown()
}

func NewAudioWriter() (aw *audioWriter, err error) {
	aw = &audioWriter{
		input:  make(chan audio, inputBufferSize),
		quit:   make(chan bool),
		device: new(audioDevice),
	}
	driverid, err := ao.DefaultDriver()
	if err != nil {
		panic(err)
	}
	aw.wg.Add(1)
	go aw.AOWriter(driverid)
	return
}

func (w *audioWriter) Close() {
	select {
	case w.quit <- true:
	default:
	}
	w.wg.Wait()
	return
}

func (w *audioWriter) WriteAudio(format sp.AudioFormat, frames []byte) int {
	select {
	case w.input <- audio{format, frames}:
		return len(frames)
	default:
		return 0
	}
}

// AudioDevice wraps a portaudio device pointer with some state, allowing us to
// handle changes in sample format closed devices neatly
type audioDevice struct {
	dev      *ao.Device
	channels int
	rate     int
}

// Construct a libao sampleformat struct suitable for libspotify use
// given channels and sample rate, the two things that can change
// (libspotify uses 16bit native endianness. I think.)
func getSampleFormat(channels, rate int) *ao.SampleFormat {
	var matrix string
	if channels == 1 {
		matrix = "M"
	} else {
		matrix = "L,R"
	}
	return &ao.SampleFormat{
		Channels:  channels,
		Matrix:    matrix,
		Rate:      rate,
		Bits:      16,
		ByteOrder: ao.EndianNative,
	}
}

//Ready the audiodevice for writing, with the given channel/rate configuration.
//Reuses existing device when it can, opens new device when needed
func (a *audioDevice) Ready(channels, rate, driver int) (err error) {
	if a.dev == nil || a.channels != channels || a.rate != rate {
		if a.dev != nil {
			//We have an open device; it just needs reconfiguring
			a.dev.Close()
		}
		a.dev, err = ao.OpenLive(driver, getSampleFormat(channels, rate), nil)
		if err != nil {
			panic(err)
		}
		a.channels = channels
		a.rate = rate
	}
	return
}

func (a *audioDevice) Close() {
	a.dev.Close()
	a.dev = nil
}

func (w *audioWriter) AOWriter(driverid int) {
	defer w.wg.Done()

	var input audio
	for {
		select {
		case input = <-w.input:
		case <-w.quit:
			return
		}
		w.device.Ready(input.format.Channels, input.format.SampleRate, driverid)
		_, err := w.device.dev.Write(input.frames)
		if err != nil {
			// Should probably close the device
			w.device.Close()
		}
	}
}
