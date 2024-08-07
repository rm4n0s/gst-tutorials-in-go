# gst-tutorials-in-go
Code from https://gstreamer.freedesktop.org/documentation/tutorials in Golang and some examples.

## Dependencies
I work on fedor, so here are the dependencies that I used
```
dnf install gstreamer1-devel gstreamer1-plugins-base-tools gstreamer1-doc gstreamer1-plugins-base-devel gstreamer1-plugins-good gstreamer1-plugins-good-extras gstreamer1-plugins-ugly gstreamer1-plugins-bad-free gstreamer1-plugins-bad-free-devel gstreamer1-plugins-bad-free-extras
```

Also I installed opencv from homebrew
```
brew install opencv
```

## Build
```bash
make build
```


## Run
The 'make build' will create a 'bin/' folder with all the binaries from each tutorial and example <br/>

```bash
# it will create ./videos folder to record each detected motion in each own video
make ex-record-motion

# start a GUI showing the webcam's feed
make ex-fyne-webcam

# record desktop to a video file in ./videos/test.mp4
make ex-record-desktop

# start a self hosted server to connect the camera from the server to Web UI through webrtc
make ex-webrtc
```
