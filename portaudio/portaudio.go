package portaudio

import (
	"fmt"
	"github.com/benoit-pereira-da-silva/raffut/console"
	"github.com/benoit-pereira-da-silva/raffut/streams"
	"github.com/gordonklaus/portaudio"
	"io"
	"log"
	"math"
	"os"
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
}

func (p *PortAudio) ReadStreamFrom(c io.Reader) error {
	portaudio.Initialize()
	defer portaudio.Terminate()
	bs := make([]byte, p.ChunkSize*4)
	floatBuffer := make([]float32, len(bs)/4)
	stream, err := portaudio.OpenDefaultStream(0, p.channels, p.sampleRate, p.ChunkSize, func(out []float32) {
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
	chk(err)
	chk(stream.Start())
	defer stream.Close()
	for {
		select {
		case <-p.done:
			chk(stream.Stop())
			return nil
		}
	}
}

func (p *PortAudio) WriteStreamTo(c io.Writer) error {
	buffer := make([]float32, p.ChunkSize)
	byteBuffer := make([]byte, len(buffer)*4)
	portaudio.Initialize()
	defer portaudio.Terminate()
	stream, err := portaudio.OpenDefaultStream(p.channels, 0, p.sampleRate, p.ChunkSize, func(in []float32) {
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
				// After one write there is always an error
				// Explanation: https://stackoverflow.com/questions/46697799/golang-udp-connection-refused-on-every-other-write
				// " Because UDP has no real connection and there is no ACK for any packets sent,
				// the best a "connected" UDP socket can do to simulate a send failure is to save the ICMP response,
				// and return it as an error on the next write."
			} else {
				if p.echo {
					console.PrintFrame(sum)
				}
			}
		}

	})
	chk(err)
	chk(stream.Start())
	defer stream.Close()
	for {
		select {
		case <-p.done:
			chk(stream.Stop())
			return nil
		}
	}
}

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

// Echo if responding true prints the flow in the stdio
func (p *PortAudio) Echo() bool {
	return p.echo
}

// Done is the cancellation channel
func (p *PortAudio) Done() chan interface{} {
	return p.done
}

func chk(err error) {
	if err != nil {
		log.Println(err)
		os.Exit(1)
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
