package portaudio

import (
	"fmt"
	"github.com/benoit-pereira-da-silva/raffut/console"
	"github.com/benoit-pereira-da-silva/raffut/streams"
	"github.com/gordonklaus/portaudio"
	"io"
	"math"
	"strings"
)

// PortAudio Streamable support.
// Source: [Portaudio](http://www.portaudio.com)
// "PortAudio is a free, cross-platform, open-source, audio I/O library.
// It lets you write simple audio programs in 'C' or C++ that will compile and run on many platforms including Windows, Macintosh OS X, and Unix (OSS/ALSA).
// It is intended to promote the exchange of audio software between developers on different platforms.
// Many applications use PortAudio for Audio I/O."
type PortAudio struct {
	streams.Streamable
	address    string
	ChunkSize  int
	channels   int
	sampleRate float64
	echo       bool
	done       chan interface{}

	// Advanced configuration this device name would be used to open the device.
	DeviceName string // e.g : "IQaudIODAC"
}

// Infos returns the streamer infos as a string.
func (p *PortAudio) Infos() string {

	s := strings.Builder{}
	err := portaudio.Initialize()
	defer portaudio.Terminate()

	if err != nil {
		s.WriteString(fmt.Sprintf("PortAudio error on Initialization: %s\n", err.Error()))
		return s.String()
	}
	s.WriteString("-- PortAudio initialized -- \n")
	s.WriteString(fmt.Sprintf("Streamer.address: %s\n", p.address))
	s.WriteString(fmt.Sprintf("Streamer.ChunkSize: %d\n", p.ChunkSize))
	s.WriteString(fmt.Sprintf("Streamer. Channels: %d\n", p.channels))
	s.WriteString(fmt.Sprintf("Streamer.SampleRate: %f\n", p.sampleRate))
	s.WriteString(fmt.Sprintf("Streamer.Echo: %t\n", p.echo))

	s.WriteString(fmt.Sprintf("portaudio.VersionText: %s\n", portaudio.VersionText()))

	ha, err := portaudio.DefaultHostApi()
	if err != nil {
		s.WriteString(fmt.Sprintf("DefaultHostApi: %s\n", err.Error()))
	} else {
		s.WriteString(fmt.Sprintf("DefaultHostApi.Name: %s\n", ha.Name))
		if ha.DefaultInputDevice == nil {
			s.WriteString(fmt.Sprintf("DefaultHostApi.DefaultOutputDevice: nil\n"))
		} else {
			s.WriteString(fmt.Sprintf("DefaultHostApi.DefaultOutputDevice: %s\n", ha.DefaultOutputDevice.Name))
		}
		if ha.DefaultInputDevice == nil {
			s.WriteString(fmt.Sprintf("DefaultHostApi.DefaultInputDevice: nil\n"))
		} else {
			s.WriteString(fmt.Sprintf("DefaultHostApi.DefaultInputDevice: %s\n", ha.DefaultInputDevice.Name))
		}
	}

	devices, errDev := portaudio.Devices()
	if errDev != nil {
		s.WriteString(fmt.Sprintf("Devices: %s\n", errDev.Error()))
	} else {
		for idx, dev := range devices {
			s.WriteString(fmt.Sprintf("-- Device %03d -- \n", idx))
			s.WriteString(fmt.Sprintf("Device.Name: %s\n", dev.Name))
			s.WriteString(fmt.Sprintf("Device.DefaultSampleRate: %v\n", dev.DefaultSampleRate))
			s.WriteString(fmt.Sprintf("Device.MaxInputChannels: %d\n", dev.MaxInputChannels))
			s.WriteString(fmt.Sprintf("Device.MaxOutputChannels: %d\n", dev.MaxOutputChannels))
			s.WriteString(fmt.Sprintf("Device.DefaultLowInputLatency: %v\n", dev.DefaultLowInputLatency))
			s.WriteString(fmt.Sprintf("Device.DefaultLowOutputLatency: %v\n", dev.DefaultLowOutputLatency))
			s.WriteString(fmt.Sprintf("Device.DefaultHighInputLatency: %v\n", dev.DefaultHighInputLatency))
			s.WriteString(fmt.Sprintf("Device.DefaultHighOutputLatency: %v\n", dev.DefaultHighOutputLatency))
			s.WriteString(fmt.Sprintf("Device.HostApi: %s\n", dev.HostApi.Name))
		}
	}
	return s.String()
}

// ReadStreamFrom  reads the stream from the given reader.
func (p *PortAudio) ReadStreamFrom(c io.Reader) error {
	portaudio.Initialize()
	defer portaudio.Terminate()
	bs := make([]byte, p.ChunkSize*4*p.channels)
	floatBuffer := make([]float32, len(bs)/4)
	stream, err := p.openStream(true, func(out []float32) {
		_, err := c.Read(bs)
		if err != nil {
			println(err.Error())
			<-p.done
		} else {
			err = bigEndianBytesToFloat32(bs, &floatBuffer)
			if err != nil {
				println(err.Error())
				<-p.done
			} else {
				sum := float32(0)
				for i := range out {
					v := floatBuffer[i]
					out[i] = v
					sum += v
				}
				if p.echo {
					console.PrintFrame(sum)
				}
			}
		}
	})
	if err != nil {
		return err
	}
	sErr := stream.Start()
	if sErr != nil {
		return sErr
	}
	defer stream.Close()
	for {
		select {
		case <-p.done:
			return stream.Stop()
		}
	}
}

// WriteStreamTo writes the stream to the given writer.
func (p *PortAudio) WriteStreamTo(c io.Writer) error {
	buffer := make([]float32, p.ChunkSize*p.channels)
	byteBuffer := make([]byte, len(buffer)*4)
	portaudio.Initialize()
	defer portaudio.Terminate()
	stream, err := p.openStream(false, func(in []float32) {
		sum := float32(0)
		for i := range buffer {
			v := in[i]
			buffer[i] = v
			sum += v
		}
		err := bigEndianFloat32ToBytes(buffer, &byteBuffer)
		if err != nil {
			println(err.Error())
			<-p.done
		} else {
			_, err = c.Write(byteBuffer)
			if err != nil {
				// UDP has no real connection and no Acknowledgement on any packet transmission.
				// If there is no receiver c.Write you get a "connection refused"
				// This is not always the case.
				println("ERROR:", err.Error())
			} else {
				if p.echo {
					console.PrintFrame(sum)
				}
			}
		}

	})
	if err != nil {
		return err
	}
	sErr := stream.Start()
	if sErr != nil {
		return sErr
	}
	defer stream.Close()
	for {
		select {
		case <-p.done:
			return stream.Stop()
		}
	}
}

// Configure the streamer
func (p *PortAudio) Configure(address string, sampleRate float64, nbChannels int, echo bool, done chan interface{}) {
	p.address = address
	p.channels = nbChannels
	p.sampleRate = sampleRate
	p.echo = echo
	p.done = done
}

// Address correspond to the <IP or Name:PORT>
func (p *PortAudio) Address() string {
	return p.address
}

// SampleRate is the sample rate :)
func (p *PortAudio) SampleRate() float64 {
	return p.sampleRate
}

func (p *PortAudio) NbChannels() int {
	return p.channels
}

// Echo if responding true prints the flow in the stdio
func (p *PortAudio) Echo() bool {
	return p.echo
}

// Done is the cancellation channel
func (p *PortAudio) Done() chan interface{} {
	return p.done
}

// openStream opens a PortAudio stream with a prefix matching the device name and specifies the input or output configuration based on the 'in' parameter.
// The prefix parameter is used to match the device name with devices obtained from 'portaudio.Devices()'.
// If a matching device is found, the stream is opened with the specified input or output parameters (sampleRate and nbChannels).
// If 'in' is true, the stream is opened for input. Otherwise, the stream is opened for output.
// The returned portaudio.Stream object can be used to read or write audio data.
func (p *PortAudio) openStream(in bool, f func([]float32)) (*portaudio.Stream, error) {
	if p.DeviceName == "" {
		// default device
		return portaudio.OpenDefaultStream(0, p.channels, p.sampleRate, p.ChunkSize, f)
	}
	devices, err := portaudio.Devices()
	if err != nil {
		return nil, err
	}
	var deviceIndex int = -1
	for i, device := range devices {
		if strings.HasPrefix(device.Name, p.DeviceName) {
			deviceIndex = i
			break
		}
	}
	if deviceIndex == -1 {
		return nil, fmt.Errorf("device not found")
	} else {
		di := devices[deviceIndex]
		if di == nil {
			return nil, fmt.Errorf("device is not available")
		}
		if in {
			sp := portaudio.HighLatencyParameters(di, nil)
			sp.Input.Channels = p.NbChannels()
			sp.SampleRate = p.SampleRate()
			sp.FramesPerBuffer = p.ChunkSize
			return portaudio.OpenStream(sp, f)
		} else {
			sp := portaudio.HighLatencyParameters(nil, di)
			sp.Output.Channels = p.NbChannels()
			sp.SampleRate = p.SampleRate()
			sp.FramesPerBuffer = p.ChunkSize
			return portaudio.OpenStream(sp, f)
		}
	}
}

// bigEndianFloat32ToBytes should be faster than binary.Write(c, binary.BigEndian, &buffer)
// It does not rely on reflexion.
// when dealing with sound faster is always better.
func bigEndianFloat32ToBytes(data []float32, result *[]byte) error {
	if len(data) != len(*result)/4 {
		return fmt.Errorf("length missmatch in bigEndianFloat32ToBytes []float32 len should be equal to []byte len / 4")
	}
	for i, x := range data {
		v := math.Float32bits(x)
		(*result)[4*i] = byte(v >> 24)
		(*result)[4*i+1] = byte(v >> 16)
		(*result)[4*i+2] = byte(v >> 8)
		(*result)[4*i+3] = byte(v)
	}
	return nil
}

// bigEndianBytesToFloat32 should be faster than binary.Read(c, binary.BigEndian, &buffer)
// It does not rely on reflexion.
func bigEndianBytesToFloat32(data []byte, result *[]float32) error {
	if len(data)/4 != len(*result) {
		return fmt.Errorf("length missmatch in bigEndianBytesToFloat32 []float32 len should be equal to []byte len / 4")
	}
	for i, _ := range *result {
		v := uint32(data[4*i+3]) | uint32(data[4*i+2])<<8 | uint32(data[4*i+1])<<16 | uint32(data[4*i])<<24
		(*result)[i] = math.Float32frombits(v)
	}
	return nil
}
