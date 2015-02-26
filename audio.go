package main

import (
	"sync"

	sp "github.com/op/go-libspotify/spotify"
	"github.com/wlcx/ao"
)

var (
	inputBufferSize  = 8
	outputBufferSize = 8192
)

type audio struct {
	format sp.AudioFormat
	frames []byte
}

type audioWriter struct {
	input chan audio
	quit  chan bool
	wg    sync.WaitGroup
}

func AudioInit() {
	ao.Init()
}

func AudioDeinit() {
	ao.Shutdown()
}

func NewAudioWriter() (aw *audioWriter, err error) {
	aw = &audioWriter{
		input: make(chan audio, inputBufferSize),
		quit:  make(chan bool),
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

func compareSpAudioFormat(a, b sp.AudioFormat) bool {
	switch {
	case &a == &b:
		return true
	case a.SampleRate != b.SampleRate:
		return false
	case a.Channels != b.Channels:
		return false
	case a.SampleType != b.SampleType:
		return false
	}
	return true
}

func (w *audioWriter) AOWriter(driver int) {
	var dev *ao.Device
	defer w.wg.Done()
	defer dev.Close()

	var input audio
	var lastSpFormat sp.AudioFormat
	for {
		select {
		case input = <-w.input:
		case <-w.quit:
			return
		}
		if !compareSpAudioFormat(lastSpFormat, input.format) { // If the sample fomat has changed
			if dev != nil {
				dev.Close()
			}
			var matrix string
			if input.format.Channels == 1 {
				matrix = "M"
			} else {
				matrix = "L,R"
			}
			sf := ao.SampleFormat{
				Channels:  input.format.Channels,
				Matrix:    matrix,
				Rate:      input.format.SampleRate,
				Bits:      16,
				ByteOrder: ao.EndianNative,
			}
			lastSpFormat = input.format
			var err error
			dev, err = ao.OpenLive(driver, &sf, nil)
			if err != nil {
				panic(err)
			}
		}
		_, err := dev.Write(input.frames)
		if err != nil {
			dev.Close()
		}
	}
}
