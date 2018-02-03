run:
	PORT=8080 go run main.go

test:
	go test ./...

fake:
	http-server -p 7070 . &

