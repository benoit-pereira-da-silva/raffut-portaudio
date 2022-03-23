# raffut-portaudio
Added Raw PortAudio support for [raffut](https://github.com/benoit-pereira-da-silva/raffut)

The portaudio flavour may be more efficient than miniaudio's.
It allows to define the udp buffer size and may provide very low latency support.

# build
1. Install "portaudio".
2.`go build -o raffut cmd/main.go`

Note that any device that uses this flavour should have installed "portaudio"