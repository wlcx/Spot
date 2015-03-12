package main

import (
	"sync"
	"time"

	sp "github.com/op/go-libspotify/spotify"
	"github.com/wlcx/ao"
)

var (
	inputBufferSize = 16
)

type audio struct {
	format sp.AudioFormat
	frames []byte
}

type AudioWriter struct {
	input      chan audio
	quit       chan bool
	wg         sync.WaitGroup
	device     *audioDevice
	timeplayed time.Duration
	Ticks      chan time.Duration
	flush      chan bool
	paused     Latch
}

func AudioInit() {
	ao.Init()
}

func AudioDeinit() {
	ao.Shutdown()
}

// Pause allows instantaneous pause and unpause of buffer playback
func (w *AudioWriter) Pause(pause bool) {
	if pause {
		w.paused.Set()
	} else {
		w.paused.Clear()
	}
}

// Flush unpauses and clear the current audio buffer
func (w *AudioWriter) Flush() {
	w.paused.Clear()
	w.flush <- true
}

func NewAudioWriter() (aw *AudioWriter, err error) {
	aw = &AudioWriter{
		input:  make(chan audio, inputBufferSize),
		quit:   make(chan bool),
		device: new(audioDevice),
		Ticks:  make(chan time.Duration),
		flush:  make(chan bool),
	}
	driverid, err := ao.DefaultDriver()
	if err != nil {
		panic(err)
	}
	aw.wg.Add(1)
	go aw.AOWriter(driverid)
	return
}

func (w *AudioWriter) Close() {
	w.Flush() // Flush and unpause first, to ensure the main AOWriter loop is unblocked
	select {
	case w.quit <- true:
	default:
	}
	w.wg.Wait()
	return
}

// The Libspotify callback for audio delivery
func (w *AudioWriter) WriteAudio(format sp.AudioFormat, frames []byte) int {
	select {
	case w.input <- audio{format, frames}:
		return len(frames)
	default:
		return 0
	}
}

// AudioDevice wraps a portaudio device pointer with some state, allowing us to
// handle changes in sample format neatly
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

// Ready the audiodevice for writing, with the given channel/rate configuration.
// Reuses existing device when it can, opens new device when needed
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

func (w *AudioWriter) AOWriter(driverid int) {
	defer w.device.Close()
	defer w.wg.Done()
	for {
		select {
		case <-w.flush:
			// Flush the input buffer (E.G. on song change) by remaking the channel.
			w.input = make(chan audio, inputBufferSize)
		case <-w.quit:
			return
		case input := <-w.input:
			// TODO: refresh the default driverid so we can 'roam' across devices.
			w.device.Ready(input.format.Channels, input.format.SampleRate, driverid)
			bytes, err := w.device.dev.Write(input.frames)
			if err != nil && w.device.dev != nil {
				// Close the device and hope it can be reopened next time round
				w.device.Close()
			} else {
				w.timeplayed += time.Duration(((bytes/input.format.Channels/2)*1000000)/input.format.SampleRate) * time.Microsecond
				if w.timeplayed > time.Duration(1)*time.Second {
					// Nonblocking send - if we can't send, don't reset timeplayed and
					// we'll try next time round.
					select {
					case w.Ticks <- w.timeplayed:
						w.timeplayed = time.Duration(0)
					default:
					}
				}
			}
			w.paused.Wait() // Block here if paused is set
		}
	}
}
