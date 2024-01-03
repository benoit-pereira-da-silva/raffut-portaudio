# raffut-portaudio
Added Raw PortAudio support for [raffut](https://github.com/benoit-pereira-da-silva/raffut)

The portaudio flavour may be more efficient than miniaudio's.
It allows to define the udp buffer size and may provide very low latency support.

# build
1. Install "portaudio".
2.`go build -o raffut cmd/main.go`

# build for raspberry pi 4,5 

Compiling on mac os is currently not possible ~~`export GOOS=linux && export GOARCH=arm64 && go build -o raffut cmd/main.go`~~

1. Copy the source on the raspberry pi :`ssh bpds@rat1.local mkdir -p /home/bpds/Documents/raffut/ && scp -rv ./*  bpds@rat1.local:/home/bpds/Documents/raffut
2. If necessary: `sudo apt-get install portaudio19-dev`
3.`go build -o raffut cmd/main.go`


1. Launch on the distant host :  `sudo ./raffut receive "192.168.1.4:8383"`
2. Launch on the raspberry host : `sudo ./raffut send 192.168.1.4:8383 IQaudIODAC`

Note that any device that uses this flavour should have installed "portaudio"
