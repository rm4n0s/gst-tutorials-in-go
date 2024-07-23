build:
	mkdir -p bin
	go build -o bin/1-basic-tutorial  1-basic-tutorial/main.go
	go build -o bin/2-basic-tutorial  2-basic-tutorial/main.go
	go build -o bin/2-basic-exercise  2-basic-tutorial/exercise/main.go
	go build -o bin/3-basic-tutorial  3-basic-tutorial/main.go
	go build -o bin/3-basic-exercise  3-basic-tutorial/exercise/main.go
	go build -o bin/4-basic-tutorial  4-basic-tutorial/main.go
	go build -o bin/6-basic-tutorial  6-basic-tutorial/main.go
	go build -o bin/7-basic-tutorial  7-basic-tutorial/main.go

	go build -o bin/fyne-webcam  examples/fyne-webcam/main.go
	go build -o bin/webrtc-webcam  examples/webrtc-webcam/*.go
	cp -r examples/webrtc-webcam/static bin/static

	go build -o bin/record-motion-detections examples/record-motion-detections/*.go

ex-webrtc:
	./bin/webrtc-webcam

ex-recorder:
	mkdir -p videos
	./bin/record-motion-detections -path ./videos