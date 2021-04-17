.PHONY: tests
tests:
	rm -f coverage.html coverage.txt
	go test -v -coverprofile=coverage.txt .
	go tool cover -html=coverage.txt -o coverage.html
