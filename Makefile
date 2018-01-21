run:
	PORT=8080 go run main.go

test:
	http-server -p 7070 . &

page1:
	make test
	TEST=1 PAGE=test_page.html PORT=8080 go run main.go

page2:
	make test
	TEST=1 PAGE=test_page2.html PORT=8080 go run main.go
