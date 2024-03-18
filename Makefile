build:
	mkdir -p bin
	go build -o bin/1-basic-tutorial  1-basic-tutorial/main.go
	go build -o bin/2-basic-tutorial  2-basic-tutorial/main.go
	go build -o bin/2-basic-exercise  2-basic-tutorial/exercise/main.go
	go build -o bin/3-basic-tutorial  3-basic-tutorial/main.go
	go build -o bin/3-basic-exercise  3-basic-tutorial/exercise/main.go