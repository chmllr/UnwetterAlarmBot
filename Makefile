test:
	http-server -p 7070 . &

page1:
	make test
	TEST=1 PAGE=test_page1.html go run main.go

page2:
	make test
	TEST=1 PAGE=test_page2.html go run main.go
