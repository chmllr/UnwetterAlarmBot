run:
	PORT=8080 go run main.go

test:
	http-server -p 7070 . &

