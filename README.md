# Spot
Spot is a minimal, lightweight, cross-platform, standalone, console Spotify client, written in Go. It is *really* not ready for primetime unless primetime involves not being feature complete and crashing.
## Installing
I plan to issue releases for different platforms. Until then, you'll have to compile yourself :poop:

## Compiling
### Requirements
- libao, consult your nearest package manager
- libspotify, available from the hästens mouth [here](https://developer.spotify.com/technologies/libspotify/#libspotify-downloads) or in your package manager or choice (probably)
- A libspotify application key, available [here](https://devaccount.spotify.com/my-account/keys/) for Spotify premium users. You might want the 'c code' option so as to copy and paste the hex byte values (see below)

### Instructions
- `go get github.com/wlcx/spot` (assuming you have a Go dev environment)
- make an `appkey` variable available in the main package, I do so with a file `appkey.go`:
```go
package main

var appkey = []byte{
  0x00, 0x01..... etc
}
```
- `script/install`
- Run `$GOPATH/bin/spot`
- Recieve stack trace up in yo face
- I told you it wasn't finished
